package projection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTargetsDiscoversRealHarnessesAndFixture(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", ".."))
	resources := filepath.Join(repoRoot, "resources")
	targets, err := LoadTargets(resources, "/tmp/operator")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 5 {
		t.Fatalf("real harness target count = %d, want 5: %#v", len(targets), targets)
	}
	for _, target := range targets {
		if target.Path == "" || target.Path[0:len("/tmp/operator")] != "/tmp/operator" {
			t.Fatalf("target did not resolve user home: %#v", target)
		}
	}

	fixture := t.TempDir()
	manifest := map[string]any{
		"name": "fixture-agent",
		"storage": map[string]any{"entries": map[string]any{
			"skills": map[string]any{
				"path":       "$USER_HOME/.fixture/skills",
				"projection": map[string]any{"environment": "FIXTURE_HOME", "project_scope": true},
			},
		}},
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(fixture, "fixture-agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "fixture-agent", "resource.json"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	fixtureTargets, err := LoadTargets(fixture, "/tmp/operator")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtureTargets) != 1 || fixtureTargets[0].Path != "/tmp/operator/.fixture/skills" {
		t.Fatalf("fixture target = %#v", fixtureTargets)
	}
}

func TestLoadTargetsNamesDuplicateRuntime(t *testing.T) {
	resources := t.TempDir()
	for _, dir := range []string{"one", "two"} {
		path := filepath.Join(resources, dir)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		data := []byte(`{"name":"same","storage":{"entries":{"skills":{"path":"$USER_HOME/.` + dir + `/skills"}}}}`)
		if err := os.WriteFile(filepath.Join(path, "resource.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadTargets(resources, "/tmp/operator"); err == nil {
		t.Fatal("expected duplicate runtime error")
	}
}

func TestProjectTargetsWritesEveryRealHarness(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", ".."))
	targets, err := LoadTargets(filepath.Join(repoRoot, "resources"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pack, err := LoadBasePack(filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "skills", "_base-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	results, err := ProjectTargets(filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "skills", "packs"), targets, pack)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 5 {
		t.Fatalf("projected target count = %d, want 5", len(results))
	}
	for _, target := range targets {
		if _, err := os.Stat(filepath.Join(target.Path, "implementation-plan-execution", "SKILL.md")); err != nil {
			t.Fatalf("target %s missing base skill: %v", target.Runtime, err)
		}
	}
}
