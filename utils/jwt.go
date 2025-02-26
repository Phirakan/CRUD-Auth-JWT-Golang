package utils

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// JWT secret key
var jwtSecret = []byte("your_secret_key_change_this")

// JWTClaim represents JWT claims
type JWTClaim struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

// GenerateJWT generates a JWT token
func GenerateJWT(userID int, role string) (string, error) {
	// Set expiration time for token
	expirationTime := time.Now().Add(1 * time.Hour)
	
	// Create claims
	claims := &JWTClaim{
		UserID: userID,
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Generate signed token
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

// ValidateToken validates JWT token and returns claims
func ValidateToken(signedToken string) (*JWTClaim, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
	)
	
	if err != nil {
		return nil, err
	}
	
	// Validate token
	claims, ok := token.Claims.(*JWTClaim)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	
	// Verify expiration
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	
	return claims, nil
}