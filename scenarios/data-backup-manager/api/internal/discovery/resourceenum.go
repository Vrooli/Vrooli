package discovery

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// ResourceRef is one platform resource as reported by the CLI: enough to read
// its durable_data declaration without hardcoding the repo layout.
type ResourceRef struct {
	Name         string
	Driver       string
	Enabled      bool
	ManifestPath string
}

// CLIResourceEnumerator enumerates resources by shelling `vrooli resource list
// --json` and keeping only enabled host-filesystem (external-cli) resources —
// the only ones that can declare durable host state. It is the single place the
// `vrooli` CLI is shelled.
//
// Failure degrades gracefully: if the CLI is missing or returns unparseable
// output, Enumerate returns no resources and no error, so discovery still works
// from the WellKnownScanner alone.
type CLIResourceEnumerator struct {
	// bin is the CLI binary to invoke (default "vrooli").
	bin string
}

// NewResourceEnumerator constructs the production enumerator.
func NewResourceEnumerator() *CLIResourceEnumerator {
	return &CLIResourceEnumerator{bin: "vrooli"}
}

// Compile-time guarantee.
var _ ResourceEnumerator = (*CLIResourceEnumerator)(nil)

// resourceListEnvelope mirrors the relevant slice of `vrooli resource list
// --json`. Unknown fields are ignored.
type resourceListEnvelope struct {
	Resources []struct {
		Name         string `json:"name"`
		Enabled      bool   `json:"enabled"`
		Driver       string `json:"driver"`
		ManifestPath string `json:"manifest_path"`
	} `json:"resources"`
}

func (e *CLIResourceEnumerator) Enumerate(ctx context.Context) ([]ResourceRef, error) {
	bin := e.bin
	if strings.TrimSpace(bin) == "" {
		bin = "vrooli"
	}
	out, err := exec.CommandContext(ctx, bin, "resource", "list", "--json").Output()
	if err != nil {
		return nil, nil // graceful: discovery falls back to the well-known scanner.
	}
	return parseEnabledExternalCLI(out), nil
}

// parseEnabledExternalCLI parses `vrooli resource list --json` output and keeps
// only enabled external-cli resources with a manifest path. Unparseable input
// yields no resources (never an error), so a CLI shape change degrades to the
// well-known scanner rather than breaking discovery.
func parseEnabledExternalCLI(out []byte) []ResourceRef {
	var env resourceListEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil
	}
	refs := make([]ResourceRef, 0, len(env.Resources))
	for _, r := range env.Resources {
		if !r.Enabled || r.Driver != "external-cli" {
			continue
		}
		if strings.TrimSpace(r.ManifestPath) == "" {
			continue
		}
		refs = append(refs, ResourceRef{
			Name:         r.Name,
			Driver:       r.Driver,
			Enabled:      r.Enabled,
			ManifestPath: r.ManifestPath,
		})
	}
	return refs
}
