package handlers

import (
	"net/http"
	"goapi/config" //change this to your module

	"github.com/gin-gonic/gin"
)

// CheckConnection 
func CheckConnection(c *gin.Context) {
	err := config.DB.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Hello This is Your Go API Project"})
}
