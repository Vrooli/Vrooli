// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
// Package storage provides shared filesystem utilities for JSON persistence.
//
// This package centralizes atomic read/write behaviors so handlers can
// persist data safely without duplicating file IO logic.
package storage

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// ReadJSONBytes reads raw JSON file bytes.
func ReadJSONBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ReadJSON reads a JSON file into the provided destination.
// Returns (false, nil) when the file does not exist.
func ReadJSON(path string, dest any) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return true, err
	}
	return true, nil
}

// WriteJSONAtomic writes JSON to disk using a temp file and atomic rename.
// Ensures the parent directory exists before writing.
func WriteJSONAtomic(path string, data any) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o750); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(parentDir, "tmp-*.json")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	defer func() {
		if rmErr := os.Remove(tempName); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Debug("storage: remove temp file failed", "err", rmErr, "path", tempName)
		}
	}()

	if _, err := tempFile.Write(bytes); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			slog.Debug("storage: close temp file failed", "err", closeErr)
		}
		return err
	}
	if err := tempFile.Sync(); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			slog.Debug("storage: close temp file failed", "err", closeErr)
		}
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tempName, 0o600); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}
