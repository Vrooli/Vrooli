package templateengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	scenariocli "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
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

	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		t.Fatalf("copyTemplate() error = %v", err)
	}
	if err := verifyTemplate(destination); err != nil {
		t.Fatalf("verifyTemplate() error = %v", err)
	}
	for _, excluded := range []string{
		filepath.Join("docs", "internal", "TEMPLATE-GENERATION-CONTRACT.md"),
		filepath.Join("docs", "internal", "TEMPLATE-MAINTENANCE.md"),
		"proto",
	} {
		if _, err := os.Stat(filepath.Join(destination, excluded)); err == nil {
			t.Fatalf("copyTemplate leaked excluded/relocated path %s", excluded)
		}
	}

	apiGoMod, err := os.ReadFile(filepath.Join(destination, "api", "go.mod"))
	if err != nil {
		t.Fatalf("read api/go.mod: %v", err)
	}
	expectedReplace := "replace github.com/vrooli/api-core => " + filepath.ToSlash(filepath.Join(wantPackagesRel, "api-core"))
	if !strings.Contains(string(apiGoMod), expectedReplace) {
		t.Fatalf("api/go.mod = %s", string(apiGoMod))
	}
	expectedCLIReplace := "replace github.com/vrooli/cli-core => " + filepath.ToSlash(filepath.Join(wantPackagesRel, "cli-core"))
	if !strings.Contains(string(apiGoMod), expectedCLIReplace) {
		t.Fatalf("api/go.mod = %s", string(apiGoMod))
	}

	issues := validateGeneratedScenario(destination, false, nil, info.Name, info.Manifest)
	if len(issues) != 0 {
		t.Fatalf("validateGeneratedScenario() issues = %#v", issues)
	}
}

func TestRunGenerateReactViteHooksWritePrimitiveEvidenceArtifact(t *testing.T) {
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
	// This test is about the scenario-local postHook pipeline, not out-of-tree
	// proto relocation/codegen side effects. Keep the real react-vite postHooks
	// intact and suppress relocations so generation stays inside t.TempDir().
	info.Manifest.Relocations = nil

	destination := filepath.Join(t.TempDir(), "evidence-app")
	var sawEvidenceHook bool
	capture := &capturedSubprocess{
		onRun: func(spec scenarioexec.SubprocessSpec) error {
			cmd := strings.Join(spec.Args, " ")
			if !strings.Contains(cmd, "UPDATE_CLI_EVIDENCE=1") || !strings.Contains(cmd, "TestPrimitiveEvidenceArtifactCurrent") {
				return nil
			}
			sawEvidenceHook = true
			if got, want := filepath.Clean(spec.Dir), filepath.Join(destination, "cli"); got != want {
				t.Fatalf("evidence hook cwd = %q, want %q", got, want)
			}
			manifestRaw, err := os.ReadFile(filepath.Join(destination, "cli", "manifest.json"))
			if err != nil {
				return fmt.Errorf("read generated cli manifest: %w", err)
			}
			sum := sha256.Sum256(manifestRaw)
			artifact := map[string]any{
				"schema":          "cli-primitive-evidence/v1",
				"source_manifest": "cli/manifest.json",
				"scenario":        "evidence-app",
				"generator":       "test",
				"manifest_hash":   hex.EncodeToString(sum[:]),
				"commands":        []any{},
			}
			raw, err := json.MarshalIndent(artifact, "", "  ")
			if err != nil {
				return err
			}
			outPath := filepath.Join(destination, ".vrooli", "generated", "cli-primitive-evidence.json")
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(outPath, append(raw, '\n'), 0o644)
		},
	}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	req := scenariocli.GenerateRequest{
		TemplateInfo: info,
		Options: scenariocli.GenerateOptions{
			Destination: destination,
			RunHooks:    true,
			Values: map[string]string{
				"SCENARIO_ID":           "evidence-app",
				"SCENARIO_DISPLAY_NAME": "Evidence App",
				"SCENARIO_DESCRIPTION":  "Generated scaffold evidence test",
			},
		},
	}
	_, err = runGenerate(deps, struct{}{}, req)
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}
	if !sawEvidenceHook {
		t.Fatal("react-vite postHooks did not run the primitive-evidence generator")
	}

	artifactPath := filepath.Join(destination, ".vrooli", "generated", "cli-primitive-evidence.json")
	raw, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("generated primitive evidence artifact missing at %s: %v", artifactPath, err)
	}
	var artifact struct {
		Schema       string `json:"schema"`
		ManifestHash string `json:"manifest_hash"`
	}
	if err := json.Unmarshal(raw, &artifact); err != nil {
		t.Fatalf("parse primitive evidence artifact: %v", err)
	}
	if artifact.Schema != "cli-primitive-evidence/v1" {
		t.Fatalf("artifact schema = %q", artifact.Schema)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(destination, "cli", "manifest.json"))
	if err != nil {
		t.Fatalf("read generated cli manifest: %v", err)
	}
	sum := sha256.Sum256(manifestRaw)
	if want := hex.EncodeToString(sum[:]); artifact.ManifestHash != want {
		t.Fatalf("artifact manifest_hash = %q, want %q", artifact.ManifestHash, want)
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

	issues := validateGeneratedScenario(destination, false, nil, "demo", scenariocli.TemplateManifest{})
	if len(issues) == 0 {
		t.Fatal("expected validation issues for broken replace target")
	}
	if !strings.Contains(issues[0].Message, "does not resolve") {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateGeneratedScenarioFlagsMissingStartDocument(t *testing.T) {
	destination := t.TempDir()
	issues := validateGeneratedScenario(destination, false, nil, "demo", scenariocli.TemplateManifest{
		StartDocument: "docs/START-HERE.md",
	})
	if len(issues) == 0 {
		t.Fatal("expected missing startDocument issue")
	}
	if issues[0].Path != "docs/START-HERE.md" || !strings.Contains(issues[0].Message, "startDocument is declared but missing") {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateGeneratedScenarioAcceptsStartDocument(t *testing.T) {
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "docs", "START-HERE.md"), []byte("# Start Here\n"), 0o644); err != nil {
		t.Fatalf("write start doc: %v", err)
	}
	issues := validateGeneratedScenario(destination, false, nil, "demo", scenariocli.TemplateManifest{
		StartDocument: "docs/START-HERE.md",
	})
	if len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestFilterTemplatesForValidation(t *testing.T) {
	templates := []scenariocli.TemplateInfo{
		{Name: "alpha"},
		{Name: "react-vite"},
	}
	all, err := filterTemplatesForValidation(templates, "")
	if err != nil {
		t.Fatalf("filterTemplatesForValidation(all) error = %v", err)
	}
	if len(all) != len(templates) {
		t.Fatalf("all templates = %#v", all)
	}
	filtered, err := filterTemplatesForValidation(templates, "react-vite")
	if err != nil {
		t.Fatalf("filterTemplatesForValidation(react-vite) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].Name != "react-vite" {
		t.Fatalf("filtered templates = %#v", filtered)
	}
	if _, err := filterTemplatesForValidation(templates, "missing"); err == nil {
		t.Fatal("expected missing template filter to fail")
	}
}

func TestRunTemplateValidateDefaultsToShallowAndSupportsTemplateFilter(t *testing.T) {
	repoRoot := t.TempDir()
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", "missing-manifest")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	var stdout, stderr strings.Builder
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, &capturedSubprocess{})

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		TemplateName: "missing-manifest",
	})
	if err != nil {
		t.Fatalf("runTemplateValidate() error = %v", err)
	}
	if report.Mode != scenariocli.TemplateValidationModeShallow || report.TemplateName != "missing-manifest" || report.Count != 1 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Issues) != 1 || !strings.Contains(report.Issues[0].Message, "template.json is missing") {
		t.Fatalf("issues = %#v", report.Issues)
	}

	if _, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{TemplateName: "missing"}); err == nil {
		t.Fatal("expected missing template filter to fail")
	}
}

func TestValidateTemplateDeepRejectsConcurrentArtifactOwner(t *testing.T) {
	deepValidationSemaphore <- struct{}{}
	defer func() { <-deepValidationSemaphore }()

	_, issues := validateTemplateDeep(HandlerDeps[struct{}]{}, struct{}{}, scenariocli.TemplateInfo{Name: "react-vite"}, scenariocli.TemplateValidateRequest{})
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "already in progress") {
		t.Fatalf("issues = %#v, want concurrent deep-validation rejection", issues)
	}
}

func TestRunTemplateValidateRoutesHookOutputToStderr(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(filepath.Join(templateDir, "proto"), 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  },
  "relocations": [
    {
      "from": "proto/",
      "to": "packages/proto/schemas/{{SCENARIO_ID}}/",
      "post": [{"cmd": "printf hook-output", "cwd": "."}]
    }
  ]
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	writeTestFile(t, filepath.Join(templateDir, "proto", "README.md"), "# {{SCENARIO_ID}}\n")
	var stdout, stderr strings.Builder
	capture := &capturedSubprocess{stdout: "hook-output"}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		TemplateName: templateName,
	})
	if err != nil {
		t.Fatalf("runTemplateValidate() error = %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("validation hook output polluted stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "[Relocation post]") || !strings.Contains(stderr.String(), "hook-output") {
		t.Fatalf("expected hook output on stderr, got %q", stderr.String())
	}
}

func TestRunTemplateValidateDeepInvokesTestGenieWithScenarioPath(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  }
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	var stdout, stderr strings.Builder
	capture := &capturedSubprocess{stdout: `{"success":true}`}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
	deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		Mode:         scenariocli.TemplateValidationModeDeep,
		TemplateName: templateName,
		TestPreset:   "quick",
	})
	if err != nil {
		t.Fatalf("runTemplateValidate(deep) error = %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if report.Mode != scenariocli.TemplateValidationModeDeep || report.Count != 1 || len(report.DeepRuns) != 1 {
		t.Fatalf("report = %#v", report)
	}
	deepRun := report.DeepRuns[0]
	if deepRun.ScenarioID != "template-validation-demo-template-deep" ||
		deepRun.TestPreset != "quick" ||
		deepRun.CleanupStatus != "removed" ||
		deepRun.RetainedTemp {
		t.Fatalf("deepRun = %#v", deepRun)
	}
	if _, err := os.Stat(deepRun.TempRoot); !os.IsNotExist(err) {
		t.Fatalf("temp root cleanup err = %v, want not exist", err)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("captured %d subprocess calls, want test-genie only", len(capture.calls))
	}
	call := capture.calls[0]
	if call.Name != "/tmp/test-genie" {
		t.Fatalf("test-genie call name = %q", call.Name)
	}
	args := strings.Join(call.Args, " ")
	for _, want := range []string{
		"execute template-validation-demo-template-deep",
		"--scenario-path " + deepRun.ScenarioPath,
		"--logical-repo-root " + repoRoot,
		"--logical-scenario-relpath scenarios/template-validation-demo-template-deep",
		"--preset quick",
		"--wait",
		"--json",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("args = %q, want %q", args, want)
		}
	}
}

func TestRunTemplateValidateDeepReportsAndFailsWarningsByPolicy(t *testing.T) {
	for _, tt := range []struct {
		name          string
		policy        scenariocli.TemplateValidationWarningPolicy
		wantIssue     bool
		wantWarnCount int
	}{
		{name: "report", policy: scenariocli.TemplateValidationWarningPolicyReport, wantWarnCount: 1},
		{name: "fail", policy: scenariocli.TemplateValidationWarningPolicyFail, wantIssue: true, wantWarnCount: 1},
		{name: "ignore", policy: scenariocli.TemplateValidationWarningPolicyIgnore},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			seedRepoContract(t, repoRoot)
			templateName := "demo-template"
			templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
			if err := os.MkdirAll(templateDir, 0o755); err != nil {
				t.Fatalf("mkdir template dir: %v", err)
			}
			writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  }
}`)
			writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
			var stdout, stderr strings.Builder
			capture := &capturedSubprocess{stdout: `{
  "success": true,
  "warningSummary": {
    "total": 1,
    "phases": [{
      "name": "performance",
      "count": 1,
      "warnings": [{
        "message": "seo: 82% below warning threshold 90%",
        "source": "observation",
        "logPath": "coverage/logs/run/performance.log",
        "artifactPath": "coverage/phase-results/performance.json"
      }]
    }]
  }
}`}
			deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
			deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

			report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
				Mode:          scenariocli.TemplateValidationModeDeep,
				TemplateName:  templateName,
				TestPreset:    "quick",
				WarningPolicy: tt.policy,
			})
			if err != nil {
				t.Fatalf("runTemplateValidate(deep) error = %v", err)
			}
			if gotIssue := len(report.Issues) > 0; gotIssue != tt.wantIssue {
				t.Fatalf("issues = %#v, want issue %t", report.Issues, tt.wantIssue)
			}
			if report.WarningSummary.Total != tt.wantWarnCount || report.DeepRuns[0].WarningSummary.Total != tt.wantWarnCount {
				t.Fatalf("warning summaries = report %#v run %#v", report.WarningSummary, report.DeepRuns[0].WarningSummary)
			}
		})
	}
}

func TestRunTemplateValidateDeepFailsWhenTestGenieJSONReportsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  }
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	var stdout, stderr strings.Builder
	capture := &capturedSubprocess{stdout: `{
  "success": false,
  "phaseSummary": {"total": 2, "passed": 1, "failed": 1},
  "phases": [
    {"name": "structure", "status": "passed"},
    {"name": "unit", "status": "failed", "error": "go test failed"}
  ]
}`}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
	deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		Mode:         scenariocli.TemplateValidationModeDeep,
		TemplateName: templateName,
		TestPreset:   "quick",
	})
	if err != nil {
		t.Fatalf("runTemplateValidate(deep) error = %v", err)
	}
	if len(report.Issues) != 1 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if !strings.Contains(report.Issues[0].Message, "unit: go test failed") {
		t.Fatalf("issue message = %q", report.Issues[0].Message)
	}
	if report.Issues[0].Path != testGenieDeepValidationPhaseResultsPath {
		t.Fatalf("issue path = %q, want stable phase-results path", report.Issues[0].Path)
	}
}

func TestRunTemplateValidateDeepParsesTestGenieJSONOnNonzeroExit(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  }
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	var stdout, stderr strings.Builder
	capture := &capturedSubprocess{
		stdout: `{
  "success": false,
  "phaseSummary": {"total": 3, "passed": 1, "failed": 2},
  "phases": [
    {"name": "structure", "status": "passed"},
    {"name": "standards", "status": "failed", "error": "26 violations"},
    {"name": "business", "status": "failed", "error": "no requirement modules found"}
  ]
}`,
		err: errors.New("suite execution completed with failures"),
	}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
	deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		Mode:         scenariocli.TemplateValidationModeDeep,
		TemplateName: templateName,
		TestPreset:   "quick",
	})
	if err != nil {
		t.Fatalf("runTemplateValidate(deep) error = %v", err)
	}
	if len(report.Issues) != 1 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	message := report.Issues[0].Message
	for _, want := range []string{"standards: 26 violations", "business: no requirement modules found", "1 passed, 2 failed, 3 total"} {
		if !strings.Contains(message, want) {
			t.Fatalf("issue message = %q, want %q", message, want)
		}
	}
}

func TestParseTestGenieJSONResultUsesTerminalResultAfterRunHandle(t *testing.T) {
	result := parseTestGenieJSONResult("react-vite", []byte(`
{"event":"run_started","run_id":"run-1","scenario":"template-validation-react-vite-deep"}
{"success":true,"phaseSummary":{"total":20,"passed":20,"failed":0}}
`))
	if result.Issue != nil || result.Success == nil || !*result.Success {
		t.Fatalf("terminal Test Genie result must win over an early run handle: %#v", result)
	}
}

func TestRunTemplateValidateDeepReportsTestGenieStartupErrorDetails(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  }
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	var stdout, stderr strings.Builder
	capture := &capturedSubprocess{
		stdout: `{
  "success": false,
  "error": "suite execution failed",
  "errors": ["start target scenario demo: exit status 2; lifecycle start output: NotesCard.tsx(134,11): error TS2322: Property 'tableTestId' does not exist"]
}`,
		err: errors.New("api error (500): suite execution failed"),
	}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
	deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		Mode:         scenariocli.TemplateValidationModeDeep,
		TemplateName: templateName,
		TestPreset:   "quick",
	})
	if err != nil {
		t.Fatalf("runTemplateValidate(deep) error = %v", err)
	}
	if len(report.Issues) != 1 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	for _, want := range []string{"start target scenario demo: exit status 2", "TS2322", "tableTestId"} {
		if !strings.Contains(report.Issues[0].Message, want) {
			t.Fatalf("issue message = %q, want %q", report.Issues[0].Message, want)
		}
	}
	if !strings.Contains(report.Issues[0].Message, "startup failed before phases") {
		t.Fatalf("issue message = %q, want explicit startup classification", report.Issues[0].Message)
	}
	if report.Issues[0].Path != testGenieDeepValidationStartupPath {
		t.Fatalf("issue path = %q, want stable startup path", report.Issues[0].Path)
	}
}

func TestParseTestGenieJSONResultUsesStableDebtPaths(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantPath string
	}{
		{
			name:     "protocol",
			output:   "not json",
			wantPath: testGenieDeepValidationProtocolPath,
		},
		{
			name: "startup",
			output: `{
  "success": false,
  "error": "scenario start failed",
  "phaseSummary": {"total": 0, "passed": 0, "failed": 0}
}`,
			wantPath: testGenieDeepValidationStartupPath,
		},
		{
			name: "phase results",
			output: `{
  "success": false,
  "phaseSummary": {"total": 3, "passed": 1, "failed": 2},
  "phases": [
    {"name": "dependencies", "status": "failed", "error": "module metadata drift"},
    {"name": "workflow", "status": "failed", "error": "target missing"}
  ]
}`,
			wantPath: testGenieDeepValidationPhaseResultsPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTestGenieJSONResult("react-vite", []byte(tt.output))
			if result.Issue == nil {
				t.Fatal("Issue = nil, want failure")
			}
			if result.Issue.Path != tt.wantPath {
				t.Fatalf("issue path = %q, want %q", result.Issue.Path, tt.wantPath)
			}
		})
	}
}

func TestRunTemplateValidateDeepKeepsRelocationsDuringTestGenie(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	writeTestFile(t, filepath.Join(repoRoot, "packages", "proto", "Makefile"), "generate:\n\t@true\n")
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(filepath.Join(templateDir, "proto"), 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  },
  "relocations": [
    {"from": "proto/", "to": "packages/proto/schemas/{{SCENARIO_ID}}/"}
  ]
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	writeTestFile(t, filepath.Join(templateDir, "proto", "README.md"), "# {{SCENARIO_ID}}\n")
	var stdout, stderr strings.Builder
	var relocatedPath string
	capture := &capturedSubprocess{
		stdout: `{"success":true}`,
		onRun: func(spec scenarioexec.SubprocessSpec) error {
			if spec.Name != "/tmp/test-genie" {
				return nil
			}
			for i, arg := range spec.Args {
				if arg == "--scenario-path" && i+1 < len(spec.Args) {
					scenarioID := filepath.Base(spec.Args[i+1])
					relocatedPath = filepath.Join(repoRoot, "packages", "proto", "schemas", scenarioID, "README.md")
					if _, err := os.Stat(relocatedPath); err != nil {
						return fmt.Errorf("relocated file should exist during test-genie: %w", err)
					}
				}
			}
			return nil
		},
	}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
	deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		Mode:         scenariocli.TemplateValidationModeDeep,
		TemplateName: templateName,
		TestPreset:   "quick",
	})
	if err != nil {
		t.Fatalf("runTemplateValidate(deep) error = %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if relocatedPath == "" {
		t.Fatal("test-genie call did not inspect relocated path")
	}
	if _, err := os.Stat(relocatedPath); !os.IsNotExist(err) {
		t.Fatalf("relocated path cleanup err = %v, want not exist", err)
	}
	if !capturedCommand(capture.calls, "make", "generate") {
		t.Fatalf("calls = %#v, want proto regeneration after relocation cleanup", capture.calls)
	}
}

func TestRunTemplateValidateDeepRetainTempKeepsRelocationsForRerun(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templateName := "demo-template"
	templateDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if err := os.MkdirAll(filepath.Join(templateDir, "proto"), 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	writeTestFile(t, filepath.Join(templateDir, "template.json"), `{
  "name": "demo-template",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name"},
    "SCENARIO_DESCRIPTION": {"flag": "description"}
  },
  "relocations": [
    {"from": "proto/", "to": "packages/proto/schemas/{{SCENARIO_ID}}/"}
  ]
}`)
	writeTestFile(t, filepath.Join(templateDir, "README.md"), "# {{SCENARIO_DISPLAY_NAME}}\n")
	writeTestFile(t, filepath.Join(templateDir, "proto", "README.md"), "# {{SCENARIO_ID}}\n")
	var stdout, stderr strings.Builder
	capture := &capturedSubprocess{stdout: `{"success":true}`}
	deps := newRelocationTestDeps(repoRoot, &stdout, &stderr, capture)
	deps.LocateTestGenieCLI = func(struct{}) (string, error) { return "/tmp/test-genie", nil }

	report, err := runTemplateValidate(deps, struct{}{}, scenariocli.TemplateValidateRequest{
		Mode:         scenariocli.TemplateValidationModeDeep,
		TemplateName: templateName,
		TestPreset:   "quick",
		RetainTemp:   true,
	})
	if err != nil {
		t.Fatalf("runTemplateValidate(deep retain) error = %v", err)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	deepRun := report.DeepRuns[0]
	if !deepRun.RetainedTemp || deepRun.CleanupStatus != "retained" || deepRun.RunID == "" || deepRun.CleanupCommand == "" {
		t.Fatalf("deepRun = %#v", deepRun)
	}
	markerPath := filepath.Join(deepRun.TempRoot, ".vrooli", "template-validation-run.json")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("retained marker missing: %v", err)
	}
	if len(deepRun.RelocationArtifacts) == 0 {
		t.Fatalf("deepRun relocation artifacts missing: %#v", deepRun)
	}
	relocatedPath := filepath.Join(repoRoot, "packages", "proto", "schemas", deepRun.ScenarioID, "README.md")
	if _, err := os.Stat(relocatedPath); err != nil {
		t.Fatalf("retained relocation missing: %v", err)
	}
	cleanupRelocationTargets([]scenariocli.ResolvedRelocation{{To: filepath.Dir(relocatedPath)}})
	_ = os.RemoveAll(deepRun.TempRoot)
}

func TestPrepareDeepValidationWorkspaceLinksCanonicalSchemas(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".vrooli", "schemas", "cli-manifest.schema.json"), `{"type":"object"}`)
	tempRoot := filepath.Join(t.TempDir(), "workspace")

	if err := prepareDeepValidationWorkspace(repoRoot, tempRoot); err != nil {
		t.Fatalf("prepareDeepValidationWorkspace() error = %v", err)
	}

	schemasPath := filepath.Join(tempRoot, ".vrooli", "schemas")
	info, err := os.Lstat(schemasPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", schemasPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("schemas path mode = %v, want symlink", info.Mode())
	}
	if _, err := os.Stat(filepath.Join(schemasPath, "cli-manifest.schema.json")); err != nil {
		t.Fatalf("shared schema unavailable from deep workspace: %v", err)
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

func TestResolveDesignRejectsNoneForRequiredTemplate(t *testing.T) {
	destination := t.TempDir()
	_, err := resolveDesign(t.TempDir(), scenariocli.TemplateInfo{
		Name: "demo",
		Manifest: scenariocli.TemplateManifest{
			Design: scenariocli.TemplateDesign{Required: true, Default: "vrooli-default", Adapter: "react-vite-tailwind"},
		},
	}, "none", destination, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "--design none is not allowed") {
		t.Fatalf("resolveDesign() error = %v", err)
	}
}

func TestValidateDesignKitRequiresSpecTokensAndStateContract(t *testing.T) {
	root := t.TempDir()
	kitDir := filepath.Join(root, "templates", "design", "demo-kit")
	adapterDir := filepath.Join(kitDir, "adapters", "react-vite-tailwind")
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatalf("mkdir adapter: %v", err)
	}
	writeTestFile(t, filepath.Join(kitDir, "metadata.json"), `{
  "id": "demo-kit",
  "name": "Demo Kit",
  "version": "0.1.0",
  "adapters": {
    "react-vite-tailwind": {
      "path": "adapters/react-vite-tailwind"
    }
  }
}`)
	writeTestFile(t, filepath.Join(kitDir, "DESIGN.md"), `---
name: Demo Kit
tokens:
  color:
    primary: "#111827"
---
# Demo Kit
`)
	writeTestFile(t, filepath.Join(adapterDir, "adapter.json"), `{"id":"react-vite-tailwind"}`)

	info, err := loadDesignKit(root, "demo-kit")
	if err != nil {
		t.Fatalf("loadDesignKit() error = %v", err)
	}
	issues := validateDesignKit(info)
	if len(issues) == 0 {
		t.Fatal("expected design validation issues")
	}
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, want := range []string{
		`missing official-style top-level "colors" token group`,
		`components must include "button-primary-loading" state guidance token`,
		"missing ## Feedback & State section",
		`missing required UX state term "validation-error"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("issues = %#v, want message containing %q", issues, want)
		}
	}
}

func TestResolveAndCopyDesignAssets(t *testing.T) {
	repoRoot := t.TempDir()
	kitDir := filepath.Join(repoRoot, "templates", "design", "demo-kit")
	adapterDir := filepath.Join(kitDir, "adapters", "react-vite-tailwind")
	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		t.Fatalf("mkdir adapter: %v", err)
	}
	writeTestFile(t, filepath.Join(kitDir, "metadata.json"), `{
  "id": "demo-kit",
  "name": "Demo Kit",
  "version": "0.1.0",
  "default": true,
  "adapters": {
    "react-vite-tailwind": {
      "path": "adapters/react-vite-tailwind",
      "supports": ["templates/scenarios/react-vite"]
    }
  }
}`)
	writeTestFile(t, filepath.Join(kitDir, "DESIGN.md"), `---
name: "{{SCENARIO_DISPLAY_NAME}}"
colors:
  primary: "#111827"
  surface: "#ffffff"
  on-surface: "#111827"
typography:
  body-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: "400"
    lineHeight: 1.5
rounded:
  md: 8px
spacing:
  touch: 44px
components:
  button-primary-loading:
    backgroundColor: "{colors.primary}"
    textColor: "#ffffff"
  button-disabled:
    backgroundColor: "#e5e7eb"
    textColor: "#6b7280"
  input-error:
    backgroundColor: "{colors.surface}"
    textColor: "#b91c1c"
  alert-error:
    backgroundColor: "#fee2e2"
    textColor: "#b91c1c"
  toast-success:
    backgroundColor: "#dcfce7"
    textColor: "#166534"
  empty-state:
    backgroundColor: "#f3f4f6"
    textColor: "#6b7280"
  skeleton:
    backgroundColor: "#e5e7eb"
    height: 1rem
  inline-progress:
    backgroundColor: "#dbeafe"
    textColor: "{colors.primary}"
  retry-action:
    backgroundColor: transparent
    textColor: "{colors.primary}"
---
# {{SCENARIO_DISPLAY_NAME}} Design

## Feedback & State

Generated UI must include loading, success, validation-error, request-error, and retry states.

## Request Lifecycle

Generated UI must describe pending, success, failure, and retry transitions.
`)
	writeTestFile(t, filepath.Join(adapterDir, "adapter.json"), `{
  "id": "react-vite-tailwind",
  "copy": [{ "from": "tokens.css", "to": "ui/src/design-tokens.css" }]
}`)
	writeTestFile(t, filepath.Join(adapterDir, "tokens.css"), ":root { --scenario-name: \"{{SCENARIO_ID}}\"; }\n")

	destination := filepath.Join(repoRoot, "scenarios", "alpha")
	values := map[string]string{"SCENARIO_ID": "alpha", "SCENARIO_DISPLAY_NAME": "Alpha"}
	design, err := resolveDesign(repoRoot, scenariocli.TemplateInfo{
		Name: "react-vite",
		Manifest: scenariocli.TemplateManifest{
			Design: scenariocli.TemplateDesign{Required: true, Default: "demo-kit", Adapter: "react-vite-tailwind"},
		},
	}, "", destination, values)
	if err != nil {
		t.Fatalf("resolveDesign() error = %v", err)
	}
	if len(design.Copies) != 2 {
		t.Fatalf("design copies = %#v", design.Copies)
	}
	if err := preflightDesignCopies(design, false); err != nil {
		t.Fatalf("preflightDesignCopies() error = %v", err)
	}
	if err := copyDesignAssets(design, values); err != nil {
		t.Fatalf("copyDesignAssets() error = %v", err)
	}
	designDoc, err := os.ReadFile(filepath.Join(destination, "DESIGN.md"))
	if err != nil {
		t.Fatalf("read generated DESIGN.md: %v", err)
	}
	if !strings.Contains(string(designDoc), "# Alpha Design") {
		t.Fatalf("DESIGN.md = %q", string(designDoc))
	}
	tokens, err := os.ReadFile(filepath.Join(destination, "ui", "src", "design-tokens.css"))
	if err != nil {
		t.Fatalf("read generated tokens: %v", err)
	}
	if !strings.Contains(string(tokens), `"alpha"`) {
		t.Fatalf("tokens = %q", string(tokens))
	}
}

func TestPreflightDesignTemplateCollisionsRejectsTemplateOwnedTargets(t *testing.T) {
	root := t.TempDir()
	templateDir := filepath.Join(root, "templates", "scenarios", "demo")
	destination := filepath.Join(root, "scenarios", "demo")
	writeTestFile(t, filepath.Join(templateDir, "ui", "tailwind.config.ts"), "export default {}\n")

	err := preflightDesignTemplateCollisions(templateDir, destination, scenariocli.ResolvedDesign{
		KitID: "demo-kit",
		Copies: []scenariocli.ResolvedDesignCopy{
			{From: filepath.Join(root, "tokens.css"), To: filepath.Join(destination, "ui", "tailwind.config.ts")},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "collides with template file") {
		t.Fatalf("preflightDesignTemplateCollisions() error = %v", err)
	}
}

// --------------------------------------------------------------------------
// Relocation tests
// --------------------------------------------------------------------------

// captureSubprocess records every SubprocessSpec passed to RunSubprocess so
// post-command tests can assert cwd, command, and call count without
// actually executing anything.
type capturedSubprocess struct {
	calls  []scenarioexec.SubprocessSpec
	stdout string
	err    error
	onRun  func(scenarioexec.SubprocessSpec) error
}

func (c *capturedSubprocess) Run(_ struct{}, spec scenarioexec.SubprocessSpec) error {
	c.calls = append(c.calls, spec)
	if c.onRun != nil {
		if err := c.onRun(spec); err != nil {
			return err
		}
	}
	if c.stdout != "" && spec.Stdout != nil {
		_, _ = io.WriteString(spec.Stdout, c.stdout)
	}
	return c.err
}

func capturedCommand(calls []scenarioexec.SubprocessSpec, name string, args ...string) bool {
	for _, call := range calls {
		if call.Name != name || len(call.Args) != len(args) {
			continue
		}
		matched := true
		for i := range args {
			if call.Args[i] != args[i] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// newRelocationTestDeps builds a HandlerDeps that captures subprocess
// invocations, so post-command tests can assert against cwd / command.
// Stdout/Stderr go to the provided buffers; CommandEnv is empty.
func newRelocationTestDeps(repoRoot string, stdout, stderr io.Writer, capture *capturedSubprocess) HandlerDeps[struct{}] {
	return HandlerDeps[struct{}]{
		Stdout:        func(struct{}) io.Writer { return stdout },
		Stderr:        func(struct{}) io.Writer { return stderr },
		Root:          func(struct{}) string { return repoRoot },
		RunSubprocess: capture.Run,
		CommandEnv:    func(struct{}) []string { return nil },
	}
}

// seedRepoContract copies the canonical .vrooli/repo-contract.json from
// the real repo into the test repoRoot so buildTemplateValues can resolve
// {{PACKAGES_REL_FROM_API}} et al. The contract validator is strict
// (resource.manifest, scenario.required_files, etc. all required), so
// reusing the real file is simpler and safer than maintaining a
// hand-trimmed copy.
func seedRepoContract(t *testing.T, repoRoot string) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed in seedRepoContract")
	}
	realRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath() error = %v", err)
	}
	contract, err := os.ReadFile(filepath.Join(realRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read canonical repo-contract.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
	// The contract loader checks that root.markers.required_dirs exist, so
	// pre-create them as empty dirs.
	for _, dir := range []string{"packages", "templates", "scenarios", "resources", "cmd", "internal"} {
		_ = os.MkdirAll(filepath.Join(repoRoot, dir), 0o755)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// writeRelocationTemplate creates a tiny template tree at templatesDir/<name>/
// with a manifest that declares one proto-shaped relocation. Returns the
// repo root for use as deps.Root. The template's `proto/` source contains
// a `{{SCENARIO_ID}}.proto` file whose body references {{SCENARIO_ID_SNAKE}}
// so substitution is exercised in both path and content.
func writeRelocationTemplate(t *testing.T, templateName string, extraManifest map[string]any) (repoRoot string, info scenariocli.TemplateInfo) {
	t.Helper()
	repoRoot = t.TempDir()
	seedRepoContract(t, repoRoot)
	templatesDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	protoSrc := filepath.Join(templatesDir, "proto", "{{SCENARIO_ID}}", "v1")
	if err := os.MkdirAll(protoSrc, 0o755); err != nil {
		t.Fatalf("mkdir proto src: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(protoSrc, "health.proto"),
		[]byte("syntax = \"proto3\";\npackage vrooli.{{SCENARIO_ID_SNAKE}}.v1.health;\n"),
		0o644,
	); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	// Also include a non-proto file inside the scenario tree (so we can
	// confirm copyTemplate's skip-list excludes proto/ from the in-tree
	// copy without affecting unrelated files).
	apiDir := filepath.Join(templatesDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	manifest := map[string]any{
		"name":         templateName,
		"requiredVars": map[string]any{"SCENARIO_ID": map[string]any{"flag": "id"}},
		"relocations": []map[string]any{
			{
				"description": "Relocate proto schemas",
				"from":        "proto/",
				"to":          "packages/proto/schemas/{{SCENARIO_ID}}/",
				"post": []map[string]any{
					{"description": "regen", "cmd": "make generate", "cwd": "packages/proto"},
				},
			},
		},
	}
	for k, v := range extraManifest {
		manifest[k] = v
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "template.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}
	info, err = loadTemplate(repoRoot, templateName)
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}
	return repoRoot, info
}

func TestRelocations_CopiesAndSubstitutes(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-copy", nil)
	destination := filepath.Join(repoRoot, "scenarios", "my-app")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "my-app",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	if got := values["SCENARIO_ID_SNAKE"]; got != "my_app" {
		t.Fatalf("SCENARIO_ID_SNAKE = %q, want %q", got, "my_app")
	}

	resolved, err := resolveRelocations(repoRoot, info, values)
	if err != nil {
		t.Fatalf("resolveRelocations: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved = %d, want 1", len(resolved))
	}
	wantTo := filepath.Join(repoRoot, "packages", "proto", "schemas", "my-app")
	if resolved[0].To != wantTo {
		t.Fatalf("resolved.To = %q, want %q", resolved[0].To, wantTo)
	}

	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	if err := runRelocations(deps, struct{}{}, info.Path, resolved, values, io.Discard); err != nil {
		t.Fatalf("runRelocations: %v", err)
	}

	// Path component substitution: {{SCENARIO_ID}} -> my-app
	out := filepath.Join(wantTo, "my-app", "v1", "health.proto")
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read relocated proto: %v", err)
	}
	// Content substitution: {{SCENARIO_ID_SNAKE}} -> my_app
	if !strings.Contains(string(body), "package vrooli.my_app.v1.health;") {
		t.Fatalf("substitution did not run; body=%s", body)
	}
}

func TestRelocations_VerifyDetectsUnresolvedPlaceholders(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-verify", nil)
	// Append a deliberately unresolved placeholder to the proto body so
	// renderTemplateString can't substitute it (the value never appears
	// in the values map).
	protoFile := filepath.Join(info.Path, "proto", "{{SCENARIO_ID}}", "v1", "health.proto")
	body, err := os.ReadFile(protoFile)
	if err != nil {
		t.Fatalf("read proto: %v", err)
	}
	if err := os.WriteFile(protoFile, append(body, []byte("\n// {{UNKNOWN_PLACEHOLDER}}\n")...), 0o644); err != nil {
		t.Fatalf("write proto with placeholder: %v", err)
	}

	destination := filepath.Join(repoRoot, "scenarios", "verify")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "verify",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	resolved, err := resolveRelocations(repoRoot, info, values)
	if err != nil {
		t.Fatalf("resolveRelocations: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	err = runRelocations(deps, struct{}{}, info.Path, resolved, values, io.Discard)
	if err == nil {
		t.Fatal("expected runRelocations to fail on unresolved placeholder")
	}
	if !strings.Contains(err.Error(), "unresolved placeholders") {
		t.Fatalf("err = %v", err)
	}
}

func TestRelocations_PostCommandsRunAtRepoRoot(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-post", nil)
	destination := filepath.Join(repoRoot, "scenarios", "post-app")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "post-app",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	resolved, err := resolveRelocations(repoRoot, info, values)
	if err != nil {
		t.Fatalf("resolveRelocations: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	if err := runRelocations(deps, struct{}{}, info.Path, resolved, values, io.Discard); err != nil {
		t.Fatalf("runRelocations: %v", err)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("captured %d calls, want 1: %#v", len(capture.calls), capture.calls)
	}
	got := capture.calls[0]
	wantCwd := filepath.Join(repoRoot, "packages", "proto")
	if got.Dir != wantCwd {
		t.Fatalf("post.Dir = %q, want %q (must run at repo root + Cwd, NOT scenario destination)", got.Dir, wantCwd)
	}
	if !strings.Contains(strings.Join(got.Args, " "), "make generate") {
		t.Fatalf("post.Args = %#v, want to contain 'make generate'", got.Args)
	}
}

func TestRunGenerate_RelocationIdempotencyGuard(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-idem", map[string]any{
		"requiredVars": map[string]any{
			"SCENARIO_ID":           map[string]any{"flag": "id"},
			"SCENARIO_DISPLAY_NAME": map[string]any{"flag": "display-name"},
			"SCENARIO_DESCRIPTION":  map[string]any{"flag": "description"},
		},
	})

	// Pre-create the relocation target so the idempotency guard fires.
	preExisting := filepath.Join(repoRoot, "packages", "proto", "schemas", "idem-app")
	if err := os.MkdirAll(preExisting, 0o755); err != nil {
		t.Fatalf("pre-create relocation target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(preExisting, "leftover.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write leftover: %v", err)
	}

	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	req := scenariocli.GenerateRequest{
		TemplateInfo: info,
		Options: scenariocli.GenerateOptions{
			Force: false,
			Values: map[string]string{
				"SCENARIO_ID":           "idem-app",
				"SCENARIO_DISPLAY_NAME": "Idem App",
				"SCENARIO_DESCRIPTION":  "Idempotency test",
			},
		},
	}
	_, err := runGenerate(deps, struct{}{}, req)
	if err == nil {
		t.Fatal("expected runGenerate to error when relocation target exists without --force")
	}
	if !strings.Contains(err.Error(), "relocation target already exists") {
		t.Fatalf("err = %v", err)
	}

	// With --force the existing target is removed.
	req.Options.Force = true
	_, err = runGenerate(deps, struct{}{}, req)
	if err != nil {
		t.Fatalf("runGenerate with --force: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(preExisting, "leftover.txt")); statErr == nil {
		t.Fatal("leftover.txt should have been removed by --force")
	}
}

func TestRunGenerate_DryRunDoesNotWriteRelocations(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-dry", map[string]any{
		"requiredVars": map[string]any{
			"SCENARIO_ID":           map[string]any{"flag": "id"},
			"SCENARIO_DISPLAY_NAME": map[string]any{"flag": "display-name"},
			"SCENARIO_DESCRIPTION":  map[string]any{"flag": "description"},
		},
	})
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	req := scenariocli.GenerateRequest{
		TemplateInfo: info,
		Options: scenariocli.GenerateOptions{
			DryRun: true,
			Values: map[string]string{
				"SCENARIO_ID":           "dry-app",
				"SCENARIO_DISPLAY_NAME": "Dry App",
				"SCENARIO_DESCRIPTION":  "Dry run",
			},
		},
	}
	result, err := runGenerate(deps, struct{}{}, req)
	if err != nil {
		t.Fatalf("dry runGenerate: %v", err)
	}
	if !result.DryRun {
		t.Fatal("result.DryRun should be true")
	}
	if len(result.Relocations) != 1 {
		t.Fatalf("dry-run result.Relocations len = %d, want 1", len(result.Relocations))
	}
	wantTo := filepath.Join(repoRoot, "packages", "proto", "schemas", "dry-app")
	if result.Relocations[0].To != wantTo {
		t.Fatalf("dry-run reloc.To = %q, want %q", result.Relocations[0].To, wantTo)
	}
	// Nothing should have been written.
	if _, err := os.Stat(filepath.Join(repoRoot, "scenarios", "dry-app")); err == nil {
		t.Fatal("dry-run wrote scenario destination")
	}
	if _, err := os.Stat(wantTo); err == nil {
		t.Fatal("dry-run wrote relocation target")
	}
	if len(capture.calls) != 0 {
		t.Fatalf("dry-run invoked %d subprocess calls, want 0", len(capture.calls))
	}
}

func TestCopyTemplate_SkipsRelocationFromDirs(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-skip", nil)
	destination := filepath.Join(repoRoot, "scenarios", "skip-app")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "skip-app",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		t.Fatalf("copyTemplate: %v", err)
	}
	// proto/ is in the manifest's relocations so it must NOT appear
	// inside the scenario destination — the in-tree skip-list pruned it.
	if _, err := os.Stat(filepath.Join(destination, "proto")); err == nil {
		t.Fatal("copyTemplate should skip relocation From dirs (proto/ leaked into scenario destination)")
	}
	// But unrelated files should still land.
	if _, err := os.Stat(filepath.Join(destination, "api", "main.go")); err != nil {
		t.Fatalf("non-relocation file missing from destination: %v", err)
	}
}

func TestCopyTemplate_SkipsManifestCopyExcludes(t *testing.T) {
	templateDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(templateDir, "template.json"), []byte(`{"name":"copy-exclude","copyExcludes":["docs/internal/TEMPLATE-MAINTENANCE.md","tmp-only"]}`), 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(templateDir, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "docs", "internal", "TEMPLATE-MAINTENANCE.md"), []byte("template only"), 0o644); err != nil {
		t.Fatalf("write maint doc: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(templateDir, "tmp-only"), 0o755); err != nil {
		t.Fatalf("mkdir tmp-only: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "tmp-only", "scratch.txt"), []byte("scratch"), 0o644); err != nil {
		t.Fatalf("write scratch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "README.md"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}

	info := scenariocli.TemplateInfo{
		Name: "copy-exclude",
		Path: templateDir,
		Manifest: scenariocli.TemplateManifest{
			Name:         "copy-exclude",
			CopyExcludes: []string{"docs/internal/TEMPLATE-MAINTENANCE.md", "tmp-only"},
		},
	}
	destination := filepath.Join(t.TempDir(), "scenario")
	if err := copyTemplate(info.Path, destination, map[string]string{}, info.Manifest); err != nil {
		t.Fatalf("copyTemplate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "docs", "internal", "TEMPLATE-MAINTENANCE.md")); err == nil {
		t.Fatal("copyTemplate should exclude template-maintenance doc")
	}
	if _, err := os.Stat(filepath.Join(destination, "tmp-only")); err == nil {
		t.Fatal("copyTemplate should exclude template-only directories")
	}
	if _, err := os.Stat(filepath.Join(destination, "README.md")); err != nil {
		t.Fatalf("copyTemplate should keep unrelated files: %v", err)
	}
}

func TestCleanupRelocationTargets_RemovesProtoGeneratedArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioID := "template-validation-react-vite"
	relocTo := filepath.Join(repoRoot, "packages", "proto", "schemas", scenarioID)
	paths := []string{
		relocTo,
		filepath.Join(repoRoot, "packages", "proto", "gen", "go", scenarioID),
		filepath.Join(repoRoot, "packages", "proto", "gen", "typescript", scenarioID),
		filepath.Join(repoRoot, "packages", "proto", "gen", "typescript", "js", scenarioID),
		filepath.Join(repoRoot, "packages", "proto", "gen", "python", scenarioID),
		filepath.Join(repoRoot, "packages", "proto", "gen", "python", "template_validation_react_vite"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	cleanupRelocationTargets([]scenariocli.ResolvedRelocation{{To: relocTo}})

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("cleanupRelocationTargets left %s behind", path)
		}
	}
}

func TestValidateRelocationProtoSources_DetectsProtoDir(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-lint", nil)
	// The validator copies template protos into a temp dir under
	// packages/proto/schemas/ before running `buf lint`, so the schemas
	// directory must exist for the lint path to be exercised.
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "proto", "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir packages/proto/schemas: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	// We don't assert on issue presence (the capture stub always returns
	// nil from RunSubprocess so no failure is reported); we assert the
	// lint call was made with the right shape.
	if len(capture.calls) != 1 {
		t.Fatalf("captured %d calls, want 1 buf lint invocation", len(capture.calls))
	}
	args := strings.Join(capture.calls[0].Args, " ")
	if !strings.Contains(args, "buf lint") {
		t.Fatalf("call args = %q, want to contain 'buf lint'", args)
	}
	// The path is a temp subdirectory under schemas/ — it should be a
	// relative path beginning with `schemas/.tmp-validate-reloc-lint-`.
	if !strings.Contains(args, "schemas/.tmp-validate-reloc-lint-") {
		t.Fatalf("call args = %q, want to lint a temp dir under schemas/", args)
	}
	if capture.calls[0].Dir != filepath.Join(repoRoot, "packages", "proto") {
		t.Fatalf("call dir = %q, want packages/proto", capture.calls[0].Dir)
	}
	_ = issues
}

// TestValidateRelocationProtoSources_SurfacesStdoutOnLintFailure pins the
// stdout-capture fix: `buf lint` writes diagnostics to stdout, not stderr,
// so the validator must surface stdout in the issue Message. Pre-fix,
// failures collapsed to a useless "buf lint failed: exit status 100" string.
func TestValidateRelocationProtoSources_SurfacesStdoutOnLintFailure(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-lint-fail", nil)
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "proto", "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir packages/proto/schemas: %v", err)
	}

	const lintDiagnostic = `Service name "Notes" should be suffixed with "Service".`
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, &capturedSubprocess{})
	deps.RunSubprocess = func(_ struct{}, spec scenarioexec.SubprocessSpec) error {
		if spec.Stdout != nil {
			_, _ = io.WriteString(spec.Stdout, lintDiagnostic+"\n")
		}
		return errors.New("exit status 100")
	}

	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want exactly 1", issues)
	}
	msg := issues[0].Message
	if !strings.Contains(msg, lintDiagnostic) {
		t.Fatalf("issue message %q must surface the stdout diagnostic %q", msg, lintDiagnostic)
	}
	if strings.Contains(msg, ".tmp-validate-") {
		t.Fatalf("issue message %q must rewrite the temp-dir prefix back to the template path; got raw temp dir", msg)
	}
}

func TestValidateRelocationProtoSources_SkipsWhenSchemasDirAbsent(t *testing.T) {
	// When packages/proto/schemas/ doesn't exist (test repo without a
	// real proto module), the validator returns no issues and runs no
	// subprocess — schema-level mistakes will surface at make-generate
	// time during an actual scenario generation, which is a separate
	// failure mode.
	repoRoot, info := writeRelocationTemplate(t, "reloc-noschemas", nil)
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want none when schemas dir absent", issues)
	}
	if len(capture.calls) != 0 {
		t.Fatalf("invoked %d subprocess calls; want 0 when schemas dir absent", len(capture.calls))
	}
}

// TestValidateRelocationProtoSources_RunsOfflineWithBSRUnreachable proves
// that template proto validation does not contact BSR after the local-plugin
// and vendored-module cutover.
//
// Mechanism: invoke validateRelocationProtoSources end-to-end against the
// real packages/proto/ module with HTTPS_PROXY=http://127.0.0.1:9 forced
// into the subprocess env. Any BSR call would hard-fail; the test asserts
// the lint still succeeds. The fixture proto imports
// `buf/validate/validate.proto` so transitive resolution through the
// vendored protovalidate workspace module is exercised — pre-CD-2 this
// import would have triggered a BSR fetch.
//
// Skipped when buf isn't on PATH or when packages/proto/schemas/ is
// missing (e.g. minimal CI environments without `vrooli setup` complete).
func TestValidateRelocationProtoSources_RunsOfflineWithBSRUnreachable(t *testing.T) {
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skipf("buf not on PATH: %v", err)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "packages", "proto", "schemas")); err != nil {
		t.Skipf("packages/proto/schemas not present in this repo layout: %v", err)
	}

	// Pollute templates/scenarios/ with a uniquely named fixture and
	// clean it up on completion. The validator copies into
	// packages/proto/schemas/.tmp-validate-<name>-<rand>/ and removes that
	// itself, so we only own the templates side.
	templateName := "offline-bsr-validation-test"
	templatesDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if _, err := os.Stat(templatesDir); err == nil {
		t.Fatalf("fixture %s already exists; another run may be in progress", templatesDir)
	}
	t.Cleanup(func() { _ = os.RemoveAll(templatesDir) })

	protoSrc := filepath.Join(templatesDir, "proto", "{{SCENARIO_ID}}", "v1")
	if err := os.MkdirAll(protoSrc, 0o755); err != nil {
		t.Fatalf("mkdir proto src: %v", err)
	}
	// The import on `buf/validate/validate.proto` is the load-bearing
	// detail: it forces buf lint to resolve a symbol from the vendored
	// protovalidate module. Pre-CD-2, buf would have fetched it from BSR.
	protoBody := strings.Join([]string{
		`syntax = "proto3";`,
		`package vrooli.{{SCENARIO_ID_SNAKE}}.v1.health;`,
		`import "buf/validate/validate.proto";`,
		`message Probe {`,
		`  string name = 1 [(buf.validate.field).string.min_len = 1];`,
		`}`,
		``,
	}, "\n")
	if err := os.WriteFile(filepath.Join(protoSrc, "health.proto"), []byte(protoBody), 0o644); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	manifest := map[string]any{
		"name":         templateName,
		"requiredVars": map[string]any{"SCENARIO_ID": map[string]any{"flag": "id"}},
		"relocations": []map[string]any{{
			"description": "Relocate proto schemas",
			"from":        "proto/",
			"to":          "packages/proto/schemas/{{SCENARIO_ID}}/",
		}},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "template.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}

	info, err := loadTemplate(repoRoot, templateName)
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}

	// Real subprocess runner; CommandEnv injects deliberately unreachable
	// proxies so any BSR HTTP would hard-fail. The test asserts the lint
	// still succeeds — proving template validation is offline-clean.
	deps := HandlerDeps[struct{}]{
		Stdout: func(struct{}) io.Writer { return io.Discard },
		Stderr: func(struct{}) io.Writer { return io.Discard },
		Root:   func(struct{}) string { return repoRoot },
		RunSubprocess: func(_ struct{}, spec scenarioexec.SubprocessSpec) error {
			return scenarioexec.RunSubprocess(spec)
		},
		CommandEnv: func(struct{}) []string {
			// Inherit PATH/HOME so `buf` and ~/.netrc are reachable, then
			// add proxies pointing at a port that nothing listens on.
			env := os.Environ()
			env = append(env,
				"HTTPS_PROXY=http://127.0.0.1:9",
				"HTTP_PROXY=http://127.0.0.1:9",
				"https_proxy=http://127.0.0.1:9",
				"http_proxy=http://127.0.0.1:9",
				"NO_PROXY=",
				"no_proxy=",
			)
			return env
		},
	}

	issues := validateRelocationProtoSources(deps, struct{}{}, info)

	if len(issues) != 0 {
		var msg strings.Builder
		for _, iss := range issues {
			fmt.Fprintf(&msg, "  - [%s] %s: %s\n", iss.Template, iss.Path, iss.Message)
		}
		t.Fatalf(
			"validateRelocationProtoSources surfaced %d issue(s) with HTTPS_PROXY=http://127.0.0.1:9 — codegen is reaching BSR (CD-1/CD-2 regressed):\n%s",
			len(issues), msg.String(),
		)
	}
}

func TestValidateRelocationProtoSources_SkipsNonProtoRelocations(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templatesDir := filepath.Join(repoRoot, "templates", "scenarios", "reloc-nonproto")
	scriptsDir := filepath.Join(templatesDir, "scripts", "{{SCENARIO_ID}}")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "deploy.sh"), []byte("#!/bin/bash\n"), 0o644); err != nil {
		t.Fatalf("write deploy.sh: %v", err)
	}
	manifest := map[string]any{
		"name": "reloc-nonproto",
		"relocations": []map[string]any{
			{"from": "scripts/", "to": "scripts/{{SCENARIO_ID}}/"},
		},
	}
	manifestBytes, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(templatesDir, "template.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}
	info, err := loadTemplate(repoRoot, "reloc-nonproto")
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	if len(issues) != 0 {
		t.Fatalf("non-proto relocation produced issues: %#v", issues)
	}
	if len(capture.calls) != 0 {
		t.Fatalf("non-proto relocation invoked %d subprocess calls; should not run buf lint", len(capture.calls))
	}
}
