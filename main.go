package main

import (
	"log"

	"goapi/config" //change this to your module
	"goapi/handlers" //change this to your module
	"goapi/middleware" //change this to your module

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	config.InitDB()
	defer config.DB.Close()
	
	// Create a new Gin router
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())
	
	r.GET("/health-check", handlers.CheckConnection)
	
	// Public routes (no authentication required)
	r.POST("/register", handlers.RegisterUser)
	r.POST("/login", handlers.LoginUser)
	r.GET("/products", handlers.GetAllProducts)
	r.GET("/products/:id", handlers.GetProduct)

	// Protected routes (authentication required)
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())
	{
		// Cart routes
		auth.GET("/cart", handlers.GetCart)
		auth.POST("/cart/items", handlers.AddToCart)
		auth.PUT("/cart/items/:id", handlers.UpdateCartItem)
		auth.DELETE("/cart/items/:id", handlers.RemoveFromCart)
		auth.DELETE("/cart", handlers.ClearCart)
		
		// Checkout route
		auth.POST("/checkout", handlers.Checkout)
		
		// Order routes
		auth.GET("/orders", handlers.GetOrders)
		auth.GET("/orders/:id", handlers.GetOrderDetails)

		 // Shipping address routes
    	auth.GET("/shipping-addresses", handlers.GetShippingAddresses)
    	auth.GET("/shipping-addresses/:id", handlers.GetShippingAddress)
   		auth.POST("/shipping-addresses", handlers.CreateShippingAddress)
    	auth.PUT("/shipping-addresses/:id", handlers.UpdateShippingAddress)
    	auth.DELETE("/shipping-addresses/:id", handlers.DeleteShippingAddress)
	}

	// Admin-only routes
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminRequired())
	{
		// Product management
		admin.POST("/products", handlers.CreateProduct)
		admin.PUT("/products/:id", handlers.UpdateProduct)
		admin.DELETE("/products/:id", handlers.DeleteProduct)
		
		// Admin user management
		admin.POST("/users", handlers.CreateAdmin)
		
		// Admin order management
    	admin.GET("/orders", handlers.GetAllOrders)
    	admin.PUT("/orders/:id/status", handlers.UpdateOrderStatus)
	}
	
	// Webhook routes (called by external services)
	r.POST("/api/webhook/payment", handlers.PaymentWebhook)
	
	// Start the server
	log.Println("Server starting on http://localhost:8080")
	r.Run(":8080")
}