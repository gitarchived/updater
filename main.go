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

var ctx = context.Background()

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

	for _, repository := range repositories {
		if repository.Deleted {
			log.Println("Skipping", repository.Name, "because it's deleted")
			continue
		}

		log.Println("Updating", repository.Owner+"/"+repository.Name)

		lastCommit, err := utils.GetLastCommit(repository.Owner, repository.Name)

		if err != nil {
			// Move the repository to the deleted state
			err = db.Model(&models.Repository{}).Where("id = ?", repository.ID).Update("deleted", true).Error

			if err != nil {
				log.Println("Error updating deleted state for", repository.Name)
				continue
			}

			log.Println("Error getting last commit for", repository.Name)
			continue
		}

		if lastCommit == repository.LastCommit {
			log.Println("No new commits for", repository.Name, "skipping...")
			continue
		}

		fullName := repository.Owner + "/" + repository.Name
		cmdClone := exec.Command("git", "clone", "--depth=100", fmt.Sprintf("https://github.com/%s", fullName))

		if err := cmdClone.Run(); err != nil {
			log.Println("Error cloning", fullName)
			continue
		}

		// Create a bunde file
		cmdBundle := exec.Command("git", "bundle", "create", fmt.Sprintf("%d.bundle", repository.ID), "--all")
		cmdBundle.Dir = fmt.Sprintf("./%s", repository.Name)

		if err := cmdBundle.Run(); err != nil {
			log.Println("Error creating bundle for", fullName)
		}

		path := utils.GetSplitPath(repository.Name, repository.ID)
		localPath := fmt.Sprintf("./%s", strings.Join(path, "/"))
		dir := strings.Join(path[:len(path)-1], "/")

		// Save file local
		err = os.MkdirAll(dir, os.ModePerm)

		if err != nil {
			log.Println("Error creating folders for", fullName)
			continue
		}

		// Move the file to the right path
		err = os.Rename(fmt.Sprintf("./%s/%d.bundle", repository.Name, repository.ID), localPath)

		if err != nil {
			log.Println("Error moving file for", fullName)
			continue
		}

		// Upload file to object storage
		_, err = storage.FPutObject(ctx, os.Getenv("STORAGE_BUCKET"), strings.Join(path, "/"), strings.Join(path, "/"), minio.PutObjectOptions{})

		if err != nil {
			log.Println("Error uploading file for", fullName)
			continue
		}

		// Remove file local (even the directories)
		err = os.RemoveAll(strings.Split(localPath, "/")[1])

		if err != nil {
			log.Println("Error removing local file for", fullName)
			continue
		}

		// Remove the repository
		err = os.RemoveAll(repository.Name)

		if err != nil {
			log.Println("Error removing local repository for", fullName)
			continue
		}

		err = db.Model(&models.Repository{}).Where("id = ?", repository.ID).Update("last_commit", lastCommit).Error

		if err != nil {
			log.Println("Error updating last commit for", fullName)
			continue
		}

		log.Println("Updated", fullName)

		time.Sleep(5 * time.Second) // wait 5 seconds to avoid rate limits
	}
}
