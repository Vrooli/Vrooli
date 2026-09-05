package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkillProjectionTargetsAcceptsFixtureHarness(t *testing.T) {
	resources := t.TempDir()
	fixture := filepath.Join(resources, "fixture-agent")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatal(err)
	}
	data := `{"name":"fixture-agent","storage":{"entries":{"skills":{"path":"$USER_HOME/.fixture/skills","projection":{"environment":"FIXTURE_HOME","project_scope":true}}}}}`
	if err := os.WriteFile(filepath.Join(fixture, "resource.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	targets, err := LoadSkillProjectionTargets(resources)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Runtime != "fixture-agent" || targets[0].Environment != "FIXTURE_HOME" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
	if got := ResolveSkillProjectionPath(targets[0].PathTemplate, "/tmp/operator"); got != "/tmp/operator/.fixture/skills" {
		t.Fatalf("unexpected resolved path %q", got)
	}
}
