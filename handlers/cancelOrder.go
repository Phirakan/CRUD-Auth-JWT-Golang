package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"goapi/config" //change this to your module

	"github.com/gin-gonic/gin"
)

// CancelOrder allows a user to cancel their order if it's still in 'pending' status
func CancelOrder(c *gin.Context) {
    // Get order ID from URL
    orderID := c.Param("id")
    
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
    defer tx.Rollback()
    
    // Get order details and verify ownership
    var dbOrderID int
    var status string
    var transactionID sql.NullString
    
    err = tx.QueryRow(`
        SELECT id, status, transaction_id 
        FROM orders 
        WHERE order_id = ? AND user_id = ?`, 
        orderID, userID).Scan(&dbOrderID, &status, &transactionID)
    
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    
    // Check if order can be cancelled
    if status != "pending" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "only pending orders can be cancelled"})
        return
    }
    
    // Update order status
    _, err = tx.Exec("UPDATE orders SET status = 'cancelled' WHERE id = ?", dbOrderID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel order"})
        return
    }
    
    // Get order items to restore stock
    rows, err := tx.Query(`
        SELECT product_id, quantity 
        FROM order_items 
        WHERE order_id = ?`, dbOrderID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch order items"})
        return
    }
    defer rows.Close()
    
    // Restore product stock
    for rows.Next() {
        var productID, quantity int
        
        err := rows.Scan(&productID, &quantity)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process order items"})
            return
        }
        
        // Update product stock
        _, err = tx.Exec(
            "UPDATE products SET stock = stock + ? WHERE id = ?",
            quantity, productID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product stock"})
            return
        }
    }
    
    // If a transaction ID exists, cancel the payment
    if transactionID.Valid {
        // Create a request to cancel the payment in the Java payment service
        req, err := http.NewRequest("POST", 
            fmt.Sprintf("https://cunning-smoothly-aphid.ngrok-free.app/api/payment/cancel/%s", transactionID.String), 
            nil)
        
        if err == nil {
            client := &http.Client{}
            _, _ = client.Do(req) // Ignore errors, we've already updated our DB
        }
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "order cancelled successfully"})
}