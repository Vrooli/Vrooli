package pipeline

import (
	"context"
	"path/filepath"
	"scenario-to-desktop-api/shared/errors"

	sharedpath "scenario-to-desktop-api/shared/path"
)

// ResolveDeploymentStage is the mandatory admission gate for bundled desktop
// delivery. It creates a target-specific plan before bundle packaging starts.
type ResolveDeploymentStage struct {
	timeProvider TimeProvider
	scenarioRoot string
}
type ResolveDeploymentStageOption func(*ResolveDeploymentStage)

func WithResolveDeploymentScenarioRoot(root string) ResolveDeploymentStageOption {
	return func(s *ResolveDeploymentStage) { s.scenarioRoot = root }
}

func NewResolveDeploymentStage(opts ...ResolveDeploymentStageOption) *ResolveDeploymentStage {
	s := &ResolveDeploymentStage{timeProvider: NewRealTimeProvider()}
	for _, opt := range opts {
		opt(s)
	}
	if s.scenarioRoot == "" {
		s.scenarioRoot = sharedpath.DetectScenariosRoot()
		if s.scenarioRoot == "" {
			s.scenarioRoot = "scenarios"
		}
	}
	return s
}
func (s *ResolveDeploymentStage) Name() string           { return StageResolveDeployment }
func (s *ResolveDeploymentStage) Dependencies() []string { return nil }
func (s *ResolveDeploymentStage) CanSkip(input *StageInput) bool {
	return input.Config.GetDeploymentMode() != DeploymentModeBundled
}

func (s *ResolveDeploymentStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)
	if s.CanSkip(input) {
		if err := validateThinClientProxyURL(input.Config.ProxyURL); err != nil {
			failStage(result, s.timeProvider, errors.New(errors.CodeValidation, err.Error()).
				WithRecovery(errors.RecoveryFixInput, "Use the Vrooli scenario proxy URL, not a Vault listener or /v1/ API endpoint."))
			return result
		}
		skipStage(result, s.timeProvider, "Skipping resource deployment resolution: deployment mode is proxy")
		return result
	}
	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}
	scenarioPath := input.ScenarioPath
	if scenarioPath == "" {
		scenarioPath = filepath.Join(s.scenarioRoot, input.Config.ScenarioName)
	}
	plan, err := resolveResourceDeploymentPlan(scenarioPath, input.Config.ResourceArtifactRoot, input.Config.Platforms)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrBundlePackagingFailed(err, scenarioPath))
		return result
	}
	input.ResourceDeploymentPlan = plan
	completeStage(result, s.timeProvider, plan)
	appendInfo(result, "Resolved %d resource deployment selections", len(plan.Resources))
	return result
}
