//nolint:goconst // test data deliberately reuses stable fixture values.
package ports

import (
	"context"
	"path/filepath"
	"sync"
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

// TestWorkspaceSandboxVariantClaimsAreGlobalAndReplayable binds the stable
// workspace-sandbox variant to the same global port-claim owner used by every
// lifecycle caller. Distinct disposable instances can race without sharing a
// port, while replaying the first instance's allocation returns its original
// claim after a lost response.
func TestWorkspaceSandboxVariantClaimsAreGlobalAndReplayable(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	writePortRegistry(t, root, nil)
	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	store := newRuntimeClaimStoreForPortTests(t)
	ctx := context.Background()
	variantA := "wsb-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	variantB := "wsb-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	instanceA, err := store.CreateInstance(ctx, scenarioruntime.Instance{Scenario: "alpha", Variant: variantA, Status: scenarioruntime.StatusStarting})
	if err != nil {
		t.Fatalf("create sandbox A runtime identity: %v", err)
	}
	instanceB, err := store.CreateInstance(ctx, scenarioruntime.Instance{Scenario: "alpha", Variant: variantB, Status: scenarioruntime.StatusStarting})
	if err != nil {
		t.Fatalf("create sandbox B runtime identity: %v", err)
	}
	item := scenario.Scenario{
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "16000-16999"},
		}},
	}
	type allocation struct {
		instance scenarioruntime.Instance
		env      Environment
		err      error
	}
	results := make(chan allocation, 2)
	var wg sync.WaitGroup
	for _, instance := range []scenarioruntime.Instance{instanceA, instanceB} {
		wg.Add(1)
		go func(instance scenarioruntime.Instance) {
			defer wg.Done()
			candidate := item
			candidate.Variant = instance.Variant
			env, buildErr := manager.BuildEnvironmentWithRuntimeClaims(candidate, nil, RuntimeClaimOptions{
				Enabled: true, Context: ctx, Store: store, InstanceID: instance.InstanceID,
			})
			results <- allocation{instance: instance, env: env, err: buildErr}
		}(instance)
	}
	wg.Wait()
	close(results)
	var allocations []allocation
	for result := range results {
		if result.err != nil {
			t.Fatalf("allocate %s: %v", result.instance.Variant, result.err)
		}
		allocations = append(allocations, result)
	}
	if len(allocations) != 2 {
		t.Fatalf("allocations=%d, want two", len(allocations))
	}
	if allocations[0].env.AllocatedPorts["api"] == allocations[1].env.AllocatedPorts["api"] {
		t.Fatalf("workspace sandboxes shared API port %d", allocations[0].env.AllocatedPorts["api"])
	}
	if got, want := allocations[0].env.EnvVars[scenarioruntime.EnvVariant], allocations[0].instance.Variant; got != want {
		t.Fatalf("sandbox A process variant=%q, want persisted identity %q", got, want)
	}
	if got, want := allocations[1].env.EnvVars[scenarioruntime.EnvVariant], allocations[1].instance.Variant; got != want {
		t.Fatalf("sandbox B process variant=%q, want persisted identity %q", got, want)
	}
	if allocations[0].env.EnvVars[scenarioruntime.EnvStorageNamespace] == allocations[1].env.EnvVars[scenarioruntime.EnvStorageNamespace] {
		t.Fatalf("workspace sandboxes shared storage namespace %q", allocations[0].env.EnvVars[scenarioruntime.EnvStorageNamespace])
	}
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{Statuses: []string{scenarioruntime.ClaimStatusReserved, scenarioruntime.ClaimStatusBound}})
	if err != nil {
		t.Fatalf("list global runtime claims: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("active global claims=%d, want two: %#v", len(claims), claims)
	}
	first := allocations[0]
	replayItem := item
	replayItem.Variant = first.instance.Variant
	replayed, err := manager.BuildEnvironmentWithRuntimeClaims(replayItem, nil, RuntimeClaimOptions{
		Enabled: true, Context: ctx, Store: store, InstanceID: first.instance.InstanceID,
	})
	if err != nil {
		t.Fatalf("replay lost allocation response: %v", err)
	}
	if got, want := replayed.AllocatedPorts["api"], first.env.AllocatedPorts["api"]; got != want {
		t.Fatalf("replayed API port=%d, want original instance port %d", got, want)
	}
	claims, err = store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{Statuses: []string{scenarioruntime.ClaimStatusReserved, scenarioruntime.ClaimStatusBound}})
	if err != nil {
		t.Fatalf("relist global runtime claims: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("replay duplicated or lost claims: %#v", claims)
	}
}

func TestBuildEnvironmentProfileRegistersOnlyProfilePorts(t *testing.T) {
	root := t.TempDir()
	writePortRegistry(t, root, nil)
	manager, err := NewManager(root, root)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	base := scenario.Scenario{
		Slug: "audio-tools",
		Path: filepath.Join(root, "scenarios", "audio-tools"),
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{
				"api": {EnvVar: "API_PORT", Range: "15000-19999"},
				"ui":  {EnvVar: "UI_PORT", Range: "20000-24999"},
			},
			Components: map[string]scenario.Component{
				"product-api": {Role: "api", Run: scenario.ComponentRun{Port: "api"}},
				"product-ui":  {Role: "ui", Run: scenario.ComponentRun{Port: "ui"}},
				"lab-api":     {Role: "api", Run: scenario.ComponentRun{Port: "api"}},
			},
			Profiles: map[string]scenario.ComponentProfile{
				"default": {Components: []string{"product-api", "product-ui"}, Ports: []string{"api", "ui"}},
				"lab":     {Components: []string{"lab-api"}, Ports: []string{"api"}},
			},
		},
	}
	product, err := base.SelectProfile("live")
	if err != nil {
		t.Fatalf("SelectProfile(live): %v", err)
	}
	productEnv, err := manager.BuildEnvironment(product, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment(live): %v", err)
	}
	lab, err := base.SelectProfile("lab")
	if err != nil {
		t.Fatalf("SelectProfile(lab): %v", err)
	}
	labEnv, err := manager.BuildEnvironment(lab, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment(lab): %v", err)
	}
	if len(productEnv.AllocatedPorts) != 2 || productEnv.EnvVars["UI_PORT"] == "" {
		t.Fatalf("live profile ports = %v, env=%v", productEnv.AllocatedPorts, productEnv.EnvVars)
	}
	if len(labEnv.AllocatedPorts) != 1 || labEnv.AllocatedPorts["api"] == 0 || labEnv.EnvVars["API_PORT"] == "" {
		t.Fatalf("lab profile ports = %v, env=%v", labEnv.AllocatedPorts, labEnv.EnvVars)
	}
	if _, exists := labEnv.AllocatedPorts["ui"]; exists || labEnv.EnvVars["UI_PORT"] != "" {
		t.Fatalf("lab profile unexpectedly registers UI port: ports=%v env=%v", labEnv.AllocatedPorts, labEnv.EnvVars)
	}
	if labEnv.AllocatedPorts["api"] == productEnv.AllocatedPorts["api"] {
		t.Fatalf("lab instance reused product API port %d", labEnv.AllocatedPorts["api"])
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
