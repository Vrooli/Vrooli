// Package pipeline provides stage skip decision helpers.
//
// This file extracts skip decision logic from individual stages into named,
// testable functions. Each function encapsulates the business rules for when
// a stage should be skipped, making the decision boundaries explicit and
// easier to reason about.
package pipeline

import (
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/shared/errors"
)

// ShouldSkipPreflight returns true if the preflight stage should be skipped.
// Skip conditions:
//   - Config.SkipPreflight is explicitly true
//   - Deployment mode is "proxy" (no bundle to validate)
func ShouldSkipPreflight(config *Config) bool {
	if config == nil {
		return false
	}
	return config.SkipPreflight || config.GetDeploymentMode() == DeploymentModeProxy
}

// ShouldSkipBundle returns true if the bundle stage should be skipped.
// Skip conditions:
//   - Deployment mode is "proxy" (no bundling needed)
func ShouldSkipBundle(config *Config) bool {
	if config == nil {
		return false
	}
	return config.GetDeploymentMode() == DeploymentModeProxy
}

// ShouldSkipSmokeTest returns true if the smoke test stage should be skipped.
// Skip conditions:
//   - Config.SkipSmokeTest is explicitly true
func ShouldSkipSmokeTest(config *Config) bool {
	if config == nil {
		return false
	}
	return config.SkipSmokeTest
}

// ShouldSkipDistribution returns true if the distribution stage should be skipped.
// Skip conditions:
//   - Config is nil
//   - Distribution is not enabled (Config.Distribute is false)
//   - No build artifacts are available
func ShouldSkipDistribution(config *Config, buildResult *build.Status) bool {
	if config == nil || !config.Distribute {
		return true
	}
	if buildResult == nil || len(buildResult.Artifacts) == 0 {
		return true
	}
	return false
}

// ValidateCanResume checks if a pipeline can be resumed from a later stage.
// Returns an error describing why resumption is not possible, or nil if valid.
func ValidateCanResume(status *Status) error {
	if status == nil {
		return errors.ErrPipelineNotFound("").WithMessage("pipeline status is nil")
	}
	if status.Status != StatusCompleted {
		return errors.ErrPipelineNotResumable(status.PipelineID,
			"status is "+status.Status+" (must be completed)")
	}
	if status.StoppedAfterStage == "" {
		return errors.ErrPipelineNotResumable(status.PipelineID,
			"was not stopped after a stage")
	}
	return nil
}

// ShouldStopAfterStage returns true if the pipeline should stop after the given stage.
func ShouldStopAfterStage(config *Config, stageName string) bool {
	if config == nil {
		return false
	}
	return config.GetStopAfterStage() == stageName
}

// ShouldSkipSigning returns true if the signing stage should be skipped.
// Skip conditions:
//   - Config.Sign is explicitly false or not set (signing is opt-in)
func ShouldSkipSigning(config *Config) bool {
	if config == nil {
		return true
	}
	return !config.Sign
}

// ShouldSkipGeneration returns true if the generation stage should be skipped.
// Skip conditions:
//   - ResumeFromStage is set to a stage after generation (i.e., resuming past generation)
//
// Note: Unlike other stages, generation cannot be skipped via config flag because
// it's required to produce the desktop wrapper for the build stage.
func ShouldSkipGeneration(config *Config) bool {
	if config == nil {
		return false
	}
	resume := config.GetResumeFromStage()
	if resume == "" {
		return false
	}
	// Skip if resuming from build or later (generation already done)
	return resume == StageBuild || resume == StageSmokeTest || resume == StageDistribution
}

// IsBuildComplete returns true if the build has finished (successfully or not).
// Use this to check if polling should stop.
func IsBuildComplete(status string) bool {
	return status == BuildStatusReady ||
		status == BuildStatusPartial ||
		status == BuildStatusFailed
}

// IsBuildFailed returns true if the build failed completely.
// A partial build (some platforms succeeded) is NOT considered failed.
func IsBuildFailed(status string) bool {
	return status == BuildStatusFailed
}

// IsBuildSuccessful returns true if at least some platforms built successfully.
// This includes both "ready" (all succeeded) and "partial" (some succeeded).
func IsBuildSuccessful(status string) bool {
	return status == BuildStatusReady || status == BuildStatusPartial
}

// ShouldSkipPlatform returns true if a platform should be skipped during build.
// Skip conditions:
//   - Windows platform on Linux without Wine installed
func ShouldSkipPlatform(platform string, wineInstalled bool) bool {
	if platform == "win" && !wineInstalled {
		return true
	}
	return false
}

// CanRunStagesInParallel documents which stages can run concurrently.
// Currently all stages are sequential due to data dependencies:
//   - Bundle must complete before Preflight (preflight validates bundled assets)
//   - Preflight must complete before Generate (ensures prerequisites)
//   - Generate must complete before Build (creates the code to build)
//   - Build must complete before SmokeTest (tests the built artifact)
//   - SmokeTest must complete before Distribution (validates before upload)
//
// Future optimization: Preflight and Bundle could potentially run in parallel
// since preflight checks system dependencies while bundle packages assets.
func CanRunStagesInParallel(stageA, stageB string) bool {
	// Currently no stages can run in parallel due to sequential dependencies.
	// Explicitly return false to document this design decision.
	// See function comment for future optimization opportunities.
	_ = stageA
	_ = stageB
	return false
}
