package util

import (
	"fmt"
	"strings"
)

func GetSplitPath(name string, id uint) []string {
	path := strings.Split(name, "")
	path = append(path, fmt.Sprintf("%d.bundle", id))

	for i, letter := range path {
		if letter == "." {
			path[i] = "-"
		}
	}

	return path
}
