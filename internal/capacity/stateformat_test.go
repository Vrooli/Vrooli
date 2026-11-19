package capacity

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/stateformat/stateformattest"
)

// schemaHistory holds the canonical DDL of every format this binary can open.
// The write version's snapshot must equal the live schema; that keeps the
// history honest and makes a schema edit show up as a test failure rather than
// as a silent reinterpretation of a version both images already know.
//
//go:embed schema_history/*.sql
var schemaHistoryFS embed.FS

func TestStateFormatDescribesTheCurrentSchema(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if store.Name != "capacity" || store.Write != stateformat.Version(SchemaVersion) {
		t.Fatalf("store = %s@v%d, want capacity@v%d", store.Name, store.Write, SchemaVersion)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if len(store.StatePaths) != 1 || store.StatePaths[0] != capacityStatePath {
		t.Fatalf("state paths = %v, want [%s]", store.StatePaths, capacityStatePath)
	}
	if got := store.Schemas[store.Write]; got != capacitySchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s; a deliberate shape change must bump SchemaVersion and re-pin", got, capacitySchemaDigest)
	}
}

// TestCapacitySchemaHistoryMatchesTheLiveSchema is the guard on the frozen
// snapshot. Regenerate it only when SchemaVersion changes:
//
//	UPDATE_SCHEMA_HISTORY=1 go test ./internal/capacity -run TestCapacitySchemaHistory
func TestCapacitySchemaHistoryMatchesTheLiveSchema(t *testing.T) {
	name := fmt.Sprintf("schema_history/v%d.sql", SchemaVersion)
	if os.Getenv("UPDATE_SCHEMA_HISTORY") != "" {
		dir := "schema_history"
		stale, err := filepath.Glob(filepath.Join(dir, "*.sql"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range stale {
			version, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(file), ".sql"))
			if err != nil || version < SchemaVersion {
				continue
			}
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
		}
		content := stateformat.CanonicalDDL(schemaSQL) + "\n"
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("v%d.sql", SchemaVersion)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data, err := schemaHistoryFS.ReadFile(name)
	if err != nil {
		t.Fatalf("read the frozen schema %s: %v", name, err)
	}
	if got, want := stateformat.DigestDDL(string(data)), stateformat.DigestDDL(schemaSQL); got != want {
		t.Fatalf("%s digest = %s, live schema digest = %s; regenerate the snapshot", name, got, want)
	}
}

// TestVocabularyCoversEveryConstant keeps the vocabulary from going stale. A
// new string constant is either a value this package can write — so a serving
// image must learn it before the candidate activates — or an explicit
// non-stored exception with a reason. Silence is not an option.
func TestVocabularyCoversEveryConstant(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	allow := make([]string, 0, len(notStoredConstants))
	for name := range notStoredConstants {
		allow = append(allow, name)
	}
	stateformattest.AssertVocabularyCoversConstants(t, stateformattest.Options{
		Dir:        ".",
		Vocabulary: store.Vocabulary,
		Allow:      allow,
	})
}
