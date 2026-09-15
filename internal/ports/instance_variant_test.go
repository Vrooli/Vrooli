//nolint:goconst // test data deliberately reuses stable fixture values.
package ports

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/portspec"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// TestBuildEnvironmentVariantIsolatesPortsAndNamespace proves that a non-live
// variant of the same scenario hashes to a different first-choice port (via the
// variant-aware CRC seed) and that the variant-aware storage namespace env vars
// (VROOLI_VARIANT + VROOLI_STORAGE_NAMESPACE) are injected, while the live
// instance is byte-identical to the pre-variant behavior. VROOLI_SCENARIO stays
// the bare slug for both.
func TestBuildEnvironmentVariantIsolatesPortsAndNamespace(t *testing.T) {
	root := t.TempDir()
	home := root
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	build := func(variant string) (Environment, scenario.Scenario) {
		item := scenario.Scenario{
			Slug:    "alpha",
			Variant: variant,
			Path:    filepath.Join(root, "scenarios", "alpha"),
			Manifest: scenario.ServiceManifest{
				Ports: map[string]scenario.Port{
					"api": {EnvVar: "API_PORT", Range: "15000-15099"},
				},
			},
		}
		env, err := manager.BuildEnvironment(item, nil)
		if err != nil {
			t.Fatalf("BuildEnvironment(%q): %v", variant, err)
		}
		return env, item
	}

	live, _ := build("")
	shadow, _ := build("shadow")

	if live.AllocatedPorts["api"] == shadow.AllocatedPorts["api"] {
		t.Fatalf("live and shadow share api port %d; variant CRC seed not applied", live.AllocatedPorts["api"])
	}

	// Live keeps its existing deterministic port (PortSeed == bare slug).
	bare, _ := build("live") // "live" normalizes to the same as ""
	if bare.AllocatedPorts["api"] != live.AllocatedPorts["api"] {
		t.Fatalf("explicit live port %d != default-variant port %d", bare.AllocatedPorts["api"], live.AllocatedPorts["api"])
	}

	// VROOLI_SCENARIO stays the bare slug for both variants.
	if live.EnvVars["VROOLI_SCENARIO"] != "alpha" || shadow.EnvVars["VROOLI_SCENARIO"] != "alpha" {
		t.Fatalf("VROOLI_SCENARIO = (%q, %q), want both \"alpha\"", live.EnvVars["VROOLI_SCENARIO"], shadow.EnvVars["VROOLI_SCENARIO"])
	}

	// Variant-aware namespace env: live ⇒ live/<scenario>, shadow ⇒ shadow/<scenario>_shadow.
	if live.EnvVars[scenarioruntime.EnvVariant] != "live" {
		t.Fatalf("live VROOLI_VARIANT = %q, want live", live.EnvVars[scenarioruntime.EnvVariant])
	}
	if live.EnvVars[scenarioruntime.EnvStorageNamespace] != "alpha" {
		t.Fatalf("live VROOLI_STORAGE_NAMESPACE = %q, want alpha", live.EnvVars[scenarioruntime.EnvStorageNamespace])
	}
	if shadow.EnvVars[scenarioruntime.EnvVariant] != "shadow" {
		t.Fatalf("shadow VROOLI_VARIANT = %q, want shadow", shadow.EnvVars[scenarioruntime.EnvVariant])
	}
	if shadow.EnvVars[scenarioruntime.EnvStorageNamespace] != "alpha_shadow" {
		t.Fatalf("shadow VROOLI_STORAGE_NAMESPACE = %q, want alpha_shadow", shadow.EnvVars[scenarioruntime.EnvStorageNamespace])
	}
}

// TestAllocateFallsBackToBandForNonLiveFixedPort proves the live-only fixed-port
// policy AND its non-live fallback: the live instance keeps the constant fixed
// port, while a non-live variant — which must never take or be preempted on that
// constant — is allocated a role-appropriate port in the same canonical band
// (here the API band) instead of being left without a port (skipping would leave
// e.g. a shadow's UI unstartable and the scenario permanently degraded).
func TestAllocateFallsBackToBandForNonLiveFixedPort(t *testing.T) {
	root := t.TempDir()
	home := root
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	fixed := 16000 // a fixed API port, inside the API band
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Port: &fixed},
		},
	}

	shadow := scenario.Scenario{Slug: "alpha", Variant: "shadow", Path: filepath.Join(root, "scenarios", "alpha"), Manifest: manifest}
	env, err := manager.BuildEnvironment(shadow, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment(shadow fixed): %v", err)
	}
	got, ok := env.AllocatedPorts["api"]
	if !ok || got == 0 {
		t.Fatalf("shadow got no api port; a non-live fixed port must fall back to a band, not be skipped")
	}
	if got == fixed {
		t.Fatalf("shadow took the live fixed port %d; it must avoid the constant", fixed)
	}
	if got < portspec.APIRangeStart || got > portspec.APIRangeEnd {
		t.Fatalf("shadow api port %d not in API band %d-%d", got, portspec.APIRangeStart, portspec.APIRangeEnd)
	}

	live := scenario.Scenario{Slug: "alpha", Path: filepath.Join(root, "scenarios", "alpha"), Manifest: manifest}
	liveEnv, err := manager.BuildEnvironment(live, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment(live fixed): %v", err)
	}
	if liveEnv.AllocatedPorts["api"] != fixed {
		t.Fatalf("live api port = %d, want fixed %d", liveEnv.AllocatedPorts["api"], fixed)
	}
}

// TestFallbackBandForFixedPort covers the band selector: env-var role is the
// primary signal, the fixed port's own canonical band is the fallback for
// role-ambiguous names, and headroom is the last resort.
func TestFallbackBandForFixedPort(t *testing.T) {
	p := func(envVar, name string, port int) scenario.PortSummary {
		return scenario.PortSummary{EnvVar: envVar, Name: name, FixedPort: &port}
	}
	cases := []struct {
		name             string
		summary          scenario.PortSummary
		wantLo, wantHigh int
	}{
		{"ui by name", p("UI_PORT", "ui", 21241), portspec.UIRangeStart, portspec.UIRangeEnd},
		{"api by name", p("API_PORT", "api", 9999), portspec.APIRangeStart, portspec.APIRangeEnd},
		{"ws by name", p("WS_PORT", "ws", 12345), portspec.WSRangeStart, portspec.WSRangeEnd},
		{"role-ambiguous name falls back to port band", p("PORT", "listener", 21500), portspec.UIRangeStart, portspec.UIRangeEnd},
		{"unknown falls back to headroom", p("PORT", "listener", 8080), portspec.ReservedHeadroomStart, portspec.ReservedHeadroomEnd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi := fallbackBandForFixedPort(tc.summary)
			if lo != tc.wantLo || hi != tc.wantHigh {
				t.Fatalf("band = %d-%d, want %d-%d", lo, hi, tc.wantLo, tc.wantHigh)
			}
		})
	}
}
