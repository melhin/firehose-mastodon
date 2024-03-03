package models

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(dbName string) {
	log.Printf("Connecting to db:%s", dbName)
	database, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})

	if err != nil {
		panic("Failed to connect to database!")
	}

	DB = database
}

func Migrate() {
	DB.AutoMigrate(&Post{})
	log.Println("Database Migration Completed!")
}
