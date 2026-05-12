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

	"flow-verifier/internal/module"
	"flow-verifier/internal/modules"
)

const (
	manifestSchema  = "../../../../scripts/scenarios/schemas/endpoints.schema.json"
	manifestVersion = "1.0.0"
	defaultOutput   = "../.vrooli/endpoints.json"
	defaultSeed     = "cmd/gen-endpoints/cli_commands_seed.json"
)

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
	flag.Parse()

	if err := run(*output, *seedPath); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func run(output, seedPath string) error {
	endpoints := modules.AllEndpoints()
	seed, err := loadSeed(seedPath)
	if err != nil {
		return fmt.Errorf("load seed: %w", err)
	}

	if err := crossCheck(endpoints, seed.CLICommands); err != nil {
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
		// The seed stores CLI command names without the flow-verifier
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

// stripBinaryPrefix removes the leading "flow-verifier " from a CLI
// command string. The CLI commands seed lists the subcommand names
// without the binary prefix because the binary name is substituted
// per-scenario at generation time.
func stripBinaryPrefix(cmd string) string {
	const prefix = "flow-verifier "
	if len(cmd) > len(prefix) && cmd[:len(prefix)] == prefix {
		return cmd[len(prefix):]
	}
	return cmd
}
