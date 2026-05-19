package uxmetrics

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/services/entitlement"
	uxmetricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxmetrics"
)

const (
	defaultAggregateLimit = 10
	maxAggregateLimit     = 100
)

type service struct {
	deps Deps
}

func (s *service) GetExecutionMetrics(
	ctx context.Context,
	req *connect.Request[uxmetricsv1.GetExecutionMetricsRequest],
) (*connect.Response[uxmetricsv1.GetExecutionMetricsResponse], error) {
	if err := requireProTier(ctx); err != nil {
		return nil, err
	}
	executionID, err := uuid.Parse(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExecutionID)
	}

	metrics, err := s.deps.Service.GetExecutionMetrics(ctx, executionID)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution metrics")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if metrics == nil {
		return nil, connect.NewError(connect.CodeNotFound, errMetricsNotFound)
	}

	payload, err := toStruct(metrics)
	if err != nil {
		s.deps.Logger.WithError(err).Error("uxmetrics.GetExecutionMetrics encode failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&uxmetricsv1.GetExecutionMetricsResponse{Metrics: payload}), nil
}

func (s *service) GetStepMetrics(
	ctx context.Context,
	req *connect.Request[uxmetricsv1.GetStepMetricsRequest],
) (*connect.Response[uxmetricsv1.GetStepMetricsResponse], error) {
	if err := requireProTier(ctx); err != nil {
		return nil, err
	}
	executionID, err := uuid.Parse(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExecutionID)
	}
	stepIndex := int(req.Msg.GetStepIndex())
	if stepIndex < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidStepIndex)
	}

	metrics, err := s.deps.Service.Analyzer().AnalyzeStep(ctx, executionID, stepIndex)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("execution_id", executionID).WithField("step_index", stepIndex).Error("Failed to analyze step")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if metrics == nil {
		return nil, connect.NewError(connect.CodeNotFound, errStepMetricsMissing)
	}

	payload, err := toStruct(metrics)
	if err != nil {
		s.deps.Logger.WithError(err).Error("uxmetrics.GetStepMetrics encode failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&uxmetricsv1.GetStepMetricsResponse{Metrics: payload}), nil
}

func (s *service) ComputeExecutionMetrics(
	ctx context.Context,
	req *connect.Request[uxmetricsv1.ComputeExecutionMetricsRequest],
) (*connect.Response[uxmetricsv1.ComputeExecutionMetricsResponse], error) {
	if err := requireProTier(ctx); err != nil {
		return nil, err
	}
	executionID, err := uuid.Parse(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExecutionID)
	}

	metrics, err := s.deps.Service.ComputeAndSaveMetrics(ctx, executionID)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("execution_id", executionID).Error("Failed to compute metrics")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	payload, err := toStruct(metrics)
	if err != nil {
		s.deps.Logger.WithError(err).Error("uxmetrics.ComputeExecutionMetrics encode failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&uxmetricsv1.ComputeExecutionMetricsResponse{Metrics: payload}), nil
}

func (s *service) GetWorkflowAggregate(
	ctx context.Context,
	req *connect.Request[uxmetricsv1.GetWorkflowAggregateRequest],
) (*connect.Response[uxmetricsv1.GetWorkflowAggregateResponse], error) {
	if err := requireProTier(ctx); err != nil {
		return nil, err
	}
	workflowID, err := uuid.Parse(req.Msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultAggregateLimit
	}
	if limit > maxAggregateLimit {
		limit = maxAggregateLimit
	}

	aggregate, err := s.deps.Service.GetWorkflowAggregate(ctx, workflowID, limit)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("workflow_id", workflowID).Error("Failed to get workflow metrics aggregate")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	payload, err := toStruct(aggregate)
	if err != nil {
		s.deps.Logger.WithError(err).Error("uxmetrics.GetWorkflowAggregate encode failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&uxmetricsv1.GetWorkflowAggregateResponse{Aggregate: payload}), nil
}

func requireProTier(ctx context.Context) error {
	ent := entitlement.FromContext(ctx)
	if ent == nil {
		return nil
	}
	if !ent.Tier.AtLeast(entitlement.TierPro) {
		return connect.NewError(connect.CodePermissionDenied, errProTierRequired)
	}
	return nil
}

// toStruct round-trips the typed payload through JSON to preserve the
// snake_case shape consumed by the existing UI store and CLI.
func toStruct(v any) (*structpb.Struct, error) {
	if v == nil {
		return structpb.NewStruct(map[string]any{})
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return structpb.NewStruct(map[string]any{})
	}
	return structpb.NewStruct(m)
}
