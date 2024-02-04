package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gitarchived/updater/models"
	"github.com/gitarchived/updater/utils"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	prod := os.Getenv("PRODUCTION")

	if prod == "" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
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

		lastCommit, err := utils.GetLastCommit(repository.Name)

		if err != nil {
			log.Println("Error getting last commit for", repository.Name)
			continue
		}

		if lastCommit == repository.LastCommit {
			log.Println("No new commits for", repository.Name, "skipping...")
			continue
		}

		name := strings.Split(repository.Name, "/")[1]

		cmdClone := exec.Command("git", "clone", fmt.Sprintf("https://github.com/%s.git", repository.Name), fmt.Sprintf("./%s", name))

		if err := cmdClone.Run(); err != nil {
			log.Println("Error cloning", repository.Name)
			continue
		}

		// Create a bunde file
		cmdBundle := exec.Command("git", "bundle", "create", fmt.Sprintf("./%s.bundle", name), "--all")

		if err := cmdBundle.Run(); err != nil {
			log.Println("Error creating bundle for", repository.Name)
			println(err.Error())
			continue
		}

		// Rename the file (neovim.bundle -> [id].bundle)
		err = os.Rename(fmt.Sprintf("./%s.bundle", name), fmt.Sprintf("./%d.bundle", repository.ID))

		// Build the path (./t/h/e/r/e/p/o/n/a/m/e/[id].zip) all the name letter need to be a folder
		path := strings.Split(name, "")
		path = append(path, fmt.Sprintf("%d.bundle", repository.ID))

		localPath := "./" + strings.Join(path, "/")

		// Save file local
		err = os.MkdirAll(strings.Join(path[:len(path)-1], "/"), 0755)

		if err != nil {
			log.Println("Error creating folders for", repository.Name)
			continue
		}

		// Move the file to the right path
		err = os.Rename(fmt.Sprintf("./%d.bundle", repository.ID), localPath)

		if err != nil {
			log.Println("Error moving file for", repository.Name)
			continue
		}

		// Upload file to object storage
		_, err = storage.FPutObject(ctx, os.Getenv("STORAGE_BUCKET"), strings.Join(path, "/"), strings.Join(path, "/"), minio.PutObjectOptions{})

		if err != nil {
			log.Println("Error uploading file for", repository.Name)
			println(err.Error())
			continue
		}

		// Remove file local (even the directories)
		err = os.RemoveAll(strings.Split(localPath, "/")[1])

		if err != nil {
			log.Println("Error removing local file for", repository.Name)
			continue
		}

		// Remove the repository
		err = os.RemoveAll(name)

		err = db.Model(&models.Repository{}).Where("id = ?", repository.ID).Update("last_commit", lastCommit).Error

		if err != nil {
			log.Println("Error updating last commit for", repository.Name)
			continue
		}

		log.Println("Updated", repository.Name)

		time.Sleep(5 * time.Second) // wait 5 seconds to avoid rate limits
	}
}
