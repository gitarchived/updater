package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gitarchived/updater/data"
	database "github.com/gitarchived/updater/db"
	"github.com/gitarchived/updater/internal/git"
	"github.com/gitarchived/updater/internal/logger"
	"github.com/gitarchived/updater/internal/util"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	db := database.Create()

	// Get `HOST` from the database
	hostName := os.Getenv("HOST")

	var host data.Host

	hostQuery := db.Where("name = ?", hostName).First(&host)

	if hostQuery.Error != nil {
		log.Fatal("Error getting host from database")
	}

	log.Info("Checking host connectivity", "host", host.Name)

	client := resty.New()
	resp, err := client.R().
		Get(host.URL)

	if err != nil || resp.StatusCode() != 200 {
		log.Fatal("Error getting host", "host", host.Name)
		os.Exit(1)
	}

	repositories, err := database.GetRepositories(db, host, *useForce)

	if err != nil {
		log.Fatal("Error getting repositories from database")
	}

	log.Info("Starting to update repositories")

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

		if err := util.Clear(r.Name, splittedPath); err != nil {
			logger.HandleError(r, host, err)
			continue
		}

		if err := db.Model(&data.Repository{}).Where("id = ?", r.ID).Update("last_commit", lastCommit); err.Error != nil {
			logger.HandleError(r, host, err.Error)
			continue
		}

		log.Info("Updated", "repository", r.Name)

		time.Sleep(10 * time.Second) // wait 5 seconds to avoid rate limits
	}
}
