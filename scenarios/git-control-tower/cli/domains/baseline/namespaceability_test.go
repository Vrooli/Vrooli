package baseline

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// withFakeDetector swaps the detectSharedStore seam for a deterministic stub and
// returns a restore func plus a pointer to a "was it called" flag.
func withFakeDetector(found bool, detail string, err error) (called *bool, restore func()) {
	prev := detectSharedStore
	flag := false
	detectSharedStore = func(_ context.Context, _ string) (bool, string, error) {
		flag = true
		return found, detail, err
	}
	return &flag, func() { detectSharedStore = prev }
}

func TestResolveSharedStoreSignalDeclaredOverride(t *testing.T) {
	// A declared flag forces live and must NOT consult the auditor at all.
	called, restore := withFakeDetector(false, "clean", nil)
	defer restore()

	v := resolveSharedStoreSignal(context.Background(), "demo", true)
	if !v.writesSharedStore {
		t.Fatalf("declared flag must force the gate, got %+v", v)
	}
	if *called {
		t.Errorf("declared override must short-circuit auto-detection")
	}
	if !strings.Contains(v.note, "declared") {
		t.Errorf("note should explain the override, got %q", v.note)
	}
}

func TestResolveSharedStoreSignalAutoDetectsHit(t *testing.T) {
	_, restore := withFakeDetector(true, "api/internal/notes/qdrant.go", nil)
	defer restore()

	v := resolveSharedStoreSignal(context.Background(), "demo", false)
	if !v.writesSharedStore {
		t.Fatalf("a detected violation must trip the gate, got %+v", v)
	}
	if !strings.Contains(v.note, storageNamespaceFindingCode) || !strings.Contains(v.note, "qdrant.go") {
		t.Errorf("note should cite the finding code + evidence, got %q", v.note)
	}
}

func TestResolveSharedStoreSignalAutoDetectsClean(t *testing.T) {
	_, restore := withFakeDetector(false, "", nil)
	defer restore()

	v := resolveSharedStoreSignal(context.Background(), "demo", false)
	if v.writesSharedStore {
		t.Fatalf("a clean scan must leave the gate open, got %+v", v)
	}
	if !strings.Contains(v.note, "namespaceable") {
		t.Errorf("note should say namespaceable, got %q", v.note)
	}
}

func TestResolveSharedStoreSignalDetectionUnavailable(t *testing.T) {
	// An unreachable auditor degrades safely: the gate stays open (shadow-eligible)
	// rather than routing every scenario to live whenever the auditor is down.
	_, restore := withFakeDetector(false, "scenario-auditor unavailable", fmt.Errorf("connection refused"))
	defer restore()

	v := resolveSharedStoreSignal(context.Background(), "demo", false)
	if v.writesSharedStore {
		t.Fatalf("unavailable detection must not trip the gate, got %+v", v)
	}
	if !strings.Contains(v.note, "unavailable") || !strings.Contains(v.note, "--writes-shared-store") {
		t.Errorf("note should flag unavailability + the override, got %q", v.note)
	}
}

// ---- detectSharedStore (parses real storage-manager validation JSON) ------

func TestDetectSharedStoreParsesStorageHealthFinding(t *testing.T) {
	// storage-manager validate exits non-zero on ERROR findings but still emits the
	// assessment JSON to stdout; the namespace signal is a WARNING that rides
	// alongside. The shared fakeRunner can't return (stdout, err) together, so stub
	// runCommand directly to reproduce that real contract.
	prev := runCommand
	defer func() { runCommand = prev }()
	var gotName string
	var gotArgs []string
	runCommand = func(_ context.Context, name string, args ...string) ([]byte, error) {
		gotName, gotArgs = name, args
		return []byte(`{
		  "scenario": "demo",
		  "status": "VALIDATION_STATUS_FAILED",
		  "assessment": {
		    "findings": [
		      {"code": "ROUTED_SEAMS_UNWIRED", "severity": "ERROR", "location": "api/main.go"},
		      {"code": "STORAGE_NAMESPACE_HARDCODED", "severity": "WARNING", "location": "api/internal/backlog/qdrant.go:42"}
		    ]
		  }
		}`), fmt.Errorf("exit status 1")
	}

	found, detail, err := detectSharedStore(context.Background(), "demo")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if !found {
		t.Fatalf("expected a hit even though validate exited non-zero")
	}
	if !strings.Contains(detail, "qdrant.go") {
		t.Errorf("detail should carry the violating file, got %q", detail)
	}
	// Must consult storage-manager (not the retired auditor) for the one finding.
	if gotName != "storage-manager" {
		t.Errorf("detection must call storage-manager, got %q", gotName)
	}
	if got := strings.Join(gotArgs, " "); !strings.Contains(got, "validate scenario demo") || !strings.Contains(got, "--json") {
		t.Errorf("detection must run `validate scenario demo --json`, got args %v", gotArgs)
	}
}

func TestDetectSharedStoreCleanValidation(t *testing.T) {
	f := newFakeRunner(t)
	f.stdout["storage-manager validate scenario demo"] = []byte(`{"scenario":"demo","status":"VALIDATION_STATUS_PASSED","assessment":{"findings":[]}}`)
	defer f.install()()

	found, _, err := detectSharedStore(context.Background(), "demo")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if found {
		t.Fatalf("a clean validation must report no hit")
	}
}

func TestDetectSharedStoreProviderDownIsUnknown(t *testing.T) {
	f := newFakeRunner(t)
	// storage-manager unreachable: the seam returns no stdout AND an error → the
	// non-parseable response must surface as "unknown", never silently "clean".
	f.failOn["storage-manager"] = fmt.Errorf("connection refused")
	defer f.install()()

	found, _, err := detectSharedStore(context.Background(), "demo")
	if err == nil {
		t.Fatalf("a provider-down validation must surface as an error (unknown, not clean)")
	}
	if found {
		t.Fatalf("found must be false on error")
	}
}

func TestDetectSharedStoreUnparseableIsUnknown(t *testing.T) {
	f := newFakeRunner(t)
	// Garbage stdout with no assessment is "unknown", never silently "clean".
	f.stdout["storage-manager validate scenario demo"] = []byte(`not json`)
	defer f.install()()

	if _, _, err := detectSharedStore(context.Background(), "demo"); err == nil {
		t.Fatalf("an unparseable response must surface as an error")
	}
}

// ---- start integration: the gate now auto-derives ------------------------

func TestStartAutoDetectsSharedStoreRoutesLive(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down") // fallback core set
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	// Auditor reports the scenario hardcodes a namespace → must route to live even
	// though the agent asked for (auto) shadow and never passed --writes-shared-store.
	called, restore := withFakeDetector(true, "api/qdrant.go", nil)
	defer restore()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeAuto, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if !*called {
		t.Fatalf("auto mode on a non-trusted scenario must consult the detector")
	}
	if res.Decision.Mode != modeLive {
		t.Fatalf("a hardcoded-namespace writer must auto-route to live, got %q", res.Decision.Mode)
	}
	if f.sawCommand("scenario start") {
		t.Errorf("live mode must not stand up a shadow; calls=%v", f.calls)
	}
	var sawNote bool
	for _, r := range res.Decision.Reasons {
		if strings.Contains(r, "namespaceability signal") && strings.Contains(r, storageNamespaceFindingCode) {
			sawNote = true
		}
	}
	if !sawNote {
		t.Errorf("decision reasons should record the auto-detected signal, got %v", res.Decision.Reasons)
	}
}

func TestStartExplicitShadowOverriddenByAutoDetectedGate(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	_, restore := withFakeDetector(true, "api/redis.go", nil)
	defer restore()

	// Namespaceability is a HARD gate: even an explicit --mode shadow loses to it.
	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeShadow, slug: "wip"})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if res.Decision.Mode != modeLive {
		t.Fatalf("hard gate must override explicit --mode shadow, got %q", res.Decision.Mode)
	}
}

func TestStartLiveModeSkipsNamespaceDetection(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	called, restore := withFakeDetector(true, "x", nil)
	defer restore()

	// Explicit live is already the gate's destination — no point scanning.
	if _, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeLive, slug: "wip"}); err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if *called {
		t.Errorf("live mode must not run namespaceability detection")
	}
}

func TestStartTrustedBaseSkipsNamespaceDetection(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	called, restore := withFakeDetector(true, "x", nil)
	defer restore()

	// git-control-tower is trusted base → hard-routed to live; the gate is moot.
	if _, err := startEngagement(nil, startParams{scenario: "git-control-tower", mode: modeAuto, slug: "wip", signals: modeSignals{operatorConfirm: true}}); err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if *called {
		t.Errorf("trusted-base scenarios must not run namespaceability detection")
	}
}

func TestStartDeclaredSharedStoreStillForcesLive(t *testing.T) {
	f := newFakeRunner(t)
	f.failOn["scenario-dependency-analyzer"] = fmt.Errorf("down")
	defer f.install()()
	defer withFakeAnchors(t, "", "clean")()
	// Auditor says clean, but the operator forces the gate via the flag.
	called, restore := withFakeDetector(false, "", nil)
	defer restore()

	res, err := startEngagement(nil, startParams{scenario: "demo-scenario", mode: modeAuto, slug: "wip", signals: modeSignals{writesSharedStore: true}})
	if err != nil {
		t.Fatalf("startEngagement: %v", err)
	}
	if res.Decision.Mode != modeLive {
		t.Fatalf("declared --writes-shared-store must force live, got %q", res.Decision.Mode)
	}
	if *called {
		t.Errorf("declared override must short-circuit the auditor call")
	}
}
