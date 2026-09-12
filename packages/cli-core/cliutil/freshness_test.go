package cliutil

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveFreshnessInputFilesReusesCanonicalExclusions(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "main.go"), "package main\n")
	mustWriteFile(t, filepath.Join(root, "main_test.go"), "package main\n")
	mustWriteFile(t, filepath.Join(root, "app"), "\x7fELFbinary")
	mustWriteFile(t, filepath.Join(root, "dist", "bundle.js"), "generated")
	spec := FreshnessSpec{SourceRoot: root, SkipSuffixes: []string{"_test.go"}}
	paths, err := ResolveFreshnessInputFiles(spec)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{filepath.Join(root, "main.go")}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("inputs = %v, want %v", paths, want)
	}
	manifest, err := ComputeFreshnessManifest(spec, "test", nil, 0)
	if err != nil || len(manifest.Files) != len(paths) {
		t.Fatalf("enumerator differs from manifest: %v, %v", manifest.Files, err)
	}
}

// TestCanonicalScenarioSpecMatchesStaleCheckerDerivation verifies that
// CanonicalScenarioGoModuleFreshnessSpec and the spec a StaleChecker would
// derive for a scenario CLI produce the same fingerprint. This is the
// invariant that prevents the install-vs-runtime disagreement bug.
func TestCanonicalScenarioSpecMatchesStaleCheckerDerivation(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "alpha")
	modulePath := filepath.Join(scenarioRoot, "cli")
	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")

	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("mkdir module: %v", err)
	}
	if err := os.WriteFile(servicePath, []byte(`{"service":{"name":"alpha"},"cli":{"enabled":true}}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulePath, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	mustWriteFile(t, filepath.Join(root, "packages", "cli-core", "connect.go"), "package cliapp\n")

	installerSpec := CanonicalScenarioGoModuleFreshnessSpec(scenarioRoot, modulePath, "alpha", nil)
	installerFP, err := ComputeFreshnessFingerprint(installerSpec)
	if err != nil {
		t.Fatalf("installer fingerprint: %v", err)
	}

	// Mimic StaleChecker.freshnessSpec: SourceContextPath=".." → ContextRoot = module/..
	runtime := StaleChecker{
		BuildSourceRoot:   modulePath,
		SourceContextPath: "..",
		FreshnessInputs:   []string{"cli/**", ".vrooli/service.json", "../../packages/cli-core"},
	}
	runtimeSpec := runtime.freshnessSpec(modulePath)
	runtimeSpec.SkipFiles = []string{"alpha"}
	runtimeFP, err := ComputeFreshnessFingerprint(runtimeSpec)
	if err != nil {
		t.Fatalf("runtime fingerprint: %v", err)
	}

	if installerFP != runtimeFP {
		t.Fatalf("fingerprint mismatch\n installer: %s\n runtime:   %s", installerFP, runtimeFP)
	}
}

func TestCanonicalSpecHonorsCustomInputs(t *testing.T) {
	spec := CanonicalScenarioGoModuleFreshnessSpec("/scenarios/alpha", "/scenarios/alpha/cli", "alpha", []string{"cli/**", "docs/**"})
	if len(spec.Inputs) != 2 || spec.Inputs[0] != "cli/**" || spec.Inputs[1] != "docs/**" {
		t.Fatalf("custom inputs not used: %#v", spec.Inputs)
	}
}

func TestCanonicalSpecFallsBackToDefaultModuleDir(t *testing.T) {
	spec := CanonicalScenarioGoModuleFreshnessSpec("/scenarios/alpha", ".", "alpha", nil)
	if spec.Inputs[0] != "cli/**" {
		t.Fatalf("expected cli/** default, got %q", spec.Inputs[0])
	}
	if spec.Inputs[2] != "../../packages/cli-core" {
		t.Fatalf("expected shared cli-core default, got %#v", spec.Inputs)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
