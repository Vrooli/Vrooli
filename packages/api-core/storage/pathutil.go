package storage

import (
	"path/filepath"
	"strings"
)

func cleanScenarioID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/\\")
	return s
}

func isValidScenarioID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

func cleanJoin(base, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", &Error{Kind: ErrInvalidInput, Message: "relative path must not be absolute", Details: rel}
	}
	cleaned := filepath.Clean(rel)
	if cleaned == "." {
		return filepath.Clean(base), nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", &Error{Kind: ErrInvalidInput, Message: "relative path must stay within class root", Details: rel}
	}
	joined := filepath.Join(base, cleaned)
	return joined, nil
}
