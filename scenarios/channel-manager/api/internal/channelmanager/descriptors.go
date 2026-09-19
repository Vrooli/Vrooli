package channelmanager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadDescriptors makes the JSON files the product's source of truth. The
// same loader accepts any additional descriptor placed in either directory.
func LoadDescriptors(root string) ([]Platform, []Program, error) {
	platforms, err := loadJSON[Platform](filepath.Join(root, "platforms"))
	if err != nil {
		return nil, nil, err
	}
	programs, err := loadJSON[Program](filepath.Join(root, "warming-programs"))
	if err != nil {
		return nil, nil, err
	}
	return platforms, programs, nil
}

func loadJSON[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		b, e := os.ReadFile(filepath.Join(dir, entry.Name()))
		if e != nil {
			return nil, e
		}
		var item T
		if e = json.Unmarshal(b, &item); e != nil {
			return nil, fmt.Errorf("decode %s: %w", entry.Name(), e)
		}
		out = append(out, item)
	}
	return out, nil
}
