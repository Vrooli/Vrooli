package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/testutil/hosthome"
)

// TestMain keeps every test in this package off the operator's real harness
// logins and fails the package if a test leaves a credential file behind.
func TestMain(m *testing.M) { os.Exit(hosthome.Main(m)) }

// TestHandlerRunsSeedCodexHomeFromDummyLoginOnly pins the contract for tests
// that start Codex runs: the login a run's CODEX_HOME is seeded from is the
// dummy one, never the operator's.
func TestHandlerRunsSeedCodexHomeFromDummyLoginOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	seed, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil || string(seed) != hosthome.DummyAuth {
		t.Fatalf("HOME/.codex/auth.json is not the dummy login (err=%v); a test would seed runs from a real login", err)
	}
}
