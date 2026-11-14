package fleetpack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCensusFixtureFleet(t *testing.T) {
	root := fixtureFleet(t)

	report, err := Census(root, true)
	if err != nil {
		t.Fatalf("Census: %v", err)
	}

	checks := []struct {
		name string
		got  int
		want int
	}{
		{name: "scenario directories", got: report.ScenarioDirectoryCount, want: 4},
		{name: "manifests", got: report.ManifestCount, want: 3},
		{name: "total steps", got: report.Lifecycle.TotalSteps, want: 12},
		{name: "live steps", got: report.Lifecycle.LiveSteps, want: 12},
		{name: "distinct shapes", got: report.Lifecycle.DistinctLiveStepShapes, want: 7},
		{name: "port declarations", got: report.Ports.DeclarationCount, want: 3},
		{name: "range ports", got: report.Ports.RangeAllocatedCount, want: 2},
		{name: "pinned ports", got: report.Ports.PinnedCount, want: 1},
		{name: "peer edges", got: report.PeerDependencies.EdgeCount, want: 1},
		{name: "peer declarers", got: report.PeerDependencies.DeclaringScenarioCount, want: 1},
		{name: "peer targets", got: report.PeerDependencies.DistinctTargetCount, want: 1},
		{name: "runtime-only edges", got: report.PeerDependencies.RuntimeOnlyCount, want: 1},
		{name: "runtime-only rationale", got: report.PeerDependencies.RuntimeOnlyRationaleCount, want: 1},
		{name: "version ranges", got: report.PeerDependencies.VersionRangeCount, want: 1},
		{name: "component adopters", got: report.Components.AdoptingManifestCount, want: 1},
		{name: "components", got: report.Components.ComponentCount, want: 1},
		{name: "scenario shell files", got: report.ShellFiles.ScenarioCount, want: 1},
		{name: "resource shell files", got: report.ShellFiles.ResourceCount, want: 1},
		{name: "schema violations", got: report.SchemaValidation.ViolationCount, want: 1},
		{name: "failing manifests", got: report.SchemaValidation.FailingManifestCount, want: 1},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if check.got != check.want {
				t.Fatalf("got %d, want %d", check.got, check.want)
			}
		})
	}

	if !reflect.DeepEqual(report.NoManifest, []string{"orphan"}) {
		t.Fatalf("no-manifest scenarios = %v, want [orphan]", report.NoManifest)
	}
	if !reflect.DeepEqual(report.Cohorts.TemplateCurrent, []string{"alpha"}) {
		t.Fatalf("template-current = %v, want [alpha]", report.Cohorts.TemplateCurrent)
	}
	if !reflect.DeepEqual(report.Cohorts.TemplatePlusExtras, []string{"beta"}) {
		t.Fatalf("template-plus-extras = %v, want [beta]", report.Cohorts.TemplatePlusExtras)
	}
	if !reflect.DeepEqual(report.Cohorts.PreTemplate, []string{"gamma"}) {
		t.Fatalf("pre-template = %v, want [gamma]", report.Cohorts.PreTemplate)
	}
	if got := len(report.ShellFiles.LifecycleInvokedReferences); got != 1 {
		t.Fatalf("lifecycle shell references = %d, want 1", got)
	}
	if got := len(report.StepsInventory["beta"]["develop"]); got != 3 {
		t.Fatalf("beta develop inventory = %d, want 3", got)
	}
	if got := report.SchemaValidation.ByManifest["gamma"]; got != 1 {
		t.Fatalf("gamma schema violations = %d, want 1", got)
	}
}

func TestCensusIsDeterministicAndCanOmitInventory(t *testing.T) {
	root := fixtureFleet(t)
	first, err := Census(root, false)
	if err != nil {
		t.Fatalf("first Census: %v", err)
	}
	second, err := Census(root, false)
	if err != nil {
		t.Fatalf("second Census: %v", err)
	}
	if first.StepsInventory != nil || second.StepsInventory != nil {
		t.Fatal("inventory was populated when includeSteps=false")
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("census output is not deterministic:\n%s\n%s", firstJSON, secondJSON)
	}
}

func TestLoadTemplatesAcceptsComponentOnlyLifecycle(t *testing.T) {
	root := t.TempDir()
	template := `{
  "service": {"name": "{{SCENARIO_ID}}"},
  "components": {"api": {"role": "api"}},
  "lifecycle": {"version": "2.0.0", "health": {}}
}`
	writeFixture(t, root, "templates/scenarios/react-vite/.vrooli/service.json", template)
	writeFixture(t, root, "templates/scenarios/landing-page-react-vite/.vrooli/service.json", template)

	templates, err := loadTemplates(root)
	if err != nil {
		t.Fatalf("loadTemplates: %v", err)
	}
	if len(templates) != 2 || len(templates[0]) != 0 || len(templates[1]) != 0 {
		t.Fatalf("component-only templates = %#v, want two empty lifecycle step sets", templates)
	}
}

func fixtureFleet(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, ".vrooli/schemas/service.schema.json", `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["service"]
}`)

	template := `{
  "service": {"name": "{{SCENARIO_ID}}"},
  "lifecycle": {
    "setup": {"steps": [
      {"name":"build-api","exec":["go","build","-o","{{SCENARIO_ID}}-api","."],"cwd":"api"},
      {"name":"install-ui-deps","exec":["pnpm","install"],"cwd":"ui"},
      {"name":"build-ui","exec":["pnpm","run","build"],"cwd":"ui"}
    ]},
    "develop": {"steps": [
      {"name":"start-api","exec":["./{{SCENARIO_ID}}-api"],"cwd":"api"},
      {"name":"start-ui","exec":["node","server.js"],"cwd":"ui"}
    ]}
  }
}`
	writeFixture(t, root, "templates/scenarios/react-vite/.vrooli/service.json", template)
	writeFixture(t, root, "templates/scenarios/landing-page-react-vite/.vrooli/service.json", template)

	alpha := `{
  "service": {"name": "alpha"},
  "ports": {"api": {"env_var":"API_PORT", "range":"15000-19999"}},
  "lifecycle": {
    "setup": {"steps": [
      {"name":"build-api","exec":["go","build","-o","alpha-api","."],"cwd":"api"},
      {"name":"install-ui-deps","exec":["pnpm","install"],"cwd":"ui"},
      {"name":"build-ui","exec":["pnpm","run","build"],"cwd":"ui"}
    ]},
    "develop": {"steps": [
      {"name":"start-api","exec":["./alpha-api"],"cwd":"api"},
      {"name":"start-ui","exec":["node","server.js"],"cwd":"ui"}
    ]}
  }
}`
	writeFixture(t, root, "scenarios/alpha/.vrooli/service.json", alpha)

	beta := `{
  "service": {"name": "beta"},
  "ports": {
    "api": {"env_var":"API_PORT", "range":"15000-19999"},
    "ui": {"env_var":"WRONG_PORT", "port":21237}
  },
  "components": {"api": {}},
  "dependencies": {"scenarios": {"alpha": {
    "runtime_only": true,
    "runtime_only_rationale": "HTTP-only peer",
    "versionRange": ">=1.0.0"
  }}},
  "lifecycle": {
    "setup": {"steps": [
      {"name":"build-api","exec":["go","build","-o","beta-api","."],"cwd":"api"},
      {"name":"install-ui-deps","exec":["pnpm","install"],"cwd":"ui"},
      {"name":"build-ui","exec":["pnpm","run","build"],"cwd":"ui"}
    ]},
    "develop": {"steps": [
      {"name":"start-api","exec":["./beta-api"],"cwd":"api"},
      {"name":"start-ui","exec":["node","server.js"],"cwd":"ui"},
      {"name":"register-hook","exec":["./scripts/hook.sh","tee","output.txt"]}
    ]}
  }
}`
	writeFixture(t, root, "scenarios/beta/.vrooli/service.json", beta)
	writeFixture(t, root, "scenarios/beta/scripts/hook.sh", "#!/bin/sh\n")

	gamma := `{
  "ports": {},
  "lifecycle": {"develop": {"steps": [
    {"name":"launch-worker","exec":["worker","--serve","worker.log"]}
  ]}}
}`
	writeFixture(t, root, "scenarios/gamma/.vrooli/service.json", gamma)
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "orphan"), 0o755); err != nil {
		t.Fatalf("create orphan scenario: %v", err)
	}
	writeFixture(t, root, "resources/example/script.sh", "#!/bin/sh\n")
	return root
}

func writeFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s parent: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}
