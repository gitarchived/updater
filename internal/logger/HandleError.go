package logger

import (
	"github.com/charmbracelet/log"
	"github.com/gitarchived/updater/data"
)

func HandleError(r data.Repository, h data.Host, err error) {
	log.Errorf("Error processing %s/%s: %s", r.Owner, r.Name, err)
}
