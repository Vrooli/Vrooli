package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quality-health/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end: writes
// the manifest to a temp file, reads it back, asserts it's valid JSON
// with the canonical envelope shape.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	seed := filepath.Join(t.TempDir(), "seed.json")
	writeSeed(t, seed, []CLICommand{
		{Name: "status", Description: "Health check", EndpointID: "health"},
		{Name: "audit run", Description: "Run audit", EndpointID: "audit_run"},
		{Name: "contracts list", Description: "List contracts", EndpointID: "contracts_list"},
		{Name: "explain finding", Description: "Explain finding", EndpointID: "explain_finding"},
		{Name: "fix-config run", Description: "Preview config fix", EndpointID: "fix_config_preview"},
		{Name: "fix-config apply", Description: "Apply config fix", EndpointID: "fix_config_apply"},
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
	if len(got.CLICommands) != 6 {
		t.Errorf("cli_commands count = %d, want 6", len(got.CLICommands))
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
				Command: "quality-health lonely subcommand",
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
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "quality-health x"}},
		{ID: "y_no_cli"}, // no CLIMapping — must be allowed
	}
	commands := []CLICommand{{Name: "x", EndpointID: "x"}}

	if err := crossCheck(endpoints, commands); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "quality-health audit run"
// must compare against the seed's "audit run".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "quality-health status", want: "status"},
		{in: "quality-health audit run", want: "audit run"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "quality-health", want: "quality-health"}, // no trailing space → preserved
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
