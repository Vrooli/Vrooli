package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGeneratedFileAtomicallyPreservesMode(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "test-genie.json")
	if err := os.WriteFile(output, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := writeGeneratedFile(output, []byte("generated\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != "generated\n" {
		t.Fatalf("generated output = %q, %v", got, err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("generated mode = %#o, want %#o", got, 0o640)
	}
}

func TestWriteGeneratedFileRejectsSymlinkOutput(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(external, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "test-genie.json")
	if err := os.Symlink(external, output); err != nil {
		t.Fatal(err)
	}
	if err := writeGeneratedFile(output, []byte("replace")); err == nil {
		t.Fatal("writeGeneratedFile accepted a symlink output")
	}
	got, err := os.ReadFile(external)
	if err != nil || string(got) != "preserve" {
		t.Fatalf("external target changed: %q, %v", got, err)
	}
}
