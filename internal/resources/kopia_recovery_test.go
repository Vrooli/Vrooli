package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKopiaRepositoryEntriesReportsAddressesWithoutValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	data := `{"version":1,"repos":[{"name":"offsite","backend":"s3"},{"name":"nightly","backend":"filesystem"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := KopiaRepositoryEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].LogicalID != "vrooli/kopia/nightly" || entries[1].LogicalID != "vrooli/kopia/offsite" {
		t.Fatalf("entries = %#v", entries)
	}
	for _, entry := range entries {
		if entry.Field != "repository-passphrase" || entry.Repository == "" {
			t.Fatalf("invalid recovery entry %#v", entry)
		}
	}
}

func TestKopiaRepositoryEntriesMissingRegistryIsEmpty(t *testing.T) {
	entries, err := KopiaRepositoryEntries(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Fatalf("entries = %#v, want nil", entries)
	}
}
