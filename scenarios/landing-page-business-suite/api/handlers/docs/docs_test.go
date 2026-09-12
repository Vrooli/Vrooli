package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWithinDirectoryRejectsSiblingPrefix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs")
	if !IsWithinDirectory(root, filepath.Join(root, "guide.md")) {
		t.Fatal("expected file under root to be accepted")
	}
	if IsWithinDirectory(root, root+"-old/guide.md") {
		t.Fatal("sibling path sharing root prefix must be rejected")
	}
}

func TestExtractTitleUsesHeadingThenFilename(t *testing.T) {
	if got := ExtractTitle("# Getting Started\n\nBody", "guide.md"); got != "Getting Started" {
		t.Fatalf("heading title = %q", got)
	}
	if got := ExtractTitle("No heading", "api-reference.md"); got != "api-reference" {
		t.Fatalf("fallback title = %q", got)
	}
}

func TestBuildTreeSortsDirectoriesAndMarkdownEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "guides"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "guides", "intro.md"), []byte("# Intro"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z-last.md"), []byte("# Last"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a-first.md"), []byte("# First"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := BuildTree(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[0].Name != "guides" || entries[1].Name != "a-first.md" || entries[2].Name != "z-last.md" {
		t.Fatalf("entries = %#v", entries)
	}
}
