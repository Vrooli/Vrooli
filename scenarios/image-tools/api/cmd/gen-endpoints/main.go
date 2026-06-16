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
	"strings"

	"image-tools/internal/module"
	"image-tools/internal/modules"
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
		// The seed stores CLI command names without the image-tools
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

// stripBinaryPrefix removes the leading "image-tools " from a CLI
// command string. The CLI commands seed lists the subcommand names
// without the binary prefix because the binary name is substituted
// per-scenario at generation time.
func stripBinaryPrefix(cmd string) string {
	const prefix = "image-tools "
	if len(cmd) > len(prefix) && cmd[:len(prefix)] == prefix {
		return cmd[len(prefix):]
	}
	return cmd
}
