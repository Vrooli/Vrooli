package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func (a *App) cmdManifestValidate(args []string) error {
	return a.postManifestOnly(args, "manifest-validate", "/api/v1/manifest/validate")
}

func readJSONFile(path string) (map[string]interface{}, error) {
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
