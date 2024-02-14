package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gitarchived/updater/models"
	"github.com/gitarchived/updater/utils"
)

func BundleRemote(r models.Repository, h models.Host) (string, error) {
	url := fmt.Sprintf("%s%s/%s.git", h.Prefix, r.Owner, r.Name)
	cloneCmd := exec.Command("git", "clone", "--depth=100", url)

	if err := cloneCmd.Run(); err != nil {
		return "", err
	}

	// Create a bunde file
	bundleCmd := exec.Command("git", "bundle", "create", fmt.Sprintf("%d.bundle", r.ID), "HEAD")
	bundleCmd.Dir = fmt.Sprintf("./%s", r.Name)

	if err := bundleCmd.Run(); err != nil {
		return "", err
	}

	path := utils.GetSplitPath(r.Name, r.ID)
	localPath := fmt.Sprintf("./%s", strings.Join(path, "/"))
	dir := strings.Join(path[:len(path)-1], "/")

	// Save file local
	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		return "", err
	}

	// Move the file to the right path
	err = os.Rename(fmt.Sprintf("./%s/%d.bundle", r.Name, r.ID), localPath)

	if err != nil {
		return "", err
	}

	return localPath, nil
}
