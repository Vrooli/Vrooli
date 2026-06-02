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

// TestBalancedProfileUnchanged is the golden guard for §8 OQ#7: the plain
// `balanced` profile must NOT carry a ladder block (so the A/B against
// `balanced-ladder` is honest).
func TestBalancedProfileUnchanged(t *testing.T) {
	p := loadProfileJSON(t, profilesDir(t), "balanced")
	if p.Ladder != nil {
		t.Fatalf("balanced profile must have NO ladder block (got %+v)", p.Ladder)
	}
	if p.ladderEnabled() {
		t.Fatal("balanced profile must not be ladder-enabled")
	}
}

// TestBalancedLadderProfileValid confirms the new profile loads, validates, and
// has the ladder enabled to R4.
func TestBalancedLadderProfileValid(t *testing.T) {
	p := loadProfileJSON(t, profilesDir(t), "balanced-ladder")
	// CreateProfile-style invariants: name comes from metadata, so set it for
	// validation (the FS repository injects it at load).
	if p.Name == "" {
		p.Name = "Balanced (Maturity Ladder)"
	}
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("balanced-ladder profile must validate: %v", err)
	}
	if !p.ladderEnabled() {
		t.Fatal("balanced-ladder must be ladder-enabled")
	}
	if p.Ladder.TopRung != "R4" {
		t.Errorf("balanced-ladder top_rung = %q, want R4", p.Ladder.TopRung)
	}
}
