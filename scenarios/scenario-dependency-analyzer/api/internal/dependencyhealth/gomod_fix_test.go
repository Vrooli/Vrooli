package dependencyhealth

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestGoSurfaceGoModsIncludesScenarioRootModule(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	writeFile(t, filepath.Join(scenarioDir, "go.mod"), "module example.com/demo\n")
	writeFile(t, filepath.Join(scenarioDir, "cli", "go.mod"), "module example.com/demo-cli\n")
	writeFile(t, filepath.Join(scenarioDir, "ui", "package.json"), "{}\n")

	got := goSurfaceGoMods(scenarioDir)
	want := []string{
		filepath.Join(scenarioDir, "cli", "go.mod"),
		filepath.Join(scenarioDir, "go.mod"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("goSurfaceGoMods = %#v, want %#v", got, want)
	}
}
