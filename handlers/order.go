package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"goapi/config" //change this to your module
	
	"github.com/gin-gonic/gin"

)

// GetOrders retrieves all orders for the authenticated user
func GetOrders(c *gin.Context) {
    // Get user ID from context
    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
        return
    }
    
    // Query orders from database
    rows, err := config.DB.Query(`
        SELECT id, order_id, total_amount, status, transaction_id, created_at 
        FROM orders 
        WHERE user_id = ? 
        ORDER BY created_at DESC`, userID)
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
            TotalAmount   float64   `json:"total_amount"`
            Status        string    `json:"status"`
            TransactionID sql.NullString `json:"transaction_id"`
            CreatedAt     time.Time `json:"created_at"`
        }
        
        err := rows.Scan(
            &order.ID,
            &order.OrderID,
            &order.TotalAmount,
            &order.Status,
            &order.TransactionID,
            &order.CreatedAt,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process orders"})
            return
        }
        
        orderMap := map[string]interface{}{
            "id":          order.ID,
            "order_id":    order.OrderID,
            "total_amount": order.TotalAmount,
            "status":      order.Status,
            "created_at":  order.CreatedAt,
        }
        
        if order.TransactionID.Valid {
            orderMap["transaction_id"] = order.TransactionID.String
        } else {
            orderMap["transaction_id"] = nil
        }
        
        orders = append(orders, orderMap)
    }
    
    c.JSON(http.StatusOK, gin.H{"orders": orders})
}