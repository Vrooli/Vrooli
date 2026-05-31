package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"development-toolchain-validator/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end: writes
// the manifest to a temp file, reads it back, asserts it's valid JSON
// with the canonical envelope shape.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	seed := filepath.Join(t.TempDir(), "seed.json")
	writeSeed(t, seed, []CLICommand{
		{Name: "status", Description: "Health check", EndpointID: "health"},
		{Name: "goldens list", Description: "List goldens", EndpointID: "golden_list"},
		{Name: "goldens get", Description: "Get golden", EndpointID: "golden_get"},
		{Name: "goldens register", Description: "Register golden", EndpointID: "golden_register"},
		{Name: "goldens update", Description: "Update golden", EndpointID: "golden_update"},
		{Name: "goldens delete", Description: "Delete golden", EndpointID: "golden_delete"},
		{Name: "goldens regenerate", Description: "Regenerate golden", EndpointID: "golden_regenerate"},
		{Name: "manifest list", Description: "List manifests", EndpointID: "manifest_list"},
		{Name: "manifest get", Description: "Get manifest", EndpointID: "manifest_get"},
		{Name: "manifest upsert", Description: "Upsert manifest", EndpointID: "manifest_upsert"},
		{Name: "manifest clear-stale", Description: "Clear stale", EndpointID: "manifest_clear_stale"},
		{Name: "skill-catalog sync", Description: "Sync skill catalog", EndpointID: "skill_catalog_sync"},
		{Name: "skill-catalog list", Description: "List skills", EndpointID: "skill_catalog_list"},
		{Name: "skill-catalog get", Description: "Get skill", EndpointID: "skill_catalog_get"},
		{Name: "staleness list", Description: "List stale", EndpointID: "staleness_list"},
		{Name: "record list", Description: "List records", EndpointID: "validation_record_list"},
		{Name: "record get", Description: "Get record", EndpointID: "validation_record_get"},
		{Name: "validation start", Description: "Start validation", EndpointID: "validation_run_start"},
		{Name: "validation get", Description: "Get validation", EndpointID: "validation_run_get"},
		{Name: "validation list-active", Description: "List active runs", EndpointID: "validation_run_list_active"},
		{Name: "report golden-summary", Description: "Summary", EndpointID: "report_golden_summary"},
		{Name: "report tuple-history", Description: "History", EndpointID: "report_tuple_history"},
		{Name: "report coverage", Description: "Coverage", EndpointID: "report_coverage"},
		{Name: "report skill-fitness", Description: "Skill fitness", EndpointID: "report_skill_fitness"},
	})

	if err := run(output, seed); err != nil {
		t.Fatalf("run: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	var got struct {
		Schema      string                      `json:"$schema"`
		Version     string                      `json:"version"`
		Endpoints   []module.EndpointDescriptor `json:"endpoints"`
		CLICommands []CLICommand                `json:"cli_commands"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal output: %v\nbody=%s", err, string(data))
	}

	if got.Schema == "" {
		t.Error("manifest must include $schema")
	}
	if got.Version != manifestVersion {
		t.Errorf("version = %q, want %q", got.Version, manifestVersion)
	}
	if len(got.Endpoints) == 0 {
		t.Error("manifest must include at least one endpoint")
	}
	if len(got.CLICommands) != 24 {
		t.Errorf("cli_commands count = %d, want 24", len(got.CLICommands))
	}

	// Trailing newline so editors don't get angry about diff noise.
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("manifest must end with a newline")
	}
}

// TestCrossCheck_FailsOnUnseededCommand pins the codegen-time guard:
// an endpoint whose cli_mapping.command isn't in cli_commands_seed
// fails fast with a message that names the missing entry.
func TestCrossCheck_FailsOnUnseededCommand(t *testing.T) {
	endpoints := []module.EndpointDescriptor{
		{
			ID: "lonely",
			CLIMapping: &module.CLIMapping{
				Command: "development-toolchain-validator lonely subcommand",
			},
		},
	}
	commands := []CLICommand{} // empty seed

	err := crossCheck(endpoints, commands)
	if err == nil {
		t.Fatal("expected crossCheck to fail when cli_commands seed is missing the entry")
	}
	if !strings.Contains(err.Error(), "lonely subcommand") {
		t.Errorf("error %q must name the missing command", err.Error())
	}
	if !strings.Contains(err.Error(), `endpoint "lonely"`) {
		t.Errorf("error %q must name the offending endpoint", err.Error())
	}
}

// TestCrossCheck_PassesWhenSeeded confirms the happy path.
func TestCrossCheck_PassesWhenSeeded(t *testing.T) {
	endpoints := []module.EndpointDescriptor{
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator x"}},
		{ID: "y_no_cli"}, // no CLIMapping — must be allowed
	}
	commands := []CLICommand{{Name: "x", EndpointID: "x"}}

	if err := crossCheck(endpoints, commands); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "development-toolchain-validator notes list"
// must compare against the seed's "notes list".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "development-toolchain-validator status", want: "status"},
		{in: "development-toolchain-validator notes list", want: "notes list"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "development-toolchain-validator", want: "development-toolchain-validator"}, // no trailing space → preserved
	}
	for _, tc := range cases {
		if got := stripBinaryPrefix(tc.in); got != tc.want {
			t.Errorf("stripBinaryPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func writeSeed(t *testing.T, path string, commands []CLICommand) {
	t.Helper()
	body, err := json.MarshalIndent(seedFile{CLICommands: commands}, "", "  ")
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
}
