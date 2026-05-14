package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-component-library/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end: writes
// the manifest to a temp file, reads it back, asserts it's valid JSON
// with the canonical envelope shape.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	seed := filepath.Join(t.TempDir(), "seed.json")
	writeSeed(t, seed, []CLICommand{
		{Name: "status", Description: "Health check", EndpointID: "health"},
		{Name: "adoptions list", Description: "List adoptions", EndpointID: "adoptions_list"},
		{Name: "adoptions create", Description: "Create adoption", EndpointID: "adoptions_create"},
		{Name: "adoptions delete", Description: "Delete adoption", EndpointID: "adoptions_delete"},
		{Name: "adoptions refresh", Description: "Refresh adoptions", EndpointID: "adoptions_refresh"},
		{Name: "components index", Description: "Index components", EndpointID: "components_index"},
		{Name: "components list", Description: "List components", EndpointID: "components_list"},
		{Name: "components get", Description: "Get component", EndpointID: "components_get"},
		{Name: "components get-by-library-id", Description: "Get by libraryId", EndpointID: "components_get_by_library_id"},
		{Name: "components content-get", Description: "Read content", EndpointID: "components_content_get"},
		{Name: "components content-set", Description: "Write content", EndpointID: "components_content_set"},
		{Name: "deps list", Description: "List declarations", EndpointID: "deps_list_declarations"},
		{Name: "deps validate", Description: "Validate adoption", EndpointID: "deps_validate_adoption"},
		{Name: "preview bundle", Description: "Bundle for preview", EndpointID: "preview_get_bundle"},
		{Name: "themes list-builtin", Description: "List built-in themes", EndpointID: "themes_list_builtin"},
		{Name: "themes get-builtin", Description: "Get built-in theme", EndpointID: "themes_get_builtin"},
		{Name: "themes get-from-scenario", Description: "Resolve from scenario", EndpointID: "themes_get_from_scenario"},
		{Name: "versions list", Description: "List versions", EndpointID: "versions_list"},
		{Name: "versions show", Description: "Show version", EndpointID: "versions_get"},
		{Name: "versions diff", Description: "Diff versions", EndpointID: "versions_diff"},
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
	if len(got.CLICommands) != 20 {
		t.Errorf("cli_commands count = %d, want 20", len(got.CLICommands))
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
				Command: "react-component-library lonely subcommand",
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
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "react-component-library x"}},
		{ID: "y_no_cli"}, // no CLIMapping — must be allowed
	}
	commands := []CLICommand{{Name: "x", EndpointID: "x"}}

	if err := crossCheck(endpoints, commands); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "react-component-library components list"
// must compare against the seed's "components list".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "react-component-library status", want: "status"},
		{in: "react-component-library components list", want: "components list"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "react-component-library", want: "react-component-library"}, // no trailing space → preserved
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
