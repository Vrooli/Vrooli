package coreactivation

import (
	"path/filepath"
	"reflect"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatIsAValidVersionedStore(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if store.Name != "core-activation" {
		t.Fatalf("store name = %q", store.Name)
	}
	if store.Write != stateformat.Version(recordSchemaVersion) || len(store.Reads) != 1 || store.Reads[0] != store.Write {
		t.Fatalf("write %d reads %v; the record schema version is %d", store.Write, store.Reads, recordSchemaVersion)
	}
	if store.Schemas[store.Write] == "" {
		t.Fatal("the write version has no schema digest")
	}
	// Additive claims are earned by a downlevel fixture. There is no older
	// read version yet, so the store must not claim one.
	if store.AdditiveFrom != 0 || store.MigratesFrom != 0 {
		t.Fatalf("store claims additive_from=%d migrates_from=%d without a proven prior reader", store.AdditiveFrom, store.MigratesFrom)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("store is invalid: %v", err)
	}
}

// The declared paths are the ones the store actually writes. A snapshot that
// missed one would restore an activation record whose witnesses disagree.
func TestStateFormatPathsAreTheStoreFiles(t *testing.T) {
	home := t.TempDir()
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	activation, err := NewStore(home)
	if err != nil {
		t.Fatal(err)
	}
	runtimeHome, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	if activation.Directory() != filepath.Join(stateRoot, "core-activation") {
		t.Fatalf("store directory = %s", activation.Directory())
	}
	// Store paths are relative to the runtime home, never to the OS home.
	declared := map[string]bool{}
	for _, relative := range store.StatePaths {
		declared[filepath.Join(runtimeHome, relative)] = true
	}
	if len(declared) != len(store.StatePaths) {
		t.Fatalf("declared state paths collide: %v", store.StatePaths)
	}
	for _, name := range []string{"current.json", "initialized", "enrollment.json", "enrollment-initialized"} {
		path := filepath.Join(activation.Directory(), name)
		if !declared[path] {
			t.Fatalf("store writes %s but does not declare it: %v", path, store.StatePaths)
		}
	}
	for path := range declared {
		if filepath.Dir(path) != activation.Directory() {
			t.Fatalf("declared path %s is outside the store directory %s", path, activation.Directory())
		}
	}
}

func TestStateFormatDigestIgnoresSourceLayout(t *testing.T) {
	first, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	// The digest is derived from the compiled shape, so it is the same on every
	// machine and every run for the same build.
	second, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if first.Schemas[1] != second.Schemas[1] {
		t.Fatalf("schema digest is unstable: %s != %s", first.Schemas[1], second.Schemas[1])
	}
}

// The stored shape is pinned so a field change cannot silently cross a format
// version. Bump Write and Reads, add the downlevel fixture, then re-pin.
func TestStateFormatDigestIsPinned(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Schemas[store.Write]; got != coreActivationSchemaDigest {
		t.Fatalf("stored shape digest = %s, pinned %s; bump the store's format version (Write and Reads) and re-pin if the change is intended", got, coreActivationSchemaDigest)
	}
}

// A stored field change must move the digest, or an incompatible image could be
// treated as an identical one.
func TestStateFormatDigestTracksTheStoredShape(t *testing.T) {
	type smaller struct {
		SchemaVersion int    `json:"schema_version"`
		ID            string `json:"id"`
	}
	type larger struct {
		SchemaVersion int    `json:"schema_version"`
		ID            string `json:"id"`
		Home          string `json:"home"`
	}
	small, err := stateformat.DigestTypes(reflect.TypeFor[smaller]())
	if err != nil {
		t.Fatal(err)
	}
	large, err := stateformat.DigestTypes(reflect.TypeFor[larger]())
	if err != nil {
		t.Fatal(err)
	}
	if small == large {
		t.Fatal("adding a stored field kept the schema digest")
	}
}
