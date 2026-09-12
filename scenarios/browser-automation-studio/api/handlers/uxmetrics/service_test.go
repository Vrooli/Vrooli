package uxmetrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/services/entitlement"
	uxsvc "github.com/vrooli/browser-automation-studio/services/uxmetrics"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
	uxmetricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxmetrics"
	uxmetricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxmetrics/uxmetricsconnect"
)

// fakeService implements uxmetrics.Service for handler tests.
type fakeService struct {
	executionMetrics *contracts.ExecutionMetrics
	executionErr     error

	stepMetrics *contracts.StepMetrics
	stepErr     error

	computed    *contracts.ExecutionMetrics
	computeErr  error
	computedFor uuid.UUID

	aggregate    *contracts.WorkflowMetricsAggregate
	aggregateErr error
	lastLimit    int
}

func (f *fakeService) Collector() uxsvc.Collector { return nil }
func (f *fakeService) Analyzer() uxsvc.Analyzer   { return f }

func (f *fakeService) AnalyzeExecution(_ context.Context, _ uuid.UUID) (*contracts.ExecutionMetrics, error) {
	return f.computed, f.computeErr
}

func (f *fakeService) AnalyzeStep(_ context.Context, _ uuid.UUID, _ int) (*contracts.StepMetrics, error) {
	return f.stepMetrics, f.stepErr
}

func (f *fakeService) GetExecutionMetrics(_ context.Context, _ uuid.UUID) (*contracts.ExecutionMetrics, error) {
	return f.executionMetrics, f.executionErr
}

func (f *fakeService) ComputeAndSaveMetrics(_ context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	f.computedFor = executionID
	return f.computed, f.computeErr
}

func (f *fakeService) GetWorkflowAggregate(_ context.Context, _ uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error) {
	f.lastLimit = limit
	return f.aggregate, f.aggregateErr
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

func newTestClient(t *testing.T, svc uxsvc.Service) uxmetricsconnect.UXMetricsServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Service: svc, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return uxmetricsconnect.NewUXMetricsServiceClient(srv.Client(), srv.URL)
}

func TestModulePanicsWithoutLogger(t *testing.T) {
	require.Panics(t, func() { Module(Deps{Service: &fakeService{}}) })
}

func TestModulePanicsWithoutService(t *testing.T) {
	require.Panics(t, func() { Module(Deps{Logger: logrus.New()}) })
}

func TestGetExecutionMetricsHappyPath(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	svc := &fakeService{executionMetrics: &contracts.ExecutionMetrics{
		ExecutionID:     execID,
		WorkflowID:      wfID,
		ComputedAt:      time.Now(),
		StepCount:       3,
		SuccessfulSteps: 3,
		OverallFriction: 12.5,
	}}
	client := newTestClient(t, svc)
	res, err := client.GetExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetExecutionMetricsRequest{
		ExecutionId: execID.String(),
	}))
	require.NoError(t, err)
	fields := res.Msg.GetMetrics().GetFields()
	require.Equal(t, execID.String(), fields["execution_id"].GetStringValue())
	require.InDelta(t, 3.0, fields["step_count"].GetNumberValue(), 0.0001)
	require.InDelta(t, 12.5, fields["overall_friction_score"].GetNumberValue(), 0.0001)
}

func TestGetExecutionMetricsInvalidUUID(t *testing.T) {
	client := newTestClient(t, &fakeService{})
	_, err := client.GetExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetExecutionMetricsRequest{
		ExecutionId: "not-a-uuid",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetExecutionMetricsNotFound(t *testing.T) {
	client := newTestClient(t, &fakeService{})
	_, err := client.GetExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetExecutionMetricsRequest{
		ExecutionId: uuid.New().String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetExecutionMetricsRepoError(t *testing.T) {
	svc := &fakeService{executionErr: errors.New("boom")}
	client := newTestClient(t, svc)
	_, err := client.GetExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetExecutionMetricsRequest{
		ExecutionId: uuid.New().String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetStepMetricsHappyPath(t *testing.T) {
	svc := &fakeService{stepMetrics: &contracts.StepMetrics{StepIndex: 4, NodeID: "n", StepType: "click"}}
	client := newTestClient(t, svc)
	res, err := client.GetStepMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetStepMetricsRequest{
		ExecutionId: uuid.New().String(),
		StepIndex:   4,
	}))
	require.NoError(t, err)
	require.InDelta(t, 4.0, res.Msg.GetMetrics().GetFields()["step_index"].GetNumberValue(), 0.0001)
}

func TestGetStepMetricsNegativeIndex(t *testing.T) {
	client := newTestClient(t, &fakeService{})
	_, err := client.GetStepMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetStepMetricsRequest{
		ExecutionId: uuid.New().String(),
		StepIndex:   -1,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetStepMetricsNotFound(t *testing.T) {
	client := newTestClient(t, &fakeService{})
	_, err := client.GetStepMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetStepMetricsRequest{
		ExecutionId: uuid.New().String(),
		StepIndex:   0,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestComputeExecutionMetricsHappyPath(t *testing.T) {
	execID := uuid.New()
	svc := &fakeService{computed: &contracts.ExecutionMetrics{
		ExecutionID: execID,
		StepCount:   2,
	}}
	client := newTestClient(t, svc)
	res, err := client.ComputeExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.ComputeExecutionMetricsRequest{
		ExecutionId: execID.String(),
	}))
	require.NoError(t, err)
	require.Equal(t, execID, svc.computedFor)
	require.InDelta(t, 2.0, res.Msg.GetMetrics().GetFields()["step_count"].GetNumberValue(), 0.0001)
}

func TestComputeExecutionMetricsError(t *testing.T) {
	svc := &fakeService{computeErr: errors.New("nope")}
	client := newTestClient(t, svc)
	_, err := client.ComputeExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.ComputeExecutionMetricsRequest{
		ExecutionId: uuid.New().String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetWorkflowAggregateDefaultsAndClamps(t *testing.T) {
	wfID := uuid.New()
	svc := &fakeService{aggregate: &contracts.WorkflowMetricsAggregate{
		WorkflowID:     wfID,
		ExecutionCount: 5,
	}}
	client := newTestClient(t, svc)

	// limit=0 → default 10
	_, err := client.GetWorkflowAggregate(context.Background(), connect.NewRequest(&uxmetricsv1.GetWorkflowAggregateRequest{
		WorkflowId: wfID.String(),
	}))
	require.NoError(t, err)
	require.Equal(t, defaultAggregateLimit, svc.lastLimit)

	// limit > max → clamped
	_, err = client.GetWorkflowAggregate(context.Background(), connect.NewRequest(&uxmetricsv1.GetWorkflowAggregateRequest{
		WorkflowId: wfID.String(),
		Limit:      999,
	}))
	require.NoError(t, err)
	require.Equal(t, maxAggregateLimit, svc.lastLimit)
}

func TestGetWorkflowAggregateInvalidUUID(t *testing.T) {
	client := newTestClient(t, &fakeService{})
	_, err := client.GetWorkflowAggregate(context.Background(), connect.NewRequest(&uxmetricsv1.GetWorkflowAggregateRequest{
		WorkflowId: "nope",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestRequireProTierBlocksFreeTier(t *testing.T) {
	svc := &fakeService{executionMetrics: &contracts.ExecutionMetrics{ExecutionID: uuid.New()}}
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Service: svc, Logger: logger})

	// Wrap the mount handler in middleware that injects a Free-tier entitlement.
	freeEnt := &entitlement.Entitlement{Tier: entitlement.TierFree}
	mux := http.NewServeMux()
	mux.Handle(mount.Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(entitlement.WithEntitlement(r.Context(), freeEnt))
		mount.Handler.ServeHTTP(w, r)
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := uxmetricsconnect.NewUXMetricsServiceClient(srv.Client(), srv.URL)

	_, err := client.GetExecutionMetrics(context.Background(), connect.NewRequest(&uxmetricsv1.GetExecutionMetricsRequest{
		ExecutionId: uuid.New().String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}
