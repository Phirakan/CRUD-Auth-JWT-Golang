package models

import (
	"time"
)

// Cart represents a user's shopping cart
type Cart struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Items     []CartItem `json:"items,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CartItem represents an item in the shopping cart
type CartItem struct {
    ID        int       `json:"id"`
    CartID    int       `json:"cart_id"`
    ProductID int       `json:"product_id"`
    SizeID    int       `json:"size_id,omitempty"`
    SizeName  string    `json:"size_name,omitempty"`
    Product   Product   `json:"product,omitempty"`
    Quantity  int       `json:"quantity"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// CartSummary provides a summary of the cart with totals
type CartSummary struct {
	CartID      int       `json:"cart_id"`
	ItemCount   int       `json:"item_count"`
	TotalItems  int       `json:"total_items"`
	TotalAmount float64   `json:"total_amount"`
	Items       []CartItem `json:"items"`
}

// CartItemInput holds data for adding/updating cart items
type CartItemInput struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}