package lint

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/lint/execution"
	"test-genie/internal/shared"
)

type stubCommandRunner struct {
	byName map[string]execution.Result
	errs   map[string]error
}

func (s stubCommandRunner) Run(_ context.Context, cmd execution.Command) (execution.Result, error) {
	if err := s.errs[filepath.Base(cmd.Name)]; err != nil {
		return execution.Result{}, err
	}
	if result, ok := s.byName[filepath.Base(cmd.Name)]; ok {
		return result, nil
	}
	return execution.Result{}, errors.New("unexpected command")
}

func TestRunner_Run_NoLintableComponentsDetected(t *testing.T) {
	config := Config{
		ScenarioDir:   t.TempDir(),
		ScenarioName:  "test-scenario",
		CommandRunner: stubCommandRunner{},
	}

	var buf bytes.Buffer
	runner := New(config, WithLogger(&buf))
	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatal("expected success when no lintable components are detected")
	}
	if len(result.Components) != 0 {
		t.Fatalf("expected no component results, got %d", len(result.Components))
	}
}

func TestRunner_Run_CommonUnconfiguredAPIIsFailure(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := New(Config{
		ScenarioDir:   tmpDir,
		ScenarioName:  "demo",
		CommandRunner: stubCommandRunner{},
	}).Run(context.Background())

	if result.Success {
		t.Fatal("expected unmatched api component to fail")
	}
	if result.Summary.PolicyErrors != 1 {
		t.Fatalf("expected one policy error, got %d", result.Summary.PolicyErrors)
	}
}

func TestRunner_Run_ComponentDiscoveryUsesTopLevelSidecars(t *testing.T) {
	tmpDir := t.TempDir()
	workerDir := filepath.Join(tmpDir, "worker")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "go.mod"), []byte("module example.com/worker\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := New(Config{
		ScenarioDir:  tmpDir,
		ScenarioName: "demo",
		CommandLookup: func(name string) (string, error) {
			return "/tmp/" + name, nil
		},
		CommandRunner: stubCommandRunner{
			byName: map[string]execution.Result{
				"golangci-lint": {Stdout: []byte(`{"Issues":[]}`), ExitCode: 0},
			},
		},
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.Summary.ComponentsLinted != 1 {
		t.Fatalf("expected one linted component, got %d", result.Summary.ComponentsLinted)
	}
	if len(result.Components) == 0 || result.Components[0].Component.Name != "worker" {
		t.Fatalf("expected worker component result, got %+v", result.Components)
	}
}

func TestRunner_Run_MixedComponentsRootOverridesIgnoreAndPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]\nname='demo'\nversion='0.1.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	workerDir := filepath.Join(tmpDir, "worker")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "go.mod"), []byte("module example.com/worker\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	frontendDir := filepath.Join(tmpDir, "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte(`{"name":"frontend"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "eslint.config.js"), []byte(`export default [];`), 0o644); err != nil {
		t.Fatal(err)
	}

	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "main.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "helper.go"), []byte("package docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	settings := DefaultSettings()
	settings.Ignore = append(settings.Ignore, "docs")
	settings.Components["frontend"] = ComponentSettings{
		Handler: HandlerNodePackage,
		Strict:  boolPtr(true),
	}

	result := New(Config{
		ScenarioDir:  tmpDir,
		ScenarioName: "demo",
		CommandLookup: func(name string) (string, error) {
			return "/tmp/" + name, nil
		},
		CommandRunner: stubCommandRunner{
			byName: map[string]execution.Result{
				"ruff":           {Stdout: []byte(`[]`), ExitCode: 0},
				"golangci-lint":  {Stdout: []byte(`{"Issues":[]}`), ExitCode: 0},
				"eslint":         {Stdout: []byte(`[]`), ExitCode: 0},
			},
		},
		Settings: settings,
	}).Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success with only policy warnings, got error: %v", result.Error)
	}
	if result.Summary.ComponentsLinted != 3 {
		t.Fatalf("expected three linted components, got %d", result.Summary.ComponentsLinted)
	}
	if result.Summary.PolicyWarnings != 1 {
		t.Fatalf("expected one policy warning for unmatched ui, got %d", result.Summary.PolicyWarnings)
	}
	if result.Summary.PolicyErrors != 0 {
		t.Fatalf("expected no policy errors, got %d", result.Summary.PolicyErrors)
	}

	seen := map[string]ComponentResult{}
	for _, component := range result.Components {
		seen[component.Component.Name] = component
		if component.Component.Name == "docs" {
			t.Fatalf("ignored docs component should not be present: %+v", result.Components)
		}
	}

	if root, ok := seen["."]; !ok || root.HandlerID != HandlerPythonProject {
		t.Fatalf("expected root component matched to python_project, got %+v", seen["."])
	}
	if worker, ok := seen["worker"]; !ok || worker.HandlerID != HandlerGoModule {
		t.Fatalf("expected worker matched to go_module, got %+v", seen["worker"])
	}
	if frontend, ok := seen["frontend"]; !ok || frontend.HandlerID != HandlerNodePackage || !frontend.Strict {
		t.Fatalf("expected frontend override to force strict node_package, got %+v", seen["frontend"])
	}
	if ui, ok := seen["ui"]; !ok || ui.Matched || len(ui.PolicyFindings) != 1 {
		t.Fatalf("expected unmatched ui with one policy finding, got %+v", seen["ui"])
	}
}

func TestRunner_Run_ComponentOverrideUnknownHandlerFails(t *testing.T) {
	tmpDir := t.TempDir()
	workerDir := filepath.Join(tmpDir, "worker")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "go.mod"), []byte("module example.com/worker\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	settings := DefaultSettings()
	settings.Components["worker"] = ComponentSettings{Handler: "does_not_exist"}

	result := New(Config{
		ScenarioDir:   tmpDir,
		ScenarioName:  "demo",
		CommandRunner: stubCommandRunner{},
		Settings:      settings,
	}).Run(context.Background())

	if result.Success {
		t.Fatal("expected unknown handler override to fail")
	}
	if result.Summary.PolicyErrors != 1 {
		t.Fatalf("expected one policy error, got %d", result.Summary.PolicyErrors)
	}
	if len(result.Components) != 1 || !strings.Contains(result.Components[0].PolicyFindings[0].Message, "unknown lint handler") {
		t.Fatalf("expected unknown handler policy finding, got %+v", result.Components)
	}
}

func TestRunner_Run_ComponentOverrideMismatchFails(t *testing.T) {
	tmpDir := t.TempDir()
	workerDir := filepath.Join(tmpDir, "worker")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, "package.json"), []byte(`{"name":"worker"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings := DefaultSettings()
	settings.Components["worker"] = ComponentSettings{Handler: HandlerGoModule}

	result := New(Config{
		ScenarioDir:   tmpDir,
		ScenarioName:  "demo",
		CommandRunner: stubCommandRunner{},
		Settings:      settings,
	}).Run(context.Background())

	if result.Success {
		t.Fatal("expected mismatched handler override to fail")
	}
	if result.Summary.PolicyErrors != 1 {
		t.Fatalf("expected one policy error, got %d", result.Summary.PolicyErrors)
	}
	if len(result.Components) != 1 || !strings.Contains(result.Components[0].PolicyFindings[0].Message, "does not match its lint contract") {
		t.Fatalf("expected handler mismatch policy finding, got %+v", result.Components)
	}
}

func TestRunner_Run_AmbiguousHandlerMatchFails(t *testing.T) {
	tmpDir := t.TempDir()
	mixedDir := filepath.Join(tmpDir, "mixed")
	if err := os.MkdirAll(mixedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mixedDir, "package.json"), []byte(`{"name":"mixed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mixedDir, "tool.py"), []byte("print('hi')\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := New(Config{
		ScenarioDir:   tmpDir,
		ScenarioName:  "demo",
		CommandRunner: stubCommandRunner{},
	}).Run(context.Background())

	if result.Success {
		t.Fatal("expected ambiguous handler match to fail")
	}
	if result.Summary.PolicyErrors != 1 {
		t.Fatalf("expected one policy error, got %d", result.Summary.PolicyErrors)
	}
	if len(result.Components) != 1 || !strings.Contains(result.Components[0].PolicyFindings[0].Message, "matches multiple lint handlers") {
		t.Fatalf("expected ambiguous match policy finding, got %+v", result.Components)
	}
}

func TestRunner_Run_StrictWarningsFailPhase(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"demo-ui"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "eslint.config.js"), []byte(`export default [];`), 0o644); err != nil {
		t.Fatal(err)
	}

	settings := DefaultSettings()
	handler := settings.Handlers[HandlerNodePackage]
	handler.Strict = true
	settings.Handlers[HandlerNodePackage] = handler

	runner := New(Config{
		ScenarioDir:  tmpDir,
		ScenarioName: "demo",
		CommandLookup: func(name string) (string, error) {
			return "/tmp/" + name, nil
		},
		CommandRunner: stubCommandRunner{
			byName: map[string]execution.Result{
				"eslint": {Stdout: []byte(`[{"filePath":"src/index.ts","messages":[{"line":1,"column":1,"message":"warn","ruleId":"demo","severity":1}]}]`), ExitCode: 1},
			},
		},
		Settings: settings,
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected strict node warnings to fail the phase")
	}
	if result.Summary.LintWarnings != 1 {
		t.Fatalf("expected one lint warning, got %d", result.Summary.LintWarnings)
	}
}

func TestRunner_Run_ContextCancelled(t *testing.T) {
	config := Config{
		ScenarioDir:   t.TempDir(),
		ScenarioName:  "test-scenario",
		CommandRunner: stubCommandRunner{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := New(config).Run(ctx)
	if result.Success {
		t.Fatal("expected failure when context is cancelled")
	}
	if result.FailureClass != FailureClassSystem {
		t.Fatalf("expected system failure class, got %v", result.FailureClass)
	}
}

func TestLintSummaryMethods(t *testing.T) {
	summary := LintSummary{
		ComponentsLinted:    2,
		ComponentsUnmatched: 1,
		TypeErrors:          1,
		LintWarnings:        3,
		PolicyWarnings:      2,
	}
	if got := summary.TotalChecks(); got != 2 {
		t.Fatalf("TotalChecks() = %d, want 2", got)
	}
	if got := summary.TotalIssues(); got != 6 {
		t.Fatalf("TotalIssues() = %d, want 6", got)
	}
	if !summary.HasTypeErrors() {
		t.Fatal("expected HasTypeErrors true")
	}
	if got := summary.String(); got == "" {
		t.Fatal("expected non-empty summary string")
	}
}

func TestNewObservations(t *testing.T) {
	section := NewSectionObservation("icon", "Section message")
	if section.Type != shared.ObservationSection {
		t.Errorf("expected ObservationSection, got %v", section.Type)
	}
	if section.Icon != "icon" {
		t.Errorf("expected icon 'icon', got %v", section.Icon)
	}

	success := NewSuccessObservation("Success message")
	if success.Type != shared.ObservationSuccess {
		t.Errorf("expected ObservationSuccess, got %v", success.Type)
	}

	warning := NewWarningObservation("Warning message")
	if warning.Type != shared.ObservationWarning {
		t.Errorf("expected ObservationWarning, got %v", warning.Type)
	}

	errorObs := NewErrorObservation("Error message")
	if errorObs.Type != shared.ObservationError {
		t.Errorf("expected ObservationError, got %v", errorObs.Type)
	}

	info := NewInfoObservation("Info message")
	if info.Type != shared.ObservationInfo {
		t.Errorf("expected ObservationInfo, got %v", info.Type)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
