package facts

import (
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestDiscoverNestedParseUnitsEmitsPolyglotDependencyEvidence(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"web/bun.lock":           `{"lockfileVersion":1}`,
		"tools/requirements.txt": "requests==2.32.0\n",
		"native/Cargo.toml":      "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	units := discoverNestedParseUnits(root)
	seen := map[string]bool{}
	for _, unit := range units {
		seen[unit.GetLanguage()] = true
	}
	for _, language := range []string{"node", "python", "rust"} {
		if !seen[language] {
			t.Errorf("language %q not discovered: %+v", language, units)
		}
	}
	for _, unit := range units {
		if unit.GetToolchain() == nil {
			t.Errorf("%s parse unit has no neutral toolchain observation", unit.GetId())
		}
	}
}

func TestParseUnitPublishesNeutralToolchainObservation(t *testing.T) {
	root := t.TempDir()
	writeFileFixture(t, filepath.Join(root, "package.json"), `{"packageManager":"pnpm@9.0.0","scripts":{"test":"vitest run"},"devDependencies":{"vitest":"1.0.0"}}`)
	writeFileFixture(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	units := discoverNestedParseUnits(root)
	var node *factsv1.ParseUnit
	for _, unit := range units {
		if unit.GetLanguage() == "node" {
			node = unit
			break
		}
	}
	if node == nil || node.GetToolchain() == nil {
		t.Fatalf("node parse unit has no toolchain observation: %+v", units)
	}
	observation := node.GetToolchain()
	if observation.GetEcosystem() != "node" || observation.GetPackageManager() != "pnpm@9.0.0" {
		t.Fatalf("toolchain observation = %+v", observation)
	}
	if len(observation.GetLockfilePaths()) != 1 || len(observation.GetRunnerIndicators()) != 2 {
		t.Fatalf("toolchain paths/indicators = %+v", observation)
	}
}
