package autosteer

import (
	"testing"
)

func resetSteerModeRegistry(t *testing.T) func() {
	t.Helper()

	steerModeRegistry.mu.Lock()
	originalCustom := steerModeRegistry.custom

	steerModeRegistry.custom = make(map[SteerMode]struct{})
	steerModeRegistry.mu.Unlock()

	return func() {
		steerModeRegistry.mu.Lock()
		steerModeRegistry.custom = originalCustom
		steerModeRegistry.mu.Unlock()
	}
}

func TestRegisterSteerModesRegistersCustomModes(t *testing.T) {
	restore := resetSteerModeRegistry(t)
	defer restore()

	modeName := "screaming-architecture-audit"
	RegisterSteerModes(SteerMode(modeName))

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

func TestBuiltInSteerModesAreValid(t *testing.T) {
	restore := resetSteerModeRegistry(t)
	defer restore()

	for _, mode := range builtInSteerModes {
		if !mode.IsValid() {
			t.Errorf("expected built-in mode %s to be valid", mode)
		}
	}
}

func TestUnregisteredModeIsInvalid(t *testing.T) {
	restore := resetSteerModeRegistry(t)
	defer restore()

	mode := SteerMode("nonexistent-custom-mode")
	if mode.IsValid() {
		t.Fatalf("expected unregistered mode %s to be invalid", mode)
	}
}
