package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"web-search/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end: writes
// the manifest to a temp file, reads it back, asserts it's valid JSON
// with the canonical envelope shape.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	seed := filepath.Join(t.TempDir(), "seed.json")
	writeSeed(t, seed, []CLICommand{
		{Name: "status", Description: "Health check", EndpointID: "health"},
		{Name: "findings list", Description: "List findings", EndpointID: "findings_list"},
		{Name: "findings get", Description: "Get finding", EndpointID: "findings_get"},
		{Name: "findings add", Description: "Add finding", EndpointID: "findings_add"},
		{Name: "findings edit", Description: "Edit finding", EndpointID: "findings_edit"},
		{Name: "findings supersede", Description: "Supersede finding", EndpointID: "findings_supersede"},
		{Name: "findings flag", Description: "Flag finding", EndpointID: "findings_flag"},
		{Name: "disputes list", Description: "List disputed findings", EndpointID: "findings_disputes_list"},
		{Name: "disputes resolve", Description: "Resolve a dispute", EndpointID: "findings_resolve"},
		{Name: "findings prune", Description: "Prune findings", EndpointID: "findings_prune"},
		{Name: "findings search", Description: "Search findings", EndpointID: "findings_search"},
		{Name: "findings count", Description: "Count findings", EndpointID: "findings_count"},
		{Name: "search", Description: "Live web search", EndpointID: "livesearch_search"},
		{Name: "research l2", Description: "L2 research", EndpointID: "research_l2"},
		{Name: "research l3", Description: "L3 research", EndpointID: "research_l3"},
		{Name: "research status", Description: "Poll L3 run", EndpointID: "research_status"},
		{Name: "research gather", Description: "Bounded GATHER near a query", EndpointID: "research_gather"},
		{Name: "findings effectiveness", Description: "List findings by usage effectiveness", EndpointID: "findings_effectiveness"},
		{Name: "findings use", Description: "Record explicit finding usage", EndpointID: "findings_record_usage"},
		{Name: "findings gc", Description: "Run the store-consistency GC", EndpointID: "findings_gc"},
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
				Command: "web-search lonely subcommand",
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
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "web-search x"}},
		{ID: "y_no_cli"}, // no CLIMapping — must be allowed
	}
	commands := []CLICommand{{Name: "x", EndpointID: "x"}}

	if err := crossCheck(endpoints, commands); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "web-search findings list"
// must compare against the seed's "findings list".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "web-search status", want: "status"},
		{in: "web-search findings list", want: "findings list"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "web-search", want: "web-search"}, // no trailing space → preserved
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
