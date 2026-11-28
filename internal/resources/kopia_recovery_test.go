package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKopiaRepositoryEntriesReportsAddressesWithoutValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	data := `{"version":1,"repos":[{"name":"offsite","backend":"s3"},{"name":"nightly","backend":"filesystem"},{"name":"mirror","backend":"sftp"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := KopiaRepositoryEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("entries = %#v", entries)
	}
	want := map[string]bool{
		"nightly:repository-passphrase": true,
		"mirror:repository-passphrase":  true,
		"mirror:sftp-private-key":       true,
		"offsite:repository-passphrase": true,
		"offsite:s3-access-key-id":      true,
		"offsite:s3-secret-access-key":  true,
	}
	for _, entry := range entries {
		key := entry.Repository + ":" + entry.Field
		if !want[key] {
			t.Fatalf("invalid recovery entry %#v", entry)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing recovery fields %v", want)
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
