package projectmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func writeServiceJSON(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestMode_Development(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"mode":"development"}`)
	SetStartDirForTesting(dir)
	if got := Mode(); got != ModeDevelopment {
		t.Fatalf("Mode() = %q, want %q", got, ModeDevelopment)
	}
	if !IsDevelopment() || IsProduction() {
		t.Fatalf("flags inconsistent: dev=%v prod=%v", IsDevelopment(), IsProduction())
	}
}

func TestMode_Production(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"mode":"production"}`)
	SetStartDirForTesting(dir)
	if got := Mode(); got != ModeProduction {
		t.Fatalf("Mode() = %q, want %q", got, ModeProduction)
	}
	if IsDevelopment() || !IsProduction() {
		t.Fatalf("flags inconsistent: dev=%v prod=%v", IsDevelopment(), IsProduction())
	}
}

func TestMode_FileMissing_DefaultsToDevelopment(t *testing.T) {
	// Use a directory tree with no .vrooli/service.json anywhere reachable.
	dir := t.TempDir()
	SetStartDirForTesting(dir)
	if got := Mode(); got != ModeDevelopment {
		t.Fatalf("Mode() with no service.json = %q, want %q", got, ModeDevelopment)
	}
}

func TestMode_InvalidValue_DefaultsToDevelopment(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"mode":"staging"}`)
	SetStartDirForTesting(dir)
	if got := Mode(); got != ModeDevelopment {
		t.Fatalf("Mode() with invalid value = %q, want %q", got, ModeDevelopment)
	}
}

func TestMode_AbsentField_DefaultsToDevelopment(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{}`)
	SetStartDirForTesting(dir)
	if got := Mode(); got != ModeDevelopment {
		t.Fatalf("Mode() with absent field = %q, want %q", got, ModeDevelopment)
	}
}

func TestMode_AscendsToFindServiceJSON(t *testing.T) {
	root := t.TempDir()
	writeServiceJSON(t, root, `{"mode":"production"}`)
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}
	SetStartDirForTesting(deep)
	if got := Mode(); got != ModeProduction {
		t.Fatalf("Mode() = %q, want %q", got, ModeProduction)
	}
}

func TestMode_MalformedJSON_DefaultsToDevelopment(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `not json`)
	SetStartDirForTesting(dir)
	if got := Mode(); got != ModeDevelopment {
		t.Fatalf("Mode() with malformed json = %q, want %q", got, ModeDevelopment)
	}
}
