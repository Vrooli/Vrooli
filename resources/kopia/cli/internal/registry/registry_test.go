package registry_test

import (
	"path/filepath"
	"resource-kopia/cli/internal/registry"
	"strings"
	"testing"
)

func newRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "registry.json")
	return registry.New(path)
}

func TestUpsertGetRemove(t *testing.T) {
	reg := newRegistry(t)

	if _, found, err := reg.Get("nope"); err != nil || found {
		t.Fatalf("Get on empty registry found=%v err=%v", found, err)
	}

	fs := registry.Entry{Name: "nightly", Backend: registry.BackendFilesystem, ConfigFile: "/cfg/nightly", Path: "/var/backups"}
	if err := reg.Upsert(fs); err != nil {
		t.Fatalf("Upsert fs error = %v", err)
	}
	s3 := registry.Entry{Name: "offsite", Backend: registry.BackendS3, ConfigFile: "/cfg/offsite", Bucket: "b", Endpoint: "minio:9000"}
	if err := reg.Upsert(s3); err != nil {
		t.Fatalf("Upsert s3 error = %v", err)
	}

	entries, err := reg.Load()
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Sorted by name: nightly, offsite.
	if entries[0].Name != "nightly" || entries[1].Name != "offsite" {
		t.Fatalf("entries not sorted: %v", entries)
	}
	if entries[0].CreatedAt == "" {
		t.Fatal("CreatedAt should be stamped")
	}

	got, found, err := reg.Get("offsite")
	if err != nil || !found {
		t.Fatalf("Get offsite found=%v err=%v", found, err)
	}
	if got.Bucket != "b" || got.Endpoint != "minio:9000" {
		t.Fatalf("offsite entry mismatch: %+v", got)
	}

	if err := reg.Remove("nightly"); err != nil {
		t.Fatalf("Remove error = %v", err)
	}
	if _, found, _ := reg.Get("nightly"); found {
		t.Fatal("nightly should be removed")
	}
}

func TestUpsertPreservesCreatedAt(t *testing.T) {
	reg := newRegistry(t)
	if err := reg.Upsert(registry.Entry{Name: "r", Backend: registry.BackendFilesystem, ConfigFile: "/a", Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	first, _, _ := reg.Get("r")
	if err := reg.Upsert(registry.Entry{Name: "r", Backend: registry.BackendFilesystem, ConfigFile: "/a", Path: "/p2"}); err != nil {
		t.Fatal(err)
	}
	second, _, _ := reg.Get("r")
	if second.CreatedAt != first.CreatedAt {
		t.Fatalf("CreatedAt changed on update: %q -> %q", first.CreatedAt, second.CreatedAt)
	}
	if second.Path != "/p2" {
		t.Fatalf("expected updated path, got %q", second.Path)
	}
}

func TestRegistryPathOutsideRepoData(t *testing.T) {
	reg := newRegistry(t)
	// The state path must not resolve into a repo-local data/ root.
	if strings.Contains(reg.Path, "/data/") {
		t.Fatalf("registry path %q must not be under repo-local data/", reg.Path)
	}
}
