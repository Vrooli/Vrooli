package readiness

import "testing"

// [REQ:PH-TIER-001] Each perf-build infra detector recognizes its piece.
func TestDetectInfraInstrumentedHasNoFindings(t *testing.T) {
	root := writeInstrumentedReactVite(t)
	if got := detectInfra(root); len(got) != 0 {
		t.Fatalf("instrumented tree should have no findings, got %d: %+v", len(got), got)
	}
	if !infraComplete(root) {
		t.Fatal("infraComplete should be true for the instrumented tree")
	}
}

func TestDetectInfraBareHasAllFindings(t *testing.T) {
	root := writeBareReactVite(t)
	got := detectInfra(root)
	if len(got) != 4 {
		t.Fatalf("bare tree should have 4 findings, got %d: %+v", len(got), got)
	}
	if infraComplete(root) {
		t.Fatal("infraComplete should be false for the bare tree")
	}
	for _, f := range got {
		if !f.Autofixable {
			t.Fatalf("finding %q should be autofixable", f.Code)
		}
		if f.Severity != "warning" {
			t.Fatalf("finding %q severity = %q, want warning", f.Code, f.Severity)
		}
	}
}

// A vite config that aliases react-dom/profiling unconditionally (no profile
// mode gate) is NOT the prescribed infra and is reported missing.
func TestViteProfileModeRequiresKeepNamesAndModeGate(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, root+"/ui")
	mustWrite(t, root+"/ui/vite.config.ts", `export default { resolve: { alias: { "react-dom/client": "react-dom/profiling" } } };`)
	if viteProfileModePresent(uiRoot(root)) {
		t.Fatal("alias without keepNames + mode gate should be reported missing")
	}
}
