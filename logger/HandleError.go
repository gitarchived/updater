package logger

import (
	"github.com/charmbracelet/log"
	"github.com/gitarchived/updater/models"
)

func HandleError(r models.Repository, h models.Host, err error) {
	log.Errorf("Error processing %s/%s: %s", r.Owner, r.Name, err)
}
