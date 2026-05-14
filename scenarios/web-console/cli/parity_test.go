package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"web-console/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAPICLIParity locks in the API↔CLI contract: every Connect-RPC
// endpoint in .vrooli/endpoints.json must declare a cli_mapping, and the
// command named there must actually be registered in the CLI domain
// tree. Drift in either direction fails the test with a punch list, so
// agents can't ship an RPC without a matching CLI command (or remove a
// CLI command without thinking about the RPC consumers).
//
// Skip mechanism: an endpoint can opt out by adding "cli:skip" to its
// rest_exception note OR by referencing one of the explicitly-skipped
// IDs in parityCLISkipIDs below. Use the IDs list only for endpoints
// that genuinely cannot have a CLI form (server streams, long-lived
// subscriptions); never to silence the test without thought.
func TestAPICLIParity(t *testing.T) {
	manifest, err := loadEndpointsManifest()
	if err != nil {
		t.Fatalf("load endpoints manifest: %v", err)
	}

	registered := registeredCLICommands()

	var (
		missingCLIMapping     []string
		missingCLIRegistered  []string
		seedReferencesMissing []string
	)

	// 1. Every Connect-RPC endpoint must have a cli_mapping (or be
	//    explicitly skipped). Connect procedures are namespaced under
	//    /vrooli. — anything else is a REST endpoint with its own check
	//    in gen-endpoints (RESTException).
	for _, e := range manifest.Endpoints {
		if !strings.HasPrefix(e.Path, "/vrooli.") {
			continue
		}
		if parityCLISkipped(e) {
			continue
		}
		if e.CLIMapping == nil {
			missingCLIMapping = append(missingCLIMapping, e.ID+" ("+e.Path+")")
			continue
		}
		want := stripBinaryPrefix(e.CLIMapping.Command)
		if _, ok := registered[want]; !ok {
			missingCLIRegistered = append(
				missingCLIRegistered,
				e.ID+" expects CLI command "+want,
			)
		}
	}

	// 2. Every cli_commands[] seed entry must be a registered command.
	//    Catches the inverse drift: a seed entry was added but the CLI
	//    side never grew the corresponding command.
	for _, c := range manifest.CLICommands {
		key := stripBinaryPrefix(c.Name)
		if _, ok := registered[key]; !ok {
			seedReferencesMissing = append(
				seedReferencesMissing,
				c.Name+" (endpoint "+c.EndpointID+")",
			)
		}
	}

	if len(missingCLIMapping) > 0 {
		sort.Strings(missingCLIMapping)
		t.Errorf(
			"Connect-RPC endpoints missing cli_mapping (every RPC must have a CLI command — add one to handlers/<domain>/endpoints.go and a matching command to cli/domains/<domain>/register.go):\n  - %s",
			strings.Join(missingCLIMapping, "\n  - "),
		)
	}
	if len(missingCLIRegistered) > 0 {
		sort.Strings(missingCLIRegistered)
		t.Errorf(
			"cli_mapping references commands not registered in cli/domains/* :\n  - %s",
			strings.Join(missingCLIRegistered, "\n  - "),
		)
	}
	if len(seedReferencesMissing) > 0 {
		sort.Strings(seedReferencesMissing)
		t.Errorf(
			"cli_commands_seed.json entries with no matching registered CLI command:\n  - %s",
			strings.Join(seedReferencesMissing, "\n  - "),
		)
	}
}

// parityCLISkipIDs lists endpoint IDs that intentionally have no CLI
// form — long-lived server streams, WebSocket upgrades, and similar.
// Adding an entry here requires a comment explaining why a CLI command
// would be impossible (not just inconvenient).
var parityCLISkipIDs = map[string]struct{}{}

func parityCLISkipped(e endpointEntry) bool {
	if _, ok := parityCLISkipIDs[e.ID]; ok {
		return true
	}
	if e.RESTException != nil && strings.Contains(e.RESTException.Note, "cli:skip") {
		return true
	}
	return false
}

// registeredCLICommands enumerates every command the CLI registers,
// returning a set keyed by the user-facing command string (minus the
// binary prefix). Aliases are included so cli_mapping references that
// hit an alias still resolve.
func registeredCLICommands() map[string]struct{} {
	app, err := NewApp()
	if err != nil {
		// NewApp failure leaves no way to enumerate commands; surface
		// the underlying error rather than masking parity drift.
		panic("parity: NewApp failed: " + err.Error())
	}
	out := make(map[string]struct{})
	for _, g := range domains.CommandGroups(app.core) {
		for _, c := range g.Commands {
			addCommand(out, "", c)
		}
	}
	for _, sg := range domains.SubcommandGroups(app.core) {
		for _, c := range sg.Subcommands {
			addCommand(out, sg.Name, c)
		}
	}
	for k := range builtinCLICommands {
		out[k] = struct{}{}
	}
	return out
}

func addCommand(out map[string]struct{}, group string, c cliapp.Command) {
	names := append([]string{c.Name}, c.Aliases...)
	for _, n := range names {
		key := n
		if group != "" {
			key = group + " " + n
		}
		out[key] = struct{}{}
	}
}

// endpointEntry mirrors the subset of fields the parity test inspects.
// Defined locally so the test stays independent of api package internals
// (which the cli module can't import without a cycle anyway).
type endpointEntry struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	CLIMapping *struct {
		Command string `json:"command"`
	} `json:"cli_mapping,omitempty"`
	RESTException *struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	} `json:"rest_exception,omitempty"`
}

type cliCommandEntry struct {
	Name       string `json:"name"`
	EndpointID string `json:"endpoint_id"`
}

type endpointsManifest struct {
	Endpoints   []endpointEntry   `json:"endpoints"`
	CLICommands []cliCommandEntry `json:"cli_commands"`
}

func loadEndpointsManifest() (*endpointsManifest, error) {
	// cli/ is a sibling of api/ and .vrooli/ — the manifest is at
	// ../.vrooli/endpoints.json relative to the cli package root. Tests
	// run from the package directory by default.
	path := filepath.Join("..", ".vrooli", "endpoints.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m endpointsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func stripBinaryPrefix(cmd string) string {
	const prefix = "web-console "
	if strings.HasPrefix(cmd, prefix) {
		cmd = cmd[len(prefix):]
	}
	// Drop flags from the lookup key: `capabilities --liveness` and
	// `capabilities` register as the same CLI command (the flag is
	// parsed inside the handler, not by the dispatcher).
	if idx := strings.Index(cmd, " --"); idx >= 0 {
		cmd = cmd[:idx]
	}
	return strings.TrimSpace(cmd)
}

// builtinCLICommands lists commands provided by cli-core's standard
// scenario app (NewStandardScenarioApp) that domain registration does
// not surface. The parity test merges these into the registered set so
// `status` / `help` etc. resolve.
var builtinCLICommands = map[string]struct{}{
	"status": {},
}
