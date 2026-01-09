package bundleruntime

import (
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/testutil"
)

// Note: Unit tests for secrets.Injector have been moved to secrets/inject_test.go.
// Unit tests for secrets.Manager are in secrets/store_test.go.
// This file contains integration tests for Supervisor secret management methods.

func timeNow() time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestUpdateSecrets(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(timeNow())

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{}, // No required secrets
	}
	sm := secrets.NewManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	sm.Set(map[string]string{"EXISTING": "value"})

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: testutil.NewMockPortAllocator(),
	}

	newSecrets := map[string]string{"NEW_KEY": "new_value"}
	err := s.UpdateSecrets(newSecrets)
	if err != nil {
		t.Fatalf("UpdateSecrets() error = %v", err)
	}

	// Check merged secrets
	currentSecrets := s.secretStore.Get()
	if currentSecrets["EXISTING"] != "value" {
		t.Error("UpdateSecrets() lost existing secret")
	}
	if currentSecrets["NEW_KEY"] != "new_value" {
		t.Error("UpdateSecrets() didn't add new secret")
	}
}

func TestUpdateSecrets_MissingRequired(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(timeNow())
	required := true

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "REQUIRED_KEY", Required: &required},
		},
	}
	sm := secrets.NewManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	sm.Set(map[string]string{})

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: testutil.NewMockPortAllocator(),
	}

	// Update without providing required secret
	err := s.UpdateSecrets(map[string]string{"OTHER": "value"})
	if err == nil {
		t.Error("UpdateSecrets() expected error for missing required secret")
	}
}
