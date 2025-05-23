package handlers

import (
	"database/sql"

	"net/http"
	"strconv"


	"goapi/config"
	"goapi/models"
	
	"github.com/gin-gonic/gin"
)
// GetCart retrieves the user's current cart
func GetCart(c *gin.Context) {
	// Get user ID from context (set by AuthMiddleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Find or create cart for user
	var cartID int
	err := config.DB.QueryRow("SELECT id FROM carts WHERE user_id = ?", userID).Scan(&cartID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new cart if one doesn't exist
			result, err := config.DB.Exec("INSERT INTO carts (user_id) VALUES (?)", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cart"})
				return
			}
			
			cartID64, err := result.LastInsertId()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cart ID"})
				return
			}
			cartID = int(cartID64)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}
	
	// Get cart items
	rows, err := config.DB.Query(`
		SELECT ci.id, ci.product_id, ci.quantity, 
		       p.name, p.description, p.price, p.stock 
		FROM cart_items ci 
		JOIN products p ON ci.product_id = p.id 
		WHERE ci.cart_id = ?`, cartID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cart items"})
		return
	}
	defer rows.Close()
	
	var items []models.CartItem
	var totalItems int
	var totalAmount float64
	
	for rows.Next() {
		var item models.CartItem
		var product models.Product
		
		err := rows.Scan(
			&item.ID, 
			&item.ProductID, 
			&item.Quantity,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Stock,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process cart items"})
			return
		}
		
		product.ID = item.ProductID
		item.Product = product
		item.CartID = cartID
		
		items = append(items, item)
		totalItems += item.Quantity
		totalAmount += float64(item.Quantity) * product.Price
	}
	
	// Create cart summary
	cartSummary := models.CartSummary{
		CartID:      cartID,
		ItemCount:   len(items),
		TotalItems:  totalItems,
		TotalAmount: totalAmount,
		Items:       items,
	}
	
	c.JSON(http.StatusOK, gin.H{"cart": cartSummary})
}

// AddToCart adds a product to the cart
func AddToCart(c *gin.Context) {
	var input struct {
		ProductID int `json:"product_id" binding:"required"`
		SizeID    int `json:"size_id"`
		Quantity  int `json:"quantity" binding:"required"`
	}
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate quantity
	if input.Quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be at least 1"})
		return
	}
	
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
	
	// Check if product exists
	var productExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = ?)", input.ProductID).Scan(&productExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	if !productExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	
	// Check if the product has sizes
	var sizeCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM product_sizes WHERE product_id = ?", input.ProductID).Scan(&sizeCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	// If product has sizes, a size ID is required
	if sizeCount > 0 && input.SizeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "size is required for this product"})
		return
	}
	
	// Check if size exists and has enough stock (if a size is specified)
	var stockAvailable int
	if input.SizeID > 0 {
		err = tx.QueryRow("SELECT stock FROM product_sizes WHERE product_id = ? AND size_id = ?", 
			input.ProductID, input.SizeID).Scan(&stockAvailable)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "size not found for this product"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			}
			return
		}
	} else {
		// If no size is specified, check overall product stock
		err = tx.QueryRow("SELECT stock FROM products WHERE id = ?", input.ProductID).Scan(&stockAvailable)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}
	
	if stockAvailable < input.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not enough stock available"})
		return
	}
	
	// Find or create cart for user
	var cartID int
	err = tx.QueryRow("SELECT id FROM carts WHERE user_id = ?", userID).Scan(&cartID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new cart if one doesn't exist
			result, err := tx.Exec("INSERT INTO carts (user_id) VALUES (?)", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cart"})
				return
			}
			
			cartIDInt64, err := result.LastInsertId()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cart ID"})
				return
			}
			cartID = int(cartIDInt64)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}
	
	// Check if item already exists in cart (including size)
	var existingItemID int
	var existingQuantity int
	var findItemQuery string
	var findItemArgs []interface{}
	
	if input.SizeID > 0 {
		findItemQuery = "SELECT id, quantity FROM cart_items WHERE cart_id = ? AND product_id = ? AND size_id = ?"
		findItemArgs = []interface{}{cartID, input.ProductID, input.SizeID}
	} else {
		findItemQuery = "SELECT id, quantity FROM cart_items WHERE cart_id = ? AND product_id = ? AND size_id IS NULL"
		findItemArgs = []interface{}{cartID, input.ProductID}
	}
	
	err = tx.QueryRow(findItemQuery, findItemArgs...).Scan(&existingItemID, &existingQuantity)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Add new item to cart
			var insertQuery string
			var insertArgs []interface{}
			
			if input.SizeID > 0 {
				insertQuery = "INSERT INTO cart_items (cart_id, product_id, size_id, quantity) VALUES (?, ?, ?, ?)"
				insertArgs = []interface{}{cartID, input.ProductID, input.SizeID, input.Quantity}
			} else {
				insertQuery = "INSERT INTO cart_items (cart_id, product_id, quantity) VALUES (?, ?, ?)"
				insertArgs = []interface{}{cartID, input.ProductID, input.Quantity}
			}
			
			_, err = tx.Exec(insertQuery, insertArgs...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add item to cart"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	} else {
		// Update existing item quantity
		newQuantity := existingQuantity + input.Quantity
		if newQuantity > stockAvailable {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not enough stock available"})
			return
		}
		
		_, err = tx.Exec(
			"UPDATE cart_items SET quantity = ? WHERE id = ?",
			newQuantity, existingItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update cart item"})
			return
		}
	}
	
	// Commit transaction
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "item added to cart successfully"})
}

// UpdateCartItem updates the quantity of a cart item
func UpdateCartItem(c *gin.Context) {
	// Get item ID from URL
	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}
	
	var input struct {
		Quantity int `json:"quantity"`
	}
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate quantity
	if input.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity cannot be negative"})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Verify user owns the cart containing this item
	var count int
	err = config.DB.QueryRow(`
		SELECT COUNT(*) FROM cart_items ci
		JOIN carts c ON ci.cart_id = c.id
		WHERE ci.id = ? AND c.user_id = ?`, 
		itemID, userID).Scan(&count)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "cart item not found or not authorized"})
		return
	}
	
	// Get product ID and check stock
	var productID int
	err = config.DB.QueryRow("SELECT product_id FROM cart_items WHERE id = ?", itemID).Scan(&productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	var productStock int
	err = config.DB.QueryRow("SELECT stock FROM products WHERE id = ?", productID).Scan(&productStock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	if input.Quantity > productStock {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not enough stock available"})
		return
	}
	
	// Update or remove item based on quantity
	if input.Quantity == 0 {
		// Remove item from cart
		_, err = config.DB.Exec("DELETE FROM cart_items WHERE id = ?", itemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove item from cart"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "item removed from cart"})
	} else {
		// Update quantity
		_, err = config.DB.Exec("UPDATE cart_items SET quantity = ? WHERE id = ?", input.Quantity, itemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update cart item"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cart item updated successfully"})
	}
}

// RemoveFromCart removes an item from the cart
func RemoveFromCart(c *gin.Context) {
	// Get item ID from URL
	itemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Verify user owns the cart containing this item
	var count int
	err = config.DB.QueryRow(`
		SELECT COUNT(*) FROM cart_items ci
		JOIN carts c ON ci.cart_id = c.id
		WHERE ci.id = ? AND c.user_id = ?`, 
		itemID, userID).Scan(&count)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "cart item not found or not authorized"})
		return
	}
	
	// Remove item from cart
	_, err = config.DB.Exec("DELETE FROM cart_items WHERE id = ?", itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove item from cart"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "item removed from cart successfully"})
}

// ClearCart removes all items from the user's cart
func ClearCart(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Get cart ID
	var cartID int
	err := config.DB.QueryRow("SELECT id FROM carts WHERE user_id = ?", userID).Scan(&cartID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No cart exists, so it's already "cleared"
			c.JSON(http.StatusOK, gin.H{"message": "cart is already empty"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}
	
	// Remove all items from cart
	_, err = config.DB.Exec("DELETE FROM cart_items WHERE cart_id = ?", cartID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear cart"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "cart cleared successfully"})
}