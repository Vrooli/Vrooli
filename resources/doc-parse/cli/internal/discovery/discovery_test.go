package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceRootFromBuildMetadata(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "bin", "resource-doc-parse")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable+".build.meta", []byte(`{"module_path":"`+root+`/cli"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, want := sourceRootFromBuildMetadata(executable), root; got != want {
		t.Fatalf("sourceRootFromBuildMetadata() = %q, want %q", got, want)
	}
}

func TestSourceRootFromBuildMetadataIgnoresMalformedMetadata(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "resource-doc-parse")
	if err := os.WriteFile(executable+".build.meta", []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := sourceRootFromBuildMetadata(executable); got != "" {
		t.Fatalf("sourceRootFromBuildMetadata() = %q, want empty root", got)
	}
}
