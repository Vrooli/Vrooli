package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	runtimeapi "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
)

// TestBuildStage tests the build stage.
func TestBuildStage(t *testing.T) {
	t.Run("NewBuildStage with options", func(t *testing.T) {
		mockTime := &mockTimeProvider{now: time.Now().Unix()}
		stage := NewBuildStage(
			WithBuildTimeProvider(mockTime),
		)
		if stage == nil {
			t.Fatal("expected stage to be created")
		}
	})

	t.Run("Name", func(t *testing.T) {
		stage := NewBuildStage()
		if stage.Name() != StageBuild {
			t.Errorf("expected name %q, got %q", StageBuild, stage.Name())
		}
	})

	t.Run("Dependencies", func(t *testing.T) {
		stage := NewBuildStage()
		deps := stage.Dependencies()
		if len(deps) != 1 || deps[0] != StageGenerate {
			t.Errorf("expected dependencies [%s], got %v", StageGenerate, deps)
		}
	})

	t.Run("CanSkip", func(t *testing.T) {
		stage := NewBuildStage()
		input := &StageInput{Config: &Config{}}
		if stage.CanSkip(input) {
			t.Error("expected CanSkip to return false")
		}
	})
}

// TestBundleStage tests the bundle stage.
func TestBundleStage(t *testing.T) {
	t.Run("NewBundleStage with options", func(t *testing.T) {
		mockTime := &mockTimeProvider{now: time.Now().Unix()}
		stage := NewBundleStage(
			WithScenarioRoot("/tmp"),
			WithBundleTimeProvider(mockTime),
		)
		if stage == nil {
			t.Fatal("expected stage to be created")
		}
	})

	t.Run("Name", func(t *testing.T) {
		stage := NewBundleStage()
		if stage.Name() != StageBundle {
			t.Errorf("expected name %q, got %q", StageBundle, stage.Name())
		}
	})

	t.Run("Dependencies", func(t *testing.T) {
		stage := NewBundleStage()
		deps := stage.Dependencies()
		if len(deps) != 0 {
			t.Errorf("expected no dependencies, got %v", deps)
		}
	})

	t.Run("CanSkip with proxy mode", func(t *testing.T) {
		stage := NewBundleStage()
		input := &StageInput{Config: &Config{DeploymentMode: "proxy"}}
		if !stage.CanSkip(input) {
			t.Error("expected CanSkip to return true for proxy mode")
		}
	})

	t.Run("CanSkip with bundled mode", func(t *testing.T) {
		stage := NewBundleStage()
		input := &StageInput{Config: &Config{DeploymentMode: "bundled"}}
		if stage.CanSkip(input) {
			t.Error("expected CanSkip to return false for bundled mode")
		}
	})
}

// TestGenerateStage tests the generate stage.
func TestGenerateStage(t *testing.T) {
	t.Run("NewGenerateStage with options", func(t *testing.T) {
		mockTime := &mockTimeProvider{now: time.Now().Unix()}
		stage := NewGenerateStage(
			WithGenerateScenarioRoot("/tmp"),
			WithGenerateTimeProvider(mockTime),
		)
		if stage == nil {
			t.Fatal("expected stage to be created")
		}
	})

	t.Run("Name", func(t *testing.T) {
		stage := NewGenerateStage()
		if stage.Name() != StageGenerate {
			t.Errorf("expected name %q, got %q", StageGenerate, stage.Name())
		}
	})

	t.Run("Dependencies", func(t *testing.T) {
		stage := NewGenerateStage()
		deps := stage.Dependencies()
		// Should depend on bundle or preflight
		if len(deps) != 1 {
			t.Errorf("expected 1 dependency, got %v", deps)
		}
	})

	t.Run("CanSkip", func(t *testing.T) {
		stage := NewGenerateStage()
		input := &StageInput{Config: &Config{}}
		if stage.CanSkip(input) {
			t.Error("expected CanSkip to return false")
		}
	})

	t.Run("default scenario root uses contract", func(t *testing.T) {
		root := newStageContractFixtureRepo(t)
		nested := filepath.Join(root, "scenarios", "scenario-to-desktop", "api")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}
		t.Setenv("VROOLI_SOURCE_ROOT", nested)
		t.Setenv("VROOLI_ROOT", "")

		stage := NewGenerateStage()
		want := filepath.Join(root, "scenarios")
		if stage.scenarioRoot != want {
			t.Fatalf("scenarioRoot = %q, want %q", stage.scenarioRoot, want)
		}
	})
}

// TestPreflightStage tests the preflight stage.
func TestPreflightStage(t *testing.T) {
	t.Run("NewPreflightStage with options", func(t *testing.T) {
		mockTime := &mockTimeProvider{now: time.Now().Unix()}
		stage := NewPreflightStage(
			WithPreflightTimeProvider(mockTime),
		)
		if stage == nil {
			t.Fatal("expected stage to be created")
		}
	})

	t.Run("Name", func(t *testing.T) {
		stage := NewPreflightStage()
		if stage.Name() != StagePreflight {
			t.Errorf("expected name %q, got %q", StagePreflight, stage.Name())
		}
	})

	t.Run("Dependencies", func(t *testing.T) {
		stage := NewPreflightStage()
		deps := stage.Dependencies()
		if len(deps) != 1 || deps[0] != StageBundle {
			t.Errorf("expected dependencies [%s], got %v", StageBundle, deps)
		}
	})

	t.Run("CanSkip when skipped in config", func(t *testing.T) {
		stage := NewPreflightStage()
		input := &StageInput{Config: &Config{SkipPreflight: true}}
		if !stage.CanSkip(input) {
			t.Error("expected CanSkip to return true when SkipPreflight is true")
		}
	})

	t.Run("CanSkip when not skipped in bundled mode", func(t *testing.T) {
		stage := NewPreflightStage()
		input := &StageInput{Config: &Config{SkipPreflight: false, DeploymentMode: DeploymentModeBundled}}
		if stage.CanSkip(input) {
			t.Error("expected CanSkip to return false when SkipPreflight is false in bundled mode")
		}
	})

	t.Run("CanSkip in proxy mode", func(t *testing.T) {
		stage := NewPreflightStage()
		input := &StageInput{Config: &Config{DeploymentMode: "proxy"}}
		if !stage.CanSkip(input) {
			t.Error("expected CanSkip to return true in proxy mode")
		}
	})
}

func TestBundleStage_DefaultScenarioRootUsesContract(t *testing.T) {
	root := newStageContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "scenario-to-desktop", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	stage := NewBundleStage()
	want := filepath.Join(root, "scenarios")
	if stage.scenarioRoot != want {
		t.Fatalf("scenarioRoot = %q, want %q", stage.scenarioRoot, want)
	}
}

func TestOrchestrator_DefaultScenarioRootUsesContract(t *testing.T) {
	root := newStageContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "scenario-to-desktop", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	orchestrator := NewOrchestrator(WithStages(&mockStage{name: "test"}))
	want := filepath.Join(root, "scenarios")
	if orchestrator.scenarioRoot != want {
		t.Fatalf("scenarioRoot = %q, want %q", orchestrator.scenarioRoot, want)
	}
}

func newStageContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := stageRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/scenario-to-desktop-stage-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"scenarios", "resources", "packages", "cmd", "internal", "templates"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func stageRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}

// TestSmokeTestStage tests the smoke test stage.
func TestSmokeTestStage(t *testing.T) {
	t.Run("NewSmokeTestStage with options", func(t *testing.T) {
		mockTime := &mockTimeProvider{now: time.Now().Unix()}
		stage := NewSmokeTestStage(
			WithSmokeTestTimeProvider(mockTime),
		)
		if stage == nil {
			t.Fatal("expected stage to be created")
		}
	})

	t.Run("Name", func(t *testing.T) {
		stage := NewSmokeTestStage()
		if stage.Name() != StageSmokeTest {
			t.Errorf("expected name %q, got %q", StageSmokeTest, stage.Name())
		}
	})

	t.Run("Dependencies", func(t *testing.T) {
		stage := NewSmokeTestStage()
		deps := stage.Dependencies()
		if len(deps) != 1 || deps[0] != StageBuild {
			t.Errorf("expected dependencies [%s], got %v", StageBuild, deps)
		}
	})

	t.Run("CanSkip when skipped in config", func(t *testing.T) {
		stage := NewSmokeTestStage()
		input := &StageInput{Config: &Config{SkipSmokeTest: true}}
		if !stage.CanSkip(input) {
			t.Error("expected CanSkip to return true when SkipSmokeTest is true")
		}
	})

	t.Run("CanSkip when not skipped", func(t *testing.T) {
		stage := NewSmokeTestStage()
		input := &StageInput{Config: &Config{SkipSmokeTest: false}}
		if stage.CanSkip(input) {
			t.Error("expected CanSkip to return false when SkipSmokeTest is false")
		}
	})
}

// TestStageExecuteWithMissingService tests execute errors.
func TestStageExecuteWithMissingService(t *testing.T) {
	ctx := context.Background()
	input := &StageInput{
		Config:       &Config{ScenarioName: "test"},
		ScenarioPath: "/tmp/test",
		Logger:       &mockLogger{},
	}

	t.Run("build stage without service", func(t *testing.T) {
		stage := NewBuildStage()
		result := stage.Execute(ctx, input)
		if result.Status != StatusFailed {
			t.Error("expected failed status when service is nil")
		}
	})

	t.Run("bundle stage without packager", func(t *testing.T) {
		stage := NewBundleStage()
		input.Config.DeploymentMode = "bundled"
		result := stage.Execute(ctx, input)
		if result.Status != StatusFailed {
			t.Error("expected failed status when packager is nil")
		}
	})

	t.Run("generate stage without service", func(t *testing.T) {
		stage := NewGenerateStage()
		result := stage.Execute(ctx, input)
		if result.Status != StatusFailed {
			t.Error("expected failed status when service is nil")
		}
	})

	t.Run("preflight stage without service", func(t *testing.T) {
		stage := NewPreflightStage()
		input.Config.DeploymentMode = "bundled"
		input.Config.SkipPreflight = false
		result := stage.Execute(ctx, input)
		if result.Status != StatusFailed {
			t.Error("expected failed status when service is nil")
		}
	})

	t.Run("smoketest stage without service", func(t *testing.T) {
		stage := NewSmokeTestStage()
		result := stage.Execute(ctx, input)
		if result.Status != StatusFailed {
			t.Error("expected failed status when service is nil")
		}
	})
}

// TestExtractValidationErrors tests the helper function for extracting validation errors.
func TestExtractValidationErrors(t *testing.T) {
	t.Run("empty validation result", func(t *testing.T) {
		result := &runtimeapi.BundleValidationResult{Valid: true}
		errs := extractValidationErrors(result)
		if len(errs) != 0 {
			t.Errorf("expected 0 errors, got %d", len(errs))
		}
	})

	t.Run("with errors", func(t *testing.T) {
		result := &runtimeapi.BundleValidationResult{
			Valid: false,
			Errors: []runtimeapi.BundleError{
				{Code: "ERR001", Service: "api", Message: "missing config"},
				{Code: "ERR002", Service: "ui", Message: "invalid entry point"},
			},
		}
		errs := extractValidationErrors(result)
		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs))
		}
		if errs[0] != "[ERR001] api: missing config" {
			t.Errorf("unexpected error format: %s", errs[0])
		}
	})

	t.Run("with missing binaries", func(t *testing.T) {
		result := &runtimeapi.BundleValidationResult{
			Valid: false,
			MissingBinaries: []runtimeapi.MissingBinary{
				{ServiceID: "api", Platform: "linux", Path: "/bin/api"},
			},
		}
		errs := extractValidationErrors(result)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0] != "missing binary: /bin/api (api/linux)" {
			t.Errorf("unexpected error format: %s", errs[0])
		}
	})

	t.Run("with missing assets", func(t *testing.T) {
		result := &runtimeapi.BundleValidationResult{
			Valid: false,
			MissingAssets: []runtimeapi.MissingAsset{
				{ServiceID: "ui", Path: "/assets/logo.png"},
			},
		}
		errs := extractValidationErrors(result)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0] != "missing asset: /assets/logo.png (ui)" {
			t.Errorf("unexpected error format: %s", errs[0])
		}
	})

	t.Run("with all types", func(t *testing.T) {
		result := &runtimeapi.BundleValidationResult{
			Valid: false,
			Errors: []runtimeapi.BundleError{
				{Code: "ERR001", Service: "api", Message: "error 1"},
			},
			MissingBinaries: []runtimeapi.MissingBinary{
				{ServiceID: "api", Platform: "linux", Path: "/bin/api"},
			},
			MissingAssets: []runtimeapi.MissingAsset{
				{ServiceID: "ui", Path: "/assets/logo.png"},
			},
		}
		errs := extractValidationErrors(result)
		if len(errs) != 3 {
			t.Errorf("expected 3 errors, got %d", len(errs))
		}
	})
}
