package main

import (
	"github/Phirakan/go-CRUD/initializers"

	"github.com/gin-gonic/gin"
)

func init() {
	initializers.LoadEnvVariables()
}

func main() {
	// // กำหนดค่าตัวแปรแบบ
	// name := "mosu"
	// age := 25
	// score := 3.14

	// // แสดงผลลัพธ์ value(%v) ของตัวแปร name
	// fmt.Printf("Hello, my name is %v \n", name)
	// // แสดงผลลัพธ์ Type(%T) ของตัวแปร age
	// fmt.Printf("I am %T years old \n", age)
	// // แสดงผลลัพธ์ float(%f) ของตัวแปร score
	// fmt.Printf("My score is %f \n", score)

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello Golang",
		})
	})

	r.Run() //  localhost:8080

}
