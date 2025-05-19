// handlers/shipping_address.go
package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"goapi/config"
	"goapi/models"
	
	"github.com/gin-gonic/gin"
)

// GetShippingAddresses retrieves all shipping addresses for the authenticated user
func GetShippingAddresses(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Query addresses from database
	rows, err := config.DB.Query(`
		SELECT id, recipient_name, phone, address_line1, address_line2, city, state, postal_code, country, is_default
		FROM shipping_addresses 
		WHERE user_id = ? 
		ORDER BY is_default DESC, id DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch addresses"})
		return
	}
	defer rows.Close()
	
	var addresses []models.ShippingAddress
	
	for rows.Next() {
		var address models.ShippingAddress
		var addressLine2 sql.NullString
		
		err := rows.Scan(
			&address.ID,
			&address.RecipientName,
			&address.Phone,
			&address.AddressLine1,
			&addressLine2,
			&address.City,
			&address.State,
			&address.PostalCode,
			&address.Country,
			&address.IsDefault,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process addresses"})
			return
		}
		
		if addressLine2.Valid {
			address.AddressLine2 = addressLine2.String
		}
		
		address.UserID = userID.(int)
		addresses = append(addresses, address)
	}
	
	c.JSON(http.StatusOK, gin.H{"addresses": addresses})
}

// GetShippingAddress retrieves a specific shipping address
func GetShippingAddress(c *gin.Context) {
	// Get address ID from URL
	addressID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address ID"})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Query address from database
	var address models.ShippingAddress
	var addressLine2 sql.NullString
	
	err = config.DB.QueryRow(`
		SELECT id, recipient_name, phone, address_line1, address_line2, city, state, postal_code, country, is_default 
		FROM shipping_addresses 
		WHERE id = ? AND user_id = ?`, 
		addressID, userID).Scan(
			&address.ID,
			&address.RecipientName,
			&address.Phone,
			&address.AddressLine1,
			&addressLine2,
			&address.City,
			&address.State,
			&address.PostalCode,
			&address.Country,
			&address.IsDefault,
		)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "address not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}
	
	if addressLine2.Valid {
		address.AddressLine2 = addressLine2.String
	}
	
	address.UserID = userID.(int)
	
	c.JSON(http.StatusOK, gin.H{"address": address})
}

// CreateShippingAddress adds a new shipping address
func CreateShippingAddress(c *gin.Context) {
	var input models.ShippingAddressInput
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	
	// If this is the default address, unset any existing default
	if input.IsDefault {
		_, err = tx.Exec(
			"UPDATE shipping_addresses SET is_default = 0 WHERE user_id = ?", 
			userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update default address"})
			return
		}
	}
	
	// Insert address into database
	result, err := tx.Exec(`
		INSERT INTO shipping_addresses (
			user_id, recipient_name, phone, address_line1, address_line2, 
			city, state, postal_code, country, is_default
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, input.RecipientName, input.Phone, input.AddressLine1, input.AddressLine2,
		input.City, input.State, input.PostalCode, input.Country, input.IsDefault,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create address"})
		return
	}
	
	// Get address ID
	addressID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get address ID"})
		return
	}
	
	// Commit transaction
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "address created successfully",
		"address_id": addressID,
	})
}

// UpdateShippingAddress updates an existing shipping address
func UpdateShippingAddress(c *gin.Context) {
	// Get address ID from URL
	addressID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address ID"})
		return
	}
	
	var input models.ShippingAddressInput
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	
	// Check if address exists and belongs to user
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM shipping_addresses WHERE id = ? AND user_id = ?", 
		addressID, userID).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "address not found"})
		return
	}
	
	// If this is the default address, unset any existing default
	if input.IsDefault {
		_, err = tx.Exec(
			"UPDATE shipping_addresses SET is_default = 0 WHERE user_id = ? AND id != ?", 
			userID, addressID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update default address"})
			return
		}
	}
	
	// Update address
	_, err = tx.Exec(`
		UPDATE shipping_addresses SET 
			recipient_name = ?, phone = ?, address_line1 = ?, address_line2 = ?,
			city = ?, state = ?, postal_code = ?, country = ?, is_default = ?
		WHERE id = ? AND user_id = ?`,
		input.RecipientName, input.Phone, input.AddressLine1, input.AddressLine2,
		input.City, input.State, input.PostalCode, input.Country, input.IsDefault,
		addressID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update address"})
		return
	}
	
	// Commit transaction
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "address updated successfully"})
}

// DeleteShippingAddress removes a shipping address
func DeleteShippingAddress(c *gin.Context) {
	// Get address ID from URL
	addressID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address ID"})
		return
	}
	
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found"})
		return
	}
	
	// Delete address
	result, err := config.DB.Exec(
		"DELETE FROM shipping_addresses WHERE id = ? AND user_id = ?", 
		addressID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete address"})
		return
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "address not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "address deleted successfully"})
}