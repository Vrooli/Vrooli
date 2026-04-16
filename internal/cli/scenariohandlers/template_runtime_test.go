package scenariohandlers

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	scenariocli "github.com/vrooli/vrooli/internal/cli/scenariocli"
)

func TestBuildTemplateValuesAndCopyTemplateRenderGeneratedGoModPaths(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath() error = %v", err)
	}
	info, err := loadTemplate(repoRoot, "react-vite")
	if err != nil {
		t.Fatalf("loadTemplate() error = %v", err)
	}

	destination := filepath.Join(t.TempDir(), "scenarios", "alpha")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID":           "alpha",
		"SCENARIO_DISPLAY_NAME": "Alpha",
		"SCENARIO_DESCRIPTION":  "Alpha scenario",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues() error = %v", err)
	}
	wantPackagesRel, err := filepath.Rel(filepath.Join(destination, "api"), filepath.Join(repoRoot, "packages"))
	if err != nil {
		t.Fatalf("filepath.Rel(packages) error = %v", err)
	}
	if got := values["PACKAGES_REL_FROM_API"]; got != filepath.ToSlash(wantPackagesRel) {
		t.Fatalf("PACKAGES_REL_FROM_API = %q, want %q", got, filepath.ToSlash(wantPackagesRel))
	}
	wantRepoRootRel, err := filepath.Rel(filepath.Join(destination, "cli"), repoRoot)
	if err != nil {
		t.Fatalf("filepath.Rel(repo root) error = %v", err)
	}
	if got := values["REPO_ROOT_REL_FROM_CLI"]; got != filepath.ToSlash(wantRepoRootRel) {
		t.Fatalf("REPO_ROOT_REL_FROM_CLI = %q, want %q", got, filepath.ToSlash(wantRepoRootRel))
	}

	if err := copyTemplate(info.Path, destination, values); err != nil {
		t.Fatalf("copyTemplate() error = %v", err)
	}
	if err := verifyTemplate(destination); err != nil {
		t.Fatalf("verifyTemplate() error = %v", err)
	}

	apiGoMod, err := os.ReadFile(filepath.Join(destination, "api", "go.mod"))
	if err != nil {
		t.Fatalf("read api/go.mod: %v", err)
	}
	expectedReplace := "replace github.com/vrooli/api-core => " + filepath.ToSlash(filepath.Join(wantPackagesRel, "api-core"))
	if !strings.Contains(string(apiGoMod), expectedReplace) {
		t.Fatalf("api/go.mod = %s", string(apiGoMod))
	}

	issues := validateGeneratedScenario(destination, false, nil, info.Name)
	if len(issues) != 0 {
		t.Fatalf("validateGeneratedScenario() issues = %#v", issues)
	}
}

func TestValidateGeneratedScenarioFlagsBrokenLocalReplaceTarget(t *testing.T) {
	destination := t.TempDir()
	moduleDir := filepath.Join(destination, "api")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	goMod := `module example.com/demo

go 1.22

replace github.com/vrooli/api-core => ../../../packages/api-core
`
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	issues := validateGeneratedScenario(destination, false, nil, "demo")
	if len(issues) == 0 {
		t.Fatal("expected validation issues for broken replace target")
	}
	if !strings.Contains(issues[0].Message, "does not resolve") {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateTemplateSourceFlagsHardcodedLocalReplaceTargets(t *testing.T) {
	templateDir := t.TempDir()
	apiDir := filepath.Join(templateDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module example.com/demo\n\nreplace github.com/vrooli/api-core => ../../../../packages/api-core\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	issues := validateTemplateSource(scenariocli.TemplateInfo{Name: "demo", Path: templateDir, Manifest: scenariocli.TemplateManifest{}})
	if len(issues) == 0 {
		t.Fatal("expected validateTemplateSource() to flag hardcoded local replace target")
	}
	if !strings.Contains(issues[0].Message, "generator-computed placeholders") {
		t.Fatalf("issues = %#v", issues)
	}
}
