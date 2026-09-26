package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFakeMediaMicrophone(t *testing.T) {
	root := t.TempDir()
	basRoot := filepath.Join(root, "bas")
	wavPath := filepath.Join(basRoot, "fixtures", "dictation-reference.wav")
	if err := os.MkdirAll(filepath.Dir(wavPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(wavPath, []byte("RIFF"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings := map[string]any{
		"fake_media": map[string]any{"microphone_wav": "fixtures/dictation-reference.wav"},
	}

	t.Run("resolves relative path against project root", func(t *testing.T) {
		got, err := resolveFakeMediaMicrophone(settings, &CompileOptions{SelectorManifestRoot: basRoot})
		if err != nil {
			t.Fatalf("resolveFakeMediaMicrophone() error = %v", err)
		}
		if got != wavPath {
			t.Fatalf("resolveFakeMediaMicrophone() = %q, want %q", got, wavPath)
		}
	})

	t.Run("climbs from bas root to scenario root", func(t *testing.T) {
		scenarioLevel := map[string]any{
			"fake_media": map[string]any{"microphone_wav": filepath.Join("bas", "fixtures", "dictation-reference.wav")},
		}
		got, err := resolveFakeMediaMicrophone(scenarioLevel, &CompileOptions{SelectorManifestRoot: basRoot})
		if err != nil {
			t.Fatalf("resolveFakeMediaMicrophone() error = %v", err)
		}
		if got != wavPath {
			t.Fatalf("resolveFakeMediaMicrophone() = %q, want %q", got, wavPath)
		}
	})

	t.Run("rejects relative path without project root", func(t *testing.T) {
		_, err := resolveFakeMediaMicrophone(settings, nil)
		if err == nil || !strings.Contains(err.Error(), "project_root") {
			t.Fatalf("resolveFakeMediaMicrophone() error = %v, want project_root requirement", err)
		}
	})

	t.Run("rejects escape from project root", func(t *testing.T) {
		escape := map[string]any{
			"fake_media": map[string]any{"microphone_wav": "../../etc/passwd"},
		}
		_, err := resolveFakeMediaMicrophone(escape, &CompileOptions{SelectorManifestRoot: basRoot})
		if err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("resolveFakeMediaMicrophone() error = %v, want escape rejection", err)
		}
	})

	t.Run("rejects missing fixture", func(t *testing.T) {
		missing := map[string]any{
			"fake_media": map[string]any{"microphone_wav": "fixtures/nope.wav"},
		}
		_, err := resolveFakeMediaMicrophone(missing, &CompileOptions{SelectorManifestRoot: basRoot})
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("resolveFakeMediaMicrophone() error = %v, want not-found error", err)
		}
	})

	t.Run("absent settings yield empty path", func(t *testing.T) {
		got, err := resolveFakeMediaMicrophone(map[string]any{}, nil)
		if err != nil || got != "" {
			t.Fatalf("resolveFakeMediaMicrophone() = (%q, %v), want empty", got, err)
		}
	})
}
