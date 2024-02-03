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

	var ctx = context.Background()

	for _, repository := range repositories {
		log.Println("Updating", repository.Name)

		url := fmt.Sprintf("https://github.com/%s/archive/refs/heads/master.zip", repository.Name)

		client := resty.New()

		resp, err := client.R().Get(url)

		if err != nil {
			log.Println("Error downloading", repository.Name)
			continue
		}

		name := strings.Split(repository.Name, "/")[1]

		// Build the path (./t/h/e/r/e/p/o/n/a/m/e/[id].zip) all the name letter need to be a folder
		path := strings.Split(name, "")
		path = append(path, fmt.Sprintf("%d.zip", repository.ID))

		localPath := "./" + strings.Join(path, "/")

		// Save file local
		err = os.MkdirAll(strings.Join(path[:len(path)-1], "/"), 0755)

		if err != nil {
			log.Println("Error creating folders for", repository.Name)
			continue
		}

		err = os.WriteFile(localPath, resp.Body(), 0644)

		if err != nil {
			log.Println("Error saving file for", repository.Name)
			continue
		}

		// Upload file to object storage
		_, err = storage.FPutObject(ctx, "github", strings.Join(path, "/"), strings.Join(path, "/"), minio.PutObjectOptions{ContentType: "application/zip"})

		if err != nil {
			log.Println("Error uploading file for", repository.Name)
			continue
		}

		// Remove file local (even the directories)
		err = os.RemoveAll(strings.Split(localPath, "/")[1])

		if err != nil {
			log.Println("Error removing local file for", repository.Name)
			continue
		}

		log.Println("Updated", repository.Name)

		time.Sleep(5 * time.Second) // wait 5 seconds to avoid rate limits
	}
}
