package handlers

import (
    "database/sql"
    "net/http"
    "strconv"

    "goapi/config"
    "goapi/models"
    
    "github.com/gin-gonic/gin"
)

// GetAllSizes retrieves all sizes
func GetAllSizes(c *gin.Context) {
    var sizes []models.Size
    
    // Query sizes from database
    query := `SELECT id, name, display_order, created_at, updated_at FROM sizes ORDER BY display_order`
    rows, err := config.DB.Query(query)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sizes"})
        return
    }
    defer rows.Close()
    
    // Iterate through rows
    for rows.Next() {
        var size models.Size
        err := rows.Scan(
            &size.ID, 
            &size.Name, 
            &size.DisplayOrder, 
            &size.CreatedAt, 
            &size.UpdatedAt,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process sizes"})
            return
        }
        sizes = append(sizes, size)
    }
    
    c.JSON(http.StatusOK, gin.H{"sizes": sizes})
}

// GetProductSizes retrieves all sizes for a specific product
func GetProductSizes(c *gin.Context) {
    // Get product ID from URL
    productID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
        return
    }
    
    // Check if product exists
    var exists int
    err = config.DB.QueryRow("SELECT COUNT(*) FROM products WHERE id = ?", productID).Scan(&exists)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    
    if exists == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
        return
    }
    
    // Get sizes for this product
    sizeRows, err := config.DB.Query(`
        SELECT ps.id, ps.product_id, ps.size_id, s.name, ps.stock, ps.created_at, ps.updated_at 
        FROM product_sizes ps
        JOIN sizes s ON ps.size_id = s.id
        WHERE ps.product_id = ?
        ORDER BY s.display_order
    `, productID)
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch product sizes"})
        return
    }
    defer sizeRows.Close()
    
    var sizes []models.ProductSize
    for sizeRows.Next() {
        var size models.ProductSize
        err := sizeRows.Scan(
            &size.ID,
            &size.ProductID,
            &size.SizeID,
            &size.SizeName,
            &size.Stock,
            &size.CreatedAt,
            &size.UpdatedAt,
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process product sizes"})
            return
        }
        sizes = append(sizes, size)
    }
    
    c.JSON(http.StatusOK, gin.H{"product_id": productID, "sizes": sizes})
}

// UpdateProductSizes updates the sizes and stock for a product
func UpdateProductSizes(c *gin.Context) {
    // Get product ID from URL
    productID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
        return
    }
    
    var input []models.ProductSizeInput
    
    // Parse request body
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Check if product exists
    var exists int
    err = config.DB.QueryRow("SELECT COUNT(*) FROM products WHERE id = ?", productID).Scan(&exists)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    
    if exists == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
        return
    }
    
    // Begin transaction
    tx, err := config.DB.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
        return
    }
    defer tx.Rollback()
    
    // For each size input, check if it exists for this product
    for _, sizeInput := range input {
        var existingSizeID int
        err := tx.QueryRow("SELECT id FROM product_sizes WHERE product_id = ? AND size_id = ?", 
            productID, sizeInput.SizeID).Scan(&existingSizeID)
        
        if err != nil {
            if err == sql.ErrNoRows {
                // Insert new product_size
                _, err = tx.Exec(
                    "INSERT INTO product_sizes (product_id, size_id, stock) VALUES (?, ?, ?)",
                    productID, sizeInput.SizeID, sizeInput.Stock)
                if err != nil {
                    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add product size"})
                    return
                }
            } else {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
                return
            }
        } else {
            // Update existing product_size
            _, err = tx.Exec(
                "UPDATE product_sizes SET stock = ? WHERE id = ?",
                sizeInput.Stock, existingSizeID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product size"})
                return
            }
        }
    }
    
    // Update total stock in products table
    _, err = tx.Exec(`
        UPDATE products 
        SET stock = (SELECT SUM(stock) FROM product_sizes WHERE product_id = ?) 
        WHERE id = ?
    `, productID, productID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product stock"})
        return
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "product sizes updated successfully"})
}

// CreateSize adds a new size
func CreateSize(c *gin.Context) {
    var input models.SizeInput
    
    // Parse request body
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Insert size into database
    result, err := config.DB.Exec(
        "INSERT INTO sizes (name, display_order) VALUES (?, ?)",
        input.Name, input.DisplayOrder)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create size"})
        return
    }
    
    // Get size ID
    sizeID, err := result.LastInsertId()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get size ID"})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "message": "size created successfully",
        "size_id": sizeID,
    })
}

// UpdateSize updates an existing size
func UpdateSize(c *gin.Context) {
    // Get size ID from URL
    sizeID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size ID"})
        return
    }
    
    var input models.SizeInput
    
    // Parse request body
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Check if size exists
    var exists bool
    err = config.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM sizes WHERE id = ?)", sizeID).Scan(&exists)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    
    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "size not found"})
        return
    }
    
    // Update size
    _, err = config.DB.Exec(
        "UPDATE sizes SET name = ?, display_order = ? WHERE id = ?",
        input.Name, input.DisplayOrder, sizeID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update size"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "size updated successfully"})
}

// DeleteSize removes a size
func DeleteSize(c *gin.Context) {
    // Get size ID from URL
    sizeID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size ID"})
        return
    }
    
    // Begin transaction
    tx, err := config.DB.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
        return
    }
    defer tx.Rollback()
    
    // Check if size is used in any products
    var count int
    err = tx.QueryRow("SELECT COUNT(*) FROM product_sizes WHERE size_id = ?", sizeID).Scan(&count)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    
    if count > 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "size is being used by products and cannot be deleted"})
        return
    }
    
    // Delete size
    _, err = tx.Exec("DELETE FROM sizes WHERE id = ?", sizeID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete size"})
        return
    }
    
    // Commit transaction
    err = tx.Commit()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "size deleted successfully"})
}