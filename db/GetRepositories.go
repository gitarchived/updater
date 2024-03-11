package db

import (
	"github.com/gitarchived/updater/data"
	"github.com/gitarchived/updater/internal/git"
	"gorm.io/gorm"
)

type Repository struct {
	data.Repository
	NewCommitHash string
}

func GetRepositories(db *gorm.DB, host data.Host, force bool) ([]Repository, error) {
	var repositories []data.Repository
	var results []Repository

	res := db.Where("host = ? AND deleted = ?", host.Name, false).Find(&repositories)

	for _, r := range repositories {
		commit, err := git.GetLastCommit(r, host)

		if err != nil {
			if err := db.Model(&data.Repository{}).Where("id = ?", r.ID).Update("deleted", true); err.Error != nil {
				continue
			}
		}

		if commit != r.LastCommit || force {
			results = append(results, Repository{Repository: r, NewCommitHash: commit})
		}
	}

	return results, res.Error
}
