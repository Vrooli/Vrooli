// DOC: docs/reference/smoke-test-pipeline.md
package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/shared/errors"
	"scenario-to-desktop-api/smoketest"
)

// SmokeTestStage implements the smoke test stage of the pipeline.
// See docs/reference/smoke-test-pipeline.md for detailed execution flow.
type SmokeTestStage struct {
	service      smoketest.Service
	store        smoketest.Store
	timeProvider TimeProvider
	matrix       *validationmatrix.Service
	discoverer   interface {
		Discover(context.Context) ([]deliveryramp.Target, error)
	}
}

// SmokeTestStageOption configures a SmokeTestStage.
type SmokeTestStageOption func(*SmokeTestStage)

// WithSmokeTestService sets the smoke test service.
func WithSmokeTestService(svc smoketest.Service) SmokeTestStageOption {
	return func(s *SmokeTestStage) {
		s.service = svc
	}
}

// WithSmokeTestStore sets the smoke test store for status polling.
func WithSmokeTestStore(store smoketest.Store) SmokeTestStageOption {
	return func(s *SmokeTestStage) {
		s.store = store
	}
}

// WithSmokeTestTimeProvider sets the time provider.
func WithSmokeTestTimeProvider(tp TimeProvider) SmokeTestStageOption {
	return func(s *SmokeTestStage) {
		s.timeProvider = tp
	}
}

// WithValidationMatrix makes the validation stage matrix-driven while
// retaining the legacy smoke-test path when no matrix service is configured.
func WithValidationMatrix(service *validationmatrix.Service) SmokeTestStageOption {
	return func(s *SmokeTestStage) { s.matrix = service }
}

// WithValidationTargetDiscovery supplies the normalized local/Bridge target
// inventory used to build one cell for every produced platform.
func WithValidationTargetDiscovery(discoverer interface {
	Discover(context.Context) ([]deliveryramp.Target, error)
},
) SmokeTestStageOption {
	return func(s *SmokeTestStage) { s.discoverer = discoverer }
}

// NewSmokeTestStage creates a new smoke test stage.
func NewSmokeTestStage(opts ...SmokeTestStageOption) *SmokeTestStage {
	s := &SmokeTestStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *SmokeTestStage) Name() string {
	return StageSmokeTest
}

// Dependencies returns stages that must complete before this one.
func (s *SmokeTestStage) Dependencies() []string {
	return []string{StageBuild}
}

// CanSkip returns whether this stage can be skipped.
func (s *SmokeTestStage) CanSkip(input *StageInput) bool {
	return ShouldSkipSmokeTest(input.Config)
}

// Execute runs the smoke test stage.
func (s *SmokeTestStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if s.CanSkip(input) {
		skipStage(result, s.timeProvider, "Skipping smoke test: explicitly skipped via config")
		return result
	}

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	if s.service == nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeServiceStartError, "smoke test service not configured").
			WithRecovery(errors.RecoveryContactSupport, "Server configuration issue - contact support").
			WithManualSteps([]string{
				"Check server startup logs for initialization errors",
				"Verify the smoke test service is properly configured",
				"Contact support if the issue persists",
			}).
			InDomain("smoketest"))
		return result
	}

	if input.BuildResult == nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeDependencyError, "build result not available from previous stage").
			WithRecovery(errors.RecoveryRetry, "Ensure build stage completes successfully first").
			WithManualSteps([]string{
				"Check if the build stage completed successfully",
				"Review build stage logs for errors",
				"Restart the pipeline from the build stage",
			}).
			InDomain("smoketest"))
		return result
	}
	if s.matrix != nil {
		return s.executeValidationMatrix(ctx, input, result)
	}

	// Find an artifact to test (prefer current platform)
	currentPlatform := s.service.CurrentPlatform()
	artifactPath := ""

	// First try current platform using artifact helper
	artifactPath = FindArtifactForPlatform(input.BuildResult.PlatformResults, currentPlatform)

	// If no current platform artifact, try any available
	if artifactPath == "" {
		currentPlatform, artifactPath = FindFirstReadyArtifact(input.BuildResult.PlatformResults)
	}

	if artifactPath == "" {
		failStage(result, s.timeProvider, errors.New(errors.CodeArtifactNotFound, "no built artifacts available for smoke testing").
			WithRecovery(errors.RecoveryRetry, "Ensure build stage produces artifacts").
			WithManualSteps([]string{
				"Check if the build stage completed successfully",
				"Verify build produced artifacts for at least one platform",
				"Review build logs for errors",
			}).
			InDomain("smoketest"))
		return result
	}

	scenarioName := input.Config.ScenarioName
	result.Logs = append(result.Logs,
		fmt.Sprintf("Running smoke test for: %s", scenarioName),
		fmt.Sprintf("Platform: %s", currentPlatform),
		fmt.Sprintf("Artifact: %s", artifactPath),
	)

	// Generate smoke test ID
	smokeTestID := fmt.Sprintf("smoke-%s-%d", scenarioName, time.Now().UnixMilli())

	// Initialize smoke test status in store BEFORE launching async goroutine.
	// The smoke test service checks if the status exists and exits immediately if not,
	// so we must create it here first.
	s.initSmokeTestStore(smokeTestID, scenarioName, currentPlatform, artifactPath, input.Config.GetDeploymentMode(), input.Config.ProxyURL, input.PipelineID)

	// Start the async smoke test
	go smoketest.RunSmokeTestRequest(ctx, s.service, smoketest.SmokeTestRequest{
		SmokeTestID: smokeTestID, ScenarioName: scenarioName, ArtifactPath: artifactPath,
		Platform: currentPlatform, DeploymentMode: input.Config.GetDeploymentMode(),
		ProxyURL: input.Config.ProxyURL, PipelineID: input.PipelineID,
		JourneyID: input.Config.JourneyID,
	})

	// Wait for smoke test to complete
	smokeStatus, waitErr := s.waitForSmokeTest(ctx, smokeTestID)
	if waitErr != nil {
		failStage(result, s.timeProvider, s.smokeTestWaitError(waitErr, smokeTestID))
		return result
	}

	// Check smoke test result
	if failed := s.handleSmokeTestResult(result, smokeStatus, smokeTestID, currentPlatform); failed {
		return result
	}

	// Update input with result
	input.SmokeTestResult = smokeStatus

	completeStage(result, s.timeProvider, smokeStatus)

	// Add logs from the smoke test
	if len(smokeStatus.Logs) > 0 {
		result.Logs = append(result.Logs, "Smoke test logs:")
		for _, log := range smokeStatus.Logs {
			result.Logs = append(result.Logs, "  "+log)
		}
	}

	return result
}

func (s *SmokeTestStage) executeValidationMatrix(ctx context.Context, input *StageInput, result *StageResult) *StageResult {
	selection, err := s.buildValidationSelection(ctx, input)
	if err != nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeDependencyError, "build validation matrix selection: "+err.Error()).InDomain("validationmatrix"))
		return result
	}
	run, err := s.matrix.Create(selection)
	if err != nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeDependencyError, "create validation matrix: "+err.Error()).InDomain("validationmatrix"))
		return result
	}
	if _, err := s.matrix.Start(run.RunID); err != nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeServiceStartError, "start validation matrix: "+err.Error()).InDomain("validationmatrix"))
		return result
	}
	completed, err := s.matrix.Wait(ctx, run.RunID)
	if err != nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeTimeout, "wait for validation matrix: "+err.Error()).InDomain("validationmatrix"))
		return result
	}
	input.ValidationMatrixResult = completed
	result.Details = completed
	result.Logs = append(result.Logs, fmt.Sprintf("Validation matrix %s completed with state %s", completed.RunID, completed.State))
	if completed.State != validationmatrix.RunCompleted || completed.Gate == nil || !completed.Gate.GetPassed() {
		failStage(result, s.timeProvider, errors.New(errors.CodeValidation, "validation matrix release gate did not pass").InDomain("validationmatrix"))
		return result
	}
	completeStage(result, s.timeProvider, completed)
	return result
}

func (s *SmokeTestStage) buildValidationSelection(ctx context.Context, input *StageInput) (validationmatrix.MatrixSelection, error) {
	if input == nil || input.BuildResult == nil {
		return validationmatrix.MatrixSelection{}, fmt.Errorf("build result is required")
	}
	selection := validationmatrix.MatrixSelection{ScenarioName: input.Config.ScenarioName, DeploymentMode: input.Config.GetDeploymentMode(), Journeys: []validationmatrix.JourneySelection{{JourneyID: input.Config.JourneyID, DisplayName: "Desktop validation", ExecutionMode: "platform", Required: true}}, EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL}, MaxConcurrency: 1, Command: "scenario-to-desktop validate-artifact", ArtifactPaths: map[string]string{}, ArtifactDigests: map[string]string{}}
	if strings.TrimSpace(selection.Journeys[0].JourneyID) == "" {
		selection.Journeys[0].JourneyID = "desktop-smoke"
	}
	if s.discoverer == nil {
		return validationmatrix.MatrixSelection{}, fmt.Errorf("validation target discovery is unavailable")
	}
	targets, discoveryErr := s.discoverer.Discover(ctx)
	localOS := runtime.GOOS
	for platform, artifact := range input.BuildResult.PlatformResults {
		if artifact == nil || artifact.Status != BuildStatusReady || strings.TrimSpace(artifact.Artifact) == "" {
			continue
		}
		data, readErr := os.ReadFile(artifact.Artifact)
		if readErr != nil {
			return validationmatrix.MatrixSelection{}, fmt.Errorf("read %s artifact: %w", platform, readErr)
		}
		sum := sha256.Sum256(data)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		kind := validationmatrix.TargetBridge
		descriptor := &domainv1.ValidationTargetDescriptor{TargetId: "bridge-unavailable-" + platform, DisplayName: platform + " target", Available: false}
		if normalizeBuildPlatform(platform) == localOS {
			kind = validationmatrix.TargetLocal
			descriptor = &domainv1.ValidationTargetDescriptor{TargetId: "local-" + platform, DisplayName: "Local host", Available: true}
		} else {
			for _, target := range targets {
				if target.OS == normalizeBuildPlatform(platform) && target.Available {
					descriptor = &domainv1.ValidationTargetDescriptor{TargetId: target.ID, DisplayName: target.Label, Available: true, Reason: stringPtr(target.Reason)}
					break
				}
			}
			if !descriptor.Available {
				reason := firstNonEmptyTargetReason(targets, platform)
				if discoveryErr != nil {
					reason = "bridge node discovery failed: " + discoveryErr.Error() + "; start or restore vrooli-bridge, then probe again"
				}
				descriptor.Reason = stringPtr("no capable target for " + platform + "; " + reason)
			}
		}
		selection.Targets = append(selection.Targets, validationmatrix.TargetSelection{Kind: kind, Descriptor: descriptor})
		selection.ArtifactPaths[descriptor.GetTargetId()] = artifact.Artifact
		selection.ArtifactDigests[descriptor.GetTargetId()] = digest
		if selection.ArtifactPath == "" {
			selection.ArtifactPath, selection.ArtifactDigest = artifact.Artifact, digest
		}
	}
	if len(selection.Targets) == 0 {
		return validationmatrix.MatrixSelection{}, fmt.Errorf("no ready build artifacts")
	}
	selection.CommandArgs = []string{"--scenario", selection.ScenarioName, "--journey", selection.Journeys[0].JourneyID, "--profile", "normal", "--evidence-output", "/tmp/evidence-bundle.tar.gz"}
	return selection, nil
}

func normalizeBuildPlatform(platform string) string {
	normalized := strings.ToLower(strings.TrimSpace(platform))
	switch {
	case normalized == "mac" || normalized == "macos" || normalized == "darwin" || strings.HasPrefix(normalized, "macos-") || strings.HasPrefix(normalized, "darwin-"):
		return "darwin"
	case normalized == "win" || normalized == "windows" || normalized == "win32" || strings.HasPrefix(normalized, "windows-") || strings.HasPrefix(normalized, "win32-"):
		return "windows"
	case normalized == "linux" || strings.HasPrefix(normalized, "linux-"):
		return "linux"
	default:
		return normalized
	}
}

func firstNonEmptyTargetReason(targets []deliveryramp.Target, platform string) string {
	for _, target := range targets {
		if normalizeBuildPlatform(target.OS) == normalizeBuildPlatform(platform) {
			if reason := actionableTargetReason(target); reason != "" {
				return reason
			}
		}
	}
	return "install the target platform toolchain and graphical session, then probe again"
}

func actionableTargetReason(target deliveryramp.Target) string {
	parts := make([]string, 0, 3)
	if reason := strings.TrimSpace(target.Reason); reason != "" {
		parts = append(parts, reason)
	}
	if missing := strings.TrimSpace(target.MissingCapability); missing != "" {
		parts = append(parts, "missing capability: "+missing)
	}
	if nextAction := strings.TrimSpace(target.NextAction); nextAction != "" {
		parts = append(parts, "next action: "+nextAction)
	}
	return strings.Join(parts, "; ")
}

// waitForSmokeTest polls for smoke test completion using the generic Poller.
func (s *SmokeTestStage) waitForSmokeTest(ctx context.Context, smokeTestID string) (*smoketest.Status, *WaitError) {
	if s.store == nil {
		return nil, &WaitError{
			Kind:       WaitErrorStore,
			EntityType: "smoke test",
			EntityID:   smokeTestID,
		}
	}

	poller := &Poller[*smoketest.Status]{
		Config: PollerConfig{
			EntityType:   "Smoke test",
			Timeout:      DefaultSmokeTestTimeout,
			PollInterval: DefaultSmokePollInterval,
			LogInterval:  10, // Log every ~20 seconds
		},
		GetStatus: s.store.Get,
		IsComplete: func(status *smoketest.Status) bool {
			switch status.Status {
			case SmokeTestStatusPassed, SmokeTestStatusFailed:
				return true
			}
			return false
		},
	}

	return poller.Wait(ctx, smokeTestID)
}

// initSmokeTestStore creates the initial smoke test status in the store.
func (s *SmokeTestStage) initSmokeTestStore(smokeTestID, scenarioName, platform, artifactPath, deploymentMode, proxyURL, pipelineID string) {
	if s.store == nil {
		return
	}
	now := time.Unix(s.timeProvider.Now(), 0)
	initialStatus := &smoketest.Status{
		SmokeTestID:    smokeTestID,
		ScenarioName:   scenarioName,
		Platform:       platform,
		Status:         "running",
		ArtifactPath:   artifactPath,
		StartedAt:      now,
		DeploymentMode: deploymentMode,
		ProxyURL:       proxyURL,
		PipelineID:     pipelineID,
		ScreenContentSource: func() string {
			if deploymentMode == DeploymentModeBundled {
				return "bundled_private"
			}
			if deploymentMode == DeploymentModeProxy {
				return "isolated_instance"
			}
			return "unknown"
		}(),
		Logs: []string{},
		RecordingConfig: &smoketest.ScreenRecordingConfig{
			Enabled:       true,
			DisplayWidth:  1920,
			DisplayHeight: 1080,
			FPS:           15,
		},
	}
	s.store.Save(initialStatus)
}

// smokeTestWaitError converts a WaitError into a domain error for the smoke test stage.
func (s *SmokeTestStage) smokeTestWaitError(waitErr *WaitError, smokeTestID string) *errors.DomainError {
	switch waitErr.Kind {
	case WaitErrorStore:
		return errors.New(errors.CodeServiceStartError, "Smoke test tracking service unavailable").
			WithRecovery(errors.RecoveryContactSupport, "Server configuration issue - contact support").
			WithManualSteps([]string{
				"Check server startup logs for initialization errors",
				"Verify the smoke test store is properly configured",
				"Contact support if the issue persists",
			}).
			InDomain("smoketest")
	case WaitErrorTimeout:
		return errors.ErrSmokeTestTimeout(
			DefaultSmokeTestTimeout.String(),
			map[string]string{"smoke_test_id": smokeTestID},
		)
	case WaitErrorCancelled:
		return errors.ErrSmokeTestCancelled()
	default:
		return errors.ErrSmokeTestExecutionFailed(
			waitErr,
			map[string]string{"smoke_test_id": smokeTestID},
		)
	}
}

// handleSmokeTestResult logs and checks the smoke test status. Returns true if the stage should fail.
func (s *SmokeTestStage) handleSmokeTestResult(result *StageResult, smokeStatus *smoketest.Status, smokeTestID, platform string) bool {
	switch smokeStatus.Status {
	case SmokeTestStatusPassed:
		result.Logs = append(result.Logs, "Smoke test passed")
		if smokeStatus.TelemetryUploaded {
			result.Logs = append(result.Logs, "Telemetry uploaded successfully")
		}
		return false
	case SmokeTestStatusFailed:
		context := map[string]string{
			"smoke_test_id": smokeTestID,
			"platform":      platform,
		}
		if smokeStatus.ErrorKind != nil {
			context["error_kind"] = smokeStatus.ErrorKind.String()
		}
		failStage(result, s.timeProvider, errors.ErrSmokeTestExecutionFailed(
			fmt.Errorf("%s", smokeStatus.Error),
			context,
		).WithRecovery(errors.RecoveryRetry, smokeStatus.SuggestedAction))
		result.Details = smokeStatus
		return true
	default:
		failStage(result, s.timeProvider, errors.New(errors.CodeSmokeTestFailed, fmt.Sprintf("unexpected smoke test status: %s", smokeStatus.Status)).
			WithDetail("smoke_test_id", smokeTestID).
			WithDetail("status", smokeStatus.Status).
			InDomain("smoketest"))
		result.Details = smokeStatus
		return true
	}
}
