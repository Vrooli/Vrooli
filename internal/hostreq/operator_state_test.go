package hostreq

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

func operatorStatePathForTest(t *testing.T, root string) string {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		t.Fatalf("resolve operator state: %v", err)
	}
	path := filepath.Join(paths.StateDir, operatorstate.StateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create operator state directory: %v", err)
	}
	return path
}

func TestLoadOperatorStateDistinguishesSafeguardChoices(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{
  "version": "1.0.0",
  "updated_at": "2026-08-05T17:24:03Z",
  "host_safeguards": {
    "enabled": {"opted_in": true},
    "declined": {"opted_in": false},
    "empty": {}
  }
}`)
	if err := os.WriteFile(operatorStatePathForTest(t, root), data, 0o644); err != nil {
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
	if err := os.WriteFile(operatorStatePathForTest(t, root), []byte(`{"version":"1.0.0","updated_at":"not-a-time"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOperatorState(root); err == nil {
		t.Fatal("malformed updated_at unexpectedly accepted")
	}
}

func TestLoadOperatorStateMissingFileIsEmpty(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	state, err := LoadOperatorState(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := state.choice(hostreqspec.KindSafeguard, "missing"); got != hostreqspec.OperatorChoiceNotRecorded {
		t.Fatalf("missing state choice = %q, want %q", got, hostreqspec.OperatorChoiceNotRecorded)
	}
	if got := state.TrustPosture().Posture; got != "personal" {
		t.Fatalf("missing state posture = %q, want personal default", got)
	}
}

func TestLoadOperatorStateReadsTypedTrustPosture(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"version":"1.0.0","updated_at":"2026-08-05T17:24:03Z","trust_posture":"shared"}`)
	if err := os.WriteFile(operatorStatePathForTest(t, root), data, 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := LoadOperatorState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.TrustPosture().Posture != "shared" || state.TrustPosture().Source == "" {
		t.Fatalf("typed posture = %+v", state.TrustPosture())
	}
}
