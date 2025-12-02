// Package fileutil provides file utility functions.
package fileutil

import (
	"strings"

	"scenario-to-desktop-runtime/infra"
)

// TailFile returns the last N lines from a file.
func TailFile(fs infra.FileSystem, path string, lines int) ([]byte, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "\n")
	if lines >= len(parts) {
		return []byte(strings.Join(parts, "\n")), nil
	}
	return []byte(strings.Join(parts[len(parts)-lines:], "\n")), nil
}
