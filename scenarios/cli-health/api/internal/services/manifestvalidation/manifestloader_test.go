package manifestvalidation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestFilesystemManifestLoader_ExplicitScenarioPath proves that when ctx carries
// an explicit scenario root (WithScenarioPath), the loader reads cli/manifest.json
// from that directory rather than resolving by name under the repo scenarios/
// tree — the path that lets deep template validation read a temp-generated
// scenario's manifest.
func TestFilesystemManifestLoader_ExplicitScenarioPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "scenarios", "generated-demo")
	if err := os.MkdirAll(filepath.Join(root, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"version":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(root, "cli", "manifest.json"), want, 0o644); err != nil {
		t.Fatal(err)
	}

	// RepoRoot points elsewhere; the ctx path must win.
	loader := NewFilesystemManifestLoader(t.TempDir())
	ctx := WithScenarioPath(context.Background(), root)

	raw, path, err := loader.Load(ctx, "generated-demo")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(raw) != string(want) {
		t.Fatalf("raw = %q, want %q", raw, want)
	}
	if path != filepath.Join(root, "cli", "manifest.json") {
		t.Fatalf("path = %q, want the explicit scenario path", path)
	}
}

// TestFilesystemManifestLoader_ExplicitPathMissingManifest proves a scenario root
// without a manifest still surfaces os.ErrNotExist (so the service emits the
// manifest-missing finding), rather than an opaque error.
func TestFilesystemManifestLoader_ExplicitPathMissingManifest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "scenarios", "no-manifest")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	loader := NewFilesystemManifestLoader(t.TempDir())
	ctx := WithScenarioPath(context.Background(), root)

	_, _, err := loader.Load(ctx, "no-manifest")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want os.ErrNotExist", err)
	}
}
