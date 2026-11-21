package resources

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireManagedServiceDirectoryPreservesInstalledArtifactOnFailure(t *testing.T) {
	for _, phase := range []string{"acquire", "validate"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			artifactPath := filepath.Join(root, "service")
			if err := os.MkdirAll(artifactPath, 0o755); err != nil {
				t.Fatal(err)
			}
			installed := filepath.Join(artifactPath, "version.txt")
			if err := os.WriteFile(installed, []byte("installed"), 0o600); err != nil {
				t.Fatal(err)
			}

			failure := errors.New("candidate failed")
			err := acquireManagedServiceDirectory(artifactPath,
				func(stagedPath string) error {
					if err := os.WriteFile(filepath.Join(stagedPath, "partial.txt"), []byte("partial"), 0o600); err != nil {
						return err
					}
					if phase == "acquire" {
						return failure
					}
					return nil
				},
				func(string) error {
					if phase == "validate" {
						return failure
					}
					return nil
				})
			if !errors.Is(err, failure) {
				t.Fatalf("acquireManagedServiceDirectory() error = %v, want candidate failure", err)
			}
			got, readErr := os.ReadFile(installed)
			if readErr != nil {
				t.Fatalf("installed artifact was not preserved: %v", readErr)
			}
			if string(got) != "installed" {
				t.Fatalf("installed artifact content = %q, want %q", got, "installed")
			}
			if _, err := os.Stat(filepath.Join(artifactPath, "partial.txt")); !os.IsNotExist(err) {
				t.Fatalf("partial candidate leaked into installed artifact, stat error = %v", err)
			}
		})
	}
}

func TestAcquireManagedServiceDirectoryCommitsValidatedArtifact(t *testing.T) {
	root := t.TempDir()
	artifactPath := filepath.Join(root, "service")
	if err := os.MkdirAll(artifactPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "version.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := acquireManagedServiceDirectory(artifactPath,
		func(stagedPath string) error {
			return os.WriteFile(filepath.Join(stagedPath, "version.txt"), []byte("new"), 0o600)
		},
		func(stagedPath string) error {
			_, err := os.Stat(filepath.Join(stagedPath, "version.txt"))
			return err
		})
	if err != nil {
		t.Fatalf("acquireManagedServiceDirectory() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(artifactPath, "version.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("committed artifact content = %q, want %q", got, "new")
	}
}
