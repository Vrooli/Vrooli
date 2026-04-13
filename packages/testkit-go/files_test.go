package testkitgo

import (
	"os"
	"path/filepath"
	"strings"
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

func TestWriteMalformedJSONPersistsInvalidPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	WriteMalformedJSON(t, path, `{`, 0o600)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "{\n" {
		t.Fatalf("malformed json = %q", string(data))
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

func TestWriteExecutableOnPathInstallsBinaryAndUpdatesPATH(t *testing.T) {
	t.Setenv("PATH", "")
	path := WriteExecutableOnPath(t, "fixture-cli", "#!/usr/bin/env bash\nexit 0\n")

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := os.Getenv("PATH"); !strings.HasPrefix(got, filepath.Dir(path)) {
		t.Fatalf("PATH = %q, want prefix %q", got, filepath.Dir(path))
	}
}

func TestWriteRelativeMalformedJSONUsesRelativePath(t *testing.T) {
	root := t.TempDir()
	WriteRelativeMalformedJSON(t, root, "nested/fixture.json", `{`, 0o644)

	data, err := os.ReadFile(filepath.Join(root, "nested", "fixture.json"))
	if err != nil {
		t.Fatalf("read relative malformed json: %v", err)
	}
	if string(data) != "{\n" {
		t.Fatalf("relative malformed json = %q", string(data))
	}
}
