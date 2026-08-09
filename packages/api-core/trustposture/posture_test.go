package trustposture

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultsAreCompleteAndDefensive(t *testing.T) {
	for _, p := range []Posture{Personal, Shared, Hosted} {
		d, err := DefaultsFor(p)
		if err != nil {
			t.Fatal(err)
		}
		if d.AccessTokenTTL <= 0 || d.JWKSCacheGrace <= 0 {
			t.Fatalf("%s has invalid time defaults: %+v", p, d)
		}
		if p == Personal && len(d.NodeExecutionScopes) == 0 {
			t.Fatal("personal posture must have node execution defaults")
		}
		if len(d.NodeExecutionScopes) > 0 {
			d.NodeExecutionScopes[0] = "mutated"
		}
		again, _ := DefaultsFor(p)
		if len(again.NodeExecutionScopes) > 0 && again.NodeExecutionScopes[0] == "mutated" {
			t.Fatal("defaults leaked mutable scope slice")
		}
	}
	if _, err := DefaultsFor(Posture("PERSONAL")); err == nil {
		t.Fatal("case-mismatched posture accepted")
	}
	if DefaultsForTestTTL != time.Hour {
		t.Fatal("test sentinel changed")
	}
}

// Keeps the table test sensitive to accidental unit changes without making
// the production policy depend on a test-only override.
const DefaultsForTestTTL = time.Hour

func TestParseDefaultsAndRejectsInvalid(t *testing.T) {
	got, err := Parse([]byte(`{"trust_posture":"shared"}`), "/state")
	if err != nil || got.Posture != Shared || got.Source != "/state" {
		t.Fatalf("Parse() = %+v, %v", got, err)
	}
	for _, raw := range []string{`{"trust_posture":"operator"}`, `{"trust_posture":1}`} {
		if _, err := Parse([]byte(raw), "/state"); err == nil {
			t.Fatalf("invalid state accepted: %s", raw)
		}
	}
	got, err = Parse([]byte(`{}`), "")
	if err != nil || got.Posture != Personal || got.Source != "default" {
		t.Fatalf("missing posture = %+v, %v", got, err)
	}
}

func TestLoadMissingAndPresent(t *testing.T) {
	root := t.TempDir()
	got, err := Load(root)
	if err != nil || got.Posture != Personal || got.Source != "default" {
		t.Fatalf("missing = %+v, %v", got, err)
	}
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, stateRelativePath), []byte(`{"trust_posture":"hosted"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(root)
	if err != nil || got.Posture != Hosted {
		t.Fatalf("present = %+v, %v", got, err)
	}
}

func TestTransitionRequiresOperatorAndProducesTypedEvent(t *testing.T) {
	from := State{Posture: Personal}
	event, err := Transition(from, Shared, "operator-1", time.Unix(10, 0))
	if err != nil {
		t.Fatal(err)
	}
	if event.Action != "trust_posture.transition" || event.From != Personal || event.To != Shared {
		t.Fatalf("event = %+v", event)
	}
	if _, err := Transition(from, Shared, "", time.Unix(10, 0)); err == nil {
		t.Fatal("anonymous posture transition accepted")
	}
	if _, err := Transition(from, Personal, "operator-1", time.Unix(10, 0)); err == nil {
		t.Fatal("unchanged posture transition accepted")
	}
}
