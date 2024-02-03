package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gitarchived/updater/models"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	storage, err := minio.New(os.Getenv("STORAGE_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("STORAGE_KEY"), os.Getenv("STORAGE_SECRET"), ""),
		Secure: os.Getenv("STORAGE_SSL") == "true",
	})

	if err != nil {
		log.Fatal("Error creating Object Storage client")
	}

	log.Println("Connected to Object Storage at " + storage.EndpointURL().Host)

	db, err := gorm.Open(postgres.Open(os.Getenv("PG_URL")), &gorm.Config{})

	if err != nil {
		log.Fatal("Error connecting to PostgreSQL")
	}

	log.Println("Connected to PostgreSQL at " + db.Dialector.Name())

	var repositories []models.Repository

	result := db.Find(&repositories)

	log.Println("Found", result.RowsAffected, "repositories. updating...")

	for _, repository := range repositories {
		log.Println("Updating", repository.Name)

		url := fmt.Sprintf("https://github.com/%s/archive/refs/heads/master.zip", repository.Name)

		client := resty.New()

		resp, err := client.R().Get(url)

		if err != nil {
			log.Println("Error downloading", repository.Name)
			continue
		}

		split := strings.Split(repository.Name, "/")[1]
		name := fmt.Sprintf("%s-%s.zip", split, fmt.Sprint(repository.ID))

		err = os.WriteFile(name, resp.Body(), 0644) // save to disk

		if err != nil {
			log.Println("Error saving", name)
			continue
		}

		info, err := storage.FPutObject(context.Background(), "github", name, "./"+name, minio.PutObjectOptions{ContentType: "application/zip"})

		if err != nil {
			log.Println("Error uploading", name)
			log.Println(err)
			continue
		}

		err = os.Remove(name) // delete from disk

		if err != nil {
			log.Println("Error deleting", name)
			continue
		}

		log.Println("Uploaded", name, "to", info.Key)

		time.Sleep(5 * time.Second) // wait 5 seconds to avoid rate limits
	}
}
