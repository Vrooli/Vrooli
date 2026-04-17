package storage

import (
	"path/filepath"
	"strings"
)

func cleanResourceID(id string) string {
	return strings.TrimSpace(id)
}

func isValidResourceID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
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
	trimmed := strings.TrimSpace(rel)
	if trimmed == "" {
		return filepath.Clean(base), nil
	}
	if filepath.IsAbs(trimmed) {
		return "", &Error{Kind: ErrInvalidInput, Message: "relative path must not be absolute", Details: rel}
	}
	candidate := filepath.Join(base, filepath.Clean(trimmed))
	baseClean := filepath.Clean(base)
	relPath, err := filepath.Rel(baseClean, candidate)
	if err != nil {
		return "", &Error{Kind: ErrResolve, Message: "resolve relative path", Details: rel, Err: err}
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", &Error{Kind: ErrInvalidInput, Message: "relative path escapes storage root", Details: rel}
	}
	return candidate, nil
}
