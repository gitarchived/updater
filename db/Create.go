package db

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Create() *gorm.DB {
	db, err := gorm.Open(postgres.Open(os.Getenv("PG_URL")), &gorm.Config{})

	if err != nil {
		log.Fatal("Error connecting to PostgreSQL")
	}

	return db
}
