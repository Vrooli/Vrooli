package modelregistry

import (
	"path/filepath"
	"testing"
)

func sampleRegistry() *Registry {
	return &Registry{
		Version: 1,
		Runners: map[string]RunnerModelRegistry{
			"codex": {
				Models: []ModelOption{
					{ID: "gpt-5-codex"},
					{ID: "gpt-5-mini", Description: "Fast model"},
				},
				Presets: map[string]string{
					"FAST":  "gpt-5-mini",
					"SMART": "gpt-5-codex",
				},
			},
		},
	}
}

func TestRegistryValidate(t *testing.T) {
	reg := sampleRegistry()
	if err := reg.Validate(); err != nil {
		t.Fatalf("expected registry to validate, got %v", err)
	}
}

func TestRegistryValidate_InvalidPreset(t *testing.T) {
	reg := sampleRegistry()
	reg.Runners["codex"] = RunnerModelRegistry{
		Models: []ModelOption{{ID: "model-a"}},
		Presets: map[string]string{
			"INVALID": "model-a",
		},
	}
	if err := reg.Validate(); err == nil {
		t.Fatalf("expected validation error for invalid preset")
	}
}

func TestRegistrySaveLoad(t *testing.T) {
	reg := sampleRegistry()
	path := filepath.Join(t.TempDir(), "registry.json")

	if err := Save(path, reg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Version != reg.Version {
		t.Fatalf("expected version %d, got %d", reg.Version, loaded.Version)
	}
	if _, ok := loaded.Runners["codex"]; !ok {
		t.Fatalf("expected runner codex in loaded registry")
	}
}
