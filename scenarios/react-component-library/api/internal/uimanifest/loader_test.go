package uimanifest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a tiny helper that fails the test on error.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

const validManifest = `{
  "contract": {
    "kind": "scenario-ui",
    "schema": "scenario-ui-manifest/v1",
    "template": "react-vite"
  },
  "slots": {
    "layout-nav": {
      "dir": "ui/src/layout",
      "pathPattern": "{dir}/{ComponentName}.tsx"
    },
    "shared-component": {
      "dir": "ui/src/components"
    }
  },
  "defaults": {"slot": "shared-component"}
}`

const validServiceJSON = `{
  "service": {"id": "demo"},
  "generation": {"template": {"id": "react-vite", "version": "0.1.0"}}
}`

func setupRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "templates", "scenarios", "react-vite", "ui", "manifest.json"), validManifest)
	writeFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), validServiceJSON)
	return root
}

func TestLoad_Happy(t *testing.T) {
	root := setupRepo(t)
	l := NewFSLoader(root)
	mf, err := l.Load("demo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if mf.Contract.Kind != "scenario-ui" || mf.Contract.Schema != "scenario-ui-manifest/v1" {
		t.Fatalf("unexpected contract: %+v", mf.Contract)
	}
	if len(mf.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(mf.Slots))
	}
	slot, ok := mf.LookupSlot("layout-nav")
	if !ok {
		t.Fatal("layout-nav slot missing")
	}
	if slot.Dir != "ui/src/layout" {
		t.Fatalf("unexpected dir: %q", slot.Dir)
	}
	if mf.Defaults.Slot != "shared-component" {
		t.Fatalf("unexpected default slot: %q", mf.Defaults.Slot)
	}
}

func TestLoad_ScenarioMissing(t *testing.T) {
	root := setupRepo(t)
	l := NewFSLoader(root)
	_, err := l.Load("does-not-exist")
	var notFound ErrScenarioNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected ErrScenarioNotFound, got %T: %v", err, err)
	}
}

func TestLoad_TemplateNotDeclared(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), `{"service":{"id":"demo"}}`)
	l := NewFSLoader(root)
	_, err := l.Load("demo")
	var notDeclared ErrTemplateNotDeclared
	if !errors.As(err, &notDeclared) {
		t.Fatalf("expected ErrTemplateNotDeclared, got %T: %v", err, err)
	}
}

func TestLoad_TemplateManifestMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), validServiceJSON)
	// No templates/scenarios/react-vite/ui/manifest.json
	l := NewFSLoader(root)
	_, err := l.Load("demo")
	var missing ErrTemplateManifestMissing
	if !errors.As(err, &missing) {
		t.Fatalf("expected ErrTemplateManifestMissing, got %T: %v", err, err)
	}
}

func TestLoad_InvalidContract(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), validServiceJSON)
	writeFile(t, filepath.Join(root, "templates", "scenarios", "react-vite", "ui", "manifest.json"), `{
		"contract": {"kind": "wrong", "schema": "scenario-ui-manifest/v1"},
		"slots": {"x": {"dir": "ui"}}
	}`)
	l := NewFSLoader(root)
	_, err := l.Load("demo")
	var invalid ErrInvalidManifest
	if !errors.As(err, &invalid) {
		t.Fatalf("expected ErrInvalidManifest, got %T: %v", err, err)
	}
}

func TestLoad_NoSlots(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), validServiceJSON)
	writeFile(t, filepath.Join(root, "templates", "scenarios", "react-vite", "ui", "manifest.json"), `{
		"contract": {"kind": "scenario-ui", "schema": "scenario-ui-manifest/v1"},
		"slots": {}
	}`)
	l := NewFSLoader(root)
	_, err := l.Load("demo")
	var invalid ErrInvalidManifest
	if !errors.As(err, &invalid) {
		t.Fatalf("expected ErrInvalidManifest, got %T: %v", err, err)
	}
}

func TestLoad_SlotMissingDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), validServiceJSON)
	writeFile(t, filepath.Join(root, "templates", "scenarios", "react-vite", "ui", "manifest.json"), `{
		"contract": {"kind": "scenario-ui", "schema": "scenario-ui-manifest/v1"},
		"slots": {"broken": {"description": "no dir"}}
	}`)
	l := NewFSLoader(root)
	_, err := l.Load("demo")
	var invalid ErrInvalidManifest
	if !errors.As(err, &invalid) {
		t.Fatalf("expected ErrInvalidManifest, got %T: %v", err, err)
	}
}
