package autosteer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// profilesDir resolves scenarios/ecosystem-manager/profiles from this test file.
func profilesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../api/pkg/autosteer/<file> → ../../../profiles
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "profiles"))
}

// loadProfileJSON reads and parses a shipped profile's profile.json.
func loadProfileJSON(t *testing.T, dir, id string) *AutoSteerProfile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, id, "profile.json"))
	if err != nil {
		t.Fatalf("read %s/profile.json: %v", id, err)
	}
	var p AutoSteerProfile
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("parse %s/profile.json: %v", id, err)
	}
	return &p
}
