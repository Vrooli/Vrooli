package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestNewAppTracksSourceFreshness(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app.StaleChecker == nil || app.StaleChecker.ManifestSourcePath != "resource.json" || app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("stale checker = %+v, want resource-rooted freshness", app.StaleChecker)
	}
	for _, input := range []string{"cli/**", "resource.json", "../../packages/cli-core"} {
		if !slices.Contains(app.StaleChecker.FreshnessInputs, input) {
			t.Fatalf("FreshnessInputs = %v, missing %s", app.StaleChecker.FreshnessInputs, input)
		}
	}
}

func TestHelpListsTheDecisionSurface(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := app.CLI.RunWithWriters([]string{"help"}, &stdout, &stderr); err != nil {
		t.Fatalf("help: %v (%s)", err, stderr.String())
	}
	for _, group := range []string{"gateway", "policy", "models", "capacity", "capacity-sync"} {
		if !strings.Contains(stdout.String(), group) {
			t.Fatalf("help output lacks %q:\n%s", group, stdout.String())
		}
	}
}
