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

func TestSelectJourneyCapabilityUsesModeAndOverride(t *testing.T) {
	tests := []struct {
		name, mode, override, want, reason string
	}{
		{name: "override", mode: "bundled", override: "custom.capability", want: "custom.capability", reason: "explicit_override"},
		{name: "proxy", mode: "proxy", want: "tier2.tier1.thin-client.v1", reason: "proxy_mode"},
		{name: "hello fixture", mode: "bundled", want: "hello-desktop", reason: "registered_fixture"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectJourneyCapability(JourneySelectionInput{ScenarioName: tt.name, DeploymentMode: tt.mode, Override: tt.override})
			if tt.name == "hello fixture" {
				got = selectJourneyCapability(JourneySelectionInput{ScenarioName: "hello-desktop", DeploymentMode: tt.mode, Override: tt.override})
			}
			if got.Capability != tt.want || got.Reason != tt.reason {
				t.Fatalf("selection = %+v, want capability=%q reason=%q", got, tt.want, tt.reason)
			}
		})
	}
}
