package hostreqkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagedFilePathRejectsAliasesAndNonDirectories(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, "foreign")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(foreign, alias); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(root, "regular")
	if err := os.WriteFile(regular, []byte("operator"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(alias, "owned.conf"), filepath.Join(alias, "nested", "owned.conf"), filepath.Join(regular, "owned.conf"), alias} {
		if err := ValidateManagedFilePath(path, nil); err == nil {
			t.Fatalf("unqualified destination admitted: %s", path)
		}
	}
	data, err := os.ReadFile(regular)
	if err != nil || string(data) != "operator" {
		t.Fatal("qualification changed operator file")
	}
	entries, err := os.ReadDir(foreign)
	if err != nil || len(entries) != 0 {
		t.Fatal("qualification wrote through alias")
	}
}

func TestManagedFilePathAllowsRegularAndMissingOwnedDestination(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "owned.conf")
	if err := os.WriteFile(regular, []byte("owned"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{regular, filepath.Join(root, "new", "nested", "owned.conf")} {
		if err := ValidateManagedFilePath(path, nil); err != nil {
			t.Fatalf("valid destination refused: %s %v", path, err)
		}
	}
}

func TestManagedFilePathPreservesUnknownAndInvalidAncestry(t *testing.T) {
	for _, path := range []string{"relative.conf", "/etc/../owned.conf", "/", "/owned\x00.conf"} {
		if err := ValidateManagedFilePath(path, nil); err == nil {
			t.Fatalf("invalid destination admitted: %q", path)
		}
	}
	if err := ValidateManagedFilePath("/owned/file.conf", func(string) (os.FileInfo, error) { return nil, os.ErrPermission }); err == nil {
		t.Fatal("unreadable ancestry admitted")
	}
}
