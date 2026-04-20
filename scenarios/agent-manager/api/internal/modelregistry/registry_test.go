package modelregistry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
				Presets: map[string]PresetChain{
					"FAST":  {"gpt-5-mini", ""},
					"SMART": {"gpt-5-codex", "gpt-5-mini", ""},
					"CHEAP": {"gpt-5-mini"},
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

func TestRegistryValidate_RejectsInvalidPresetKey(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["INVALID"] = PresetChain{"gpt-5-mini"}
	reg.Runners["codex"] = runner
	if err := reg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid preset key")
	}
}

func TestRegistryValidate_RejectsEmptyChain(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["FAST"] = PresetChain{}
	reg.Runners["codex"] = runner
	if err := reg.Validate(); err == nil {
		t.Fatal("expected validation error for empty preset chain")
	}
}

func TestRegistryValidate_RejectsChainWithOnlySentinel(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["FAST"] = PresetChain{""}
	reg.Runners["codex"] = runner
	err := reg.Validate()
	if err == nil {
		t.Fatal("expected validation error for sentinel-only chain")
	}
	if !strings.Contains(err.Error(), "at least one concrete") {
		t.Fatalf("expected 'at least one concrete' error, got %v", err)
	}
}

func TestRegistryValidate_RejectsDuplicateEntry(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["FAST"] = PresetChain{"gpt-5-mini", "gpt-5-mini"}
	reg.Runners["codex"] = runner
	if err := reg.Validate(); err == nil {
		t.Fatal("expected validation error for duplicate entry")
	}
}

func TestRegistryValidate_RejectsUnknownModelInChain(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["FAST"] = PresetChain{"does-not-exist"}
	reg.Runners["codex"] = runner
	if err := reg.Validate(); err == nil {
		t.Fatal("expected validation error for unknown model id")
	}
}

func TestRegistryValidate_RejectsSentinelInMiddle(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["FAST"] = PresetChain{"gpt-5-mini", "", "gpt-5-codex"}
	reg.Runners["codex"] = runner
	err := reg.Validate()
	if err == nil {
		t.Fatal("expected validation error for sentinel in middle")
	}
	if !strings.Contains(err.Error(), "final entry") {
		t.Fatalf("expected 'final entry' error, got %v", err)
	}
}

func TestRegistryValidate_RejectsCheapSentinel(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["CHEAP"] = PresetChain{"gpt-5-mini", ""}
	reg.Runners["codex"] = runner
	err := reg.Validate()
	if err == nil {
		t.Fatal("expected validation error for CHEAP with sentinel tail")
	}
	if !strings.Contains(err.Error(), "CHEAP") {
		t.Fatalf("expected CHEAP in error, got %v", err)
	}
}

func TestRegistryValidate_RejectsWhitespaceInEntry(t *testing.T) {
	reg := sampleRegistry()
	runner := reg.Runners["codex"]
	runner.Presets["FAST"] = PresetChain{" gpt-5-mini"}
	reg.Runners["codex"] = runner
	if err := reg.Validate(); err == nil {
		t.Fatal("expected validation error for whitespace-padded entry")
	}
}

func TestPresetChain_UnmarshalJSON_RejectsScalarString(t *testing.T) {
	var chain PresetChain
	err := json.Unmarshal([]byte(`"gpt-5-codex"`), &chain)
	if err == nil {
		t.Fatal("expected scalar string to be rejected")
	}
	if !strings.Contains(err.Error(), "arrays of model IDs") {
		t.Fatalf("expected legacy-shape error message, got %v", err)
	}
}

func TestPresetChain_UnmarshalJSON_AcceptsArray(t *testing.T) {
	var chain PresetChain
	if err := json.Unmarshal([]byte(`["a", "b", ""]`), &chain); err != nil {
		t.Fatalf("expected array to parse, got %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(chain))
	}
}

func TestRegistryLoad_RejectsLegacyScalarPreset(t *testing.T) {
	legacy := `{
		"version": 1,
		"runners": {
			"codex": {
				"models": [{"id": "gpt-5-codex"}],
				"presets": {"SMART": "gpt-5-codex"}
			}
		}
	}`
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected Load to reject legacy scalar preset shape")
	}
}

func TestPresetChain_PrimaryAndAccessors(t *testing.T) {
	empty := PresetChain{}
	if empty.Primary() != "" {
		t.Fatalf("expected empty primary, got %q", empty.Primary())
	}
	if empty.AllowRunnerDefault() {
		t.Fatal("empty chain should not allow runner default")
	}

	chain := PresetChain{"a", "b", ""}
	if chain.Primary() != "a" {
		t.Fatalf("expected primary 'a', got %q", chain.Primary())
	}
	if !chain.AllowRunnerDefault() {
		t.Fatal("expected AllowRunnerDefault true for chain with trailing empty entry")
	}
	concrete := chain.ConcreteModels()
	if len(concrete) != 2 || concrete[0] != "a" || concrete[1] != "b" {
		t.Fatalf("ConcreteModels = %v", concrete)
	}

	noParachute := PresetChain{"only"}
	if noParachute.AllowRunnerDefault() {
		t.Fatal("single-entry chain must not allow runner default")
	}

	if _, ok := chain.At(-1); ok {
		t.Fatal("At(-1) should return false")
	}
	if v, ok := chain.At(2); !ok || v != "" {
		t.Fatalf("At(2) = (%q, %v), expected (\"\", true)", v, ok)
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
	runner, ok := loaded.Runners["codex"]
	if !ok {
		t.Fatal("expected runner codex in loaded registry")
	}
	smart := runner.Presets["SMART"]
	if len(smart) != 3 || smart[0] != "gpt-5-codex" || smart[2] != "" {
		t.Fatalf("SMART chain round-trip mismatch: %v", smart)
	}
}

func TestResolvePresetReturnsChain(t *testing.T) {
	reg := sampleRegistry()
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := Save(path, reg); err != nil {
		t.Fatalf("save: %v", err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	chain, ok := store.ResolvePreset("codex", "SMART")
	if !ok {
		t.Fatal("expected SMART preset to resolve")
	}
	if chain.Primary() != "gpt-5-codex" {
		t.Fatalf("expected primary gpt-5-codex, got %q", chain.Primary())
	}
	if len(chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(chain))
	}

	// Defensive copy: mutating the returned chain must not affect the store.
	chain[0] = "tampered"
	again, _ := store.ResolvePreset("codex", "SMART")
	if again[0] == "tampered" {
		t.Fatal("ResolvePreset returned a reference to internal state")
	}

	if _, ok := store.ResolvePreset("codex", "UNKNOWN"); ok {
		t.Fatal("expected UNKNOWN preset to miss")
	}
	if _, ok := store.ResolvePreset("not-a-runner", "SMART"); ok {
		t.Fatal("expected unknown runner to miss")
	}
}
