package middleware

import (
	"net/http"
	"strings"

	"goapi/utils" //change this to your module
	
	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles authentication check
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}
		
		// Extract token from "Bearer <token>"
		splitToken := strings.Split(authHeader, "Bearer ")
		if len(splitToken) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			c.Abort()
			return
		}
		
		tokenString := splitToken[1]
		
		// Validate token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		
		// Set user ID and role in context
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		
		c.Next()
	}
}

// AdminRequired ensures user has admin role
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context (set by AuthMiddleware)
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		
		// Check if user is admin
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin privileges required"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}