package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"device-sync-hub/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end against this
// scenario's REAL committed cli/cli-commands.gen.json (not a synthetic
// fixture), so it stays scenario-agnostic: it proves the generator runs,
// crossCheck passes against the actual registration tree, and the output is
// valid JSON with the canonical envelope shape. The cli-commands.gen.json must
// be up to date (run `make endpoints`); a stale artifact fails crossCheck here
// exactly as it would in CI.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	// Path is relative to the test working directory (api/cmd/gen-endpoints):
	// up to the scenario root, then into cli/.
	cliCommands := filepath.Join("..", "..", "..", "cli", "cli-commands.gen.json")

	if err := run(output, cliCommands); err != nil {
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
	// cli_commands[] mirrors the registered tree, so it must be non-empty.
	if len(got.CLICommands) == 0 {
		t.Error("cli_commands must be non-empty (mirrors the registration tree)")
	}

	// Trailing newline so editors don't get angry about diff noise.
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("manifest must end with a newline")
	}
}

// TestCrossCheck_FailsOnUnregisteredCommand pins the codegen-time guard:
// an endpoint whose cli_mapping.command isn't a registered CLI command
// fails fast with a message that names the missing entry.
func TestCrossCheck_FailsOnUnregisteredCommand(t *testing.T) {
	endpoints := []module.EndpointDescriptor{
		{
			ID: "lonely",
			CLIMapping: &module.CLIMapping{
				Command: "device-sync-hub lonely subcommand",
			},
		},
	}
	registered := []registeredCommand{} // empty tree

	err := crossCheck(endpoints, registered)
	if err == nil {
		t.Fatal("expected crossCheck to fail when the CLI tree is missing the command")
	}
	if !strings.Contains(err.Error(), "lonely subcommand") {
		t.Errorf("error %q must name the missing command", err.Error())
	}
	if !strings.Contains(err.Error(), `endpoint "lonely"`) {
		t.Errorf("error %q must name the offending endpoint", err.Error())
	}
}

// TestCrossCheck_PassesWhenRegistered confirms the happy path, including
// alias resolution.
func TestCrossCheck_PassesWhenRegistered(t *testing.T) {
	endpoints := []module.EndpointDescriptor{
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "device-sync-hub x"}},
		{ID: "y_alias", CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices ll"}},
		{ID: "z_no_cli"}, // no CLIMapping — must be allowed
	}
	registered := []registeredCommand{
		{Name: "x"},
		{Name: "devices list", Aliases: []string{"devices ll"}},
	}

	if err := crossCheck(endpoints, registered); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestBuildCLICommands_ResolvesEndpointIDs confirms membership/order follow
// the registered tree and endpoint_id is matched from the endpoints.
func TestBuildCLICommands_ResolvesEndpointIDs(t *testing.T) {
	endpoints := []module.EndpointDescriptor{
		{ID: "health", CLIMapping: &module.CLIMapping{Command: "device-sync-hub status"}},
		{ID: "devices_list", CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices list"}},
	}
	registered := []registeredCommand{
		{Name: "configure"},
		{Name: "devices list"},
		{Name: "status"},
	}
	got := buildCLICommands(endpoints, registered)
	if len(got) != 3 {
		t.Fatalf("expected 3 cli_commands, got %d", len(got))
	}
	want := map[string]string{"configure": "", "devices list": "devices_list", "status": "health"}
	for _, c := range got {
		if want[c.Name] != c.EndpointID {
			t.Errorf("%s endpoint_id = %q, want %q", c.Name, c.EndpointID, want[c.Name])
		}
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "device-sync-hub devices list"
// must compare against the artifact's "devices list".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "device-sync-hub status", want: "status"},
		{in: "device-sync-hub devices list", want: "devices list"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "device-sync-hub", want: "device-sync-hub"}, // no trailing space → preserved
	}
	for _, tc := range cases {
		if got := stripBinaryPrefix(tc.in); got != tc.want {
			t.Errorf("stripBinaryPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
