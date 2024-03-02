package database

import (
	"github.com/gitarchived/updater/git"
	"github.com/gitarchived/updater/models"
	"gorm.io/gorm"
)

type Repository struct {
	models.Repository
	NewCommitHash string
}

func GetRepositories(db *gorm.DB, host models.Host, force bool) ([]Repository, error) {
	var repositories []models.Repository
	var results []Repository

	data := db.Where("host = ? AND deleted = ?", host.Name, false).Find(&repositories)

	for _, r := range repositories {
		commit, err := git.GetLastCommit(r, host)

		if err != nil {
			if err := db.Model(&models.Repository{}).Where("id = ?", r.ID).Update("deleted", true); err.Error != nil {
				continue
			}
		}

		if commit != r.LastCommit || force {
			results = append(results, Repository{Repository: r, NewCommitHash: commit})
		}
	}

	return results, data.Error
}
