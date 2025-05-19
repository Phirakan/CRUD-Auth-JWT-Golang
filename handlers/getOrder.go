package handlers
import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"goapi/config" //change this to your module
	"github.com/gin-gonic/gin"
)

// GetAllOrders retrieves all orders (admin only)
func GetAllOrders(c *gin.Context) {
    // Pagination parameters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    status := c.Query("status")
    
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 10
    }
    
    offset := (page - 1) * limit
    
    // Build query
    query := `
        SELECT o.id, o.order_id, o.user_id, u.username, o.total_amount, o.status, 
               o.transaction_id, o.created_at, COUNT(oi.id) as item_count
        FROM orders o
        JOIN users u ON o.user_id = u.id
        JOIN order_items oi ON o.id = oi.order_id
    `
    countQuery := `SELECT COUNT(*) FROM orders o`
    
    // Add status filter if provided
    var args []interface{}
    if status != "" {
        query += " WHERE o.status = ?"
        countQuery += " WHERE o.status = ?"
        args = append(args, status)
    }
    
    // Group by and order
    query += ` GROUP BY o.id ORDER BY o.created_at DESC LIMIT ? OFFSET ?`
    args = append(args, limit, offset)
    
    // Execute count query for pagination
    var totalOrders int
    if err := config.DB.QueryRow(countQuery, args[:len(args)-2]...).Scan(&totalOrders); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count orders"})
        return
    }
    
    // Execute main query
    rows, err := config.DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch orders"})
        return
    }
    defer rows.Close()
    
    var orders []map[string]interface{}
    
    for rows.Next() {
        var order struct {
            ID            int       `json:"id"`
            OrderID       string    `json:"order_id"`
            UserID        int       `json:"user_id"`
            Username      string    `json:"username"`
            TotalAmount   float64   `json:"total_amount"`
            Status        string    `json:"status"`
            TransactionID sql.NullString `json:"transaction_id"`
            CreatedAt     time.Time `json:"created_at"`
            ItemCount     int       `json:"item_count"`
        }
        
        err := rows.Scan(
            &order.ID,
            &order.OrderID,
            &order.UserID,
            &order.Username,
            &order.TotalAmount,
            &order.Status,
            &order.TransactionID,
            &order.CreatedAt,
            &order.ItemCount,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process orders"})
            return
        }
        
        orderMap := map[string]interface{}{
            "id":           order.ID,
            "order_id":     order.OrderID,
            "user_id":      order.UserID,
            "username":     order.Username,
            "total_amount": order.TotalAmount,
            "status":       order.Status,
            "created_at":   order.CreatedAt,
            "item_count":   order.ItemCount,
        }
        
        if order.TransactionID.Valid {
            orderMap["transaction_id"] = order.TransactionID.String
        } else {
            orderMap["transaction_id"] = nil
        }
        
        orders = append(orders, orderMap)
    }
    
    // Calculate pagination info
    totalPages := (totalOrders + limit - 1) / limit
    
    c.JSON(http.StatusOK, gin.H{
        "orders": orders,
        "pagination": gin.H{
            "total":      totalOrders,
            "page":       page,
            "limit":      limit,
            "total_pages": totalPages,
        },
    })
}

// UpdateOrderStatus allows an admin to update an order's status
func UpdateOrderStatus(c *gin.Context) {
    // Get order ID from URL
    orderID := c.Param("id")
    
    var input struct {
        Status string `json:"status" binding:"required"`
    }
    
    // Parse request body
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Validate status
    validStatuses := map[string]bool{
        "pending":   true,
        "paid":      true,
        "shipped":   true,
        "delivered": true,
        "cancelled": true,
    }
    
    if !validStatuses[input.Status] {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
        return
    }
    
    // Begin transaction
    tx, err := config.DB.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
        return
    }
    defer tx.Rollback()
    
    // Get current order status
    var currentStatus string
    var dbOrderID int
    
    err = tx.QueryRow("SELECT id, status FROM orders WHERE order_id = ?", orderID).Scan(&dbOrderID, &currentStatus)
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    
    // Handle special cases for status changes
    if currentStatus == "cancelled" && input.Status != "cancelled" {
        // If reactivating a cancelled order, restore the stock reservation
        rows, err := tx.Query("SELECT product_id, quantity FROM order_items WHERE order_id = ?", dbOrderID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch order items"})
            return
        }
        defer rows.Close()
        
        for rows.Next() {
            var productID, quantity int
            
            err := rows.Scan(&productID, &quantity)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process order items"})
                return
            }
            
            // Check if we have enough stock
            var currentStock int
            err = tx.QueryRow("SELECT stock FROM products WHERE id = ?", productID).Scan(&currentStock)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get product stock"})
                return
            }
            
            if currentStock < quantity {
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": fmt.Sprintf("Not enough stock to fulfill this order (Product ID: %d)", productID),
                })
                return
            }
            
            // Deduct stock
            _, err = tx.Exec("UPDATE products SET stock = stock - ? WHERE id = ?", quantity, productID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product stock"})
                return
            }
        }
    } else if currentStatus != "cancelled" && input.Status == "cancelled" {
        // If cancelling an order, release the stock reservation
        rows, err := tx.Query("SELECT product_id, quantity FROM order_items WHERE order_id = ?", dbOrderID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch order items"})
            return
        }
        defer rows.Close()
        
        for rows.Next() {
            var productID, quantity int
            
            err := rows.Scan(&productID, &quantity)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process order items"})
                return
            }
            
            // Restore stock
            _, err = tx.Exec("UPDATE products SET stock = stock + ? WHERE id = ?", quantity, productID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product stock"})
                return
            }
        }
    }
    
    // Update order status
    _, err = tx.Exec("UPDATE orders SET status = ? WHERE id = ?", input.Status, dbOrderID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order status"})
        return
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "order status updated successfully"})
}