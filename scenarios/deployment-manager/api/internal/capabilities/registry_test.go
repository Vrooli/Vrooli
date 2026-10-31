package capabilities

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestKnownCapabilitiesDoNotDuplicateManifestDependencies(t *testing.T) {
	if len(Known) != 0 {
		t.Fatalf("manifest dependencies must not be repeated in Known: %#v", Known)
	}
}

func TestScenarioCheckerDegradesWithoutAConfiguredOrReachableScenario(t *testing.T) {
	if status, reason := (ScenarioChecker{}).Check(context.Background()); status != StatusUnavailable || reason == "" {
		t.Fatalf("empty checker = %q/%q", status, reason)
	}
	if status, reason := (ScenarioChecker{Slug: "scenario-that-does-not-exist-for-deployment-manager-tests"}).Check(context.Background()); status != StatusUnavailable || reason == "" {
		t.Fatalf("unreachable checker = %q/%q", status, reason)
	}
}

func TestScenarioCheckerRecognizesHealthyAndStoppedOutput(t *testing.T) {
	binDir := t.TempDir()
	stub := filepath.Join(binDir, "vrooli")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s' \"$VROOLI_STATUS\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("VROOLI_STATUS", "healthy")
	if status, reason := (ScenarioChecker{Slug: "demo"}).Check(context.Background()); status != StatusAvailable || reason != "scenario is healthy" {
		t.Fatalf("healthy checker=%q/%q", status, reason)
	}
	t.Setenv("VROOLI_STATUS", "running")
	if status, reason := (ScenarioChecker{Slug: "demo"}).Check(context.Background()); status != StatusAvailable || reason != "scenario is healthy" {
		t.Fatalf("running checker=%q/%q", status, reason)
	}
	t.Setenv("VROOLI_STATUS", "installed")
	if status, reason := (ScenarioChecker{Slug: "demo"}).Check(context.Background()); status != StatusUnavailable || reason != "scenario is installed but stopped" {
		t.Fatalf("stopped checker=%q/%q", status, reason)
	}
}

type scenarioStatusRunnerFunc func(context.Context, string) ([]byte, error)

func (f scenarioStatusRunnerFunc) Status(ctx context.Context, slug string) ([]byte, error) {
	return f(ctx, slug)
}

func TestScenarioCheckerUsesInjectedStatusRunner(t *testing.T) {
	called := ""
	checker := ScenarioChecker{
		Slug: "demo",
		runner: scenarioStatusRunnerFunc(func(_ context.Context, slug string) ([]byte, error) {
			called = slug
			return []byte(`{"health_status":"healthy"}`), nil
		}),
	}

	status, reason := checker.Check(context.Background())
	if status != StatusAvailable || reason != "scenario is healthy" || called != "demo" {
		t.Fatalf("injected checker = %q/%q, called %q", status, reason, called)
	}
}
