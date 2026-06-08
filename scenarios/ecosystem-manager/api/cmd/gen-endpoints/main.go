// gen-endpoints regenerates the Connect-RPC slice of .vrooli/endpoints.json
// from the modules registry, while PRESERVING the hand-authored REST entries
// for domains that have not migrated yet.
//
// ecosystem-manager is mid-migration: ~79 gorilla/mux REST routes coexist with
// the Connect-RPC domains. A naive "regenerate the whole file from the module
// registry" (the react-vite template's behaviour) would erase every REST
// entry the moment it runs, because un-migrated domains are not in the
// registry. This variant instead:
//
//  1. loads the existing endpoints.json, keeping $schema, version, and
//     websockets in their original order;
//  2. keeps every existing endpoint whose path is NOT a Connect procedure
//     ("/vrooli.") and whose category is NOT a migrated domain — preserved as
//     raw JSON so fields the contract struct does not model (e.g. params) are
//     not dropped;
//  3. appends the freshly-generated Connect descriptors from
//     modules.AllEndpoints() (validated by validateTransport);
//  4. writes the merged endpoints[] back, leaving the other keys untouched.
//
// With no migrated domains the output is byte-identical to the input.
//
// Usage:
//
//	go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json --seed cmd/gen-endpoints/cli_commands_seed.json
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ecosystem-manager/api/internal/module"
	"github.com/ecosystem-manager/api/internal/modules"
)

const (
	defaultOutput = "../.vrooli/endpoints.json"
	defaultSeed   = "cmd/gen-endpoints/cli_commands_seed.json"
)

// manifest mirrors .vrooli/endpoints.json with its top-level keys in their
// canonical order. endpoints[] is kept as raw JSON so preserved REST entries
// round-trip byte-for-byte. Other keys ($schema, version, websockets) pass
// through untouched.
type manifest struct {
	Schema     json.RawMessage   `json:"$schema,omitempty"`
	Version    json.RawMessage   `json:"version,omitempty"`
	Endpoints  []json.RawMessage `json:"endpoints"`
	Websockets json.RawMessage   `json:"websockets,omitempty"`
}

// endpointHead peeks at the routing-relevant fields of an existing endpoint
// entry without modelling (or dropping) the rest.
type endpointHead struct {
	Path     string `json:"path"`
	Category string `json:"category"`
}

// CLICommand mirrors a cli_commands_seed.json entry. The seed is the
// cross-check source: every CLIMapping.Command on a generated Connect endpoint
// must have a matching seed entry, so an endpoint and its CLI command cannot
// drift.
type CLICommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	EndpointID  string `json:"endpoint_id"`
}

type seedFile struct {
	CLICommands []CLICommand `json:"cli_commands"`
}

func main() {
	output := flag.String("output", defaultOutput, "path to write the merged endpoints.json")
	seedPath := flag.String("seed", defaultSeed, "path to cli_commands_seed.json")
	flag.Parse()

	if err := run(*output, *seedPath); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func run(output, seedPath string) error {
	generated := modules.AllEndpoints()
	if err := validateTransport(generated); err != nil {
		return err
	}

	seed, err := loadSeed(seedPath)
	if err != nil {
		return fmt.Errorf("load seed: %w", err)
	}
	if err := crossCheck(generated, seed.CLICommands); err != nil {
		return err
	}

	raw, err := os.ReadFile(output)
	if err != nil {
		return fmt.Errorf("read existing manifest %s: %w", output, err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("unmarshal existing manifest: %w", err)
	}

	migrated := make(map[string]struct{})
	for _, d := range modules.MigratedDomains() {
		migrated[d] = struct{}{}
	}

	merged := make([]json.RawMessage, 0, len(m.Endpoints)+len(generated))
	for _, rawEntry := range m.Endpoints {
		var head endpointHead
		if err := json.Unmarshal(rawEntry, &head); err != nil {
			return fmt.Errorf("unmarshal existing endpoint head: %w", err)
		}
		if strings.HasPrefix(head.Path, "/vrooli.") {
			continue // regenerated below
		}
		if _, isMigrated := migrated[head.Category]; isMigrated {
			continue // REST entry superseded by a Connect descriptor
		}
		merged = append(merged, rawEntry) // preserve verbatim
	}
	for _, e := range generated {
		entryJSON, err := marshalIndent(e)
		if err != nil {
			return fmt.Errorf("marshal generated endpoint %q: %w", e.ID, err)
		}
		merged = append(merged, entryJSON)
	}
	m.Endpoints = merged

	out, err := marshalIndent(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(output, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}
	return nil
}

// marshalIndent encodes with HTML escaping disabled so '<', '>', '&' in
// endpoint paths and example curls stay readable.
func marshalIndent(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func loadSeed(path string) (*seedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &seedFile{}, nil
		}
		return nil, err
	}
	var s seedFile
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &s, nil
}

// crossCheck enforces that every CLIMapping.Command on a generated Connect
// endpoint has a matching cli_commands_seed.json entry, so an endpoint and its
// CLI command cannot drift.
func crossCheck(endpoints []module.EndpointDescriptor, commands []CLICommand) error {
	cmdSet := make(map[string]struct{}, len(commands))
	for _, c := range commands {
		cmdSet[c.Name] = struct{}{}
	}
	for _, e := range endpoints {
		if e.CLIMapping == nil {
			continue
		}
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

// validateTransport enforces the proto/Connect-RPC anti-drift contract for the
// GENERATED endpoints only: every generated EndpointDescriptor.Path must be a
// Connect procedure ("/vrooli.") or carry one of the four allowed
// RESTExceptions. Preserved REST entries (un-migrated domains) are exempt —
// they are the migration remainder, not new drift.
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
					"reference a generated *Procedure constant or tag RESTException{Reason: one of "+
					"multipart_upload|webhook_receiver|third_party_shape|ops_probe}",
				e.ID, e.Path, "/vrooli."))
		case !isConnect && hasException:
			if !validRESTReason(e.RESTException.Reason) {
				violations = append(violations, fmt.Sprintf(
					"endpoint %q: RESTException.Reason %q is not an allowed reason", e.ID, e.RESTException.Reason))
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

func stripBinaryPrefix(cmd string) string {
	const prefix = "ecosystem-manager "
	if strings.HasPrefix(cmd, prefix) {
		return cmd[len(prefix):]
	}
	return cmd
}
