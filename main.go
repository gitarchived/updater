package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gitarchived/updater/git"
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
	useEnv := flag.Bool("env", false, "Use .env file")
	useForce := flag.Bool("force", false, "Force update")
	useNoSSL := flag.Bool("no-ssl", false, "Use no SSL")

	flag.Parse()

	if *useEnv {
		err := godotenv.Load()

		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	storage, err := minio.New(os.Getenv("STORAGE_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("STORAGE_KEY"), os.Getenv("STORAGE_SECRET"), ""),
		Secure: !*useNoSSL,
	})

	if err != nil {
		log.Fatal("Error creating Object Storage client")
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("PG_URL")), &gorm.Config{})

	if err != nil {
		log.Fatal("Error connecting to PostgreSQL")
	}

	// Get `HOST` from the database
	hostName := os.Getenv("HOST")

	var host models.Host

	hostQuery := db.Where("name = ?", hostName).First(&host)

	if hostQuery.Error != nil {
		log.Fatal("Error getting host from database")
	}

	var repositories []models.Repository

	result := db.Where("host = ?", host.Name).Find(&repositories)

	log.Info("Starting to update repositories", "repositories", result.RowsAffected)

	for _, r := range repositories {
		if r.Deleted {
			log.Warn("Skipping, repository is deleted", "repository", r.Name)
			continue
		}

		log.Info("Updating", "repository", r.Name)

		lastCommit, err := git.GetLastCommit(r, host)

		if err != nil {
			// Move the repository to the deleted state
			err = db.Model(&models.Repository{}).Where("id = ?", r.ID).Update("deleted", true).Error

			if err != nil {
				log.Error("Error updating deleted state", "repository", r.Name)
				continue
			}

			log.Error("Error getting last commit", "repository", r.Name)
			continue
		}

		if !*useForce && lastCommit == r.LastCommit {
			log.Warn("Skipping, no new commits", "repository", r.Name)
			continue
		}

		_, err = git.BundleRemote(r, host)
		splittedPath := utils.GetSplitPath(r.Name, r.ID)

		if err != nil {
			log.Error("Error creating bundle", "repository", r.Name)
			continue
		}

		// Upload file to object storage
		_, err = storage.FPutObject(
			ctx,
			os.Getenv("STORAGE_BUCKET"),
			strings.Join(splittedPath, "/"),
			strings.Join(splittedPath, "/"),
			minio.PutObjectOptions{ContentType: "application/octet-stream"},
		)

		if err != nil {
			log.Error("Error uploading bundle to object storage", "repository", r.Name)
			continue
		}

		// Remove file local (even the directories)
		err = os.RemoveAll(splittedPath[0])

		if err != nil {
			println(err.Error())
			log.Error("Error removing local bundle", "repository", r.Name)
			continue
		}

		// Remove the repository
		err = os.RemoveAll(r.Name)

		if err != nil {
			log.Error("Error removing local repository", "repository", r.Name)
			continue
		}

		err = db.Model(&models.Repository{}).Where("id = ?", r.ID).Update("last_commit", lastCommit).Error

		if err != nil {
			log.Error("Error updating last commit", "repository", r.Name)
			continue
		}

		log.Info("Updated", "repository", r.Name)

		time.Sleep(5 * time.Second) // wait 5 seconds to avoid rate limits
	}
}
