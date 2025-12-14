package db

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type Node struct {
	ID            string `gorm:"primaryKey"`
	IPAddress     string
	Status        string
	LastSeen      time.Time
	Tokens        int64
	WalletAddress  string
	Specs          string
	BenchmarkScore int
}

type Job struct {
	ID     uint `gorm:"primaryKey"`
	NodeID string
	Image  string
	Status string
	Result string
}

type Customer struct {
	ID      string `gorm:"primaryKey"`
	ApiKey  string `gorm:"uniqueIndex"`
	Credits int64
}

func InitDB(dsn string) {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection established")

	// Auto Migrate
	err = DB.AutoMigrate(&Node{}, &Job{}, &Customer{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed Demo Customer
	var count int64
	DB.Model(&Customer{}).Count(&count)
	if count == 0 {
		demoCustomer := Customer{
			ID:      uuid.New().String(),
			ApiKey:  "sk_live_demo12345",
			Credits: 1000,
		}
		if err := DB.Create(&demoCustomer).Error; err != nil {
			log.Printf("Failed to seed demo customer: %v", err)
		} else {
			log.Println("Seeded Demo Customer (API Key: sk_live_demo12345)")
		}
	}
}
