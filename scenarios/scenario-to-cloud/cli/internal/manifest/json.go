// Package manifest provides shared manifest utilities for CLI domain packages.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadJSONFile reads and parses a JSON file into a generic map.
// Used by CLI commands that need to send manifest data to the API.
func ReadJSONFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}
