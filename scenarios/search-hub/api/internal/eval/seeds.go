package eval

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// seedFS holds the SHRINKING set of starter eval suites search-hub still ships,
// as protojson files, for providers that have NOT yet adopted corpus
// self-registration. A scenario that owns a `.vrooli/search.json` with a `tests`
// block self-registers its corpus at boot (searchregister.Register → the eval
// store mirrors the file — corpusStoreMirrorsFile), and so needs no seed here:
// knowledge-observatory graduated this way (its starter seed was migrated into
// scenarios/knowledge-observatory/.vrooli/search.json and deleted). The remaining
// seeds (swarm-manager.records, ui-health.surfaces) persist ONLY because those
// scenarios do not yet self-register a search provider at all (no search.json, no
// boot wiring); migrating them is provider-adoption work tracked separately. When
// the last scenario adopts, this whole mechanism is deleted.
//
//go:embed seeds/*.json
var seedFS embed.FS

const seedDir = "seeds"

// Seeds parses every embedded suite and returns them keyed by suite_id, in
// deterministic id order via SeedIDs. A malformed seed is a programmer error (it
// ships in the binary), so this panics — the seed-validity test calls it so the
// failure surfaces at test time, never at runtime.
func Seeds() map[string]*evalv1.EvalSuite {
	entries, err := fs.ReadDir(seedFS, seedDir)
	if err != nil {
		panic(fmt.Sprintf("eval: read seed dir: %v", err))
	}
	out := make(map[string]*evalv1.EvalSuite, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		blob, err := seedFS.ReadFile(path.Join(seedDir, e.Name()))
		if err != nil {
			panic(fmt.Sprintf("eval: read seed %s: %v", e.Name(), err))
		}
		s := &evalv1.EvalSuite{}
		if err := protojson.Unmarshal(blob, s); err != nil {
			panic(fmt.Sprintf("eval: parse seed %s: %v", e.Name(), err))
		}
		out[s.GetSuiteId()] = s
	}
	return out
}

// SeedIDs returns the suite_ids of all embedded seeds, sorted.
func SeedIDs() []string {
	seeds := Seeds()
	ids := make([]string, 0, len(seeds))
	for id := range seeds {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// RegisterSeeds upserts every embedded suite into the store. Idempotent (upsert
// keyed by suite_id), so it runs at every boot — re-registering shipped suites
// without duplicating rows. A validation failure on a shipped seed is a
// programmer error and is returned so boot fails loudly.
func RegisterSeeds(ctx context.Context, store Store) error {
	for _, id := range SeedIDs() {
		s := Seeds()[id]
		if _, err := store.UpsertSuite(ctx, s); err != nil {
			return fmt.Errorf("register seed suite %q: %w", id, err)
		}
	}
	return nil
}
