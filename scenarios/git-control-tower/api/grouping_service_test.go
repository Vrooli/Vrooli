package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGroupingRules_FileExists(t *testing.T) {
	cfg := GroupingRulesConfig{
		Enabled: true,
		Rules: []GroupingRule{
			{ID: "r1", Label: "Resources", Prefixes: []string{"resources/"}, Mode: "prefix"},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	fs := NewFakeFileIO().WithFile("/config/grouping-rules.json", string(data))
	deps := GroupingDeps{FS: fs, ConfigPath: "/config/grouping-rules.json"}

	result, err := LoadGroupingRules(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Enabled {
		t.Fatal("expected enabled=true")
	}
	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}
	if result.Rules[0].ID != "r1" {
		t.Fatalf("expected rule ID r1, got %s", result.Rules[0].ID)
	}
}

func TestLoadGroupingRules_FileNotFound(t *testing.T) {
	fs := NewFakeFileIO()
	deps := GroupingDeps{FS: fs, ConfigPath: "/config/nonexistent.json"}

	result, err := LoadGroupingRules(deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Enabled {
		t.Fatal("expected enabled=false for empty config")
	}
	if len(result.Rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(result.Rules))
	}
}

func TestSaveGroupingRules_Writes(t *testing.T) {
	// Uses a real temp dir since SaveGroupingRules delegates to storage.WriteFileAtomic.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "grouping-rules.json")

	deps := GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath}
	cfg := GroupingRulesConfig{
		Enabled: true,
		Rules: []GroupingRule{
			{ID: "s1", Label: "Scenarios", Prefixes: []string{"scenarios/"}, Mode: "segment"},
		},
	}

	if err := SaveGroupingRules(deps, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}

	var loaded GroupingRulesConfig
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !loaded.Enabled {
		t.Fatal("expected enabled=true")
	}
	if len(loaded.Rules) != 1 || loaded.Rules[0].ID != "s1" {
		t.Fatalf("unexpected rules: %+v", loaded.Rules)
	}
}

func TestLoadGroupingRules_InvalidJSON(t *testing.T) {
	fs := NewFakeFileIO().WithFile("/config/grouping-rules.json", "not valid json{{{")
	deps := GroupingDeps{FS: fs, ConfigPath: "/config/grouping-rules.json"}

	_, err := LoadGroupingRules(deps)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
