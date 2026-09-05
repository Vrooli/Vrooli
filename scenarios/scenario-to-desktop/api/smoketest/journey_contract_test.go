package smoketest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCapabilityForScenarioUsesManifestPresence(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "second-paid-app", ".vrooli", "monetization.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"version":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", root)
	if got := capabilityForScenario("second-paid-app"); got != "monetization.trust-boundary.v1" {
		t.Fatalf("capability = %q", got)
	}
	if got := capabilityForScenario("unmonetized-app"); got == "monetization.trust-boundary.v1" {
		t.Fatal("unmonetized app received monetization journey")
	}
}
