package driverpref

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"workspace-sandbox/internal/driverid"
)

func TestLoadMissingPreference(t *testing.T) {
	_, err := Load(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load error = %v, want ErrNotFound", err)
	}
}

func TestSaveAndLoadPreference(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, driverid.OverlayfsUserNS); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != driverid.OverlayfsUserNS {
		t.Fatalf("Load = %q, want %q", got, driverid.OverlayfsUserNS)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("Load succeeded for malformed JSON")
	}
}

func TestLoadRejectsUnknownDriver(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"driverId":"made-up"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("Load succeeded for unknown driver")
	}
}

func TestSaveRejectsUnknownDriver(t *testing.T) {
	if err := Save(t.TempDir(), driverid.ID("made-up")); err == nil {
		t.Fatal("Save succeeded for unknown driver")
	}
}
