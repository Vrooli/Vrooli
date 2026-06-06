package ports

import (
	"path/filepath"
	"testing"

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

// TestAllocateSkipsFixedPortForNonLiveVariant proves the live-only fixed-port
// policy: a non-live variant never claims a fixed port (so it can never preempt
// or collide with the live instance on it), while the live instance does.
func TestAllocateSkipsFixedPortForNonLiveVariant(t *testing.T) {
	root := t.TempDir()
	home := root
	writePortRegistry(t, root, nil)

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	fixed := 28123
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
	if port, ok := env.AllocatedPorts["api"]; ok {
		t.Fatalf("shadow allocated fixed api port %d; fixed ports are live-only", port)
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
