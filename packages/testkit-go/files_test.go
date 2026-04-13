package testkitgo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAddsTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	WriteJSON(t, path, map[string]any{"name": "alpha"})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatalf("expected trailing newline, got %q", string(data))
	}
}

func TestWriteRawJSONNormalizesMissingTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	WriteRawJSON(t, path, `{"name":"alpha"}`, 0o644)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "{\"name\":\"alpha\"}\n" {
		t.Fatalf("raw json = %q", string(data))
	}
}

func TestWriteExecutableUsesExecutableMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.sh")
	WriteExecutable(t, path, "#!/usr/bin/env bash\nexit 0\n")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o, want 755", info.Mode().Perm())
	}
}
