package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/vrooli/api-core/storage"
)

// GroupingDeps contains dependencies for grouping rule operations.
type GroupingDeps struct {
	AuthContext context.Context
	FS          FileIO
	ConfigPath  string
}

// LoadGroupingRules reads the grouping configuration from disk.
// Returns an empty config (not an error) when the file does not exist.
func LoadGroupingRules(deps GroupingDeps) (*GroupingRulesConfig, error) {
	raw, err := deps.FS.ReadFile(deps.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &GroupingRulesConfig{}, nil
		}
		return nil, err
	}
	var cfg GroupingRulesConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveGroupingRules persists the grouping configuration atomically.
func SaveGroupingRules(deps GroupingDeps, config GroupingRulesConfig) error {
	if err := requireHumanMutation(deps.AuthContext, "save grouping rules"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(deps.ConfigPath, data, 0o644)
}
