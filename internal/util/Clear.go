package util

import "os"

// Clear file system from the splitted path and the repository
func Clear(path string, splittedPath []string) error {
	if err := os.RemoveAll(splittedPath[0]); err != nil {
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}
