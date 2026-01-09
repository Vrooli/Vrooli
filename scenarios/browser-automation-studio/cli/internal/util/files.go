package util

import (
	"fmt"
	"path/filepath"
	"strings"
)

func CleanPlaybookFolder(input string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return "", fmt.Errorf("folder must not be empty")
	}

	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("folder must not contain '..' segments")
		}
	}
	return filepath.ToSlash(filepath.Clean(trimmed)), nil
}
