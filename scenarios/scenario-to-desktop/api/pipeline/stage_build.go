package pipeline

import (
	"context"
	"fmt"
	"time"

	"scenario-to-desktop-api/build"
)

// BuildStage implements the Electron build stage of the pipeline.
type BuildStage struct {
	service      build.Service
	store        build.Store
	timeProvider TimeProvider
}

// BuildStageOption configures a BuildStage.
type BuildStageOption func(*BuildStage)

// WithBuildService sets the build service.
func WithBuildService(svc build.Service) BuildStageOption {
	return func(s *BuildStage) {
		s.service = svc
	}
}

// WithBuildStore sets the build store for status polling.
func WithBuildStore(store build.Store) BuildStageOption {
	return func(s *BuildStage) {
		s.store = store
	}
}

// WithBuildTimeProvider sets the time provider.
func WithBuildTimeProvider(tp TimeProvider) BuildStageOption {
	return func(s *BuildStage) {
		s.timeProvider = tp
	}
}

// NewBuildStage creates a new build stage.
func NewBuildStage(opts ...BuildStageOption) *BuildStage {
	s := &BuildStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *BuildStage) Name() string {
	return StageBuild
}

// Dependencies returns stages that must complete before this one.
func (s *BuildStage) Dependencies() []string {
	return []string{StageGenerate}
}

// CanSkip returns whether this stage can be skipped.
// Build is never skipped - it's always required.
func (s *BuildStage) CanSkip(input *StageInput) bool {
	return false
}

// Execute runs the Electron build stage.
func (s *BuildStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	if s.service == nil {
		failStage(result, s.timeProvider, "build service not configured")
		return result
	}

	desktopPath := input.DesktopPath
	if desktopPath == "" {
		failStage(result, s.timeProvider, "desktop path not available from generation stage")
		return result
	}

	scenarioName := input.Config.ScenarioName
	platforms := input.Config.Platforms

	result.Logs = append(result.Logs,
		fmt.Sprintf("Building for scenario: %s", scenarioName),
		fmt.Sprintf("Desktop path: %s", desktopPath),
		fmt.Sprintf("Platforms: %v", platforms),
	)

	// Generate build ID
	buildID := fmt.Sprintf("build-%s-%d", scenarioName, time.Now().UnixMilli())

	// Start the async build
	go s.service.PerformScenarioDesktopBuild(
		buildID,
		scenarioName,
		desktopPath,
		platforms,
		input.Config.Clean,
	)

	// Wait for build to complete
	buildStatus, err := s.waitForBuild(ctx, buildID)
	if err != nil {
		failStage(result, s.timeProvider, err.Error())
		return result
	}

	// Check build result
	switch buildStatus.Status {
	case "ready":
		result.Logs = append(result.Logs, "All platforms built successfully")
	case "partial":
		result.Logs = append(result.Logs, "Build completed with some platform failures")
	case "failed":
		errMsg := "build failed"
		if len(buildStatus.ErrorLog) > 0 {
			errMsg = buildStatus.ErrorLog[len(buildStatus.ErrorLog)-1]
		}
		failStage(result, s.timeProvider, errMsg)
		result.Details = buildStatus
		return result
	}

	// Update input for next stage
	input.BuildResult = buildStatus

	completeStage(result, s.timeProvider, buildStatus)

	// Log platform results
	for platform, platResult := range buildStatus.PlatformResults {
		if platResult.Status == "ready" {
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: built (%s)", platform, platResult.Artifact))
		} else if platResult.Status == "skipped" {
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: skipped (%s)", platform, platResult.SkipReason))
		} else {
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: %s", platform, platResult.Status))
		}
	}

	return result
}

// waitForBuild polls for build completion.
func (s *BuildStage) waitForBuild(ctx context.Context, buildID string) (*build.Status, error) {
	if s.store == nil {
		return nil, fmt.Errorf("build store not configured for status polling")
	}

	// Poll with timeout (builds can take a long time)
	timeout := time.After(30 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("build cancelled")
		case <-timeout:
			return nil, fmt.Errorf("build timed out after 30 minutes")
		case <-ticker.C:
			status, ok := s.store.Get(buildID)
			if !ok {
				// Build not yet registered, keep waiting
				continue
			}

			switch status.Status {
			case "ready", "partial":
				return status, nil
			case "failed":
				return status, nil // Let caller handle the failure
			}
			// Still building, continue polling
		}
	}
}
