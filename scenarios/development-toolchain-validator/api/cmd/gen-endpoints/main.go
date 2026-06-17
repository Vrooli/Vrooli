// gen-endpoints emits .vrooli/endpoints.json from the shared modules
// registry's AllEndpoints() — the same registry main.go consumes for
// AllSchemas. The center never accumulates endpoint metadata; the
// registry collects what each handler package exports.
//
// The cli_commands[] section is derived from cli/cli-commands.gen.json,
// the generated bridge artifact that `cli/cmd/gen-cli-commands` emits from
// the live CLI registration tree (the single source of truth). The API
// module reads that artifact as data — there is no API↔CLI Go import.
//
// Usage:
//
//	go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json
//
// CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json`
// so a regenerated file that differs from what's checked in fails the
// build with an actionable diff. `make endpoints` is two-step: it first
// regenerates cli/cli-commands.gen.json, then this generator. The fix is
// always: run `make endpoints` locally and commit.
//
// Adding a new domain: register it once in
// api/internal/modules/registry.go (one line in AllEndpoints, one in
// AllSchemas) and register the matching CLI command in cli/domains/<dom>.
// The cross-check at codegen time (every cli_mapping.command in
// endpoints[] must be a registered CLI command) catches drift before the
// diff gate ever sees it.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"development-toolchain-validator/internal/module"
	"development-toolchain-validator/internal/modules"
)

const (
	manifestSchema     = "../../../../scripts/scenarios/schemas/endpoints.schema.json"
	manifestVersion    = "1.0.0"
	defaultOutput      = "../.vrooli/endpoints.json"
	defaultCLICommands = "../cli/cli-commands.gen.json"
)

// CLICommand mirrors the cli_commands[] entry shape in
// .vrooli/endpoints.json.
type CLICommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	EndpointID  string `json:"endpoint_id"`
}

// registeredCommand is one entry in cli/cli-commands.gen.json — a command from
// the live CLI registration tree, with a group-qualified name (binary prefix
// stripped), its description, and any group-qualified aliases.
type registeredCommand struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases,omitempty"`
}

type cliCommandsArtifact struct {
	Commands []registeredCommand `json:"commands"`
}

type manifest struct {
	Schema      string                      `json:"$schema"`
	Version     string                      `json:"version"`
	Endpoints   []module.EndpointDescriptor `json:"endpoints"`
	CLICommands []CLICommand                `json:"cli_commands"`
}

func main() {
	output := flag.String("output", defaultOutput, "path to write the generated endpoints.json")
	cliCommands := flag.String("cli-commands", defaultCLICommands, "path to the generated cli/cli-commands.gen.json")
	flag.Parse()

	if err := run(*output, *cliCommands); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func run(output, cliCommandsPath string) error {
	endpoints := modules.AllEndpoints()
	registered, err := loadCLICommands(cliCommandsPath)
	if err != nil {
		return fmt.Errorf("load cli commands: %w", err)
	}

	if err := crossCheck(endpoints, registered); err != nil {
		return err
	}

	if err := validateTransport(endpoints); err != nil {
		return err
	}

	m := manifest{
		Schema:      manifestSchema,
		Version:     manifestVersion,
		Endpoints:   endpoints,
		CLICommands: buildCLICommands(endpoints, registered),
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

func loadCLICommands(path string) ([]registeredCommand, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a cliCommandsArtifact
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return a.Commands, nil
}

// buildCLICommands renders the cli_commands[] section from the registered CLI
// command tree. Membership, order, name and description all come from the
// generated artifact (the registration tree is the source of truth); the
// endpoint_id is resolved by matching each command against the endpoint whose
// cli_mapping.command names it, or left empty for commands with no RPC
// (e.g. configure). endpoint_id is descriptive linkage only — no code consumes
// it — so an empty value is fine.
func buildCLICommands(endpoints []module.EndpointDescriptor, registered []registeredCommand) []CLICommand {
	endpointByCommand := make(map[string]string)
	for _, e := range endpoints {
		if e.CLIMapping == nil {
			continue
		}
		key := stripBinaryPrefix(e.CLIMapping.Command)
		if _, exists := endpointByCommand[key]; !exists {
			endpointByCommand[key] = e.ID
		}
	}
	out := make([]CLICommand, 0, len(registered))
	for _, c := range registered {
		out = append(out, CLICommand{
			Name:        c.Name,
			Description: c.Description,
			EndpointID:  endpointByCommand[c.Name],
		})
	}
	return out
}

// crossCheck enforces: every cli_mapping.command in endpoints[] must be a
// command registered in the CLI tree (present in cli-commands.gen.json, either
// as a canonical name or an alias). Catches the failure mode where a developer
// adds an endpoint with a CLI mapping but never registers the command (or
// renames the command without updating the endpoint). The error names the
// missing entry so the fix is mechanical.
func crossCheck(endpoints []module.EndpointDescriptor, registered []registeredCommand) error {
	cmdSet := make(map[string]struct{}, len(registered))
	for _, c := range registered {
		cmdSet[c.Name] = struct{}{}
		for _, a := range c.Aliases {
			cmdSet[a] = struct{}{}
		}
	}
	for _, e := range endpoints {
		if e.CLIMapping == nil {
			continue
		}
		// The CLI artifact stores command names without the development-toolchain-validator
		// binary prefix (the binary name is per-scenario). The endpoint
		// descriptor includes the prefix in its Command field. Strip it for
		// comparison.
		want := stripBinaryPrefix(e.CLIMapping.Command)
		if _, ok := cmdSet[want]; !ok {
			return fmt.Errorf(
				"cli-commands.gen.json missing command %q (referenced by endpoint %q); regenerate with `make endpoints`",
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

// stripBinaryPrefix turns an endpoint's cli_mapping.command into the registered
// command path. The binary is always the first whitespace-delimited token
// (whatever it's called — some endpoints use a short alias, others the full
// name), so drop the first token; then drop any trailing flags/args, since a
// mapping like "<bin> artifacts generate --scenario <id>" resolves to the
// registered command "artifacts generate" (flags are parsed inside the handler,
// not by the dispatcher). A command with no space is returned unchanged. The
// CLI command artifact lists names without the binary prefix, so the result
// compares directly.
func stripBinaryPrefix(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if i := strings.IndexByte(cmd, ' '); i >= 0 {
		cmd = cmd[i+1:]
	} else {
		return cmd
	}
	if idx := strings.Index(cmd, " --"); idx >= 0 {
		cmd = cmd[:idx]
	}
	return strings.TrimSpace(cmd)
}
