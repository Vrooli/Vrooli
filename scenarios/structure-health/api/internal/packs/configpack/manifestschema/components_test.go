package manifestschema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioComponentRuleFleetMotivators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenarios", "browser-automation-studio", ".vrooli", "service.json")
	tests := []struct {
		name       string
		motivator  string
		manifest   string
		wantPhrase string
	}{
		{
			name:       "failing missing component declaration",
			motivator:  "the scenario fleet",
			manifest:   `{"ports":{"api":{}}}`,
			wantPhrase: "must declare at least one component",
		},
		{
			name:      "passing sidecar and reused binary",
			motivator: "browser-automation-studio and scenario-to-mcp",
			manifest: `{
  "ports":{"api":{},"registry":{},"playwright_driver":{}},
  "components":{
    "api":{"build":{"kind":"go_module"},"run":{"port":"api"}},
    "registry":{"build":{"reuse":"api"},"run":{"port":"registry","depends_on":[{"component":"api"}]}},
    "playwright-driver":{"build":{"kind":"node_bundle"},"run":{"port":"playwright_driver","supervised_by":"api"}}
  }
}`,
		},
		{
			name:       "failing sidecar port",
			motivator:  "browser-automation-studio",
			manifest:   `{"ports":{"api":{}},"components":{"playwright-driver":{"build":{"kind":"node_bundle"},"run":{"port":"playwright_driver"}}}}`,
			wantPhrase: `missing ports key "playwright_driver"`,
		},
		{
			name:       "failing reused component cycle",
			motivator:  "scenario-to-mcp",
			manifest:   `{"components":{"api":{"build":{"reuse":"registry"},"run":{}},"registry":{"build":{"reuse":"api"},"run":{}}}}`,
			wantPhrase: "component dependency cycle",
		},
	}
	for _, test := range tests {
		t.Run(test.name+" motivated by "+test.motivator, func(t *testing.T) {
			violations := CheckScenarioComponents([]byte(test.manifest), path)
			assertComponentRuleResult(t, violations, test.wantPhrase)
		})
	}
}

func TestScenarioPeerBindingRuleFleetMotivators(t *testing.T) {
	root := t.TempDir()
	writePeerManifest(t, root, "landing-page-business-suite", `{"ports":{"api":{}}}`)
	writePeerManifest(t, root, "scenario-authenticator", `{"ports":{"api":{}}}`)
	tests := []struct {
		name       string
		motivator  string
		scenario   string
		manifest   string
		wantPhrase string
	}{
		{
			name:       "failing retired binding projection",
			motivator:  "browser-automation-studio",
			scenario:   "browser-automation-studio",
			manifest:   `{"dependencies":{"scenarios":{"landing-page-business-suite":{"startup_policy":"try_start","degraded_behavior":"Local work remains available.","bindings":[{"env_var":"BAS_ENTITLEMENT_SERVICE_URL","form":"http_base_url","port":"api","when_unavailable":"omit"}]}}}}`,
			wantPhrase: "retired bindings",
		},
		{
			name:       "failing undeclared peer port",
			motivator:  "calendar",
			scenario:   "calendar",
			manifest:   `{"dependencies":{"scenarios":{"scenario-authenticator":{"startup_policy":"must_start","bindings":[{"env_var":"AUTH_SERVICE_URL","form":"http_base_url","port":"hardcoded_15785","when_unavailable":"fail"}]}}}}`,
			wantPhrase: "retired bindings",
		},
		{
			name:       "failing omitted peer without degradation",
			motivator:  "browser-automation-studio",
			scenario:   "browser-automation-studio",
			manifest:   `{"dependencies":{"scenarios":{"landing-page-business-suite":{"startup_policy":"try_start","bindings":[{"env_var":"BAS_ENTITLEMENT_SERVICE_URL","form":"http_base_url","port":"api","when_unavailable":"omit"}]}}}}`,
			wantPhrase: "retired bindings",
		},
	}
	for _, test := range tests {
		t.Run(test.name+" motivated by "+test.motivator, func(t *testing.T) {
			path := filepath.Join(root, "scenarios", test.scenario, ".vrooli", "service.json")
			violations := CheckScenarioPeerBindings([]byte(test.manifest), path)
			assertComponentRuleResult(t, violations, test.wantPhrase)
		})
	}
}

func TestScenarioBuildKindRuleFleetMotivators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenarios", "browser-automation-studio", ".vrooli", "service.json")
	tests := []struct {
		name       string
		motivator  string
		kind       string
		wantPhrase string
	}{
		{name: "passing TypeScript sidecar", motivator: "browser-automation-studio", kind: "node_bundle"},
		{name: "failing anonymous custom builder", motivator: "browser-automation-studio", kind: "custom", wantPhrase: `unknown build kind "custom"`},
	}
	for _, test := range tests {
		t.Run(test.name+" motivated by "+test.motivator, func(t *testing.T) {
			manifest := `{"components":{"playwright-driver":{"build":{"kind":"` + test.kind + `"},"run":{}}}}`
			violations := CheckScenarioBuildKinds([]byte(manifest), path)
			assertComponentRuleResult(t, violations, test.wantPhrase)
		})
	}
}

func TestComponentAndPeerBindingRulesPassLiveFleet(t *testing.T) {
	if os.Getenv("STRUCTURE_HEALTH_FLEET_CONTRACT") != "1" {
		t.Skip("set STRUCTURE_HEALTH_FLEET_CONTRACT=1 for the serialized live-fleet gate")
	}
	repoRoot := findRepositoryRoot(t)
	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatalf("read scenarios directory: %v", err)
	}
	scenarioDirs := 0
	manifestCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenarioDirs++
		path := filepath.Join(scenariosDir, entry.Name(), ".vrooli", "service.json")
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			t.Errorf("%s has no .vrooli/service.json; every scenario directory must declare one", entry.Name())
			continue
		}
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		manifestCount++
		if violations := CheckScenarioComponents(content, path); len(violations) != 0 {
			t.Errorf("%s component violations: %#v", entry.Name(), violations)
		}
		if violations := CheckScenarioPeerBindings(content, path); len(violations) != 0 {
			t.Errorf("%s peer binding violations: %#v", entry.Name(), violations)
		}
		if violations := CheckScenarioUIServesBuild(content, path); len(violations) != 0 {
			t.Errorf("%s UI serving violations: %#v", entry.Name(), violations)
		}
	}
	// Coverage, not population: the fleet may grow or shrink, but every scenario
	// directory must be checked and the walk must never silently check none.
	if scenarioDirs == 0 {
		t.Fatal("no scenario directories discovered; the fleet walk found nothing to check")
	}
	if manifestCount != scenarioDirs {
		t.Fatalf("checked %d of %d scenario directories; every scenario must carry a canonical manifest", manifestCount, scenarioDirs)
	}
}

func TestScenarioUIServesBuildRejectsDevelopmentServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenarios", "graph-studio", ".vrooli", "service.json")
	bad := []byte(`{"components":{"ui":{"role":"ui","build":{"kind":"pnpm_vite"},"run":{"argv":["pnpm","run","preview"]}}}}`)
	assertComponentRuleResult(t, CheckScenarioUIServesBuild(bad, path), "development server")
	good := []byte(`{"components":{"ui":{"role":"ui","build":{"kind":"pnpm_vite"},"run":{"argv":["node","server.js"]}}}}`)
	assertComponentRuleResult(t, CheckScenarioUIServesBuild(good, path), "")
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			if info, err := os.Stat(filepath.Join(dir, "scenarios")); err == nil && info.IsDir() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

func writePeerManifest(t *testing.T, root, scenarioName, content string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", scenarioName, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir peer manifest: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write peer manifest: %v", err)
	}
}

func assertComponentRuleResult(t *testing.T, violations []Violation, wantPhrase string) {
	t.Helper()
	if wantPhrase == "" {
		if len(violations) != 0 {
			t.Fatalf("unexpected violations: %#v", violations)
		}
		return
	}
	for _, violation := range violations {
		if strings.Contains(violation.Description, wantPhrase) {
			return
		}
	}
	t.Fatalf("violations %#v do not contain %q", violations, wantPhrase)
}
