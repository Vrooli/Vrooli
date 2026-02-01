package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/shared/errors"
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
		failStage(result, s.timeProvider, errors.ErrBuildServiceNotConfigured())
		return result
	}

	desktopPath := input.DesktopPath
	if desktopPath == "" {
		failStage(result, s.timeProvider, errors.ErrBuildDesktopPathMissing())
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
	buildStatus, waitErr := s.waitForBuild(ctx, buildID)
	if waitErr != nil {
		// Handle wait error based on its kind
		if waitErr.Kind == WaitErrorStore {
			failStage(result, s.timeProvider, errors.ErrBuildStoreNotConfigured())
		} else if waitErr.Kind == WaitErrorTimeout {
			failStage(result, s.timeProvider, errors.ErrBuildTimedOut(buildID, DefaultBuildTimeout.String()))
		} else if waitErr.Kind == WaitErrorCancelled {
			failStage(result, s.timeProvider, errors.New(errors.CodePipelineCancelled, "build cancelled").InDomain("build"))
		} else {
			failStage(result, s.timeProvider, errors.ErrBuildStartFailed(waitErr, strings.Join(platforms, ",")))
		}
		return result
	}

	// Check build result
	switch buildStatus.Status {
	case BuildStatusReady:
		result.Logs = append(result.Logs, "All platforms built successfully")
	case BuildStatusPartial:
		result.Logs = append(result.Logs, "Build completed with some platform failures")
	case BuildStatusFailed:
		lastOutput := ""
		if len(buildStatus.ErrorLog) > 0 {
			lastOutput = buildStatus.ErrorLog[len(buildStatus.ErrorLog)-1]
		}
		// Determine which platform failed
		failedPlatform := ""
		for platform, platResult := range buildStatus.PlatformResults {
			if platResult.Status == BuildStatusFailed {
				failedPlatform = platform
				break
			}
		}
		failStage(result, s.timeProvider, errors.ErrBuildPlatformFailed(
			fmt.Errorf("build failed: %s", lastOutput),
			failedPlatform,
			lastOutput,
		))
		result.Details = buildStatus
		return result
	}

	// Update input for next stage
	input.BuildResult = buildStatus

	completeStage(result, s.timeProvider, buildStatus)

	// Log platform results
	for platform, platResult := range buildStatus.PlatformResults {
		if platResult.Status == BuildStatusReady {
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: built (%s)", platform, platResult.Artifact))
		} else if platResult.Status == BuildStatusSkipped {
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: skipped (%s)", platform, platResult.SkipReason))
		} else {
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: %s", platform, platResult.Status))
		}
	}

	return result
}

// waitForBuild polls for build completion using the generic Poller.
func (s *BuildStage) waitForBuild(ctx context.Context, buildID string) (*build.Status, *WaitError) {
	if s.store == nil {
		return nil, &WaitError{
			Kind:       WaitErrorStore,
			EntityType: "build",
			EntityID:   buildID,
		}
	}

	poller := &Poller[*build.Status]{
		Config: PollerConfig{
			EntityType:   "Build",
			Timeout:      DefaultBuildTimeout,
			PollInterval: DefaultBuildPollInterval,
			LogInterval:  10, // Log every ~20 seconds
		},
		GetStatus: s.store.Get,
		IsComplete: func(status *build.Status) bool {
			switch status.Status {
			case BuildStatusReady, BuildStatusPartial, BuildStatusFailed:
				return true
			}
			return false
		},
	}

	return poller.Wait(ctx, buildID)
}
