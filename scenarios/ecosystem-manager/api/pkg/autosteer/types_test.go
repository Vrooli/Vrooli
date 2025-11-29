package autosteer

import (
	"os"
	"path/filepath"
	"testing"
)

func resetSteerModeRegistry(t *testing.T) func() {
	t.Helper()

	steerModeRegistry.mu.Lock()
	originalDir := steerModeRegistry.phasesDir
	originalCustom := steerModeRegistry.custom

	steerModeRegistry.phasesDir = ""
	steerModeRegistry.custom = make(map[SteerMode]struct{})
	steerModeRegistry.mu.Unlock()

	return func() {
		steerModeRegistry.mu.Lock()
		steerModeRegistry.phasesDir = originalDir
		steerModeRegistry.custom = originalCustom
		steerModeRegistry.mu.Unlock()
	}
}

func TestRegisterSteerModesFromDirRegistersCustomModes(t *testing.T) {
	restore := resetSteerModeRegistry(t)
	defer restore()

	root := t.TempDir()
	phasesDir := filepath.Join(root, "phases")
	if err := os.MkdirAll(phasesDir, 0o755); err != nil {
		t.Fatalf("failed to create phases dir: %v", err)
	}

	modeName := "screaming-architecture-audit"
	if err := os.WriteFile(filepath.Join(phasesDir, modeName+".md"), []byte("# prompt"), 0o644); err != nil {
		t.Fatalf("failed to write phase prompt: %v", err)
	}

	if err := RegisterSteerModesFromDir(phasesDir); err != nil {
		t.Fatalf("RegisterSteerModesFromDir returned error: %v", err)
	}

	if !SteerMode(modeName).IsValid() {
		t.Fatalf("expected mode %s to be valid after registration", modeName)
	}

	found := false
	for _, mode := range AllowedSteerModes() {
		if mode == SteerMode(modeName) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected AllowedSteerModes to include %s", modeName)
	}
}

func TestSteerModeIsValidLazyRegistersNewPrompt(t *testing.T) {
	restore := resetSteerModeRegistry(t)
	defer restore()

	root := t.TempDir()
	phasesDir := filepath.Join(root, "phases")
	if err := os.MkdirAll(phasesDir, 0o755); err != nil {
		t.Fatalf("failed to create phases dir: %v", err)
	}

	if err := RegisterSteerModesFromDir(phasesDir); err != nil {
		t.Fatalf("RegisterSteerModesFromDir returned error: %v", err)
	}

	mode := SteerMode("temporal-flow-audit")
	if mode.IsValid() {
		t.Fatalf("expected %s to be invalid before prompt exists", mode)
	}

	if err := os.WriteFile(filepath.Join(phasesDir, string(mode)+".md"), []byte("# new prompt"), 0o644); err != nil {
		t.Fatalf("failed to write new phase prompt: %v", err)
	}

	if !mode.IsValid() {
		t.Fatalf("expected %s to become valid after prompt file creation without restart", mode)
	}
}
