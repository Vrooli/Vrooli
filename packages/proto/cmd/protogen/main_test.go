package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProtoRootFromRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	proto := filepath.Join(root, "packages", "proto")
	writeProtoMarkers(t, proto)

	got, err := resolveProtoRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != proto {
		t.Fatalf("resolveProtoRoot = %q, want %q", got, proto)
	}
}

func TestResolveProtoRootFromNestedPackagesDirectory(t *testing.T) {
	root := t.TempDir()
	proto := filepath.Join(root, "packages", "proto")
	writeProtoMarkers(t, proto)

	got, err := resolveProtoRoot(filepath.Join(root, "packages", "proto", "cmd", "protogen"))
	if err != nil {
		t.Fatal(err)
	}
	if got != proto {
		t.Fatalf("resolveProtoRoot = %q, want %q", got, proto)
	}
}

func writeProtoMarkers(t *testing.T, proto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(proto, "schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proto, "buf.yaml"), []byte("version: v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
