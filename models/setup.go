package models

import (
	"log"
	"time"

	"os"

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
	// Enable connection pooling and set the max number of connections
	sqlDB, err := database.DB()

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = database
}

func Migrate() {
	DB.AutoMigrate(&Post{})
	log.Println("Database Migration Completed!")
}

func DBSetup(dbName string) {
	ConnectDatabase(dbName)
	Migrate()
}

func RemoveDb(dbName string) {
	os.Remove(dbName)
}
