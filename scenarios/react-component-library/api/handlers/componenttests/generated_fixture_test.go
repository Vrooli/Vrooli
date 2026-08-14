package componenttests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveRequiredSurfaceTokenPreservesOtherSurfaceFamilies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "design-tokens.css")
	content := ":root {\n  --color-surface: #fff;\n  --color-surface-muted: #f8fafc;\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := removeRequiredSurfaceToken(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != ":root {\n  --color-surface-muted: #f8fafc;\n}\n" {
		t.Fatalf("removed the wrong declarations: %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o640); got != want {
		t.Fatalf("mode = %o, want %o", got, want)
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"--color-border", "--color-surface"}, "--color-surface") {
		t.Fatal("contains returned false for a present token")
	}
	if contains([]string{"--color-border"}, "--color-surface") {
		t.Fatal("contains returned true for an absent token")
	}
}
