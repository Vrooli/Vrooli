// Package fileutil provides file utility functions.
package fileutil

import (
	"strings"

	"scenario-to-desktop-runtime/infra"
)

// TailFile returns the last N lines from a file.
func TailFile(fs infra.FileSystem, path string, lines int) ([]byte, error) {
	if lines <= 0 {
		return []byte{}, nil
	}

	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if lines >= len(parts) {
		return []byte(strings.Join(parts, "\n")), nil
	}
	return []byte(strings.Join(parts[len(parts)-lines:], "\n")), nil
}
