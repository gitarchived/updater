package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gitarchived/updater/database"
	"github.com/gitarchived/updater/events"
	"github.com/gitarchived/updater/git"
	"github.com/gitarchived/updater/logger"
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
	useEvents := flag.Bool("events", false, "Use events")

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

	repositories, err := database.GetRepositories(db, host, *useForce)

	if err != nil {
		log.Fatal("Error getting repositories from database")
	}

	log.Info("Starting to update repositories")

	updated := 0

	for _, repo := range repositories {
		r := repo.Repository
		lastCommit := repo.NewCommitHash

		log.Info("Updating", "repository", r.Name)

		path, splittedPath, err := git.BundleRemote(r, host)

		if err != nil {
			logger.HandleError(r, host, err)
			continue
		}

		// Upload file to object storage
		_, err = storage.FPutObject(
			ctx,
			os.Getenv("STORAGE_BUCKET"),
			path,
			path,
			minio.PutObjectOptions{ContentType: "application/octet-stream"},
		)

		if err != nil {
			logger.HandleError(r, host, err)
			continue
		}

		if err := utils.Clear(r.Name, splittedPath); err != nil {
			logger.HandleError(r, host, err)
			continue
		}

		if err := db.Model(&models.Repository{}).Where("id = ?", r.ID).Update("last_commit", lastCommit); err.Error != nil {
			logger.HandleError(r, host, err.Error)
			continue
		}

		log.Info("Updated", "repository", r.Name)

		updated++
		time.Sleep(5 * time.Second) // wait 5 seconds to avoid rate limits
	}

	if *useEvents {
		err := events.PropagateEnd(len(repositories), len(repositories)) // Needs some work from the events api side

		if err != nil {
			log.Error("Error propagating end event", "event", "end")
		}
	}
}
