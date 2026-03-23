package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveScenarioRoot_DefaultsToSwarmManager(t *testing.T) {
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("")
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	base := filepath.Base(got)
	if base != "swarm-manager" {
		t.Errorf("expected basename swarm-manager, got %q", base)
	}
}

func TestResolveScenarioRoot_TrimsWhitespace(t *testing.T) {
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("  my-scenario  ")
	base := filepath.Base(got)
	if base != "my-scenario" {
		t.Errorf("expected basename my-scenario, got %q", base)
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

func TestResolveScenarioRoot_VrooliRootEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", tmp)

	got := ResolveScenarioRoot("test-scenario")
	want := filepath.Join(tmp, "scenarios", "test-scenario")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveScenarioRoot_WalkUpFindsScenariosDir(t *testing.T) {
	// Create a temp directory tree: tmp/scenarios/my-sc/
	tmp := t.TempDir()
	scenarioDir := filepath.Join(tmp, "scenarios", "my-sc")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Set cwd to a child directory under tmp.
	child := filepath.Join(tmp, "some", "deep", "dir")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(child); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("my-sc")
	if got != scenarioDir {
		t.Errorf("expected walk-up to find %q, got %q", scenarioDir, got)
	}
}

func TestResolveScenarioRoot_FallbackWhenNothingMatches(t *testing.T) {
	tmp := t.TempDir() // empty, no scenarios/ subdir
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")

	got := ResolveScenarioRoot("nonexistent")
	want := filepath.Join(tmp, "scenarios", "nonexistent")
	if got != want {
		t.Errorf("expected fallback %q, got %q", want, got)
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
