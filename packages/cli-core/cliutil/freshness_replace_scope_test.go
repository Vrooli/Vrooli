package cliutil

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestScenarioReplacementFreshnessKeepsImportedPackageAssetsAndExplicitInputs(t *testing.T) {
	repo := t.TempDir()
	scenarioRoot := filepath.Join(repo, "scenarios", "demo")
	modulePath := filepath.Join(scenarioRoot, "cli")

	mustWriteFile(t, filepath.Join(modulePath, "go.mod"), `module demo/cli

go 1.25.0

replace demo => ../api
`)
	mustWriteFile(t, filepath.Join(modulePath, "main.go"), `package main

import _ "demo/internal/used"

func main() {}
`)
	mustWriteFile(t, filepath.Join(scenarioRoot, ".vrooli", "service.json"), `{}`)
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "go.mod"), "module demo\n\ngo 1.25.0\n")
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "internal", "used", "used.go"), `package used

import _ "embed"

//go:embed rules.json
var rules string
`)
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "internal", "used", "rules.json"), `{"rule":"current"}`)
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "internal", "used", "other.json"), `{"asset":"sibling"}`)
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "internal", "unused", "heartbeat.go"), "package unused\n")
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "internal", "unused", "heartbeat_test.go"), "package unused\n")
	mustWriteFile(t, filepath.Join(scenarioRoot, "api", "contracts", "producers.json"), `{}`)

	spec := CanonicalScenarioGoModuleFreshnessSpec(scenarioRoot, modulePath, "demo", []string{
		"cli/**",
		".vrooli/service.json",
		"api/contracts/producers.json",
		"api/internal/used/rules.json",
	})
	for _, want := range []string{"api/go.mod", "api/internal/used", "api/contracts/producers.json", "api/internal/used/rules.json"} {
		if !slices.Contains(spec.Inputs, want) {
			t.Fatalf("canonical inputs omitted %q: %#v", want, spec.Inputs)
		}
	}
	for _, input := range spec.Inputs {
		if input == "api" || input == "api/**" || input == "api/internal/unused" {
			t.Fatalf("canonical inputs retained broad/unimported API path %q: %#v", input, spec.Inputs)
		}
	}

	fingerprint := func() string {
		t.Helper()
		value, err := ComputeFreshnessFingerprint(spec)
		if err != nil {
			t.Fatalf("ComputeFreshnessFingerprint: %v", err)
		}
		return value
	}
	for _, edit := range []struct {
		path  string
		stale bool
	}{
		{"api/internal/used/used.go", true},
		{"api/internal/used/rules.json", true},
		{"api/internal/used/other.json", true},
		{"api/go.mod", true},
		{"api/contracts/producers.json", true},
		{"api/internal/unused/heartbeat.go", false},
		{"api/internal/unused/heartbeat_test.go", false},
	} {
		before := fingerprint()
		path := filepath.Join(scenarioRoot, filepath.FromSlash(edit.path))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mustWriteFile(t, path, string(data)+"\n// changed\n")
		if changed := fingerprint() != before; changed != edit.stale {
			t.Errorf("editing %s: fingerprint changed = %v, want %v", edit.path, changed, edit.stale)
		}
	}
}
