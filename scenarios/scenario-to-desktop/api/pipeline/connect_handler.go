package pipeline

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/smoketest"

	"connectrpc.com/connect"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConnectService exposes the existing pipeline orchestration domain through the
// generated Connect contract. It deliberately delegates to Handler's
// orchestrator instead of routing Connect requests through HTTP handlers.
type ConnectService struct {
	pipelineconnect.UnimplementedPipelineServiceHandler
	handler *Handler
}

var _ pipelineconnect.PipelineServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService {
	return &ConnectService{handler: handler}
}

func (s *ConnectService) Run(ctx context.Context, req *connect.Request[pipelinev1.PipelineRunRequest]) (*connect.Response[pipelinev1.PipelineRunResponse], error) {
	if err := s.requireOrchestrator(); err != nil {
		return nil, err
	}
	config, err := configFromProto(req.Msg.GetConfig())
	if err != nil {
		return nil, err
	}
	status, err := s.handler.orchestrator.RunPipeline(ctx, config)
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	message := "Pipeline started successfully"
	return connect.NewResponse(&pipelinev1.PipelineRunResponse{
		PipelineId: status.PipelineID,
		Message:    &message,
	}), nil
}

func (s *ConnectService) Get(_ context.Context, req *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error) {
	if err := s.requireOrchestrator(); err != nil {
		return nil, err
	}
	status, ok := s.handler.orchestrator.GetStatus(req.Msg.GetPipelineId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pipeline %q was not found", req.Msg.GetPipelineId()))
	}
	return connect.NewResponse(statusToProto(status)), nil
}

// GetReleaseGate exposes the approval-gate view as a first-class contract.
// The gate state is derived from the authoritative pipeline status, but the
// separate RPC prevents clients from treating a generic status read as an
// approval decision and gives policy tooling a precise capability to govern.
func (s *ConnectService) GetReleaseGate(_ context.Context, req *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error) {
	if err := s.requireOrchestrator(); err != nil {
		return nil, err
	}
	status, ok := s.handler.orchestrator.GetStatus(req.Msg.GetPipelineId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pipeline %q was not found", req.Msg.GetPipelineId()))
	}
	return connect.NewResponse(statusToProto(status)), nil
}

func (s *ConnectService) Resume(ctx context.Context, req *connect.Request[pipelinev1.PipelineResumeRequest]) (*connect.Response[pipelinev1.PipelineResumeResponse], error) {
	if err := s.requireOrchestrator(); err != nil {
		return nil, err
	}
	var config *Config
	if req.Msg.Config != nil {
		var err error
		config, err = configFromProto(req.Msg.Config)
		if err != nil {
			return nil, err
		}
	}
	status, err := s.handler.orchestrator.ResumePipeline(ctx, req.Msg.GetPipelineId(), config)
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	message := fmt.Sprintf("Pipeline resumed from stage: %s", status.Config.ResumeFromStage)
	stage := stageNameToProto(status.Config.ResumeFromStage)
	return connect.NewResponse(&pipelinev1.PipelineResumeResponse{
		PipelineId:       status.PipelineID,
		ParentPipelineId: req.Msg.GetPipelineId(),
		ResumeFromStage:  stage,
		Message:          &message,
	}), nil
}

func (s *ConnectService) Cancel(_ context.Context, req *connect.Request[pipelinev1.PipelineCancelRequest]) (*connect.Response[pipelinev1.PipelineCancelResponse], error) {
	if err := s.requireOrchestrator(); err != nil {
		return nil, err
	}
	if s.handler.orchestrator.CancelPipeline(req.Msg.GetPipelineId()) {
		message := "Pipeline cancellation requested"
		return connect.NewResponse(&pipelinev1.PipelineCancelResponse{Status: "cancelling", Message: &message}), nil
	}
	status, ok := s.handler.orchestrator.GetStatus(req.Msg.GetPipelineId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pipeline %q was not found", req.Msg.GetPipelineId()))
	}
	if status.IsComplete() {
		message := "Pipeline has already completed"
		return connect.NewResponse(&pipelinev1.PipelineCancelResponse{Status: status.Status, Message: &message}), nil
	}
	return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pipeline %q cannot be cancelled", req.Msg.GetPipelineId()))
}

func (s *ConnectService) List(_ context.Context, req *connect.Request[pipelinev1.PipelineListRequest]) (*connect.Response[pipelinev1.PipelineListResponse], error) {
	if err := s.requireOrchestrator(); err != nil {
		return nil, err
	}
	items := make([]*pipelinev1.PipelineListItem, 0)
	for _, status := range s.handler.orchestrator.ListPipelines() {
		if scenario := req.Msg.GetScenarioName(); scenario != "" && status.ScenarioName != scenario {
			continue
		}
		item := &pipelinev1.PipelineListItem{
			PipelineId:      status.PipelineID,
			ScenarioName:    status.ScenarioName,
			Status:          stageStatusToProto(status.Status),
			ProgressPercent: int32(status.ProgressPercent),
			CreatedAt:       unixTimestamp(status.StartedAt),
			CanResume:       status.CanResume(),
		}
		if status.CurrentStage != "" {
			stage := stageNameToProto(status.CurrentStage)
			item.CurrentStage = &stage
		}
		if status.CompletedAt != 0 {
			item.CompletedAt = unixTimestamp(status.CompletedAt)
		}
		items = append(items, item)
	}
	total := int32(len(items))
	return connect.NewResponse(&pipelinev1.PipelineListResponse{Pipelines: items, Total: &total}), nil
}

func (s *ConnectService) GetActive(ctx context.Context, req *connect.Request[pipelinev1.GetActivePipelineRequest]) (*connect.Response[pipelinev1.ActivePipelineResponse], error) {
	if err := s.requireManager(); err != nil {
		return nil, err
	}
	if !req.Msg.GetAutoCreate() {
		status, ok := s.handler.manager.GetActivePipelineStatus(req.Msg.GetScenarioName())
		if !ok {
			return connect.NewResponse(&pipelinev1.ActivePipelineResponse{}), nil
		}
		return connect.NewResponse(&pipelinev1.ActivePipelineResponse{Pipeline: statusToProto(status)}), nil
	}
	status, created, err := s.handler.manager.GetOrCreateActivePipeline(ctx, req.Msg.GetScenarioName(), nil)
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	return connect.NewResponse(&pipelinev1.ActivePipelineResponse{Pipeline: statusToProto(status), Created: created}), nil
}

func (s *ConnectService) CreateActive(ctx context.Context, req *connect.Request[pipelinev1.CreatePipelineRequest]) (*connect.Response[pipelinev1.CreatePipelineResponse], error) {
	if err := s.requireManager(); err != nil {
		return nil, err
	}
	config, err := scenarioConfigFromProto(req.Msg.GetConfig(), req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	status, archivedID, err := s.handler.manager.CreateNewPipeline(ctx, req.Msg.GetScenarioName(), config)
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	response := &pipelinev1.CreatePipelineResponse{Pipeline: statusToProto(status)}
	if archivedID != "" {
		response.ArchivedPipelineId = stringPtr(archivedID)
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) ResetActive(_ context.Context, req *connect.Request[pipelinev1.ScenarioPipelineRequest]) (*connect.Response[pipelinev1.ResetPipelineResponse], error) {
	if err := s.requireManager(); err != nil {
		return nil, err
	}
	archivedID, err := s.handler.manager.ResetActivePipeline(req.Msg.GetScenarioName())
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	response := &pipelinev1.ResetPipelineResponse{Cleared: true}
	if archivedID != "" {
		response.ArchivedPipelineId = stringPtr(archivedID)
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) GetHistory(_ context.Context, req *connect.Request[pipelinev1.PipelineHistoryRequest]) (*connect.Response[pipelinev1.PipelineHistoryResponse], error) {
	if err := s.requireManager(); err != nil {
		return nil, err
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	statuses, total, err := s.handler.manager.GetPipelineHistory(req.Msg.GetScenarioName(), limit)
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	response := &pipelinev1.PipelineHistoryResponse{Total: int32(total)}
	for _, status := range statuses {
		response.Pipelines = append(response.Pipelines, statusToProto(status))
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) StartActive(ctx context.Context, req *connect.Request[pipelinev1.StartActivePipelineRequest]) (*connect.Response[pipelinev1.StartActivePipelineResponse], error) {
	if err := s.requireManager(); err != nil {
		return nil, err
	}
	config, err := scenarioConfigFromProto(req.Msg.GetConfigOverrides(), req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	status, err := s.handler.manager.StartActivePipeline(ctx, req.Msg.GetScenarioName(), config)
	if err != nil {
		return nil, pipelineConnectError(err)
	}
	message := "Active pipeline started"
	return connect.NewResponse(&pipelinev1.StartActivePipelineResponse{
		Pipeline: statusToProto(status),
		Message:  &message,
	}), nil
}

func (s *ConnectService) CleanBundle(_ context.Context, req *connect.Request[pipelinev1.BundleCleanRequest]) (*connect.Response[pipelinev1.BundleCleanResponse], error) {
	if s.handler == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pipeline handler is not configured"))
	}
	result, err := s.handler.cleanBundle(req.Msg.GetScenarioName(), FrameworkElectron, req.Msg.GetLocationMode(), req.Msg.GetPipelineId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &pipelinev1.BundleCleanResponse{ScenarioName: result.ScenarioName, LocationMode: result.LocationMode, Path: result.Path, Removed: result.Removed}
	if result.PipelineID != "" {
		response.PipelineId = stringPtr(result.PipelineID)
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) requireOrchestrator() error {
	if s.handler == nil || s.handler.orchestrator == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pipeline orchestrator is not configured"))
	}
	return nil
}

func (s *ConnectService) requireManager() error {
	if s.handler == nil || s.handler.manager == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pipeline manager is not configured"))
	}
	return nil
}

func scenarioConfigFromProto(value *pipelinev1.PipelineConfig, scenarioName string) (*Config, error) {
	if value == nil {
		return nil, nil
	}
	copy := proto.Clone(value).(*pipelinev1.PipelineConfig)
	copy.ScenarioName = scenarioName
	return configFromProto(copy)
}

func pipelineConnectError(err error) error {
	message := err.Error()
	switch {
	case strings.Contains(message, "required"), strings.Contains(message, "invalid"), strings.Contains(message, "unsupported"):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case strings.Contains(message, "not found"):
		return connect.NewError(connect.CodeNotFound, err)
	case strings.Contains(message, "not resumable"), strings.Contains(message, "already"):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func configFromProto(value *pipelinev1.PipelineConfig) (*Config, error) {
	if value == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config is required"))
	}
	if framework := value.GetFramework(); framework != sharedv1.Framework_FRAMEWORK_UNSPECIFIED && framework != sharedv1.Framework_FRAMEWORK_ELECTRON {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported framework %q: only %q is supported", framework, sharedv1.Framework_FRAMEWORK_ELECTRON))
	}
	for _, platform := range value.GetPlatforms() {
		if platformFromProto(platform) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported platform %q", platform))
		}
	}
	for _, stage := range value.GetStages() {
		if stageNameFromProto(stage) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported stage %q", stage))
		}
	}
	updateConfig, err := updateConfigFromProto(value.GetUpdateConfig(), resourcedeployment.ArtifactTrustMode(value.GetArtifactTrustMode()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	config := &Config{
		ScenarioName:         value.GetScenarioName(),
		Platforms:            platformsFromProto(value.GetPlatforms()),
		DeploymentMode:       deploymentModeFromProto(value.GetDeploymentMode()),
		Framework:            frameworkFromProto(value.GetFramework()),
		TemplateType:         templateTypeFromProto(value.GetTemplateType()),
		PreflightSecrets:     value.GetPreflightSecrets(),
		ResourceArtifactRoot: value.GetResourceArtifactRoot(),
		ToolArtifactRoot:     value.GetToolArtifactRoot(),
		ArtifactTrustMode:    resourcedeployment.ArtifactTrustMode(value.GetArtifactTrustMode()),
		LocationMode:         value.GetLocationMode(),
		Stages:               stagesFromProto(value.GetStages()),
		UpdateConfig:         updateConfig,
	}
	applyOptionalConfigFromProto(config, value)
	if config.ScenarioName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name is required"))
	}
	return config, nil
}

func applyOptionalConfigFromProto(config *Config, value *pipelinev1.PipelineConfig) {
	if value.SkipPreflight != nil {
		config.SkipPreflight = value.GetSkipPreflight()
	}
	if value.SkipSmokeTest != nil {
		config.SkipSmokeTest = value.GetSkipSmokeTest()
	}
	if value.StopOnFailure != nil {
		config.StopOnFailure = boolPtr(value.GetStopOnFailure())
	}
	if value.WebhookUrl != nil {
		config.WebhookURL = value.GetWebhookUrl()
	}
	if value.ProxyUrl != nil {
		config.ProxyURL = value.GetProxyUrl()
	}
	if value.BundleManifestPath != nil {
		config.BundleManifestPath = value.GetBundleManifestPath()
	}
	applyOptionalConfigExecutionFromProto(config, value)
}

func applyOptionalConfigExecutionFromProto(config *Config, value *pipelinev1.PipelineConfig) {
	if value.Clean != nil {
		config.Clean = value.GetClean()
	}
	if value.Sign != nil {
		config.Sign = value.GetSign()
	}
	if value.Publish != nil {
		config.Publish = value.GetPublish()
	}
	if value.Version != nil {
		config.Version = value.GetVersion()
	}
	if value.PreflightTimeoutSeconds != nil {
		config.PreflightTimeoutSeconds = int(value.GetPreflightTimeoutSeconds())
	}
	if value.StopAfterStage != nil {
		config.StopAfterStage = stageNameFromProto(value.GetStopAfterStage())
	}
	if value.ResumeFromStage != nil {
		config.ResumeFromStage = stageNameFromProto(value.GetResumeFromStage())
	}
	if value.ParentPipelineId != nil {
		config.ParentPipelineID = value.GetParentPipelineId()
	}
	if value.IdempotencyKey != nil {
		config.IdempotencyKey = value.GetIdempotencyKey()
	}
	if value.ArtifactTrustMode != nil {
		config.ArtifactTrustMode = resourcedeployment.ArtifactTrustMode(value.GetArtifactTrustMode())
	}
}

func updateConfigFromProto(value *sharedv1.UpdateConfig, trustMode resourcedeployment.ArtifactTrustMode) (*generation.UpdateConfig, error) {
	if value == nil {
		return nil, nil
	}
	provider := strings.TrimSpace(value.GetProvider())
	if provider == "" {
		provider = "generic"
	}
	if provider != "generic" && provider != "none" && provider != "github" {
		return nil, fmt.Errorf("unsupported update provider %q", provider)
	}
	config := &generation.UpdateConfig{Provider: provider, Channel: strings.TrimSpace(value.GetChannel()), AutoCheck: value.GetAutoCheck()}
	if value.Generic != nil {
		feedURL := strings.TrimSpace(value.GetGeneric().GetUrl())
		parsed, err := url.Parse(feedURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, fmt.Errorf("generic update URL must be an absolute http(s) URL")
		}
		if trustMode == resourcedeployment.ArtifactTrustProduction && parsed.Scheme != "https" {
			return nil, fmt.Errorf("production update URL must use HTTPS")
		}
		config.Generic = &generation.GenericUpdateConfig{URL: feedURL, ChannelPath: strings.TrimSpace(value.GetGeneric().GetChannelPath())}
	}
	if provider == "generic" && config.Generic == nil {
		return nil, fmt.Errorf("generic update provider requires update_config.generic.url")
	}
	if provider == "github" && value.Github != nil {
		config.GitHub = &generation.GitHubUpdateConfig{Owner: value.GetGithub().GetOwner(), Repo: value.GetGithub().GetRepo(), Private: value.GetGithub().GetPrivate()}
	}
	return config, nil
}

func statusToProto(status *Status) *pipelinev1.PipelineStatus {
	result := &pipelinev1.PipelineStatus{
		PipelineId:      status.PipelineID,
		ScenarioName:    status.ScenarioName,
		Status:          stageStatusToProto(status.Status),
		ProgressPercent: int32(status.ProgressPercent),
		Config:          configToProto(status.Config),
		StartedAt:       unixTimestamp(status.StartedAt),
		FinalArtifacts:  status.FinalArtifacts,
		Stages:          make(map[string]*pipelinev1.StageResult, len(status.Stages)),
	}
	if status.CurrentStage != "" {
		stage := stageNameToProto(status.CurrentStage)
		result.CurrentStage = &stage
	}
	if status.ProgressMessage != "" {
		result.ProgressMessage = stringPtr(status.ProgressMessage)
	}
	if status.CurrentState != "" {
		result.CurrentState = stringPtr(string(status.CurrentState))
	}
	if status.CompletedAt != 0 {
		result.CompletedAt = unixTimestamp(status.CompletedAt)
	}
	if status.Error != "" {
		result.Error = stringPtr(status.Error)
	}
	if status.StoppedAfterStage != "" {
		stage := stageNameToProto(status.StoppedAfterStage)
		result.StoppedAfterStage = &stage
	}
	if status.ParentPipelineID != "" {
		result.ParentPipelineId = stringPtr(status.ParentPipelineID)
	}
	if status.IdempotencyKey != "" {
		result.IdempotencyKey = stringPtr(status.IdempotencyKey)
	}
	for _, stage := range status.StageOrder {
		result.StageOrder = append(result.StageOrder, stageNameToProto(stage))
	}
	for name, stage := range status.Stages {
		if stage == nil {
			continue
		}
		result.Stages[name] = stageResultToProto(stage)
	}
	return result
}

func stageResultToProto(value *StageResult) *pipelinev1.StageResult {
	result := &pipelinev1.StageResult{
		Stage:     stageNameToProto(value.Stage),
		Status:    stageStatusToProto(value.Status),
		StartedAt: unixTimestamp(value.StartedAt),
		Logs:      append([]string(nil), value.Logs...),
	}
	if value.CompletedAt != 0 {
		result.CompletedAt = unixTimestamp(value.CompletedAt)
	}
	if value.Error != "" {
		result.Error = stringPtr(value.Error)
	}
	if details := stageDetailsToProto(value.Details); details != nil {
		result.Details = details
	}
	return result
}

func stageDetailsToProto(value any) *pipelinev1.StageDetails {
	switch details := value.(type) {
	case *ResourceDeploymentPlan:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_ResolveDeployment{ResolveDeployment: resourceDeploymentPlanToProto(details)}}
	case *bundle.PackageResult:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_Bundle{Bundle: bundleStageDetailsToProto(details)}}
	case *preflight.Response:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_Preflight{Preflight: preflight.ResponseToProto(details)}}
	case *generation.GenerateResponse:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_Generate{Generate: generateResponseToProto(details)}}
	case *build.Status:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_Build{Build: build.StatusToProto(details)}}
	case *smoketest.Status:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_SmokeTest{SmokeTest: smoketest.StatusToProto(details)}}
	case *DeployResult:
		return &pipelinev1.StageDetails{Kind: &pipelinev1.StageDetails_Deploy{Deploy: deployStageDetailsToProto(details)}}
	default:
		return nil
	}
}

func resourceDeploymentPlanToProto(value *ResourceDeploymentPlan) *pipelinev1.ResourceDeploymentPlan {
	if value == nil {
		return nil
	}
	result := &pipelinev1.ResourceDeploymentPlan{SchemaVersion: value.SchemaVersion, ArtifactTrustMode: string(value.ArtifactTrustMode), Promotable: value.Promotable}
	for _, resource := range value.Resources {
		item := &pipelinev1.ResourceDeploymentPlanItem{
			RequestedResource: resource.RequestedResource,
			Resource:          resource.Resource,
			Os:                resource.OS,
			Architecture:      resource.Architecture,
			Mode:              resource.Mode,
			Support:           resource.Support,
			Privilege:         resource.Privilege,
			Bundling:          resource.Bundling,
			Eligibility:       resource.Eligibility,
			EligibilityReason: resource.EligibilityReason,
			Requires:          append([]string(nil), resource.Requires...),
			Limitations:       append([]string(nil), resource.Limitations...),
			Evidence:          append([]string(nil), resource.Evidence...),
			Artifact:          optionalString(resource.Artifact),
		}
		if resource.SelectedFallback != nil {
			item.SelectedFallback = &pipelinev1.ResourceDeploymentFallback{Resource: resource.SelectedFallback.Resource, Reason: resource.SelectedFallback.Reason}
		}
		for _, file := range resource.Files {
			item.Files = append(item.Files, &pipelinev1.ResourceDeploymentArtifact{Name: file.Name, Sha256: file.SHA256})
		}
		if resource.Service != nil {
			service := resource.Service
			item.Service = &pipelinev1.ResourceDeploymentService{
				ProviderPolicy: &pipelinev1.ResourceProviderPolicy{
					DefaultMode:                string(service.ProviderPolicy.DefaultMode),
					AllowedModes:               stringSlice(service.ProviderPolicy.AllowedModes),
					SharedReuseRequiresConsent: service.ProviderPolicy.SharedReuseRequiresConsent,
					ExternalManagement:         service.ProviderPolicy.ExternalManagement,
					ExternalAccessCapabilities: stringSlice(service.ProviderPolicy.ExternalAccessCapabilities),
					TargetDefaults:             map[string]string{},
				},
				Artifact:    service.Artifact,
				Layout:      service.Layout,
				EntryPath:   optionalString(service.EntryPath),
				Version:     service.Version,
				Sha256:      service.SHA256,
				Arguments:   append([]string(nil), service.Arguments...),
				Environment: copyStringMap(service.Environment),
			}
			for target, mode := range service.ProviderPolicy.TargetDefaults {
				item.Service.ProviderPolicy.TargetDefaults[string(target)] = string(mode)
			}
			if service.Config != nil {
				item.Service.Config = &pipelinev1.ResourceDeploymentServiceConfig{Path: service.Config.Path, Content: service.Config.Content}
			}
			for _, port := range service.Ports {
				item.Service.Ports = append(item.Service.Ports, &pipelinev1.ResourceDeploymentServicePort{Name: port.Name, Host: int32(port.Host)})
			}
			for _, check := range service.HealthChecks {
				expected := make([]int32, 0, len(check.ExpectedStatus))
				for _, status := range check.ExpectedStatus {
					expected = append(expected, int32(status))
				}
				item.Service.HealthChecks = append(item.Service.HealthChecks, &pipelinev1.ResourceDeploymentHealthCheck{Type: check.Type, Target: check.Target, ExpectedStatus: expected, TimeoutSeconds: int32(check.TimeoutSeconds)})
			}
			for _, file := range service.Files {
				item.Service.Files = append(item.Service.Files, &pipelinev1.ResourceDeploymentArtifact{Name: file.Name, Sha256: file.SHA256})
			}
		}
		result.Resources = append(result.Resources, item)
	}
	for _, requirement := range value.HostRequirements {
		result.HostRequirements = append(result.HostRequirements, &pipelinev1.HostRequirementPlanItem{
			Name:         requirement.Name,
			Kind:         requirement.Kind,
			Os:           requirement.OS,
			Architecture: requirement.Architecture,
			Privilege:    requirement.Privilege,
			Bundling:     requirement.Bundling,
			Required:     requirement.Required,
			Verdict:      requirement.Verdict,
			Reason:       requirement.Reason,
			Artifact:     optionalString(requirement.Artifact),
			Provenance:   append([]string(nil), requirement.Provenance...),
		})
	}
	return result
}

func bundleStageDetailsToProto(value *bundle.PackageResult) *pipelinev1.BundleStageDetails {
	if value == nil {
		return nil
	}
	result := &pipelinev1.BundleStageDetails{BundleDir: value.BundleDir, ManifestPath: value.ManifestPath, RuntimeBinaries: copyStringMap(value.RuntimeBinaries), CopiedArtifacts: append([]string(nil), value.CopiedArtifacts...), TotalSizeBytes: value.TotalSizeBytes, TotalSizeHuman: value.TotalSizeHuman}
	if value.SizeWarning != nil {
		result.SizeWarning = &pipelinev1.BundleSizeWarning{Level: value.SizeWarning.Level, Message: value.SizeWarning.Message, TotalBytes: value.SizeWarning.TotalBytes, TotalHuman: value.SizeWarning.TotalHuman}
		for _, file := range value.SizeWarning.LargeFiles {
			result.SizeWarning.LargeFiles = append(result.SizeWarning.LargeFiles, &pipelinev1.BundleLargeFile{Path: file.Path, SizeBytes: file.SizeBytes, SizeHuman: file.SizeHuman})
		}
	}
	return result
}

func generateResponseToProto(value *generation.GenerateResponse) *pipelinev1.GenerateResponse {
	if value == nil {
		return nil
	}
	result := &pipelinev1.GenerateResponse{PipelineId: value.PipelineID, Status: value.Status, ScenarioName: value.ScenarioName, DesktopPath: optionalString(value.DesktopPath), InstallInstructions: optionalString(value.InstallInstructions), TestCommand: optionalString(value.TestCommand)}
	if metadata := value.DetectedMetadata; metadata != nil {
		result.DetectedMetadata = &sharedv1.ScenarioMetadata{Name: metadata.Name, DisplayName: optionalString(metadata.DisplayName), Description: optionalString(metadata.Description), Version: optionalString(metadata.Version), Author: optionalString(metadata.Author), License: optionalString(metadata.License), AppId: optionalString(metadata.AppID), HasUi: metadata.HasUI, UiDistPath: optionalString(metadata.UIDistPath), ScenarioPath: metadata.ScenarioPath, Category: optionalString(metadata.Category), Tags: append([]string(nil), metadata.Tags...), ServiceJsonPath: optionalString(metadata.ServiceJSONPath), PackageJsonPath: optionalString(metadata.PackageJSONPath)}
	}
	return result
}

func deployStageDetailsToProto(value *DeployResult) *pipelinev1.DeployStageDetails {
	if value == nil {
		return nil
	}
	result := &pipelinev1.DeployStageDetails{UpdateUrl: optionalString(value.UpdateURL)}
	for _, artifact := range value.Artifacts {
		result.Artifacts = append(result.Artifacts, &pipelinev1.DeployArtifactResult{ArtifactId: artifact.ArtifactID, Platform: platformToProto(artifact.Platform)})
	}
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func copyStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func stringSlice[T ~string](value []T) []string {
	result := make([]string, 0, len(value))
	for _, item := range value {
		result = append(result, string(item))
	}
	return result
}

func configToProto(config *Config) *pipelinev1.PipelineConfig {
	if config == nil {
		return nil
	}
	result := &pipelinev1.PipelineConfig{ScenarioName: config.ScenarioName, Platforms: platformsToProto(config.Platforms), DeploymentMode: deploymentModeToProto(config.DeploymentMode), Framework: frameworkToProto(config.Framework), TemplateType: templateTypeToProto(config.TemplateType), PreflightSecrets: config.PreflightSecrets, Stages: stagesToProto(config.Stages)}
	applyOptionalProtoConfig(result, config)
	return result
}

func applyOptionalProtoConfig(result *pipelinev1.PipelineConfig, config *Config) {
	if config.SkipPreflight {
		result.SkipPreflight = boolPtr(true)
	}
	if config.SkipSmokeTest {
		result.SkipSmokeTest = boolPtr(true)
	}
	if config.StopOnFailure != nil {
		result.StopOnFailure = boolPtr(*config.StopOnFailure)
	}
	if config.WebhookURL != "" {
		result.WebhookUrl = stringPtr(config.WebhookURL)
	}
	if config.ProxyURL != "" {
		result.ProxyUrl = stringPtr(config.ProxyURL)
	}
	if config.BundleManifestPath != "" {
		result.BundleManifestPath = stringPtr(config.BundleManifestPath)
	}
	applyOptionalProtoExecutionConfig(result, config)
}

func applyOptionalProtoExecutionConfig(result *pipelinev1.PipelineConfig, config *Config) {
	if config.ResourceArtifactRoot != "" {
		result.ResourceArtifactRoot = stringPtr(config.ResourceArtifactRoot)
	}
	if config.ToolArtifactRoot != "" {
		result.ToolArtifactRoot = stringPtr(config.ToolArtifactRoot)
	}
	if config.ArtifactTrustMode != "" {
		result.ArtifactTrustMode = stringPtr(string(config.ArtifactTrustMode))
	}
	if config.LocationMode != "" {
		result.LocationMode = stringPtr(config.LocationMode)
	}
	if config.Clean {
		result.Clean = boolPtr(true)
	}
	if config.Sign {
		result.Sign = boolPtr(true)
	}
	if config.Publish {
		result.Publish = boolPtr(true)
	}
	if config.Version != "" {
		result.Version = stringPtr(config.Version)
	}
	if config.PreflightTimeoutSeconds != 0 {
		timeout := int32(config.PreflightTimeoutSeconds)
		result.PreflightTimeoutSeconds = &timeout
	}
	if config.StopAfterStage != "" {
		stage := stageNameToProto(config.StopAfterStage)
		result.StopAfterStage = &stage
	}
	if config.ResumeFromStage != "" {
		stage := stageNameToProto(config.ResumeFromStage)
		result.ResumeFromStage = &stage
	}
	if config.ParentPipelineID != "" {
		result.ParentPipelineId = stringPtr(config.ParentPipelineID)
	}
	if config.IdempotencyKey != "" {
		result.IdempotencyKey = stringPtr(config.IdempotencyKey)
	}
	if config.UpdateConfig != nil {
		result.UpdateConfig = updateConfigToProto(config.UpdateConfig)
	}
}

func updateConfigToProto(value *generation.UpdateConfig) *sharedv1.UpdateConfig {
	if value == nil {
		return nil
	}
	result := &sharedv1.UpdateConfig{}
	if value.Channel != "" {
		result.Channel = stringPtr(value.Channel)
	}
	if value.Provider != "" {
		result.Provider = stringPtr(value.Provider)
	}
	if value.AutoCheck {
		result.AutoCheck = boolPtr(true)
	}
	if value.Generic != nil {
		result.Generic = &sharedv1.GenericUpdateConfig{Url: value.Generic.URL}
		if value.Generic.ChannelPath != "" {
			result.Generic.ChannelPath = stringPtr(value.Generic.ChannelPath)
		}
	}
	if value.GitHub != nil {
		result.Github = &sharedv1.GitHubUpdateConfig{Owner: value.GitHub.Owner, Repo: value.GitHub.Repo}
		if value.GitHub.Private {
			result.Github.Private = boolPtr(true)
		}
	}
	return result
}

func platformToProto(value string) sharedv1.Platform {
	switch strings.ToLower(value) {
	case "win", "windows":
		return sharedv1.Platform_PLATFORM_WIN
	case "mac", "darwin", "macos":
		return sharedv1.Platform_PLATFORM_MAC
	case "linux":
		return sharedv1.Platform_PLATFORM_LINUX
	default:
		return sharedv1.Platform_PLATFORM_UNSPECIFIED
	}
}

func platformFromProto(value sharedv1.Platform) string {
	switch value {
	case sharedv1.Platform_PLATFORM_WIN:
		return "win"
	case sharedv1.Platform_PLATFORM_MAC:
		return "mac"
	case sharedv1.Platform_PLATFORM_LINUX:
		return "linux"
	default:
		return ""
	}
}

func platformsToProto(values []string) []sharedv1.Platform {
	result := make([]sharedv1.Platform, 0, len(values))
	for _, value := range values {
		result = append(result, platformToProto(value))
	}
	return result
}

func platformsFromProto(values []sharedv1.Platform) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if platform := platformFromProto(value); platform != "" {
			result = append(result, platform)
		}
	}
	return result
}

func stageNameToProto(value string) sharedv1.StageName {
	switch value {
	case StageResolveDeployment:
		return sharedv1.StageName_STAGE_NAME_RESOLVE_DEPLOYMENT
	case StageBundle:
		return sharedv1.StageName_STAGE_NAME_BUNDLE
	case StagePreflight:
		return sharedv1.StageName_STAGE_NAME_PREFLIGHT
	case StageGenerate:
		return sharedv1.StageName_STAGE_NAME_GENERATE
	case StageBuild:
		return sharedv1.StageName_STAGE_NAME_BUILD
	case StageSmokeTest:
		return sharedv1.StageName_STAGE_NAME_SMOKE_TEST
	case StageDeploy:
		return sharedv1.StageName_STAGE_NAME_DEPLOY
	default:
		return sharedv1.StageName_STAGE_NAME_UNSPECIFIED
	}
}

func stageNameFromProto(value sharedv1.StageName) string {
	switch value {
	case sharedv1.StageName_STAGE_NAME_RESOLVE_DEPLOYMENT:
		return StageResolveDeployment
	case sharedv1.StageName_STAGE_NAME_BUNDLE:
		return StageBundle
	case sharedv1.StageName_STAGE_NAME_PREFLIGHT:
		return StagePreflight
	case sharedv1.StageName_STAGE_NAME_GENERATE:
		return StageGenerate
	case sharedv1.StageName_STAGE_NAME_BUILD:
		return StageBuild
	case sharedv1.StageName_STAGE_NAME_SMOKE_TEST:
		return StageSmokeTest
	case sharedv1.StageName_STAGE_NAME_DEPLOY:
		return StageDeploy
	default:
		return ""
	}
}

func stagesToProto(values []string) []sharedv1.StageName {
	result := make([]sharedv1.StageName, 0, len(values))
	for _, value := range values {
		result = append(result, stageNameToProto(value))
	}
	return result
}

func stagesFromProto(values []sharedv1.StageName) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if stage := stageNameFromProto(value); stage != "" {
			result = append(result, stage)
		}
	}
	return result
}

func stageStatusToProto(value string) sharedv1.StageStatus {
	switch value {
	case StatusIdle:
		return sharedv1.StageStatus_STAGE_STATUS_IDLE
	case StatusPending:
		return sharedv1.StageStatus_STAGE_STATUS_PENDING
	case StatusRunning:
		return sharedv1.StageStatus_STAGE_STATUS_RUNNING
	case StatusCompleted:
		return sharedv1.StageStatus_STAGE_STATUS_COMPLETED
	case StatusFailed:
		return sharedv1.StageStatus_STAGE_STATUS_FAILED
	case StatusSkipped:
		return sharedv1.StageStatus_STAGE_STATUS_SKIPPED
	case StatusCancelled:
		return sharedv1.StageStatus_STAGE_STATUS_CANCELLED
	default:
		return sharedv1.StageStatus_STAGE_STATUS_UNSPECIFIED
	}
}

func deploymentModeToProto(value string) sharedv1.DeploymentMode {
	if value == DeploymentModeProxy {
		return sharedv1.DeploymentMode_DEPLOYMENT_MODE_PROXY
	}
	return sharedv1.DeploymentMode_DEPLOYMENT_MODE_BUNDLED
}

func deploymentModeFromProto(value sharedv1.DeploymentMode) string {
	if value == sharedv1.DeploymentMode_DEPLOYMENT_MODE_PROXY {
		return DeploymentModeProxy
	}
	return DeploymentModeBundled
}
func frameworkToProto(_ string) sharedv1.Framework { return sharedv1.Framework_FRAMEWORK_ELECTRON }
func frameworkFromProto(value sharedv1.Framework) string {
	if value == sharedv1.Framework_FRAMEWORK_UNSPECIFIED || value == sharedv1.Framework_FRAMEWORK_ELECTRON {
		return FrameworkElectron
	}
	return ""
}

func templateTypeToProto(value string) sharedv1.TemplateType {
	switch value {
	case "advanced":
		return sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED
	case "multi-window":
		return sharedv1.TemplateType_TEMPLATE_TYPE_MULTI_WINDOW
	case "kiosk":
		return sharedv1.TemplateType_TEMPLATE_TYPE_KIOSK
	default:
		return sharedv1.TemplateType_TEMPLATE_TYPE_BASIC
	}
}

func templateTypeFromProto(value sharedv1.TemplateType) string {
	switch value {
	case sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED:
		return "advanced"
	case sharedv1.TemplateType_TEMPLATE_TYPE_MULTI_WINDOW:
		return "multi-window"
	case sharedv1.TemplateType_TEMPLATE_TYPE_KIOSK:
		return "kiosk"
	default:
		return "basic"
	}
}

func unixTimestamp(value int64) *timestamppb.Timestamp {
	if value == 0 {
		return nil
	}
	return timestamppb.New(time.Unix(value, 0))
}
func boolPtr(value bool) *bool       { return &value }
func stringPtr(value string) *string { return &value }
