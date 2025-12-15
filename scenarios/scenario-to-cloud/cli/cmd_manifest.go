package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdManifestValidate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud manifest-validate <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	payload, err := a.api.Request("POST", "/api/v1/manifest/validate", nil, manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
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
