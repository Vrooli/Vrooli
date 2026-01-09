package main

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestEnsureScenarioAPIReadyNoopWhenExplicitAPIBaseOverride(t *testing.T) {
	app := newTestApp(t)
	startScenarioFunc = func(ctx context.Context, scenarioName string) error {
		t.Fatalf("startScenarioFunc should not be called")
		return nil
	}
	healthCheckFunc = func(ctx context.Context, base string) error {
		t.Fatalf("healthCheckFunc should not be called")
		return nil
	}
	t.Cleanup(resetAutoStartHooks)

	err := ensureScenarioAPIReady(app.core, cliapp.GlobalOptions{APIBaseOverride: "http://example.com"}, appName)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEnsureScenarioAPIReadyStartsWhenHealthFails(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", "http://127.0.0.1:1")

	started := false
	startScenarioFunc = func(ctx context.Context, scenarioName string) error {
		started = true
		return nil
	}
	canAutostartFunc = func(scenarioName string) bool {
		return true
	}
	attempt := 0
	healthCheckFunc = func(ctx context.Context, base string) error {
		attempt++
		if attempt == 1 {
			return errors.New("down")
		}
		return nil
	}
	t.Cleanup(resetAutoStartHooks)

	err := ensureScenarioAPIReady(app.core, cliapp.GlobalOptions{}, appName)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !started {
		t.Fatalf("expected scenario to be started")
	}
}

func resetAutoStartHooks() {
	startScenarioFunc = startScenarioViaVrooli
	healthCheckFunc = checkHealth
	canAutostartFunc = canAutostartScenario
}
