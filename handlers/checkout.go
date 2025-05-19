package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"goapi/config"
	"goapi/models"
	
	"github.com/gin-gonic/gin"
)

// Checkout converts a cart to an order and initiates payment
func Checkout(c *gin.Context) {
    // Parse the request
    var input struct {
        ShippingAddressID *int `json:"shipping_address_id"`
        ShippingAddress   *models.ShippingAddressInput `json:"shipping_address"`
    }
    
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Get user ID from context
    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
        return
    }
    
    // Begin transaction
    tx, err := config.DB.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
        return
    }
    defer tx.Rollback() // Will be ignored if transaction is committed
    
    // Get shipping address information
    var shippingAddressJSON string
    
    if input.ShippingAddressID != nil {
        // Verify the address exists and belongs to the user
        var address models.ShippingAddress
        var addressLine2 sql.NullString
        
        err := tx.QueryRow(`
            SELECT id, recipient_name, phone, address_line1, address_line2, city, state, postal_code, country 
            FROM shipping_addresses 
            WHERE id = ? AND user_id = ?`, 
            *input.ShippingAddressID, userID).Scan(
                &address.ID,
                &address.RecipientName,
                &address.Phone,
                &address.AddressLine1,
                &addressLine2,
                &address.City,
                &address.State,
                &address.PostalCode,
                &address.Country,
            )
        
        if err != nil {
            if err == sql.ErrNoRows {
                c.JSON(http.StatusBadRequest, gin.H{"error": "shipping address not found"})
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            }
            return
        }
        
        if addressLine2.Valid {
            address.AddressLine2 = addressLine2.String
        }
        
        // Convert address to JSON
        addressBytes, err := json.Marshal(address)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process shipping address"})
            return
        }
        
        shippingAddressJSON = string(addressBytes)
    } else if input.ShippingAddress != nil {
        // Use the provided address
        addressBytes, err := json.Marshal(input.ShippingAddress)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process shipping address"})
            return
        }
        
        shippingAddressJSON = string(addressBytes)
        
        // Optionally save this address to the user's saved addresses
        if input.ShippingAddress.IsDefault {
            // If this will be the default address, unset any existing default
            _, err = tx.Exec(
                "UPDATE shipping_addresses SET is_default = 0 WHERE user_id = ?", 
                userID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update default address"})
                return
            }
        }
        
        // Insert new address
        _, err = tx.Exec(`
            INSERT INTO shipping_addresses (
                user_id, recipient_name, phone, address_line1, address_line2, 
                city, state, postal_code, country, is_default
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
            userID, input.ShippingAddress.RecipientName, input.ShippingAddress.Phone, 
            input.ShippingAddress.AddressLine1, input.ShippingAddress.AddressLine2,
            input.ShippingAddress.City, input.ShippingAddress.State, 
            input.ShippingAddress.PostalCode, input.ShippingAddress.Country, 
            input.ShippingAddress.IsDefault,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save shipping address"})
            return
        }
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "shipping address information is required"})
        return
    }
    
    // Get cart and verify it has items
    var cartID int
    err = tx.QueryRow("SELECT id FROM carts WHERE user_id = ?", userID).Scan(&cartID)
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusBadRequest, gin.H{"error": "no active cart found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    
    // Check if cart has items
    var itemCount int
    err = tx.QueryRow("SELECT COUNT(*) FROM cart_items WHERE cart_id = ?", cartID).Scan(&itemCount)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    
    if itemCount == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "cart is empty"})
        return
    }
    
    // Get user info for payment
    var user models.User
    err = tx.QueryRow("SELECT username, email FROM users WHERE id = ?", userID).Scan(
        &user.Username, &user.Email,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user information"})
        return
    }
    
    // Get cart items and calculate total
    rows, err := tx.Query(`
        SELECT ci.product_id, ci.quantity, p.name, p.price, p.stock
        FROM cart_items ci 
        JOIN products p ON ci.product_id = p.id 
        WHERE ci.cart_id = ?`, cartID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cart items"})
        return
    }
    defer rows.Close()
    
    var cartItems []struct {
        ProductID   int     `json:"product_id"`
        Quantity    int     `json:"quantity"`
        Name        string  `json:"name"`
        Price       float64 `json:"price"`
        CurrentStock int    `json:"current_stock"`
    }
    
    var totalAmount float64
    var orderDescription strings.Builder
    
    for rows.Next() {
        var item struct {
            ProductID   int     `json:"product_id"`
            Quantity    int     `json:"quantity"`
            Name        string  `json:"name"`
            Price       float64 `json:"price"`
            CurrentStock int    `json:"current_stock"`
        }
        
        err := rows.Scan(
            &item.ProductID,
            &item.Quantity,
            &item.Name,
            &item.Price,
            &item.CurrentStock,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process cart items"})
            return
        }
        
        // Check stock availability again
        if item.Quantity > item.CurrentStock {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": fmt.Sprintf("Not enough stock for %s. Available: %d, Requested: %d", 
                    item.Name, item.CurrentStock, item.Quantity),
            })
            return
        }
        
        cartItems = append(cartItems, item)
        itemTotal := float64(item.Quantity) * item.Price
        totalAmount += itemTotal
        
        if orderDescription.Len() > 0 {
            orderDescription.WriteString(", ")
        }
        orderDescription.WriteString(fmt.Sprintf("%s x%d", item.Name, item.Quantity))
    }
    
    // Generate unique order ID
    orderID := fmt.Sprintf("ORD-%d-%d", userID, time.Now().Unix())
    
    // Insert order into database with shipping address
    orderResult, err := tx.Exec(`
        INSERT INTO orders (
            order_id, user_id, total_amount, status, shipping_address, created_at, updated_at
        ) VALUES (?, ?, ?, 'pending', ?, NOW(), NOW())`,
        orderID, userID, totalAmount, shippingAddressJSON)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
        return
    }
    
    dbOrderID, err := orderResult.LastInsertId()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order ID"})
        return
    }
    
    // Insert order items
    for _, item := range cartItems {
        _, err = tx.Exec(`
            INSERT INTO order_items (order_id, product_id, quantity, price)
            VALUES (?, ?, ?, ?)`,
            dbOrderID, item.ProductID, item.Quantity, item.Price)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order items"})
            return
        }
        
        // Update product stock
        _, err = tx.Exec(
            "UPDATE products SET stock = stock - ? WHERE id = ?",
            item.Quantity, item.ProductID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product stock"})
            return
        }
    }
    
    // Clear the cart
    _, err = tx.Exec("DELETE FROM cart_items WHERE cart_id = ?", cartID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear cart"})
        return
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    
    // Extract shipping address details for payment request
    var shippingInfo struct {
        RecipientName string `json:"recipient_name"`
        AddressLine1  string `json:"address_line1"`
        City          string `json:"city"`
        PostalCode    string `json:"postal_code"`
        Phone         string `json:"phone"`
    }
    
    err = json.Unmarshal([]byte(shippingAddressJSON), &shippingInfo)
    if err != nil {
        // Don't fail the order, just use defaults
        shippingInfo.RecipientName = user.Username
    }
    
    // Create payment request for the Java payment service
    paymentRequest := map[string]interface{}{
        "firstname":      shippingInfo.RecipientName,
        "lastname":       "",
        "email":          user.Email,
        "phone":          shippingInfo.Phone,
        "amount":         totalAmount,
        "description":    orderDescription.String(),
        "address":        fmt.Sprintf("%s, %s %s", shippingInfo.AddressLine1, shippingInfo.City, shippingInfo.PostalCode),
        "message":        "Order: " + orderID,
        "feeType":        "include",
        "orderId":        orderID,
        "paymentType":    "QRNONE",
        "agreement":      1,
    }
    
    paymentJSON, err := json.Marshal(paymentRequest)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create payment request"})
        return
    }
    
    // Send payment request to Java payment service
    req, err := http.NewRequest("POST", "http://localhost:8088/api/payment/create-qr", bytes.NewBuffer(paymentJSON))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create payment request"})
        return
    }
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{
        Timeout: 10 * time.Second,
    }
    resp, err := client.Do(req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to communicate with payment service: " + err.Error()})
        return
    }
    defer resp.Body.Close()
    
    // Check response status
    if resp.StatusCode != http.StatusOK {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "payment service returned error: " + resp.Status})
        return
    }
    
    // Read payment service response
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read payment service response"})
        return
    }
    
    // Parse payment response
    var paymentResponse map[string]interface{}
    err = json.Unmarshal(body, &paymentResponse)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse payment service response"})
        return
    }
    
    // Update the order with payment information
    if transactionID, ok := paymentResponse["transactionId"].(string); ok {
        _, err = config.DB.Exec(
            "UPDATE orders SET transaction_id = ? WHERE id = ?",
            transactionID, dbOrderID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order with transaction ID"})
            return
        }
    }
    
    // Return payment information to the client
    c.JSON(http.StatusOK, gin.H{
        "message": "order created successfully",
        "order_id": orderID,
        "payment": paymentResponse,
    })
}