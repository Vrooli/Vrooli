package executions

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// stubExecutor implements Executor for tests. Only the methods exercised by
// the active test set are non-trivial; the rest panic to flag accidental
// surface broadening.
type stubExecutor struct {
	listFn          func(ctx context.Context, wf, proj *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error)
	getFn           func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	hydrateFn       func(ctx context.Context, e *database.ExecutionIndex) (*basexecution.Execution, error)
	stopFn          func(ctx context.Context, id uuid.UUID) error
	resumeFn        func(ctx context.Context, id uuid.UUID, params map[string]any) (*database.ExecutionIndex, error)
	timelineFn      func(ctx context.Context, id uuid.UUID) (*workflowservice.ExecutionTimeline, error)
	timelineProtoFn func(ctx context.Context, id uuid.UUID) (*bastimeline.ExecutionTimeline, error)
	screenshotsFn   func(ctx context.Context, id uuid.UUID) ([]*basexecution.ExecutionScreenshot, error)
	videosFn        func(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionVideoArtifact, error)
	tracesFn        func(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionFileArtifact, error)
	harFn           func(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionFileArtifact, error)
}

func (s *stubExecutor) ListExecutions(ctx context.Context, wf, proj *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	return s.listFn(ctx, wf, proj, limit, offset)
}

func (s *stubExecutor) GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return s.getFn(ctx, id)
}

func (s *stubExecutor) HydrateExecutionProto(ctx context.Context, e *database.ExecutionIndex) (*basexecution.Execution, error) {
	return s.hydrateFn(ctx, e)
}

func (s *stubExecutor) StopExecution(ctx context.Context, id uuid.UUID) error {
	return s.stopFn(ctx, id)
}

func (s *stubExecutor) ResumeExecution(ctx context.Context, id uuid.UUID, p map[string]any) (*database.ExecutionIndex, error) {
	return s.resumeFn(ctx, id, p)
}

func (s *stubExecutor) GetExecutionTimeline(ctx context.Context, id uuid.UUID) (*workflowservice.ExecutionTimeline, error) {
	return s.timelineFn(ctx, id)
}

func (s *stubExecutor) GetExecutionTimelineProto(ctx context.Context, id uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	return s.timelineProtoFn(ctx, id)
}

func (s *stubExecutor) GetExecutionScreenshots(ctx context.Context, id uuid.UUID) ([]*basexecution.ExecutionScreenshot, error) {
	return s.screenshotsFn(ctx, id)
}

func (s *stubExecutor) GetExecutionVideoArtifacts(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionVideoArtifact, error) {
	return s.videosFn(ctx, id)
}

func (s *stubExecutor) GetExecutionTraceArtifacts(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionFileArtifact, error) {
	return s.tracesFn(ctx, id)
}

func (s *stubExecutor) GetExecutionHarArtifacts(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionFileArtifact, error) {
	return s.harFn(ctx, id)
}

// Unused interface methods (we only test the transport surface).
func (s *stubExecutor) ExecuteWorkflow(context.Context, uuid.UUID, map[string]any) (*database.ExecutionIndex, error) {
	panic("not implemented")
}

func (s *stubExecutor) ExecuteWorkflowAPI(context.Context, *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	panic("not implemented")
}

func (s *stubExecutor) ExecuteWorkflowAPIWithOptions(context.Context, *basapi.ExecuteWorkflowRequest, *workflowservice.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	panic("not implemented")
}

func (s *stubExecutor) ExecuteAdhocWorkflowAPI(context.Context, *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	panic("not implemented")
}

func (s *stubExecutor) ExecuteAdhocWorkflowAPIWithOptions(context.Context, *basexecution.ExecuteAdhocRequest, *workflowservice.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	panic("not implemented")
}

func (s *stubExecutor) UpdateExecution(context.Context, *database.ExecutionIndex) error {
	panic("not implemented")
}

func (s *stubExecutor) DescribeExecutionExport(context.Context, uuid.UUID) (*workflowservice.ExecutionExportPreview, error) {
	panic("not implemented")
}

func (s *stubExecutor) ExportToFolder(context.Context, uuid.UUID, string, storage.StorageInterface) error {
	panic("not implemented")
}

type stubScheduler struct {
	calls []schedCall
	err   error
}

type schedCall struct{ exec, scenario, token string }

func (s *stubScheduler) Schedule(exec, scenario, token string) error {
	s.calls = append(s.calls, schedCall{exec, scenario, token})
	return s.err
}

func newTestService(t *testing.T, exec Executor, sched SeedScheduler) (apiconnect.ExecutionsServiceClient, func()) {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	mount := Module(Deps{Executor: exec, SeedScheduler: sched, Logger: log})
	srv := httptest.NewServer(mount.Handler)
	client := apiconnect.NewExecutionsServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func TestModule_PanicsOnMissingExecutor(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on missing executor")
		}
	}()
	Module(Deps{Logger: logrus.New()})
}

func TestModule_PanicsOnMissingLogger(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on missing logger")
		}
	}()
	Module(Deps{Executor: &stubExecutor{}})
}

func TestGetExecution_InvalidUUID(t *testing.T) {
	client, stop := newTestService(t, &stubExecutor{}, nil)
	defer stop()

	_, err := client.GetExecution(context.Background(), connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: "not-a-uuid"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestGetExecution_NotFound(t *testing.T) {
	id := uuid.New()
	exec := &stubExecutor{
		getFn: func(_ context.Context, _ uuid.UUID) (*database.ExecutionIndex, error) {
			return nil, database.ErrNotFound
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	_, err := client.GetExecution(context.Background(), connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: id.String()}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v (%v)", connect.CodeOf(err), err)
	}
}

func TestGetExecution_Success(t *testing.T) {
	id := uuid.New()
	exec := &stubExecutor{
		getFn: func(_ context.Context, got uuid.UUID) (*database.ExecutionIndex, error) {
			if got != id {
				return nil, errors.New("wrong id")
			}
			return &database.ExecutionIndex{ID: id, StartedAt: time.Now()}, nil
		},
		hydrateFn: func(_ context.Context, e *database.ExecutionIndex) (*basexecution.Execution, error) {
			return &basexecution.Execution{ExecutionId: e.ID.String()}, nil
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	resp, err := client.GetExecution(context.Background(), connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: id.String()}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetExecution().GetExecutionId() != id.String() {
		t.Fatalf("unexpected id %q", resp.Msg.GetExecution().GetExecutionId())
	}
}

func TestListExecutions_FiltersInvalidWorkflow(t *testing.T) {
	client, stop := newTestService(t, &stubExecutor{}, nil)
	defer stop()

	_, err := client.ListExecutions(context.Background(), connect.NewRequest(&basapi.ListExecutionsRequest{WorkflowId: pstr("nope")}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestListExecutions_Success(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	exec := &stubExecutor{
		listFn: func(_ context.Context, _, _ *uuid.UUID, _ int, _ int) ([]*database.ExecutionIndex, error) {
			return []*database.ExecutionIndex{{ID: a}, {ID: b}}, nil
		},
		hydrateFn: func(_ context.Context, e *database.ExecutionIndex) (*basexecution.Execution, error) {
			return &basexecution.Execution{ExecutionId: e.ID.String()}, nil
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	resp, err := client.ListExecutions(context.Background(), connect.NewRequest(&basapi.ListExecutionsRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Msg.GetExecutions()); got != 2 {
		t.Fatalf("expected 2 executions, got %d", got)
	}
}

func TestStopExecution(t *testing.T) {
	id := uuid.New()
	var got uuid.UUID
	exec := &stubExecutor{
		stopFn: func(_ context.Context, eid uuid.UUID) error { got = eid; return nil },
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	resp, err := client.StopExecution(context.Background(), connect.NewRequest(&basapi.StopExecutionRequest{ExecutionId: id.String()}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetStatus() != "stopped" {
		t.Fatalf("expected status=stopped, got %q", resp.Msg.GetStatus())
	}
	if got != id {
		t.Fatalf("executor invoked with %v, want %v", got, id)
	}
}

func TestResumeExecution_FailedPrecondition(t *testing.T) {
	id := uuid.New()
	exec := &stubExecutor{
		resumeFn: func(context.Context, uuid.UUID, map[string]any) (*database.ExecutionIndex, error) {
			return nil, errors.New("execution cannot be resumed: terminal status")
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	_, err := client.ResumeExecution(context.Background(), connect.NewRequest(&basapi.ResumeExecutionRequest{ExecutionId: id.String()}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v (%v)", connect.CodeOf(err), err)
	}
}

func TestResumeExecution_MergesResumeURL(t *testing.T) {
	id := uuid.New()
	var capturedParams map[string]any
	exec := &stubExecutor{
		resumeFn: func(_ context.Context, _ uuid.UUID, params map[string]any) (*database.ExecutionIndex, error) {
			capturedParams = params
			return &database.ExecutionIndex{ID: id}, nil
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	_, err := client.ResumeExecution(context.Background(), connect.NewRequest(&basapi.ResumeExecutionRequest{
		ExecutionId: id.String(),
		ResumeUrl:   "https://example.com",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := capturedParams["resume_url"]; got != "https://example.com" {
		t.Fatalf("resume_url not merged into params; got %v", capturedParams)
	}
}

func TestScheduleSeedCleanup_Success(t *testing.T) {
	id := uuid.New()
	sched := &stubScheduler{}
	client, stop := newTestService(t, &stubExecutor{}, sched)
	defer stop()

	resp, err := client.ScheduleExecutionSeedCleanup(context.Background(), connect.NewRequest(&basapi.ScheduleSeedCleanupRequest{
		ExecutionId:  id.String(),
		CleanupToken: "tok",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetStatus() != "scheduled" {
		t.Fatalf("expected status=scheduled, got %q", resp.Msg.GetStatus())
	}
	if len(sched.calls) != 1 || sched.calls[0].scenario != defaultSeedScenario {
		t.Fatalf("expected one call with default scenario, got %+v", sched.calls)
	}
}

func TestScheduleSeedCleanup_MissingToken(t *testing.T) {
	client, stop := newTestService(t, &stubExecutor{}, &stubScheduler{})
	defer stop()

	_, err := client.ScheduleExecutionSeedCleanup(context.Background(), connect.NewRequest(&basapi.ScheduleSeedCleanupRequest{
		ExecutionId: uuid.NewString(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestScheduleSeedCleanup_NoScheduler(t *testing.T) {
	client, stop := newTestService(t, &stubExecutor{}, nil)
	defer stop()

	_, err := client.ScheduleExecutionSeedCleanup(context.Background(), connect.NewRequest(&basapi.ScheduleSeedCleanupRequest{
		ExecutionId:  uuid.NewString(),
		CleanupToken: "tok",
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", connect.CodeOf(err))
	}
}

func TestGetExecutionRecordedVideos_Success(t *testing.T) {
	id := uuid.New()
	size := int64(42)
	exec := &stubExecutor{
		videosFn: func(_ context.Context, _ uuid.UUID) ([]workflowservice.ExecutionVideoArtifact, error) {
			return []workflowservice.ExecutionVideoArtifact{{
				ArtifactID: "a.webm", Label: "a", SizeBytes: &size, ContentType: "video/webm",
			}}, nil
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	resp, err := client.GetExecutionRecordedVideos(context.Background(), connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: id.String()}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Msg.GetVideos()); got != 1 {
		t.Fatalf("expected 1 video, got %d", got)
	}
	if resp.Msg.GetVideos()[0].GetSizeBytes() != 42 {
		t.Fatalf("size mismatch")
	}
}

func pstr(s string) *string { return &s }
