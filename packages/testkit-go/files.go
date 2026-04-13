package testkitgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func WriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(normalizeTrailingNewline(contents, false)), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func WriteExecutable(t *testing.T, path, contents string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(normalizeTrailingNewline(contents, false)), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func WriteExecutableOnPath(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	WriteExecutable(t, path, contents)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func WriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	WriteJSONMode(t, path, value, 0o644)
}

func WriteJSONMode(t *testing.T, path string, value any, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func WriteRawJSON(t *testing.T, path, raw string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(normalizeTrailingNewline(raw, true)), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func WriteMalformedJSON(t *testing.T, path, raw string, mode os.FileMode) {
	t.Helper()
	WriteRawJSON(t, path, raw, mode)
}

func ReadJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return parsed
}

func ReadJSONFileInto[T any](t *testing.T, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed T
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return parsed
}

func WriteRelativeFile(t *testing.T, root, relPath, contents string) {
	t.Helper()
	WriteFile(t, filepath.Join(root, filepath.FromSlash(relPath)), contents)
}

func WriteRelativeExecutable(t *testing.T, root, relPath, contents string) string {
	t.Helper()
	return WriteExecutable(t, filepath.Join(root, filepath.FromSlash(relPath)), contents)
}

func WriteRelativeMalformedJSON(t *testing.T, root, relPath, raw string, mode os.FileMode) {
	t.Helper()
	WriteMalformedJSON(t, filepath.Join(root, filepath.FromSlash(relPath)), raw, mode)
}

func normalizeTrailingNewline(contents string, force bool) string {
	contents = strings.ReplaceAll(contents, "\r\n", "\n")
	if force && !strings.HasSuffix(contents, "\n") {
		return contents + "\n"
	}
	return contents
}
