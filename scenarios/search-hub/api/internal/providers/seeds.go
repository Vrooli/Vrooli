package providers

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// seedFS holds the canonical descriptors for live providers, shipped as
// protojson files. They are the single source of truth an operator (or the
// Phase 8 bulk-registration step) feeds to `search-hub providers register`.
// Embedding them keeps the descriptor that the adapter test maps against and
// the descriptor an operator registers literally the same bytes — no drift.
//
//go:embed seeds/*.json
var seedFS embed.FS

const seedDir = "seeds"

// Seeds parses every embedded descriptor and returns them keyed by provider_id,
// in deterministic id order via SeedIDs. A malformed seed is a programmer error
// (it ships in the binary), so this panics — the parity test calls it so the
// failure surfaces at test time, never at runtime.
func Seeds() map[string]*registryv1.ProviderDescriptor {
	entries, err := fs.ReadDir(seedFS, seedDir)
	if err != nil {
		panic(fmt.Sprintf("providers: read seed dir: %v", err))
	}
	out := make(map[string]*registryv1.ProviderDescriptor, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		blob, err := seedFS.ReadFile(path.Join(seedDir, e.Name()))
		if err != nil {
			panic(fmt.Sprintf("providers: read seed %s: %v", e.Name(), err))
		}
		d := &registryv1.ProviderDescriptor{}
		if err := protojson.Unmarshal(blob, d); err != nil {
			panic(fmt.Sprintf("providers: parse seed %s: %v", e.Name(), err))
		}
		out[d.GetProviderId()] = d
	}
	return out
}

// SeedIDs returns the provider_ids of all embedded seeds, sorted.
func SeedIDs() []string {
	seeds := Seeds()
	ids := make([]string, 0, len(seeds))
	for id := range seeds {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// CLIHealthCommands returns the canonical cli-health.commands descriptor — the
// first live provider registered with the hub (Phase 3). It is a convenience
// accessor over the embedded seed of the same id.
func CLIHealthCommands() *registryv1.ProviderDescriptor {
	const id = "cli-health.commands"
	d, ok := Seeds()[id]
	if !ok {
		panic("providers: missing embedded seed " + id)
	}
	return d
}
