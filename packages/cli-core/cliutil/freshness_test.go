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

func TestSourcePackageDirsKeepsNestedCoveragePackages(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".vrooli", "repo-contract.json"), "{}\n")
	mustWriteFile(t, filepath.Join(root, ".gitignore"), "/coverage/\npackages/*/coverage/\nscenarios/*/ui/coverage/\ntemplates/scenarios/*/ui/coverage/\n")
	mustWriteFile(t, filepath.Join(root, "api", "internal", "checks", "coverage", "check.go"), "package coverage\n")
	mustWriteFile(t, filepath.Join(root, "api", "internal", "ui", "coverage", "check.go"), "package coverage\n")
	mustWriteFile(t, filepath.Join(root, "scenarios", "demo", "ui", "coverage", "report.go"), "package coverage\n")

	got := sourcePackageDirs(root)
	seen := make(map[string]bool, len(got))
	for _, dir := range got {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		seen[filepath.ToSlash(rel)] = true
	}
	for _, want := range []string{"api/internal/checks/coverage", "api/internal/ui/coverage"} {
		if !seen[want] {
			t.Errorf("source package walk omitted %q: %v", want, got)
		}
	}
	if seen["scenarios/demo/ui/coverage"] {
		t.Fatalf("source package walk included repo-declared report root: %v", got)
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
		FreshnessInputs:   []string{"cli/**", ".vrooli/service.json", "../../packages/cli-core", "../../packages/proto/gen/.vrooli-proto-artifact.json"},
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
	if len(spec.Inputs) != 4 || spec.Inputs[0] != "cli/**" || spec.Inputs[1] != "docs/**" || spec.Inputs[2] != "../../packages/cli-core" || spec.Inputs[3] != "../../packages/proto/gen/.vrooli-proto-artifact.json" {
		t.Fatalf("custom inputs not used: %#v", spec.Inputs)
	}
}

func TestCanonicalResourceSpecRetainsSharedCLICoreWithCustomInputs(t *testing.T) {
	spec := CanonicalResourceGoModuleFreshnessSpec("/resources/codex", "/resources/codex/cli", "resource-codex", []string{"cli/**", "resource.json"})
	if len(spec.Inputs) != 4 || spec.Inputs[2] != "../../packages/cli-core" || spec.Inputs[3] != "../../packages/proto/gen/.vrooli-proto-artifact.json" {
		t.Fatalf("custom resource inputs dropped shared cli-core: %#v", spec.Inputs)
	}
}

func TestCanonicalSpecFallsBackToDefaultModuleDir(t *testing.T) {
	spec := CanonicalScenarioGoModuleFreshnessSpec("/scenarios/alpha", ".", "alpha", nil)
	if spec.Inputs[0] != "cli/**" {
		t.Fatalf("expected cli/** default, got %q", spec.Inputs[0])
	}
	if spec.Inputs[2] != "../../packages/cli-core" || spec.Inputs[3] != "../../packages/proto/gen/.vrooli-proto-artifact.json" {
		t.Fatalf("expected shared cli-core default, got %#v", spec.Inputs)
	}
}

// replaceFixture lays out a repo whose resource CLI builds from local replace
// targets: one imported directly (harness), one only through harness (deep),
// one through the repo-root replace (internal/shared), and the generated proto
// tree that the artifact stamp represents. Unused targets sit beside them.
func replaceFixture(t *testing.T) (repo, resourceRoot, modulePath string) {
	t.Helper()
	repo = t.TempDir()
	resourceRoot = filepath.Join(repo, "resources", "demo")
	modulePath = filepath.Join(resourceRoot, "cli")
	mustWriteFile(t, filepath.Join(modulePath, "go.mod"), `module example.com/demo

go 1.25.0

replace example.com/harness => ../../../packages/harness

replace (
	example.com/deep => ../../../packages/deep
	example.com/unused => ../../../packages/unused // never imported
	example.com/repo/packages/proto => ../../../packages/proto
	example.com/repo => ../../..
	example.com/remote => example.com/fork v1.0.0
)
`)
	mustWriteFile(t, filepath.Join(modulePath, "main.go"), `package main

import (
	"fmt"

	"example.com/harness"
	"example.com/repo/internal/shared"
	"example.com/repo/packages/proto/gen/go/x"
)

func main() { fmt.Println(harness.Name, shared.Name, x.Name) }
`)
	mustWriteFile(t, filepath.Join(resourceRoot, "resource.json"), "{}\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "cli-core", "core.go"), "package core\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "harness", "go.mod"), "module example.com/harness\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "harness", "harness.go"), "package harness\n\nimport _ \"example.com/deep/sub\"\n\nconst Name = \"harness\"\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "deep", "go.mod"), "module example.com/deep\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "deep", "sub", "sub.go"), "package sub\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "unused", "go.mod"), "module example.com/unused\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "unused", "unused.go"), "package unused\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "proto", "go.mod"), "module example.com/repo/packages/proto\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "proto", "gen", ".vrooli-proto-artifact.json"), "{}\n")
	mustWriteFile(t, filepath.Join(repo, "packages", "proto", "gen", "go", "x", "x.pb.go"), "package x\n\nconst Name = \"x\"\n")
	mustWriteFile(t, filepath.Join(repo, "internal", "shared", "shared.go"), "package shared\n\nconst Name = \"shared\"\n")
	mustWriteFile(t, filepath.Join(repo, "internal", "other", "other.go"), "package other\n")
	return repo, resourceRoot, modulePath
}

func TestResourceFreshnessFollowsLocalReplaceTargetsTheCLIImports(t *testing.T) {
	repo, resourceRoot, modulePath := replaceFixture(t)
	fingerprint := func() string {
		t.Helper()
		fp, err := ComputeFreshnessFingerprint(CanonicalResourceGoModuleFreshnessSpec(resourceRoot, modulePath, "resource-demo", []string{"cli/**", "resource.json"}))
		if err != nil {
			t.Fatalf("fingerprint: %v", err)
		}
		return fp
	}

	for _, edit := range []struct {
		path  string
		stale bool
	}{
		{"packages/harness/harness.go", true},
		{"packages/deep/sub/sub.go", true},
		{"packages/deep/go.mod", true},
		{"internal/shared/shared.go", true},
		{"packages/proto/gen/.vrooli-proto-artifact.json", true},
		{"packages/unused/unused.go", false},
		{"internal/other/other.go", false},
		{"packages/proto/gen/go/x/x.pb.go", false},
	} {
		before := fingerprint()
		path := filepath.Join(repo, filepath.FromSlash(edit.path))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mustWriteFile(t, path, string(data)+"// edited\n")
		if changed := fingerprint() != before; changed != edit.stale {
			t.Errorf("editing %s: fingerprint changed = %v, want %v", edit.path, changed, edit.stale)
		}
	}
}

func TestReplaceExpansionIsIdempotentAndKeepsDeclaredInputs(t *testing.T) {
	_, resourceRoot, modulePath := replaceFixture(t)
	declared := []string{"cli/**", "resource.json", "../../packages/harness"}
	spec := CanonicalResourceGoModuleFreshnessSpec(resourceRoot, modulePath, "resource-demo", declared)
	if !reflect.DeepEqual(spec.Inputs[:len(declared)], declared) {
		t.Fatalf("declared inputs not kept first: %#v", spec.Inputs)
	}
	for _, input := range spec.Inputs {
		if input == "../../packages/harness/go.mod" || input == "../.." || input == "../../internal" {
			t.Errorf("input %q is already covered or is the repo root: %#v", input, spec.Inputs)
		}
	}
	if again := WithGoReplaceFreshnessInputs(spec); !reflect.DeepEqual(again.Inputs, spec.Inputs) {
		t.Fatalf("expansion is not idempotent:\n first:  %#v\n second: %#v", spec.Inputs, again.Inputs)
	}
}

// A CLI whose inputs come from the caller (not the baked list) must still agree
// with the installer, or every invocation rebuilds.
func TestStaleCheckerDerivesInstallerReplaceInputs(t *testing.T) {
	_, resourceRoot, modulePath := replaceFixture(t)
	installer := CanonicalResourceGoModuleFreshnessSpec(resourceRoot, modulePath, "resource-demo", nil)
	runtime := StaleChecker{SourceContextPath: "..", FreshnessInputs: []string{"cli/**", "resource.json", "../../packages/cli-core", "../../packages/proto/gen/.vrooli-proto-artifact.json"}}
	if got := runtime.freshnessSpec(modulePath).Inputs; !reflect.DeepEqual(got, installer.Inputs) {
		t.Fatalf("runtime inputs differ from installer inputs:\n runtime:   %#v\n installer: %#v", got, installer.Inputs)
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
