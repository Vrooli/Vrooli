package workflows

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/credits"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const defaultSeedScenario = "browser-automation-studio"

// service implements apiconnect.WorkflowsServiceHandler.
type service struct {
	deps Deps
}

func (s *service) log() *logrus.Logger { return s.deps.Logger }

func parseWorkflowID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, errInvalidWorkflowID
	}
	return id, nil
}

// ---------------------------------------------------------------------------
// ListWorkflows
// ---------------------------------------------------------------------------

func (s *service) ListWorkflows(
	ctx context.Context,
	req *connect.Request[basapi.ListWorkflowsRequest],
) (*connect.Response[basapi.ListWorkflowsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.ListWorkflows(ctx, req.Msg)
	if err != nil {
		s.log().WithError(err).Error("list workflows failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// GetWorkflow
// ---------------------------------------------------------------------------

func (s *service) GetWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.GetWorkflowRequest],
) (*connect.Response[basapi.GetWorkflowResponse], error) {
	if _, err := parseWorkflowID(req.Msg.GetWorkflowId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.GetWorkflowAPI(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.log().WithError(err).WithField("workflow_id", req.Msg.GetWorkflowId()).Error("get workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// CreateWorkflow
// ---------------------------------------------------------------------------

func (s *service) CreateWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.CreateWorkflowRequest],
) (*connect.Response[basapi.CreateWorkflowResponse], error) {
	if strings.TrimSpace(req.Msg.GetProjectId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project_id is required"))
	}
	if strings.TrimSpace(req.Msg.GetName()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.CreateWorkflow(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowNameConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, errWorkflowNameConflict)
		}
		if errors.Is(err, workflowservice.ErrInvalidWorkflowFormat) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowFormat)
		}
		s.log().WithError(err).Error("create workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// UpdateWorkflow
// ---------------------------------------------------------------------------

func (s *service) UpdateWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.UpdateWorkflowRequest],
) (*connect.Response[basapi.UpdateWorkflowResponse], error) {
	if _, err := parseWorkflowID(req.Msg.GetWorkflowId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.UpdateWorkflow(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowVersionConflict) {
			return nil, connect.NewError(connect.CodeAborted, errWorkflowVersionConfl)
		}
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		if errors.Is(err, workflowservice.ErrInvalidWorkflowFormat) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowFormat)
		}
		s.log().WithError(err).WithField("workflow_id", req.Msg.GetWorkflowId()).Error("update workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// DeleteWorkflow
// ---------------------------------------------------------------------------

func (s *service) DeleteWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.DeleteWorkflowRequest],
) (*connect.Response[basapi.DeleteWorkflowResponse], error) {
	if _, err := parseWorkflowID(req.Msg.GetWorkflowId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.DeleteWorkflow(ctx, req.Msg)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.log().WithError(err).WithField("workflow_id", req.Msg.GetWorkflowId()).Error("delete workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// ExecuteWorkflow
// ---------------------------------------------------------------------------

func (s *service) ExecuteWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.ExecuteWorkflowRequest],
) (*connect.Response[basapi.ExecuteWorkflowResponse], error) {
	if _, err := parseWorkflowID(req.Msg.GetWorkflowId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	opts := executeOptionsFromProto(req.Msg.GetOptions())

	seedPlan, err := s.applySeedIfNeeded(ctx, req.Msg.GetOptions(), &req.Msg.Parameters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	resp, err := s.deps.Executor.ExecuteWorkflowAPIWithOptions(ctx, req.Msg, opts)
	if err != nil {
		if seedPlan != nil && seedPlan.cleanup != nil {
			_ = seedPlan.cleanup(context.Background())
		}
		var seedErr *workflowservice.SeedRequirementError
		if errors.As(err, &seedErr) {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				fmt.Errorf("%s (missing: %s)", seedErr.Error(), strings.Join(seedErr.MissingKeys, ", ")),
			)
		}
		s.log().WithError(err).Error("execute workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.chargeExecutionCredits(ctx, req.Msg.GetWorkflowId(), resp.GetExecutionId())

	if err := s.finalizeSeedPlan(ctx, seedPlan, resp.GetExecutionId(), req.Msg.GetWaitForCompletion()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// ExecuteAdhocWorkflow
// ---------------------------------------------------------------------------

func (s *service) ExecuteAdhocWorkflow(
	ctx context.Context,
	req *connect.Request[basexecution.ExecuteAdhocRequest],
) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
	if req.Msg.GetFlowDefinition() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errCurrentFlowRequired)
	}

	// Adhoc callers explicitly opt into synchronous execution with
	// wait_for_completion. Use the completion budget rather than the generic
	// extended request timeout, which is intentionally short for CRUD calls.
	ctx, cancel := context.WithTimeout(ctx, constants.ExecutionCompletionTimeout)
	defer cancel()

	opts := executeOptionsFromProto(req.Msg.GetOptions())

	seedPlan, err := s.applySeedIfNeeded(ctx, req.Msg.GetOptions(), &req.Msg.Parameters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	resp, err := s.deps.Executor.ExecuteAdhocWorkflowAPIWithOptions(ctx, req.Msg, opts)
	if err != nil {
		if seedPlan != nil && seedPlan.cleanup != nil {
			_ = seedPlan.cleanup(context.Background())
		}
		var seedErr *workflowservice.SeedRequirementError
		if errors.As(err, &seedErr) {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				fmt.Errorf("%s (missing: %s)", seedErr.Error(), strings.Join(seedErr.MissingKeys, ", ")),
			)
		}
		s.log().WithError(err).Error("execute adhoc workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.chargeExecutionCredits(ctx, "", resp.GetExecutionId())

	if err := s.finalizeSeedPlan(ctx, seedPlan, resp.GetExecutionId(), req.Msg.GetWaitForCompletion()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// ValidateWorkflow
// ---------------------------------------------------------------------------

func (s *service) ValidateWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.ValidateWorkflowRequest],
) (*connect.Response[basapi.ValidateWorkflowResponse], error) {
	if req.Msg.GetWorkflow() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errWorkflowPayload)
	}
	result := s.deps.Validator.ValidateDefinition(ctx, req.Msg.GetWorkflow())
	return connect.NewResponse(&basapi.ValidateWorkflowResponse{Result: result}), nil
}

// ---------------------------------------------------------------------------
// ValidateResolvedWorkflow
// ---------------------------------------------------------------------------

func (s *service) ValidateResolvedWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.ValidateWorkflowRequest],
) (*connect.Response[basapi.ValidateWorkflowResponse], error) {
	// Resolved validation runs the same semantic lint over a WorkflowDefinitionV2;
	// the additional "tokens must be substituted" check is enforced by the
	// validator's V2 lint pass already.
	if req.Msg.GetWorkflow() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errWorkflowPayload)
	}
	result := s.deps.Validator.ValidateDefinition(ctx, req.Msg.GetWorkflow())
	return connect.NewResponse(&basapi.ValidateWorkflowResponse{Result: result}), nil
}

// ---------------------------------------------------------------------------
// ModifyWorkflow
// ---------------------------------------------------------------------------

func (s *service) ModifyWorkflow(
	ctx context.Context,
	req *connect.Request[basapi.ModifyWorkflowRequest],
) (*connect.Response[basapi.UpdateWorkflowResponse], error) {
	workflowID, err := parseWorkflowID(req.Msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	prompt := strings.TrimSpace(req.Msg.GetModificationPrompt())
	if prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errModifyPromptRequired)
	}
	if req.Msg.GetCurrentFlow() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errCurrentFlowRequired)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.ModifyWorkflowAPI(ctx, workflowID, prompt, req.Msg.GetCurrentFlow())
	if err != nil {
		var aiErr *workflowservice.AIWorkflowError
		if errors.As(err, &aiErr) {
			return nil, connect.NewError(connect.CodeInvalidArgument, aiErr)
		}
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.log().WithError(err).WithField("workflow_id", workflowID).Error("modify workflow failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// ListWorkflowVersions
// ---------------------------------------------------------------------------

func (s *service) ListWorkflowVersions(
	ctx context.Context,
	req *connect.Request[basapi.ListWorkflowVersionsRequest],
) (*connect.Response[basapi.WorkflowVersionList], error) {
	workflowID, err := parseWorkflowID(req.Msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	list, err := s.deps.Catalog.ListWorkflowVersionsAPI(ctx, workflowID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.log().WithError(err).WithField("workflow_id", workflowID).Error("list versions failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(list), nil
}

// ---------------------------------------------------------------------------
// GetWorkflowVersion
// ---------------------------------------------------------------------------

func (s *service) GetWorkflowVersion(
	ctx context.Context,
	req *connect.Request[basapi.GetWorkflowVersionRequest],
) (*connect.Response[basapi.WorkflowVersion], error) {
	workflowID, err := parseWorkflowID(req.Msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetVersion() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("version must be >= 1"))
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	version, err := s.deps.Catalog.GetWorkflowVersionAPI(ctx, workflowID, req.Msg.GetVersion())
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowVersionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.log().WithError(err).WithField("workflow_id", workflowID).Error("get workflow version failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(version), nil
}

// ---------------------------------------------------------------------------
// RestoreWorkflowVersion
// ---------------------------------------------------------------------------

func (s *service) RestoreWorkflowVersion(
	ctx context.Context,
	req *connect.Request[basapi.RestoreWorkflowVersionRequest],
) (*connect.Response[basapi.RestoreWorkflowVersionResponse], error) {
	workflowID, err := parseWorkflowID(req.Msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.GetVersion() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("version must be >= 1"))
	}

	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	resp, err := s.deps.Catalog.RestoreWorkflowVersionAPI(ctx, workflowID, req.Msg.GetVersion(), req.Msg.GetChangeDescription())
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowVersionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.log().WithError(err).WithField("workflow_id", workflowID).Error("restore workflow version failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// executeOptionsFromProto bridges the proto ExecuteWorkflowOptions to the
// workflowservice.ExecuteOptions struct understood by the executor.
func executeOptionsFromProto(opts *basexecution.ExecuteWorkflowOptions) *workflowservice.ExecuteOptions {
	if opts == nil {
		return nil
	}
	if !opts.GetRequiresVideo() &&
		!opts.GetRequiresTrace() &&
		!opts.GetRequiresHar() &&
		!opts.GetFrameStreaming() && opts.GetElectronTarget() == nil && opts.GetValidationContext() == nil {
		return nil
	}
	out := &workflowservice.ExecuteOptions{
		RequiresVideo:        opts.GetRequiresVideo(),
		RequiresTrace:        opts.GetRequiresTrace(),
		RequiresHAR:          opts.GetRequiresHar(),
		EnableFrameStreaming: opts.GetFrameStreaming(),
	}
	if opts.FrameStreamingQuality != nil {
		out.FrameStreamingQuality = int(opts.GetFrameStreamingQuality())
	}
	if opts.FrameStreamingFps != nil {
		out.FrameStreamingFPS = int(opts.GetFrameStreamingFps())
	}
	if target := opts.GetElectronTarget(); target != nil {
		out.ElectronTarget = &autodriver.ElectronTarget{
			TargetID: target.GetTargetId(), CDPEndpoint: target.GetCdpEndpoint(),
			RendererID: target.GetRendererId(), RendererURL: target.GetRendererUrl(),
			RendererTitle: target.GetRendererTitle(), ScenarioName: target.GetScenarioName(),
			ArtifactDigest: target.GetArtifactDigest(), ContextID: target.GetContextId(),
			CDPTransport: target.GetCdpTransport(),
		}
	}
	if validationContext := opts.GetValidationContext(); validationContext != nil {
		out.ValidationContext = &autodriver.ValidationContext{
			ContextID: validationContext.GetContextId(), ScenarioName: validationContext.GetScenarioName(),
			ArtifactDigest: validationContext.GetArtifactDigest(), TargetID: validationContext.GetTargetId(),
			WorkflowID: validationContext.GetWorkflowId(), ProfileID: validationContext.GetProfileId(),
			IsolationLeaseID: validationContext.GetIsolationLeaseId(),
		}
	}
	return out
}

// seedCleanupPlan tracks deferred cleanup of an applied seed.
type seedCleanupPlan struct {
	cleanup      func(context.Context) error
	cleanupToken string
	seedScenario string
}

// applySeedIfNeeded resolves the seed_mode option, applies the seed via the
// configured SeedRunner, and merges the resulting state into the execution
// parameters. Returns a cleanup plan when a seed was applied.
func (s *service) applySeedIfNeeded(
	ctx context.Context,
	opts *basexecution.ExecuteWorkflowOptions,
	params **basexecution.ExecutionParameters,
) (*seedCleanupPlan, error) {
	if opts == nil {
		return nil, nil
	}
	mode := strings.ToLower(strings.TrimSpace(opts.GetSeedMode()))
	if mode == "" {
		return nil, nil
	}
	if mode != "needs-applying" {
		return nil, fmt.Errorf("%w: %q", errUnsupportedSeedMode, mode)
	}
	if s.deps.SeedRunner == nil {
		return nil, errSeedCleanupUnavail
	}
	scenario := strings.TrimSpace(opts.GetSeedScenario())
	if scenario == "" {
		scenario = defaultSeedScenario
	}
	if strings.EqualFold(scenario, defaultSeedScenario) {
		return nil, errSeedSelfReference
	}

	cleanupToken, seedState, err := s.deps.SeedRunner.ApplySeed(ctx, scenario, false)
	if err != nil {
		return nil, err
	}
	mergeSeedState(params, seedState)

	plan := &seedCleanupPlan{
		cleanupToken: cleanupToken,
		seedScenario: scenario,
		cleanup: func(cleanupCtx context.Context) error {
			return s.deps.SeedRunner.CleanupSeed(cleanupCtx, scenario, cleanupToken)
		},
	}
	return plan, nil
}

// finalizeSeedPlan either runs cleanup synchronously (wait_for_completion=true)
// or schedules deferred cleanup. Returns an error only when no scheduler is
// configured.
func (s *service) finalizeSeedPlan(
	ctx context.Context,
	plan *seedCleanupPlan,
	executionID string,
	waitForCompletion bool,
) error {
	if plan == nil {
		return nil
	}
	if waitForCompletion {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultRequestTimeout)
		defer cancel()
		if err := plan.cleanup(cleanupCtx); err != nil {
			return fmt.Errorf("seed cleanup failed: %w", err)
		}
		_ = ctx
		return nil
	}
	if s.deps.SeedScheduler == nil {
		return errSeedCleanupUnavail
	}
	if err := s.deps.SeedScheduler.Schedule(executionID, plan.seedScenario, plan.cleanupToken); err != nil {
		return fmt.Errorf("seed cleanup scheduling failed: %w", err)
	}
	return nil
}

func mergeSeedState(params **basexecution.ExecutionParameters, seedState map[string]any) {
	if params == nil {
		return
	}
	if *params == nil {
		*params = &basexecution.ExecutionParameters{}
	}
	if (*params).InitialParams == nil {
		(*params).InitialParams = map[string]*commonv1.JsonValue{}
	}
	for key, value := range seedState {
		if _, exists := (*params).InitialParams[key]; exists {
			continue
		}
		(*params).InitialParams[key] = typeconv.AnyToJsonValue(value)
	}
	if (*params).Env == nil {
		(*params).Env = map[string]*commonv1.JsonValue{}
	}
	if _, exists := (*params).Env["seed_applied"]; !exists {
		(*params).Env["seed_applied"] = typeconv.AnyToJsonValue(true)
	}
}

// chargeExecutionCredits logs but does not abort on charge failure: the
// execution already started and a transient billing error must not break the
// request.
func (s *service) chargeExecutionCredits(ctx context.Context, workflowID, executionID string) {
	if s.deps.CreditService == nil || executionID == "" {
		return
	}
	identity := s.deps.UserIdentity(ctx)
	_, err := s.deps.CreditService.Charge(ctx, credits.ChargeRequest{
		UserIdentity: identity,
		Operation:    credits.OpExecutionRun,
		Metadata: credits.ChargeMetadata{
			WorkflowID:  workflowID,
			ExecutionID: executionID,
		},
	})
	if err != nil {
		s.log().WithError(err).WithField("execution_id", executionID).Warn("failed to charge credits for execution")
	}
}
