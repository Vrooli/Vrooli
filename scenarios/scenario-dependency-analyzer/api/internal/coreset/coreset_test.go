package coreset

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeService writes a minimal .vrooli/service.json for the named scenario
// under root, declaring the given scenario dependencies. deps maps a dependency
// name to its `required` flag.
func writeService(t *testing.T, root, name string, deps map[string]bool) {
	t.Helper()
	dir := filepath.Join(root, name, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	var b []byte
	b = append(b, []byte(`{"name":"`+name+`","dependencies":{"scenarios":{`)...)
	first := true
	for dep, required := range deps {
		if !first {
			b = append(b, ',')
		}
		first = false
		req := "false"
		if required {
			req = "true"
		}
		b = append(b, []byte(`"`+dep+`":{"enabled":true,"required":`+req+`}`)...)
	}
	b = append(b, []byte(`}}}`)...)

	if err := os.WriteFile(filepath.Join(dir, "service.json"), b, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}

func has(set []string, name string) bool {
	for _, v := range set {
		if v == name {
			return true
		}
	}
	return false
}

// TestEmptyDirKeepsSeed proves the seed is always included even when no
// scenario on disk can be loaded — over-inclusion is safe.
func TestEmptyDirKeepsSeed(t *testing.T) {
	root := t.TempDir()
	res := Compute(root)

	if res.Source != "computed" {
		t.Errorf("Source = %q, want computed", res.Source)
	}
	if len(res.CoreSet) != 9 {
		t.Errorf("expected 9 (seed-only) core members, got %d: %v", len(res.CoreSet), res.CoreSet)
	}
	for _, want := range res.Seed {
		if !has(res.CoreSet, want) {
			t.Errorf("seed member %q dropped from core set", want)
		}
	}
	if len(res.LoadErrors) != 9 {
		t.Errorf("expected a non-fatal load error for each missing seed, got %d", len(res.LoadErrors))
	}
	if len(res.AddedByClosure) != 0 {
		t.Errorf("no closure additions expected, got %v", res.AddedByClosure)
	}
}

// TestEmptyStringDirFallsBack proves an unusable directory yields the safe
// fallback seed rather than an error.
func TestEmptyStringDirFallsBack(t *testing.T) {
	res := Compute("   ")
	if res.Source != "fallback" {
		t.Errorf("Source = %q, want fallback", res.Source)
	}
	if len(res.CoreSet) != 9 {
		t.Errorf("fallback core set must be the 9-seed, got %d", len(res.CoreSet))
	}
}

// TestRequiredClosureAddsTransitively proves a Required edge chain from a seed
// pulls non-seed scenarios into the core set transitively.
func TestRequiredClosureAddsTransitively(t *testing.T) {
	root := t.TempDir()
	// test-genie (seed) --required--> foo --required--> bar
	writeService(t, root, "test-genie", map[string]bool{"foo": true})
	writeService(t, root, "foo", map[string]bool{"bar": true})
	writeService(t, root, "bar", nil)

	res := Compute(root)

	if !has(res.CoreSet, "foo") {
		t.Errorf("expected closure to add 'foo', core set = %v", res.CoreSet)
	}
	if !has(res.CoreSet, "bar") {
		t.Errorf("expected closure to add 'bar' transitively, core set = %v", res.CoreSet)
	}
	if !has(res.AddedByClosure, "foo") || !has(res.AddedByClosure, "bar") {
		t.Errorf("AddedByClosure should list foo+bar, got %v", res.AddedByClosure)
	}
	// The seed remains fully present.
	if len(res.CoreSet) != 11 { // 9 seed + foo + bar
		t.Errorf("expected 11 members (9 seed + foo + bar), got %d: %v", len(res.CoreSet), res.CoreSet)
	}
}

// TestNonRequiredEdgeNotAdded proves required==false edges never expand the set.
func TestNonRequiredEdgeNotAdded(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "test-genie", map[string]bool{"optional-dep": false})
	writeService(t, root, "optional-dep", nil)

	res := Compute(root)

	if has(res.CoreSet, "optional-dep") {
		t.Errorf("required=false edge must not add 'optional-dep', core set = %v", res.CoreSet)
	}
	if len(res.AddedByClosure) != 0 {
		t.Errorf("no closure additions expected, got %v", res.AddedByClosure)
	}
}

// TestMisMarkedSeedNeverDropped is the GCT case: git-control-tower declares all
// its scenario deps required=false, yet it (and the rest of the seed) must
// never be dropped from the core set.
func TestMisMarkedSeedNeverDropped(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "git-control-tower", map[string]bool{
		"test-genie":        false,
		"agent-manager":     false,
		"workspace-sandbox": false,
	})

	res := Compute(root)

	for _, want := range []string{"git-control-tower", "data-backup-manager", "scenario-dependency-analyzer"} {
		if !has(res.CoreSet, want) {
			t.Errorf("seed member %q wrongly dropped despite required=false edges", want)
		}
	}
}

// TestTrustedBaseSubset proves the trusted-base subset is reported and is a
// subset of the core set.
func TestTrustedBaseSubset(t *testing.T) {
	res := Compute(t.TempDir())
	if len(res.TrustedBase) == 0 {
		t.Fatal("trusted base subset must not be empty")
	}
	for _, name := range res.TrustedBase {
		if !has(res.CoreSet, name) {
			t.Errorf("trusted-base member %q not in core set", name)
		}
	}
	for _, want := range []string{"git-control-tower", "test-genie", "data-backup-manager"} {
		if !has(res.TrustedBase, want) {
			t.Errorf("expected %q in trusted base subset", want)
		}
	}
}

func TestValidateTrustedBaseClosureRejectsInconsistentGrant(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	writeService(t, scenariosRoot, "trusted", map[string]bool{"outside": true})

	stateDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	state := `{"version":"1.0.0","core":{"seed":["trusted"],"trusted_base":["trusted"]}}`
	if err := os.WriteFile(filepath.Join(stateDir, "operator-state.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateTrustedBaseClosure(root); err == nil {
		t.Fatal("inconsistent trusted-base grant was accepted")
	}
}

func TestRepositoryTrustedBaseClosure(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../../.."))
	if err := ValidateTrustedBaseClosure(repoRoot); err != nil {
		t.Fatalf("repository trusted-base closure is invalid: %v", err)
	}
}
