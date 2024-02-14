package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gitarchived/updater/models"
)

func GetLastCommit(r models.Repository, h models.Host) (string, error) {
	url := fmt.Sprintf("%s%s/%s.git", h.Prefix, r.Owner, r.Name)
	lsRemoteCmd := exec.Command("git", "ls-remote", url, "HEAD")

	out, err := lsRemoteCmd.Output()

	if err != nil {
		return "", err
	}

	commit := strings.Fields(string(out))[0]

	return commit, nil
}
