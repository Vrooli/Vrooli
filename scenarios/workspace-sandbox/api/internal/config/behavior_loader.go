package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// behaviorFileShape mirrors only the subset of `<scenarioDir>/.vrooli/config.json`
// that this package cares about. Other top-level keys (environment, etc.)
// are intentionally ignored at the loader boundary.
type behaviorFileShape struct {
	Behavior *BehaviorConfig `json:"behavior,omitempty"`
}

// LoadBehavior reads `<scenarioDir>/.vrooli/config.json` and returns the
// parsed `behavior` block merged onto DefaultBehavior().
//
// Greenfield contract: this function never panics, never bubbles up a
// "config missing" error — a missing file or missing `behavior` key both
// resolve to DefaultBehavior(). Only malformed JSON (file present + parse
// error) returns a non-nil error so startup can surface it loudly.
func LoadBehavior(scenarioDir string) (BehaviorConfig, error) {
	defaults := DefaultBehavior()
	if scenarioDir == "" {
		return defaults, nil
	}
	path := filepath.Join(scenarioDir, ".vrooli", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return defaults, nil
	}
	var shape behaviorFileShape
	if err := json.Unmarshal(data, &shape); err != nil {
		return defaults, fmt.Errorf("parse %s: %w", path, err)
	}
	if shape.Behavior == nil {
		return defaults, nil
	}
	return mergeBehavior(defaults, *shape.Behavior), nil
}

// mergeBehavior returns the result of layering `override` onto `base`. The
// merge rule is "non-zero override fields win" — empty slices and empty
// strings in the override leave the base value untouched. This keeps the
// JSON file additive: an operator can specify just the field they want to
// change without restating the rest.
func mergeBehavior(base, override BehaviorConfig) BehaviorConfig {
	merged := base
	if len(override.Protected.GitAllowlist) > 0 {
		merged.Protected.GitAllowlist = append([]string(nil), override.Protected.GitAllowlist...)
	}
	if override.Protected.GitDenyMessageTemplate != "" {
		merged.Protected.GitDenyMessageTemplate = override.Protected.GitDenyMessageTemplate
	}
	if override.Protected.GitNoVerbMessageTemplate != "" {
		merged.Protected.GitNoVerbMessageTemplate = override.Protected.GitNoVerbMessageTemplate
	}
	return merged
}
