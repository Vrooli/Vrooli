package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"image-tools/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end: writes
// the generated CLI-commands artifact to a temp file, runs the generator,
// reads the result back, and asserts it's valid JSON with the canonical
// envelope shape.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	cliCommands := filepath.Join(t.TempDir(), "cli-commands.gen.json")
	// The fixture mirrors this scenario's real registered command tree
	// (cli/cli-commands.gen.json): status + jobs/models/ops domains. The notes
	// reference domain was deleted in Phase 1, so a notes-based fixture would not
	// cover the jobs/models/ops endpoints crossCheck validates against.
	writeCLICommands(t, cliCommands, []registeredCommand{
		{Name: "configure", Description: "View or update CLI settings"},
		{Name: "status", Description: "Check API health"},
		{Name: "jobs get", Description: "Get a job"},
		{Name: "jobs wait", Description: "Wait for a job"},
		{Name: "jobs list", Description: "List jobs"},
		{Name: "jobs cancel", Description: "Cancel a job"},
		{Name: "jobs watch", Description: "Watch a job"},
		{Name: "models list", Description: "List models"},
		{Name: "models get", Description: "Get a model"},
		{Name: "models operations", Description: "List operations"},
		{Name: "models select", Description: "Preview selection"},
		{Name: "models enable", Description: "Enable/disable a model"},
		{Name: "models blocklist", Description: "List blocklist"},
		{Name: "ops list", Description: "List operations"},
		{Name: "ops resize", Description: "Resize"},
		{Name: "ops crop", Description: "Crop"},
		{Name: "ops rotate", Description: "Rotate"},
		{Name: "ops flip", Description: "Flip"},
		{Name: "ops deskew", Description: "Deskew"},
		{Name: "ops thumbnail", Description: "Thumbnail"},
		{Name: "ops canvas", Description: "Canvas"},
		{Name: "ops adjust", Description: "Adjust"},
		{Name: "ops filter", Description: "Filter"},
		{Name: "ops convert", Description: "Convert"},
		{Name: "ops compress", Description: "Compress"},
		{Name: "ops overlay", Description: "Overlay"},
		{Name: "ops metadata", Description: "Metadata"},
	})

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
	// cli_commands[] mirrors the registered tree (membership and order from the
	// artifact), so its length equals the artifact's command count.
	if len(got.CLICommands) != 27 {
		t.Errorf("cli_commands count = %d, want 27", len(got.CLICommands))
	}
	// endpoint_id is resolved from the matching endpoint; configure has no RPC
	// so it must carry an empty endpoint_id.
	for _, c := range got.CLICommands {
		if c.Name == "configure" && c.EndpointID != "" {
			t.Errorf("configure endpoint_id = %q, want empty", c.EndpointID)
		}
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
				Command: "image-tools lonely subcommand",
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
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "image-tools x"}},
		{ID: "y_alias", CLIMapping: &module.CLIMapping{Command: "image-tools notes ll"}},
		{ID: "z_no_cli"}, // no CLIMapping — must be allowed
	}
	registered := []registeredCommand{
		{Name: "x"},
		{Name: "notes list", Aliases: []string{"notes ll"}},
	}

	if err := crossCheck(endpoints, registered); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestBuildCLICommands_ResolvesEndpointIDs confirms membership/order follow
// the registered tree and endpoint_id is matched from the endpoints.
func TestBuildCLICommands_ResolvesEndpointIDs(t *testing.T) {
	endpoints := []module.EndpointDescriptor{
		{ID: "health", CLIMapping: &module.CLIMapping{Command: "image-tools status"}},
		{ID: "notes_list", CLIMapping: &module.CLIMapping{Command: "image-tools notes list"}},
	}
	registered := []registeredCommand{
		{Name: "configure"},
		{Name: "notes list"},
		{Name: "status"},
	}
	got := buildCLICommands(endpoints, registered)
	if len(got) != 3 {
		t.Fatalf("expected 3 cli_commands, got %d", len(got))
	}
	want := map[string]string{"configure": "", "notes list": "notes_list", "status": "health"}
	for _, c := range got {
		if want[c.Name] != c.EndpointID {
			t.Errorf("%s endpoint_id = %q, want %q", c.Name, c.EndpointID, want[c.Name])
		}
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "image-tools notes list"
// must compare against the artifact's "notes list".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "image-tools status", want: "status"},
		{in: "image-tools notes list", want: "notes list"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "image-tools", want: "image-tools"}, // no trailing space → preserved
	}
	for _, tc := range cases {
		if got := stripBinaryPrefix(tc.in); got != tc.want {
			t.Errorf("stripBinaryPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func writeCLICommands(t *testing.T, path string, commands []registeredCommand) {
	t.Helper()
	body, err := json.MarshalIndent(cliCommandsArtifact{Commands: commands}, "", "  ")
	if err != nil {
		t.Fatalf("marshal cli commands: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write cli commands: %v", err)
	}
}
