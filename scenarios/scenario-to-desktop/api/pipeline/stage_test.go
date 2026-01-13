package pipeline

import (
	"context"
	"testing"
	"time"
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

	t.Run("CanSkip when not skipped", func(t *testing.T) {
		stage := NewPreflightStage()
		input := &StageInput{Config: &Config{SkipPreflight: false}}
		if stage.CanSkip(input) {
			t.Error("expected CanSkip to return false when SkipPreflight is false")
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

