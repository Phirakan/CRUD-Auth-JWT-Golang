package models

import (
	"time"
)

// ShippingAddress represents a user's shipping address
type ShippingAddress struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	RecipientName string   `json:"recipient_name"`
	Phone        string    `json:"phone"`
	AddressLine1 string    `json:"address_line1"`
	AddressLine2 string    `json:"address_line2,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ShippingAddressInput is used for creating/updating shipping addresses
type ShippingAddressInput struct {
	RecipientName string `json:"recipient_name" binding:"required"`
	Phone        string  `json:"phone" binding:"required"`
	AddressLine1 string  `json:"address_line1" binding:"required"`
	AddressLine2 string  `json:"address_line2"`
	City         string  `json:"city" binding:"required"`
	State        string  `json:"state" binding:"required"`
	PostalCode   string  `json:"postal_code" binding:"required"`
	Country      string  `json:"country" binding:"required"`
	IsDefault    bool    `json:"is_default"`
}