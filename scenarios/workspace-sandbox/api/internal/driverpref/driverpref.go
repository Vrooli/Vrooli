package driverpref

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"workspace-sandbox/internal/driverid"
)

const FileName = "driver-preference.json"

var ErrNotFound = errors.New("driver preference not found")

// Preference is the durable launch-time driver preference contract.
type Preference struct {
	DriverID string `json:"driverId"`
}

// Path returns the preference file path under baseDir.
func Path(baseDir string) string {
	return filepath.Join(baseDir, FileName)
}

// Save writes the selected driver preference under baseDir.
func Save(baseDir string, id driverid.ID) error {
	if !driverid.Known(id) {
		return fmt.Errorf("unknown driver ID: %s", id)
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Preference{DriverID: string(id)}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(baseDir), data, 0o644)
}

// Load reads and validates the selected driver preference under baseDir.
func Load(baseDir string) (driverid.ID, error) {
	data, err := os.ReadFile(Path(baseDir))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", err
	}
	var pref Preference
	if err := json.Unmarshal(data, &pref); err != nil {
		return "", fmt.Errorf("parse %s: %w", Path(baseDir), err)
	}
	id, err := driverid.Parse(pref.DriverID)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", Path(baseDir), err)
	}
	return id, nil
}
