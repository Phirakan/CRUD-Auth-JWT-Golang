package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"goapi/config" //change this to your module

	"github.com/gin-gonic/gin"
)

// GetOrderDetails retrieves detailed information about a specific order
func GetOrderDetails(c *gin.Context) {
    // Get order ID from URL
    orderID := c.Param("id")
    
    // Get user ID from context
    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
        return
    }
    
    // First verify the order belongs to the user
    var dbOrderID int
    var orderDetails struct {
        OrderID       string    `json:"order_id"`
        TotalAmount   float64   `json:"total_amount"`
        Status        string    `json:"status"`
        TransactionID sql.NullString `json:"transaction_id"`
        CreatedAt     time.Time `json:"created_at"`
    }
    
    err := config.DB.QueryRow(`
        SELECT id, order_id, total_amount, status, transaction_id, created_at 
        FROM orders 
        WHERE order_id = ? AND user_id = ?`, 
        orderID, userID).Scan(
            &dbOrderID,
            &orderDetails.OrderID,
            &orderDetails.TotalAmount,
            &orderDetails.Status,
            &orderDetails.TransactionID,
            &orderDetails.CreatedAt,
        )
    
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    
    // Get order items
    rows, err := config.DB.Query(`
        SELECT oi.product_id, oi.quantity, oi.price, p.name, p.description 
        FROM order_items oi
        JOIN products p ON oi.product_id = p.id
        WHERE oi.order_id = ?`, 
        dbOrderID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch order items"})
        return
    }
    defer rows.Close()
    
    var items []map[string]interface{}
    
    for rows.Next() {
        var item struct {
            ProductID   int     `json:"product_id"`
            Quantity    int     `json:"quantity"`
            Price       float64 `json:"price"`
            Name        string  `json:"name"`
            Description string  `json:"description"`
        }
        
        err := rows.Scan(
            &item.ProductID,
            &item.Quantity,
            &item.Price,
            &item.Name,
            &item.Description,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process order items"})
            return
        }
        
        items = append(items, map[string]interface{}{
            "product_id":  item.ProductID,
            "quantity":    item.Quantity,
            "price":       item.Price,
            "total_price": item.Price * float64(item.Quantity),
            "name":        item.Name,
            "description": item.Description,
        })
    }
    
    // Check payment status if transaction ID exists
    var paymentStatus string
    if orderDetails.TransactionID.Valid {
        // You can add code here to call your payment service API to get payment status
        paymentStatus = "unknown" // Default value
        
        // Create a request to your Java payment service
        req, err := http.NewRequest("GET", 
            fmt.Sprintf("https://cunning-smoothly-aphid.ngrok-free.app/api/payment/status/%s", orderDetails.TransactionID.String), 
            nil)
        
        if err == nil {
            client := &http.Client{}
            resp, err := client.Do(req)
            
            if err == nil && resp.StatusCode == http.StatusOK {
                defer resp.Body.Close()
                
                body, err := ioutil.ReadAll(resp.Body)
                if err == nil {
                    var paymentResponse map[string]interface{}
                    err = json.Unmarshal(body, &paymentResponse)
                    
                    if err == nil && paymentResponse["status"] != nil {
                        paymentStatus = paymentResponse["status"].(string)
                    }
                }
            }
        }
    } else {
        paymentStatus = "not_initiated"
    }
    
    // Prepare response
    response := map[string]interface{}{
        "order_id":      orderDetails.OrderID,
        "total_amount":  orderDetails.TotalAmount,
        "status":        orderDetails.Status,
        "created_at":    orderDetails.CreatedAt,
        "items":         items,
        "payment_status": paymentStatus,
    }
    
    if orderDetails.TransactionID.Valid {
        response["transaction_id"] = orderDetails.TransactionID.String
    }
    
    c.JSON(http.StatusOK, gin.H{"order": response})
}