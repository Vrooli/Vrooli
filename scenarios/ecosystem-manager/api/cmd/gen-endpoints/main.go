// gen-endpoints regenerates the Connect-RPC slice of .vrooli/endpoints.json
// from the modules registry, while PRESERVING the hand-authored REST entries
// for domains that have not migrated yet.
//
// ecosystem-manager is mid-migration: ~79 gorilla/mux REST routes coexist with
// the Connect-RPC domains. A naive "regenerate the whole file from the module
// registry" (the react-vite template's behaviour, and the shared
// api-core/endpoints/gen.Generate) would erase every preserved REST entry and
// the websockets section the moment it runs, because un-migrated domains are
// not in the registry. This variant instead:
//
//  1. loads the existing endpoints.json, keeping $schema, version, and
//     websockets in their original order;
//  2. keeps every existing endpoint whose path is NOT a Connect procedure
//     ("/vrooli.") and whose category is NOT a migrated domain — preserved as
//     raw JSON so fields the contract struct does not model (e.g. params) are
//     not dropped;
//  3. appends the freshly-generated Connect descriptors from
//     modules.AllEndpoints() (validated by validateTransport);
//  4. cross-checks the generated Connect descriptors against cli/manifest.json
//     — the single source of truth for the CLI surface — so an endpoint and
//     its CLI command (or its declared omission) cannot drift;
//  5. writes the merged endpoints[] back, leaving the other keys untouched.
//
// With no migrated domains the output is byte-identical to the input.
//
// Usage:
//
//	go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json --manifest ../cli/manifest.json
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	gen "github.com/vrooli/api-core/endpoints/gen"

	"github.com/ecosystem-manager/api/internal/modules"
)

const (
	defaultOutput   = "../.vrooli/endpoints.json"
	defaultManifest = "../cli/manifest.json"
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

func main() {
	output := flag.String("output", defaultOutput, "path to write the merged endpoints.json")
	manifestPath := flag.String("manifest", defaultManifest, "path to the scenario cli manifest (CLI-surface SSOT)")
	flag.Parse()

	if err := run(*output, *manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}

func run(output, manifestPath string) error {
	generated := modules.AllEndpoints()
	// Enforce the transport + API↔CLI mapping contracts on the generated
	// Connect descriptors using the SHARED validators (the same code that backs
	// the fleet-wide gen.Generate), so this merge variant cannot drift from the
	// single-sourced contract. Preserved REST entries are not passed in — they
	// are hand-authored, un-migrated, and exempt by design.
	if err := gen.ValidateAgainstManifest(generated, manifestPath); err != nil {
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
