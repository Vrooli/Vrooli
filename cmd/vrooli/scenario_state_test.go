package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadScenarioDetailMissingScenario(t *testing.T) {
	_, _, _, err := loadScenarioDetailForTest(t.TempDir(), "missing")
	if err == nil || !strings.Contains(err.Error(), `scenario "missing" not found`) {
		t.Fatalf("loadScenarioDetail error = %v", err)
	}
}

func TestLoadScenarioStateFiltersUnknownProcessDirectories(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))
	writeScenarioProcessRecord(t, home, "ghost", "start-api", os.Getpid(), 28080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)

	scenarios, runtimes, err := loadScenarioStateForTest(root)
	if err != nil {
		t.Fatalf("loadScenarioState: %v", err)
	}
	if len(scenarios) != 1 || scenarios[0].Slug != "alpha" {
		t.Fatalf("scenarios = %#v", scenarios)
	}
	if _, ok := runtimes["alpha"]; !ok {
		t.Fatalf("expected alpha runtime, got %#v", runtimes)
	}
	if _, ok := runtimes["ghost"]; ok {
		t.Fatalf("unexpected stale runtime for undiscovered scenario: %#v", runtimes)
	}
}

func TestLoadScenarioDetailRejectsBrokenProcessMetadata(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestFile(t, home, ".vrooli/processes/scenarios/alpha/broken.json", "{broken")

	t.Setenv("HOME", home)

	if _, _, _, err := loadScenarioDetailForTest(root, "alpha"); err == nil {
		t.Fatalf("expected invalid process metadata to fail scenario detail loading")
	}
}
