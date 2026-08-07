package hostreq

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestLoadOperatorStateDistinguishesSafeguardChoices(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{
  "version": "1.0.0",
  "updated_at": "2026-08-05T17:24:03Z",
  "host_safeguards": {
    "enabled": {"opted_in": true},
    "declined": {"opted_in": false},
    "empty": {}
  }
}`)
	if err := os.WriteFile(filepath.Join(root, operatorStateFileName), data, 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := LoadOperatorState(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]hostreqspec.OperatorChoice{
		"enabled":  hostreqspec.OperatorChoiceOptedIn,
		"declined": hostreqspec.OperatorChoiceDeclined,
		"empty":    hostreqspec.OperatorChoiceNotRecorded,
		"missing":  hostreqspec.OperatorChoiceNotRecorded,
	}
	for name, want := range cases {
		if got := state.choice(hostreqspec.KindSafeguard, name); got != want {
			t.Errorf("choice(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestLoadOperatorStateRejectsMalformedState(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, operatorStateFileName), []byte(`{"version":"1.0.0","updated_at":"not-a-time"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOperatorState(root); err == nil {
		t.Fatal("malformed updated_at unexpectedly accepted")
	}
}

func TestLoadOperatorStateMissingFileIsEmpty(t *testing.T) {
	state, err := LoadOperatorState(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := state.choice(hostreqspec.KindSafeguard, "missing"); got != hostreqspec.OperatorChoiceNotRecorded {
		t.Fatalf("missing state choice = %q, want %q", got, hostreqspec.OperatorChoiceNotRecorded)
	}
}
