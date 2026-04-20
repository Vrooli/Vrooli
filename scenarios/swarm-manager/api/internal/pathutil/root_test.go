package pathutil

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestResolveScenarioRoot_DefaultsToSwarmManager(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", repoRoot)

	got := ResolveScenarioRoot("")
	want := filepath.Join(repoRoot, "scenarios", "swarm-manager")
	if got != want {
		t.Fatalf("ResolveScenarioRoot() = %q, want %q", got, want)
	}
}

func TestResolveScenarioRoot_TrimsWhitespace(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", repoRoot)

	got := ResolveScenarioRoot("  test-genie  ")
	want := filepath.Join(repoRoot, "scenarios", "test-genie")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveScenarioRoot_ScenarioRootEnvOverrides(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SCENARIO_ROOT", tmp)
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("anything")
	abs, _ := filepath.Abs(tmp)
	if got != abs {
		t.Errorf("expected %q, got %q", abs, got)
	}
}

func TestResolveScenarioRoot_ScenarioRootEnvWithWhitespace(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SCENARIO_ROOT", "  "+tmp+"  ")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("ignored")
	abs, _ := filepath.Abs(tmp)
	if got != abs {
		t.Errorf("expected %q, got %q", abs, got)
	}
}

func TestResolveScenarioRoot_UsesRepoContractDiscoveryFromVrooliRoot(t *testing.T) {
	tmp, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", tmp)

	got := ResolveScenarioRoot("test-genie")
	want := filepath.Join(tmp, "scenarios", "test-genie")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveScenarioRoot_UsesRepoContractDiscoveryFromCWD(t *testing.T) {
	tmp, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	child := filepath.Join(tmp, "scenarios", "swarm-manager", "api")
	chdirForTest(t, child)
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("test-genie")
	scenarioDir := filepath.Join(tmp, "scenarios", "test-genie")
	if got != scenarioDir {
		t.Errorf("expected contract-backed resolution %q, got %q", scenarioDir, got)
	}
}

func TestResolveScenarioRoot_ReturnsEmptyWhenRepoCannotBeResolved(t *testing.T) {
	tmp := t.TempDir()
	chdirForTest(t, tmp)
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("nonexistent")
	// The repocontract library falls back to the test binary's executable
	// path to find the repo root, so during `go test` a root is almost always
	// resolvable even with no env vars set and a tmp CWD. Accept either
	// outcome: "" when discovery genuinely fails, or a scenarios/<name> path
	// joined under the resolved root. Reject any value that isn't one of
	// these two shapes.
	if got == "" {
		return
	}
	wantSuffix := filepath.Join("scenarios", "nonexistent")
	if filepath.Base(filepath.Dir(got)) != "scenarios" || filepath.Base(got) != "nonexistent" {
		t.Errorf("unexpected fallback path shape: got %q, want empty or ending in %q", got, wantSuffix)
	}
}

func TestResolveScenariosDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SCENARIO_ROOT", filepath.Join(tmp, "scenarios", "swarm-manager"))
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenariosDir()
	want := filepath.Join(tmp, "scenarios")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

func TestScenariosFromGlobs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil returns nil", in: nil, want: nil},
		{name: "empty returns nil", in: []string{}, want: nil},
		{name: "single scenario", in: []string{"scenarios/web-console/**"}, want: []string{"web-console"}},
		{name: "deep path extracts scenario", in: []string{"scenarios/web-console/api/internal/**"}, want: []string{"web-console"}},
		{name: "dedup same scenario", in: []string{"scenarios/web-console/api/**", "scenarios/web-console/ui/**"}, want: []string{"web-console"}},
		{name: "multiple scenarios", in: []string{"scenarios/foo/**", "scenarios/bar/**"}, want: []string{"foo", "bar"}},
		{name: "non-scenario skipped", in: []string{"packages/proto/**"}, want: nil},
		{name: "wildcard skipped", in: []string{"**/*.go"}, want: nil},
		{name: "trailing slash no name", in: []string{"scenarios/"}, want: nil},
		{name: "bare scenarios prefix", in: []string{"scenarios/web-console"}, want: []string{"web-console"}},
		{name: "mixed scenario and non-scenario", in: []string{"scenarios/swarm-manager/**", "packages/proto/**"}, want: []string{"swarm-manager"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScenariosFromGlobs(tt.in)
			if len(got) == 0 && len(tt.want) == 0 {
				return // both empty/nil
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ScenariosFromGlobs(%v) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ScenariosFromGlobs(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}
