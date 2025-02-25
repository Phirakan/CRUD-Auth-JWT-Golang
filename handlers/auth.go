package handlers

import (
	"database/sql"
	"net/http"

	"goapi/config"
	"goapi/models"
	"goapi/utils"
	
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// RegisterUser creates a new user account
func RegisterUser(c *gin.Context) {
	var input models.UserRegister
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}
	
	// Insert user into database
	query := `INSERT INTO users (username, password, email, role) VALUES (?, ?, ?, 'user')`
	result, err := config.DB.Exec(query, input.Username, hashedPassword, input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	
	// Get user ID
	userID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user ID"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user_id": userID,
	})
}

// LoginUser authenticates a user and returns JWT token
func LoginUser(c *gin.Context) {
	var input models.UserLogin
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Query user from database
	var user models.User
	query := `SELECT id, username, password, email, role FROM users WHERE username = ?`
	err := config.DB.QueryRow(query, input.Username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email, &user.Role,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	
	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	
	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"token": token,
		"user": gin.H{
			"id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"role": user.Role,
		},
	})
}

// CreateAdmin creates an admin account (only callable by another admin)
func CreateAdmin(c *gin.Context) {
	var input models.UserRegister
	
	// Parse request body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}
	
	// Insert admin into database
	query := `INSERT INTO users (username, password, email, role) VALUES (?, ?, ?, 'admin')`
	result, err := config.DB.Exec(query, input.Username, hashedPassword, input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin"})
		return
	}
	
	// Get user ID
	userID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get admin ID"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "admin created successfully",
		"user_id": userID,
	})
}