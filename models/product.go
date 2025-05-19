package models

import (
	"time"
)

// Product represents product data in the system
type Product struct {
    ID          int           `json:"id"`
    Name        string        `json:"name"`
    Description string        `json:"description"`
    Price       float64       `json:"price"`
    Stock       int           `json:"stock"` // Total stock across all sizes
    Sizes       []ProductSize `json:"sizes,omitempty"`
    CreatedBy   int           `json:"created_by"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
}

// ProductInput with sizes
type ProductInput struct {
    Name        string             `json:"name" binding:"required"`
    Description string             `json:"description"`
    Price       float64            `json:"price" binding:"required"`
    Sizes       []ProductSizeInput `json:"sizes"`
}