package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vrooli/api-core/storage"
)

func saveJSONStore(path, kind string, value any) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s data: %w", kind, err)
	}
	if err := storage.WriteFileAtomic(path, data, storage.DefaultFilePerm); err != nil {
		return fmt.Errorf("failed to write %s data to %s: %w", kind, path, err)
	}
	return nil
}

func loadJSONStore[T any](path, kind string) (T, error) {
	var value T
	if path == "" {
		return value, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return value, nil
		}
		return value, fmt.Errorf("failed to read %s data from %s: %w", kind, path, err)
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("failed to unmarshal %s data from %s: %w", kind, path, err)
	}
	return value, nil
}
