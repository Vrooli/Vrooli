package main

import (
	"context"
	"errors"
	"testing"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// stubRunner is a vroolicli.Runner that returns canned output/error for tests.
type stubRunner struct {
	out []byte
	err error
}

func (s stubRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return s.out, s.err
}

func (s stubRunner) RunCombined(context.Context, string, ...string) ([]byte, error) {
	return s.out, s.err
}

// swapCLIClient installs a client backed by the given runner for the duration
// of a test, restoring the original afterward.
func swapCLIClient(t *testing.T, runner vroolicli.Runner) {
	t.Helper()
	prev := cliClient
	cliClient = vroolicli.New(vroolicli.WithRunner(runner))
	t.Cleanup(func() { cliClient = prev })
}

func TestScenarioLocatorListMapsTypedContract(t *testing.T) {
	swapCLIClient(t, stubRunner{out: []byte(`{
		"success": true,
		"scenarios": [
			{"name":"alpha","display_name":"Alpha","description":"first","status":"running","tags":["a","b"],"runtime":"registry"},
			{"name":"beta","display_name":"Beta","status":"stopped","tags":[],"runtime":"registry"},
			{"name":"","status":"running"}
		]
	}`)})

	got, err := NewScenarioLocator(time.Minute).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// The blank-name entry is dropped.
	if len(got) != 2 {
		t.Fatalf("want 2 scenarios, got %d: %+v", len(got), got)
	}

	alpha := got[0]
	if alpha.Name != "alpha" || alpha.DisplayName != "Alpha" || alpha.Description != "first" ||
		alpha.Status != "running" || alpha.Runtime != "registry" {
		t.Errorf("alpha mapped wrong: %+v", alpha)
	}
	if len(alpha.Tags) != 2 || alpha.Tags[0] != "a" {
		t.Errorf("alpha tags wrong: %+v", alpha.Tags)
	}
	if alpha.HealthStatus != nil {
		t.Errorf("HealthStatus should be nil (not in contract), got %v", *alpha.HealthStatus)
	}
	// Nil tags are normalized to an empty slice.
	if got[1].Tags == nil {
		t.Errorf("beta tags should be non-nil empty slice")
	}
}

func TestScenarioLocatorListPropagatesError(t *testing.T) {
	swapCLIClient(t, stubRunner{err: errors.New("vrooli boom")})

	_, err := NewScenarioLocator(time.Minute).List(context.Background())
	if err == nil {
		t.Fatal("expected error to propagate, got nil (must never degrade to an empty list)")
	}
}
