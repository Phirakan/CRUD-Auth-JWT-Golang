package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"goapi/config"
	"goapi/models"
	
	"github.com/gin-gonic/gin"
)

// CreateProduct adds a new product
func CreateProduct(c *gin.Context) {
	var input models.ProductInput
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user ID from context (set by AuthMiddleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Insert product into database
	query := `INSERT INTO products (name, description, price, stock, created_by) VALUES (?, ?, ?, ?, ?)`
	result, err := config.DB.Exec(query, input.Name, input.Description, input.Price, input.Stock, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}
	
	// Get product ID
	productID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get product ID"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "product created successfully",
		"product_id": productID,
	})
}

// GetAllProducts retrieves all products
func GetAllProducts(c *gin.Context) {
	var products []models.Product
	
	// Query products from database
	query := `SELECT id, name, description, price, stock, created_by, created_at, updated_at FROM products`
	rows, err := config.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	defer rows.Close()
	
	// Iterate through rows
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID, 
			&product.Name, 
			&product.Description, 
			&product.Price, 
			&product.Stock, 
			&product.CreatedBy, 
			&product.CreatedAt, 
			&product.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process products"})
			return
		}
		products = append(products, product)
	}
	
	c.JSON(http.StatusOK, gin.H{"products": products})
}

// GetProduct retrieves a specific product by ID
func GetProduct(c *gin.Context) {
	// Get product ID from URL
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}
	
	// Query product from database
	var product models.Product
	query := `SELECT id, name, description, price, stock, created_by, created_at, updated_at FROM products WHERE id = ?`
	err = config.DB.QueryRow(query, productID).Scan(
		&product.ID, 
		&product.Name, 
		&product.Description, 
		&product.Price, 
		&product.Stock, 
		&product.CreatedBy, 
		&product.CreatedAt, 
		&product.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"product": product})
}

// UpdateProduct updates a specific product
func UpdateProduct(c *gin.Context) {
	// Get product ID from URL
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}
	
	var input models.ProductInput
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user role from context
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user role not found"})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Check if product exists and user has permission
	var createdBy int
	err = config.DB.QueryRow("SELECT created_by FROM products WHERE id = ?", productID).Scan(&createdBy)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	// Only admin or creator can update
	if role != "admin" && userID != createdBy {
		c.JSON(http.StatusForbidden, gin.H{"error": "you don't have permission to update this product"})
		return
	}
	
	// Update product in database
	query := `UPDATE products SET name = ?, description = ?, price = ?, stock = ? WHERE id = ?`
	_, err = config.DB.Exec(query, input.Name, input.Description, input.Price, input.Stock, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "product updated successfully"})
}

// DeleteProduct removes a specific product
func DeleteProduct(c *gin.Context) {
	// Get product ID from URL
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}
	
	// Get user role from context
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user role not found"})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Check if product exists and user has permission
	var createdBy int
	err = config.DB.QueryRow("SELECT created_by FROM products WHERE id = ?", productID).Scan(&createdBy)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	// Only admin or creator can delete
	if role != "admin" && userID != createdBy {
		c.JSON(http.StatusForbidden, gin.H{"error": "you don't have permission to delete this product"})
		return
	}
	
	// Delete product from database
	_, err = config.DB.Exec("DELETE FROM products WHERE id = ?", productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "product deleted successfully"})
}