package pipeline

import (
	"context"
	"fmt"

	"scenario-to-desktop-api/deploy"
	sharedenv "scenario-to-desktop-api/shared/env"
	"scenario-to-desktop-api/shared/errors"
)

// LPBSClientFactory creates an LPBSClient for the given scenario and token.
type LPBSClientFactory func(scenarioName, serviceToken string) *deploy.LPBSClient

// DeployStage implements the deploy stage of the pipeline.
// It uploads built artifacts to a remote LPBS instance via the local LPBS proxy.
type DeployStage struct {
	clientFactory LPBSClientFactory
	targetRepo    *deploy.TargetRepository
	timeProvider  TimeProvider
}

// DeployStageOption configures a DeployStage.
type DeployStageOption func(*DeployStage)

// WithDeployClientFactory sets the LPBS client factory.
func WithDeployClientFactory(f LPBSClientFactory) DeployStageOption {
	return func(s *DeployStage) {
		s.clientFactory = f
	}
}

// WithDeployTargetRepo sets the deploy target repository.
func WithDeployTargetRepo(repo *deploy.TargetRepository) DeployStageOption {
	return func(s *DeployStage) {
		s.targetRepo = repo
	}
}

// WithDeployTimeProvider sets the time provider.
func WithDeployTimeProvider(tp TimeProvider) DeployStageOption {
	return func(s *DeployStage) {
		s.timeProvider = tp
	}
}

// NewDeployStage creates a new deploy stage.
func NewDeployStage(opts ...DeployStageOption) *DeployStage {
	s := &DeployStage{
		clientFactory: func(scenarioName, serviceToken string) *deploy.LPBSClient {
			return deploy.NewLPBSClient(scenarioName, serviceToken)
		},
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *DeployStage) Name() string {
	return StageDeploy
}

// Dependencies returns stages that must complete before this one.
func (s *DeployStage) Dependencies() []string {
	return []string{StageSmokeTest}
}

// CanSkip returns whether this stage can be skipped.
func (s *DeployStage) CanSkip(input *StageInput) bool {
	return ShouldSkipDeploy(input.Config)
}

// Execute runs the deploy stage.
func (s *DeployStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	cfg := input.Config.DeployConfig
	if cfg == nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeValidation, "deploy config not set").
			WithRecovery(errors.RecoveryRetry, "Provide --deploy-target or --deploy-to flags").
			InDomain("deploy"))
		return result
	}

	// Resolve deploy target (saved vs inline)
	scenarioName, remoteProfile, err := s.resolveTarget(cfg)
	if err != nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeValidation, err.Error()).
			WithRecovery(errors.RecoveryRetry, "Check deploy target configuration").
			InDomain("deploy"))
		return result
	}
	appendInfo(result, "Deploy target: scenario=%s, profile=%s, app=%s", scenarioName, remoteProfile, cfg.AppKey)

	// Get service token
	serviceToken := sharedenv.ResolveSecret("LPBS_SERVICE_SECRET")
	if serviceToken == "" {
		failStage(result, s.timeProvider, errors.New(errors.CodeUnauthorized, "LPBS_SERVICE_SECRET is not set (checked env and .vrooli/secrets.json)").
			WithRecovery(errors.RecoveryProvideCredentials, "Set LPBS_SERVICE_SECRET to enable service-to-service auth via scenario-to-cloud secrets command").
			WithManualSteps([]string{
				"Set LPBS_SERVICE_SECRET using scenario-to-cloud secrets set ... --targets scenario,deployment",
				"Use the same secret configured in the LPBS instance and this runtime",
			}).
			InDomain("deploy"))
		return result
	}

	// Create LPBS client
	client := s.clientFactory(scenarioName, serviceToken)

	// Validate remote profile session
	appendInfo(result, "Testing remote profile %q session...", remoteProfile)
	if err := client.TestRemoteProfile(ctx, remoteProfile); err != nil {
		failStage(result, s.timeProvider, errors.ErrDeployFailed(
			fmt.Errorf("remote profile test failed: %w", err), remoteProfile))
		return result
	}
	appendInfo(result, "Remote profile %q session is active", remoteProfile)

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	// Derive update URL if not provided
	updateURL := cfg.UpdateURL
	if updateURL == "" {
		derived, err := client.DeriveUpdateURL(ctx, remoteProfile, cfg.AppKey)
		if err != nil {
			appendWarn(result, "Could not derive update URL: %v", err)
		} else {
			updateURL = derived
			appendInfo(result, "Derived update URL: %s", updateURL)
		}
	}

	// Collect artifacts from build result
	artifacts := collectArtifacts(input)
	if len(artifacts) == 0 {
		failStage(result, s.timeProvider, errors.New(errors.CodeArtifactNotFound, "no built artifacts available for deployment").
			WithRecovery(errors.RecoveryRetry, "Ensure build stage produces artifacts").
			WithManualSteps([]string{
				"Check if the build stage completed successfully",
				"Verify build produced artifacts for at least one platform",
			}).
			InDomain("deploy"))
		return result
	}
	appendInfo(result, "Found %d artifact(s) to deploy", len(artifacts))

	// Upload each artifact
	var uploadResults []DeployArtifactResult
	releaseVersion := ""
	if input.Config != nil {
		releaseVersion = input.Config.Version
	}
	gitCommitHash := ""
	if input.Provenance != nil {
		gitCommitHash = input.Provenance.GitCommitHash
	}

	for platform, artifactPath := range artifacts {
		if checkCancellation(ctx, result, s.timeProvider) {
			return result
		}

		appendInfo(result, "Uploading %s artifact: %s", platform, artifactPath)
		uploadResult, err := client.UploadArtifact(ctx, &deploy.UploadRequest{
			RemoteProfile:  remoteProfile,
			AppKey:         cfg.AppKey,
			Platform:       platform,
			FilePath:       artifactPath,
			ReleaseVersion: releaseVersion,
			GitCommitHash:  gitCommitHash,
		})
		if err != nil {
			failStage(result, s.timeProvider, errors.ErrDeployFailed(
				fmt.Errorf("upload %s artifact: %w", platform, err), remoteProfile))
			return result
		}
		appendInfo(result, "Uploaded %s artifact: artifact_id=%d", platform, uploadResult.ArtifactID)
		uploadResults = append(uploadResults, DeployArtifactResult{
			ArtifactID: uploadResult.ArtifactID,
			Platform:   uploadResult.Platform,
		})
	}

	// Build deploy result
	deployResult := &DeployResult{
		Artifacts: uploadResults,
		UpdateURL: updateURL,
	}

	// Update input for subsequent stages
	input.DeployResult = deployResult

	completeStage(result, s.timeProvider, deployResult)
	return result
}

// resolveTarget determines the LPBS scenario name and remote profile from config.
// If TargetName is set, looks up the saved target. Otherwise uses inline config.
func (s *DeployStage) resolveTarget(cfg *DeployConfig) (scenarioName, remoteProfile string, err error) {
	if cfg.TargetName != "" {
		if s.targetRepo == nil {
			return "", "", fmt.Errorf("deploy target %q requested but target repository not configured", cfg.TargetName)
		}
		target, err := s.targetRepo.Get(cfg.TargetName)
		if err != nil {
			return "", "", fmt.Errorf("resolve deploy target %q: %w", cfg.TargetName, err)
		}
		return target.ScenarioName, target.RemoteProfile, nil
	}

	// Inline config
	if cfg.ScenarioName == "" {
		return "", "", fmt.Errorf("either target_name or scenario_name must be provided")
	}
	if cfg.RemoteProfile == "" {
		return "", "", fmt.Errorf("remote_profile is required when using inline deploy config")
	}
	return cfg.ScenarioName, cfg.RemoteProfile, nil
}
