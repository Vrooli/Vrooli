package capstatus

import (
	"os"
	"path/filepath"
	"testing"

	"experience-manager/internal/reconcile"
)

// registryDir walks up to the repo root and returns the live capability registry
// so these tests exercise the real data rather than a fixture that can drift.
func registryDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for range 8 {
		candidate := filepath.Join(dir, "scenarios", "experience-manager", "capabilities")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("capability registry not found from working directory")
	return ""
}

func liveSupport(t *testing.T) Support {
	profiles, err := reconcile.CaptureProfilesFromAxes(filepath.Join(registryDir(t), "axes.json"), 12)
	if err != nil {
		t.Fatalf("load capture profiles: %v", err)
	}
	axes := map[string][]string{}
	for _, a := range reconcile.WiredAxesFromProfiles(profiles) {
		axes[a.Axis] = a.Values
	}
	return Support{
		Axes:       axes,
		Evidence:   reconcile.AvailableEvidence(),
		ClaimTypes: reconcile.ImplementedClaimTypes(),
	}
}

func TestLoadReadsLiveRegistry(t *testing.T) {
	reg, err := Load(registryDir(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(reg.Axes) == 0 {
		t.Fatal("no axes loaded")
	}
	if len(reg.Evidence) == 0 {
		t.Fatal("no evidence kinds loaded")
	}
	if len(reg.Capabilities) == 0 {
		t.Fatal("no capabilities loaded")
	}
	for _, c := range reg.Capabilities {
		if c.Group == "" {
			t.Errorf("capability %q has no group", c.ID)
		}
		if len(c.Facets) == 0 {
			t.Errorf("capability %q has no facets", c.ID)
		}
	}
}

// TestDeriveAgainstLiveReconciler is the headline assertion: status must reflect
// what the reconciler can do, and the two capabilities we know are provable today
// must come out provable while the known-blocked ones must not.
func TestDeriveAgainstLiveReconciler(t *testing.T) {
	reg, err := Load(registryDir(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	rep := Derive(reg, liveSupport(t))

	if rep.PromiseTotal == 0 {
		t.Fatal("no promise capabilities derived")
	}
	byID := map[string]Result{}
	for _, r := range rep.Results {
		byID[r.Capability] = r
	}

	// tap-target-size needs only viewport:mobile, ax-tree, layout-box, and an
	// evaluator that exists. It is the canonical provable capability.
	if got := byID["tap-target-size"].Status; got != StatusProvable {
		t.Errorf("tap-target-size: want provable, got %s (%v)", got, byID["tap-target-size"].Blockers)
	}
	// hover-contrast is now provable because the baseline matrix transmits the
	// interaction-state axis and BAS produces computed-style evidence.
	hover := byID["hover-contrast"]
	if hover.Status != StatusProvable {
		t.Errorf("hover-contrast: want provable, got %s (%v)", hover.Status, hover.Blockers)
	}
	// contrast-floor has no axis requirement and now has its computed-style
	// evidence channel.
	if got := byID["contrast-floor"].Status; got != StatusProvable {
		t.Errorf("contrast-floor: want provable, got %s (%v)", got, byID["contrast-floor"].Blockers)
	}
	// ramp-conformance names a claim type nobody has implemented.
	ramp := byID["ramp-conformance"]
	if ramp.Status != StatusEvidenceMissing && ramp.Status != StatusNoChecker {
		t.Errorf("ramp-conformance: want evidence-missing or no-checker, got %s", ramp.Status)
	}
	// Ports carry no proves block and must never be counted as unprovable work.
	if got := byID["router-adapter"].Status; got != StatusPortOnly {
		t.Errorf("router-adapter: want port-only, got %s", got)
	}
}

func TestDeriveIsPurelyComputed(t *testing.T) {
	reg, err := Load(registryDir(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// With everything available, every promise capability must become provable.
	// If any stays blocked, the derivation is reading something it should not.
	all := Support{Axes: map[string][]string{}, Evidence: nil, ClaimTypes: nil}
	for id, axis := range reg.Axes {
		values := make([]string, 0, len(axis.Values))
		for v := range axis.Values {
			values = append(values, v)
		}
		all.Axes[id] = values
	}
	for e := range reg.Evidence {
		all.Evidence = append(all.Evidence, e)
	}
	for _, c := range reg.Capabilities {
		if c.Proves != nil {
			all.ClaimTypes = append(all.ClaimTypes, c.Proves.ClaimTypes...)
		}
	}
	rep := Derive(reg, all)
	if rep.ProvableTotal != rep.PromiseTotal {
		for _, r := range rep.Results {
			if r.Status != StatusProvable && r.Status != StatusPortOnly {
				t.Errorf("with full support, %s is still %s: %v", r.Capability, r.Status, r.Blockers)
			}
		}
	}
}

func TestBlockerCountsOrderWorkByLeverage(t *testing.T) {
	reg, err := Load(registryDir(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	counts := BlockerCounts(Derive(reg, liveSupport(t)))
	if len(counts) == 0 {
		t.Fatal("expected blockers against the live reconciler")
	}
	for i := 1; i < len(counts); i++ {
		if counts[i-1].Capabilities < counts[i].Capabilities {
			t.Fatal("blocker counts must be ordered most-blocking first")
		}
	}
	t.Logf("top blocker: %s (%d capabilities)", counts[0].Blocker, counts[0].Capabilities)
}
