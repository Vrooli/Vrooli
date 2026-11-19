package ai

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateMeshArtifactRejectsNonArchive proves a provider that returns
// something other than a bundle fails the job rather than being stored as a
// successful mesh.
func TestValidateMeshArtifactRejectsNonArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.zip")
	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMeshArtifact(path); err == nil {
		t.Fatal("non-archive mesh output accepted")
	}
}

func TestValidateMeshArtifactRejectsMissingFile(t *testing.T) {
	if err := validateMeshArtifact(filepath.Join(t.TempDir(), "absent.zip")); err == nil {
		t.Fatal("missing mesh bundle accepted")
	}
}
