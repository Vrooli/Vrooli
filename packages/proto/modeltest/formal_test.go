package modeltest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFormalArtifactPathUsesAbsoluteSourceAndTrimpathWorkingDirectory(t *testing.T) {
	temp := t.TempDir()
	moduleRoot := filepath.Join(temp, "module")
	workingDir := filepath.Join(moduleRoot, "internal", "flow")
	if err := os.MkdirAll(workingDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.test/device-studio\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("absolute source location", func(t *testing.T) {
		caller := filepath.Join(moduleRoot, "internal", "flow", "generated", "replay.go")
		got, err := resolveFormalArtifactPath(caller, "artifact.json", workingDir)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(moduleRoot, "internal", "flow", "generated", "artifact.json")
		if got != want {
			t.Fatalf("resolved path = %q, want %q", got, want)
		}
	})

	t.Run("trimpath generated subpackage location", func(t *testing.T) {
		caller := filepath.Join("example.test", "device-studio", "internal", "flow", "generated", "replay.go")
		got, err := resolveFormalArtifactPath(caller, "artifact.json", workingDir)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(moduleRoot, "internal", "flow", "generated", "artifact.json")
		if got != want {
			t.Fatalf("resolved path = %q, want %q", got, want)
		}
	})

	t.Run("trimpath package test location", func(t *testing.T) {
		caller := filepath.Join("example.test", "device-studio", "internal", "flow", "flow_test.go")
		got, err := resolveFormalArtifactPath(caller, "generated/artifact.json", workingDir)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(moduleRoot, "internal", "flow", "generated", "artifact.json")
		if got != want {
			t.Fatalf("resolved path = %q, want %q", got, want)
		}
	})

	t.Run("rejects trimpath caller outside module", func(t *testing.T) {
		caller := filepath.Join("unrelated", "internal", "flow", "generated", "replay.go")
		if _, err := resolveFormalArtifactPath(caller, "artifact.json", workingDir); err == nil {
			t.Fatal("expected an out-of-module caller to be rejected")
		}
	})
}
