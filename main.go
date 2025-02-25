package main

import (
	"log"

	"goapi/config"
	"goapi/handlers"
	"goapi/middleware"
	
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	config.InitDB()
	defer config.DB.Close()
	
	// Create a new Gin router
	r := gin.Default()
	
	// Public routes (no authentication required)
	r.POST("/register", handlers.RegisterUser)
	r.POST("/login", handlers.LoginUser)
	r.GET("/products", handlers.GetAllProducts)
	r.GET("/products/:id", handlers.GetProduct)
	
	// Protected routes (authentication required)
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())
	{
		// Product routes (both users and admins)
		auth.POST("/products", handlers.CreateProduct)
		auth.PUT("/products/:id", handlers.UpdateProduct)
		auth.DELETE("/products/:id", handlers.DeleteProduct)
	}
	
	// Admin-only routes
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminRequired())
	{
		// Admin user management
		admin.POST("/users", handlers.CreateAdmin)
		// Add more admin-specific routes as needed
	}
	
	// Start the server
	log.Println("Server starting on http://localhost:8080")
	r.Run(":8080")
}