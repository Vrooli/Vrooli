package storage

import (
	"os"
	"path/filepath"
	"testing"
)

type samplePayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestWriteJSONAtomicCreatesFileAndDir(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "nested", "payload.json")
	payload := samplePayload{Name: "alpha", Count: 3}

	if err := WriteJSONAtomic(path, payload); err != nil {
		t.Fatalf("expected write to succeed: %v", err)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	var decoded samplePayload
	if _, err := ReadJSON(path, &decoded); err != nil {
		t.Fatalf("expected read to succeed: %v", err)
	}

	if decoded != payload {
		t.Fatalf("expected payload %+v, got %+v", payload, decoded)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file stat to succeed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected permissions 0600, got %04o", perm)
	}
	if len(bytes) == 0 {
		t.Fatalf("expected file to contain JSON bytes")
	}
}

func TestWriteJSONAtomicRejectsUnsupportedData(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "payload.json")

	data := map[string]any{
		"bad": func() {},
	}

	if err := WriteJSONAtomic(path, data); err == nil {
		t.Fatalf("expected error for unsupported data")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file not to exist, got %v", err)
	}
}

func TestReadJSONMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	payload := samplePayload{Name: "existing", Count: 9}

	ok, err := ReadJSON(path, &payload)
	if err != nil {
		t.Fatalf("expected no error for missing file: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false for missing file")
	}
	if payload.Name != "existing" || payload.Count != 9 {
		t.Fatalf("expected payload to remain unchanged, got %+v", payload)
	}
}

func TestReadJSONInvalidFile(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "bad.json")

	if err := os.WriteFile(path, []byte("{bad"), 0o644); err != nil {
		t.Fatalf("expected write to succeed: %v", err)
	}

	var payload samplePayload
	ok, err := ReadJSON(path, &payload)
	if !ok {
		t.Fatalf("expected ok=true for existing file")
	}
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestReadJSONBytes(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "raw.json")
	expected := []byte(`{"name":"beta"}`)

	if err := os.WriteFile(path, expected, 0o644); err != nil {
		t.Fatalf("expected write to succeed: %v", err)
	}

	bytes, err := ReadJSONBytes(path)
	if err != nil {
		t.Fatalf("expected ReadJSONBytes to succeed: %v", err)
	}
	if string(bytes) != string(expected) {
		t.Fatalf("expected %q, got %q", string(expected), string(bytes))
	}
}
