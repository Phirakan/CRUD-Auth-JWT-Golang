package handlers
import (
	"net/http"
	"strings"
	"goapi/config" //change this to your module
	"github.com/gin-gonic/gin"
)

// PaymentWebhook handles payment status updates from the payment service
func PaymentWebhook(c *gin.Context) {
    var payload struct {
        OrderID        string `json:"orderId"`
        TransactionID  string `json:"transactionId"`
        Status         string `json:"status"`
        Amount         float64 `json:"amount,omitempty"`
    }
    
    // Parse request body
    if err := c.ShouldBindJSON(&payload); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Validate required fields
    if payload.OrderID == "" || payload.TransactionID == "" || payload.Status == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
        return
    }
    
    // Map payment status to order status
    var orderStatus string
    switch strings.ToUpper(payload.Status) {
    case "SUCCESS":
        orderStatus = "paid"
    case "CANCELLED", "FAILED":
        orderStatus = "cancelled"
        // Optionally restore product stock here
    default:
        orderStatus = "pending"
    }
    
    // Update order status
    result, err := config.DB.Exec(
        "UPDATE orders SET status = ? WHERE order_id = ? AND transaction_id = ?",
        orderStatus, payload.OrderID, payload.TransactionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order status"})
        return
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "order status updated successfully"})
}