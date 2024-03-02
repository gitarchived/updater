package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gitarchived/updater/models"
	"github.com/gitarchived/updater/utils"
)

func BundleRemote(r models.Repository, h models.Host) (string, []string, error) {
	url := fmt.Sprintf("%s%s/%s.git", h.Prefix, r.Owner, r.Name)
	cloneCmd := exec.Command("git", "clone", "--depth=100", url)

	if err := cloneCmd.Run(); err != nil {
		return "", nil, err
	}

	// Create a bunde file
	bundleCmd := exec.Command("git", "bundle", "create", fmt.Sprintf("%d.bundle", r.ID), "HEAD")
	bundleCmd.Dir = fmt.Sprintf("./%s", r.Name)

	if err := bundleCmd.Run(); err != nil {
		return "", nil, err
	}

	path := utils.GetSplitPath(r.Name, r.ID)
	localPath := fmt.Sprintf("./%s", strings.Join(path, "/"))
	dir := strings.Join(path[:len(path)-1], "/")

	// Save file local
	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		return "", nil, err
	}

	// Move the file to the right path
	err = os.Rename(fmt.Sprintf("./%s/%d.bundle", r.Name, r.ID), localPath)

	if err != nil {
		return "", nil, err
	}

	// Why not return localPath? It's because S3 dosn't support ./ or ../ similiar symbols in front of the path
	// https://stackoverflow.com/questions/30518899/amazon-s3-how-to-fix-the-request-signature-we-calculated-does-not-match-the-s
	return strings.Join(path, "/"), path, nil
}
