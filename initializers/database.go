package initializers

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB // ตัวแปรเก็บการเชื่อมต่อ DB

func ConnectToDB() {
	dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("❌ ไม่สามารถเชื่อมต่อกับฐานข้อมูล:", err)
		return
	}
	fmt.Println("✅ เชื่อมต่อกับฐานข้อมูลสำเร็จ!")
}
