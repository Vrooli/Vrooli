// gen-endpoints emits .vrooli/endpoints.json from the shared modules
// registry's AllEndpoints() — the same registry main.go consumes for
// AllSchemas. The center never accumulates endpoint metadata; the
// registry collects what each handler package exports.
//
// Usage:
//
//	go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json
//
// CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json`
// so a regenerated file that differs from what's checked in fails the
// build with an actionable diff. The fix is always: regenerate locally
// and commit.
//
// Adding a new domain: register it once in
// api/internal/modules/registry.go (one line in AllEndpoints, one in
// AllSchemas), and add the matching entry to cli_commands_seed.json.
// The cross-check at codegen time (every cli_mapping.command in
// endpoints[] must have a cli_commands[] entry) catches drift before
// the diff gate ever sees it.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"architecture-cartographer/internal/module"
	"architecture-cartographer/internal/modules"
)

const (
	manifestSchema  = "../../../../scripts/scenarios/schemas/endpoints.schema.json"
	manifestVersion = "1.0.0"
	defaultOutput   = "../.vrooli/endpoints.json"
	defaultSeed     = "cmd/gen-endpoints/cli_commands_seed.json"
	defaultCLIMani  = "../cli/manifest.json"
)

// cliCoreBuiltins are the commands cli-core's StandardScenarioApp
// registers automatically (not declared in cli/manifest.json). A seed
// entry naming one of these resolves without a manifest command.
var cliCoreBuiltins = map[string]struct{}{
	"status":    {},
	"configure": {},
	"version":   {},
}

// CLICommand mirrors the cli_commands[] entry shape in
// .vrooli/endpoints.json. Hand-maintained in cli_commands_seed.json
// because the CLI is a separate Go module from the API; cross-module
// codegen would force one to import the other.
type CLICommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	EndpointID  string `json:"endpoint_id"`
}

type seedFile struct {
	CLICommands []CLICommand `json:"cli_commands"`
}

type manifest struct {
	Schema      string                      `json:"$schema"`
	Version     string                      `json:"version"`
	Endpoints   []module.EndpointDescriptor `json:"endpoints"`
	CLICommands []CLICommand                `json:"cli_commands"`
}

func main() {
	output := flag.String("output", defaultOutput, "path to write the generated endpoints.json")
	seedPath := flag.String("seed", defaultSeed, "path to cli_commands_seed.json")
	cliManifest := flag.String("cli-manifest", defaultCLIMani, "path to cli/manifest.json (for the seed↔registered-command parity check)")
	flag.Parse()

	if err := run(*output, *seedPath, *cliManifest); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func run(output, seedPath, cliManifestPath string) error {
	endpoints := modules.AllEndpoints()
	seed, err := loadSeed(seedPath)
	if err != nil {
		return fmt.Errorf("load seed: %w", err)
	}

	if err := crossCheck(endpoints, seed.CLICommands); err != nil {
		return err
	}

	registered, err := loadRegisteredCommands(cliManifestPath)
	if err != nil {
		return fmt.Errorf("load cli manifest: %w", err)
	}
	if err := verifySeedRegistered(seed.CLICommands, registered); err != nil {
		return err
	}

	if err := validateTransport(endpoints); err != nil {
		return err
	}

	m := manifest{
		Schema:      manifestSchema,
		Version:     manifestVersion,
		Endpoints:   endpoints,
		CLICommands: seed.CLICommands,
	}
	// json.Encoder lets us disable HTML-safe escaping so '<', '>' and '&'
	// stay readable in the generated file. Endpoint paths and example
	// curls contain those characters; HTML-escaping them as < etc.
	// would still be valid JSON but useless for humans reading the
	// manifest.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(output, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}
	return nil
}

func loadSeed(path string) (*seedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s seedFile
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &s, nil
}

// crossCheck enforces: every cli_mapping.command in endpoints[] must
// have a matching cli_commands[].name. Catches the failure mode where
// a developer adds an endpoint with a CLI mapping but forgets to seed
// the CLI command (or vice versa). The error message names the
// missing entry so the fix is mechanical.
func crossCheck(endpoints []module.EndpointDescriptor, commands []CLICommand) error {
	cmdSet := make(map[string]struct{}, len(commands))
	for _, c := range commands {
		cmdSet[c.Name] = struct{}{}
	}
	for _, e := range endpoints {
		if e.CLIMapping == nil {
			continue
		}
		// The seed stores CLI command names without the architecture-cartographer
		// binary prefix (the binary name is per-scenario). The endpoint
		// descriptor includes the prefix in its Command field. Strip it
		// for comparison.
		want := stripBinaryPrefix(e.CLIMapping.Command)
		if _, ok := cmdSet[want]; !ok {
			return fmt.Errorf(
				"cli_commands_seed.json missing entry %q (referenced by endpoint %q)",
				want, e.ID,
			)
		}
	}
	return nil
}

// cliManifest is the minimal shape of cli/manifest.json this codegen
// reads to verify the seed. The CLI is a separate Go module, so the
// gate reads the manifest as data rather than importing cli-core's
// richer cliapp.Manifest type.
type cliManifest struct {
	Groups []struct {
		Name     string `json:"name"`
		Commands []struct {
			Name string `json:"name"`
		} `json:"commands"`
	} `json:"groups"`
}

// loadRegisteredCommands reads cli/manifest.json and returns the set of
// registered command names in "group command" form (matching the seed's
// naming convention), which is what the reverse parity check compares
// against.
func loadRegisteredCommands(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m cliManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	out := make(map[string]struct{})
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			out[g.Name+" "+c.Name] = struct{}{}
		}
	}
	return out, nil
}

// verifySeedRegistered closes the hollow-gate hole: every cli_commands[]
// entry must resolve to a command actually registered in cli/manifest.json
// (group + command) or to a cli-core StandardScenarioApp built-in. A seed
// entry that names no real CLI command is a build failure — previously the
// codegen only checked endpoints↔seed consistency, so the seed could claim
// commands that no CLI registered and the gate stayed green.
func verifySeedRegistered(commands []CLICommand, registered map[string]struct{}) error {
	var missing []string
	for _, c := range commands {
		if _, ok := cliCoreBuiltins[c.Name]; ok {
			continue
		}
		if _, ok := registered[c.Name]; ok {
			continue
		}
		missing = append(missing, c.Name)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf(
			"cli_commands_seed.json entries do not resolve to a registered CLI command "+
				"(neither a cli/manifest.json group+command nor a cli-core built-in): %s. "+
				"Either register the command in cli/manifest.json + cli/domains, or remove the seed entry "+
				"(and drop the endpoint's CLIMapping if the RPC is intentionally not CLI-exposed)",
			strings.Join(missing, ", "))
	}
	return nil
}

// validateTransport enforces the proto/Connect-RPC anti-drift contract:
// every EndpointDescriptor.Path must either be a generated Connect
// procedure constant (which always starts with "/vrooli." because
// Vrooli proto services are namespaced under
// vrooli.<scenario>.v1.<domain>.<Service>), or carry a RESTException
// declaring one of the four mechanically-allowed REST reasons.
//
// This is the structural fix for the drift mode where agents hand-write
// EndpointDescriptor{Path: "/api/v1/foo"} as a literal string instead
// of authoring a proto service and importing the generated *Procedure
// constant. Without this pass, nothing in the template makes that
// illegal — and the parity test (which walks proto FileDescriptors) is
// silent because there's no proto service to walk.
//
// If a domain genuinely needs REST (multipart, webhook, third-party
// shape, ops probe), tag the descriptor explicitly:
//
//	RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "..."}
//
// The four allowed reasons are defined in api/internal/module/module.go.
// Adding a new reason is a deliberate architectural decision; do not
// add one to silence this check.
func validateTransport(endpoints []module.EndpointDescriptor) error {
	var violations []string
	for _, e := range endpoints {
		isConnect := strings.HasPrefix(e.Path, "/vrooli.")
		hasException := e.RESTException != nil
		switch {
		case isConnect && hasException:
			violations = append(violations, fmt.Sprintf(
				"endpoint %q: Path %q is a Connect procedure but has RESTException set; remove RESTException",
				e.ID, e.Path))
		case !isConnect && !hasException:
			violations = append(violations, fmt.Sprintf(
				"endpoint %q: Path %q is not a Connect procedure (must start with %q) and has no RESTException; "+
					"either reference a generated *Procedure constant from packages/proto/gen, or tag with "+
					"RESTException{Reason: one of multipart_upload|webhook_receiver|third_party_shape|ops_probe}",
				e.ID, e.Path, "/vrooli."))
		case !isConnect && hasException:
			if !validRESTReason(e.RESTException.Reason) {
				violations = append(violations, fmt.Sprintf(
					"endpoint %q: RESTException.Reason %q is not one of the allowed reasons "+
						"(multipart_upload, webhook_receiver, third_party_shape, ops_probe)",
					e.ID, e.RESTException.Reason))
			}
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("transport validation failed:\n  - %s", strings.Join(violations, "\n  - "))
	}
	return nil
}

func validRESTReason(r module.RESTReason) bool {
	switch r {
	case module.RESTReasonMultipartUpload,
		module.RESTReasonWebhookReceiver,
		module.RESTReasonThirdPartyShape,
		module.RESTReasonOpsProbe:
		return true
	}
	return false
}

// stripBinaryPrefix removes the leading binary-name prefix from a CLI
// command string. The CLI commands seed lists subcommand names without
// the binary prefix because the binary name is substituted per-scenario
// at generation time. Both the scenario-id form ("architecture-cartographer ")
// and the short binary name ("arch-cart ") are recognized.
func stripBinaryPrefix(cmd string) string {
	for _, prefix := range []string{"architecture-cartographer ", "arch-cart "} {
		if len(cmd) > len(prefix) && cmd[:len(prefix)] == prefix {
			return cmd[len(prefix):]
		}
	}
	return cmd
}
