package models

import (
    "time"
)

// Size represents a clothing size
type Size struct {
    ID          int       `json:"id"`
    Name        string    `json:"name"`
    DisplayOrder int      `json:"display_order"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// ProductSize represents the relationship between a product and a size
type ProductSize struct {
    ID        int       `json:"id"`
    ProductID int       `json:"product_id"`
    SizeID    int       `json:"size_id"`
    SizeName  string    `json:"size_name"` // For convenience in API responses
    Stock     int       `json:"stock"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// SizeInput is used for adding/updating sizes
type SizeInput struct {
    Name        string `json:"name" binding:"required"`
    DisplayOrder int    `json:"display_order"`
}

// ProductSizeInput is used for adding/updating product sizes
type ProductSizeInput struct {
    SizeID int `json:"size_id" binding:"required"`
    Stock  int `json:"stock" binding:"required"`
}