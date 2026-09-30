package repocontract

import (
	"path/filepath"
	"testing"
)

func TestIsFullRepoScope(t *testing.T) {
	contract := validContract(t)

	for _, scope := range []string{"", ".", "/"} {
		if !contract.IsFullRepoScope(scope) {
			t.Fatalf("IsFullRepoScope(%q) = false, want true", scope)
		}
	}
	if contract.IsFullRepoScope("scenarios/demo") {
		t.Fatal(`IsFullRepoScope("scenarios/demo") = true, want false`)
	}
}

func TestScenarioScopeMatch(t *testing.T) {
	contract := validContract(t)
	scenarioDir := contract.Layout().ScenarioDir

	tests := []struct {
		scenario string
		scope    string
		want     bool
	}{
		{scenario: "demo", scope: "", want: true},
		{scenario: "demo", scope: ".", want: true},
		{scenario: "demo", scope: scenarioDir, want: true},
		{scenario: "demo", scope: filepath.ToSlash(filepath.Join(scenarioDir, "demo")), want: true},
		{scenario: "demo", scope: filepath.ToSlash(filepath.Join(scenarioDir, "demo", "api")), want: true},
		{scenario: "demo", scope: filepath.ToSlash(filepath.Join(scenarioDir, "other")), want: false},
		{scenario: "demo", scope: "packages/cli-core", want: false},
	}

	for _, tt := range tests {
		if got := contract.ScenarioScopeMatch(tt.scenario, tt.scope); got != tt.want {
			t.Fatalf("ScenarioScopeMatch(%q, %q) = %v, want %v", tt.scenario, tt.scope, got, tt.want)
		}
	}
}

func TestResolveSandboxScenarioPath(t *testing.T) {
	contract := validContract(t)
	merged := "/tmp/sandbox/merged"
	scenarioDir := contract.Layout().ScenarioDir
	scenarioRel := filepath.Join(scenarioDir, "demo")

	tests := []struct {
		name     string
		scenario string
		scope    string
		wantPath string
		wantOK   bool
	}{
		{name: "full repo", scenario: "demo", scope: "", wantPath: filepath.Join(merged, scenarioRel), wantOK: true},
		{name: "scenarios dir", scenario: "demo", scope: scenarioDir, wantPath: filepath.Join(merged, "demo"), wantOK: true},
		{name: "exact scenario", scenario: "demo", scope: filepath.ToSlash(scenarioRel), wantPath: merged, wantOK: true},
		{name: "deeper scope", scenario: "demo", scope: filepath.ToSlash(filepath.Join(scenarioRel, "api")), wantPath: filepath.Join(merged, scenarioRel), wantOK: true},
		{name: "out of scope", scenario: "demo", scope: filepath.ToSlash(filepath.Join(scenarioDir, "other")), wantPath: "", wantOK: false},
	}

	for _, tt := range tests {
		got, ok, err := contract.ResolveSandboxScenarioPath(merged, tt.scope, tt.scenario)
		if err != nil {
			t.Fatalf("%s: ResolveSandboxScenarioPath() error = %v", tt.name, err)
		}
		if ok != tt.wantOK {
			t.Fatalf("%s: ResolveSandboxScenarioPath() ok = %v, want %v", tt.name, ok, tt.wantOK)
		}
		if got != tt.wantPath {
			t.Fatalf("%s: ResolveSandboxScenarioPath() = %q, want %q", tt.name, got, tt.wantPath)
		}
	}
}
