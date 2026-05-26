package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"architecture-cartographer/internal/module"
)

// TestRun_ProducesValidJSON exercises the codegen end-to-end: runs the
// pipeline against the canonical seed (the real cli_commands_seed.json),
// reads the output back, and asserts it's valid JSON with the canonical
// envelope shape.
func TestRun_ProducesValidJSON(t *testing.T) {
	output := filepath.Join(t.TempDir(), "endpoints.json")
	// Use the on-disk seed so we exercise the same data the production
	// build uses. The test runs from cmd/gen-endpoints so the relative
	// path resolves the same way `go run ./cmd/gen-endpoints` would.
	seed := "cli_commands_seed.json"
	// cli/manifest.json sits at the scenario root; from cmd/gen-endpoints
	// that is three levels up (cmd/gen-endpoints -> cmd -> api -> root).
	cliManifest := filepath.Join("..", "..", "..", "cli", "manifest.json")

	if err := run(output, seed, cliManifest); err != nil {
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
	// At least the canonical health entry must be present; production
	// scenarios add many more — we don't pin the exact count because new
	// domains land alongside their CLI entries in the same commit.
	if len(got.CLICommands) == 0 {
		t.Error("cli_commands must be non-empty")
	}
	foundStatus := false
	for _, c := range got.CLICommands {
		if c.Name == "status" {
			foundStatus = true
			break
		}
	}
	if !foundStatus {
		t.Error("cli_commands must include the canonical 'status' entry")
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
				Command: "architecture-cartographer lonely subcommand",
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
		{ID: "x", CLIMapping: &module.CLIMapping{Command: "architecture-cartographer x"}},
		{ID: "y_no_cli"}, // no CLIMapping — must be allowed
	}
	commands := []CLICommand{{Name: "x", EndpointID: "x"}}

	if err := crossCheck(endpoints, commands); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestVerifySeedRegistered_FailsOnUnregisteredCommand pins the tightened
// (no-longer-hollow) gate: a seed command that resolves to neither a
// cli/manifest.json group+command nor a cli-core built-in fails codegen.
func TestVerifySeedRegistered_FailsOnUnregisteredCommand(t *testing.T) {
	registered := map[string]struct{}{"conflicts list": {}}
	commands := []CLICommand{
		{Name: "status"},          // built-in — allowed
		{Name: "conflicts list"},  // registered — allowed
		{Name: "conflicts ghost"}, // neither — must fail
	}

	err := verifySeedRegistered(commands, registered)
	if err == nil {
		t.Fatal("expected verifySeedRegistered to fail on an unregistered seed command")
	}
	if !strings.Contains(err.Error(), "conflicts ghost") {
		t.Errorf("error %q must name the unregistered command", err.Error())
	}
	if strings.Contains(err.Error(), "conflicts list") {
		t.Errorf("error %q must not flag the registered command", err.Error())
	}
}

// TestVerifySeedRegistered_PassesWhenAllResolve confirms the happy path:
// built-ins and registered group+command names both resolve.
func TestVerifySeedRegistered_PassesWhenAllResolve(t *testing.T) {
	registered := map[string]struct{}{"conflicts list": {}, "apply plan": {}}
	commands := []CLICommand{
		{Name: "status"},
		{Name: "conflicts list"},
		{Name: "apply plan"},
	}
	if err := verifySeedRegistered(commands, registered); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

// TestSeedResolvesAgainstRealManifest is the end-to-end regression guard:
// the on-disk seed must resolve against the on-disk cli/manifest.json, so
// the two cannot silently drift.
func TestSeedResolvesAgainstRealManifest(t *testing.T) {
	seed, err := loadSeed("cli_commands_seed.json")
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	registered, err := loadRegisteredCommands(filepath.Join("..", "..", "..", "cli", "manifest.json"))
	if err != nil {
		t.Fatalf("load cli manifest: %v", err)
	}
	if err := verifySeedRegistered(seed.CLICommands, registered); err != nil {
		t.Fatalf("seed does not resolve against cli/manifest.json: %v", err)
	}
}

// TestStripBinaryPrefix is the smallest unit on the command-name
// normalisation step: the endpoint's "architecture-cartographer alpha list"
// must compare against the seed's "alpha list".
func TestStripBinaryPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "architecture-cartographer status", want: "status"},
		{in: "architecture-cartographer alpha list", want: "alpha list"},
		{in: "already-stripped", want: "already-stripped"},
		{in: "architecture-cartographer", want: "architecture-cartographer"}, // no trailing space → preserved
	}
	for _, tc := range cases {
		if got := stripBinaryPrefix(tc.in); got != tc.want {
			t.Errorf("stripBinaryPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
