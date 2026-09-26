package executions

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// stubExecutor implements Executor for tests. Only the methods exercised by
// the active test set are non-trivial; the rest panic to flag accidental
// surface broadening.
type stubExecutor struct {
	listFn          func(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error)
	getFn           func(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	hydrateFn       func(ctx context.Context, e *database.ExecutionIndex) (*basexecution.Execution, error)
	stopFn          func(ctx context.Context, id uuid.UUID) error
	resumeFn        func(ctx context.Context, id uuid.UUID, params map[string]any) (*database.ExecutionIndex, error)
	timelineFn      func(ctx context.Context, id uuid.UUID) (*workflowservice.ExecutionTimeline, error)
	timelineProtoFn func(ctx context.Context, id uuid.UUID) (*bastimeline.ExecutionTimeline, error)
	replayFn        func(ctx context.Context, id uuid.UUID) (*basevidence.ReplayPackage, error)
	screenshotsFn   func(ctx context.Context, id uuid.UUID) ([]*basexecution.ExecutionScreenshot, error)
	videosFn        func(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionVideoArtifact, error)
	tracesFn        func(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionFileArtifact, error)
	harFn           func(ctx context.Context, id uuid.UUID) ([]workflowservice.ExecutionFileArtifact, error)
}

func (s *stubExecutor) ListExecutions(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	return s.listFn(ctx, query)
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

func (s *stubExecutor) GetExecutionReplayPackage(ctx context.Context, id uuid.UUID) (*basevidence.ReplayPackage, error) {
	if s.replayFn == nil {
		return nil, errors.New("replay package not configured")
	}
	return s.replayFn(ctx, id)
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
		listFn: func(_ context.Context, _ database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
			return []*database.ExecutionIndex{{ID: a}, {ID: b}}, 2, nil
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

func TestStopExecutionReturnsOwnerUnavailableError(t *testing.T) {
	t.Log("[REQ:BAS-RH-J07] stop with unavailable owner returns no stopped response")
	exec := &stubExecutor{
		stopFn: func(context.Context, uuid.UUID) error {
			return errors.New("execution is running but this API process has no cancellation owner")
		},
	}
	client, stop := newTestService(t, exec, nil)
	defer stop()

	resp, err := client.StopExecution(context.Background(), connect.NewRequest(&basapi.StopExecutionRequest{ExecutionId: uuid.NewString()}))
	if err == nil {
		t.Fatalf("StopExecution response = %+v, want owner-unavailable error", resp)
	}
	if resp != nil {
		t.Fatalf("StopExecution response = %+v alongside error, want no stopped response", resp)
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("StopExecution error code = %s, want internal error", got)
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

func TestGetExecutionReplayPackage_Success(t *testing.T) {
	id := uuid.New()
	exec := &stubExecutor{replayFn: func(_ context.Context, got uuid.UUID) (*basevidence.ReplayPackage, error) {
		if got != id {
			t.Fatalf("execution ID = %s", got)
		}
		return &basevidence.ReplayPackage{Id: uuid.NewString(), SchemaVersion: "bas-replay/v1", ExecutionId: id.String()}, nil
	}}
	client, stop := newTestService(t, exec, nil)
	defer stop()
	resp, err := client.GetExecutionReplayPackage(context.Background(), connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: id.String()}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetExecutionId() != id.String() || resp.Msg.GetSchemaVersion() != "bas-replay/v1" {
		t.Fatalf("unexpected replay package: %#v", resp.Msg)
	}
}

func pstr(s string) *string { return &s }

func TestListExecutions_QueryContract(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "history.db")+"?_pragma=foreign_keys(ON)&_time_format=sqlite")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchemaRegistry(ctx, db, ""); err != nil {
		t.Fatal(err)
	}
	repo := database.NewRepository(&database.DB{DB: db}, logrus.New())
	projects := []uuid.UUID{uuid.New(), uuid.New()}
	for i, id := range projects {
		if err := repo.CreateProject(ctx, &database.ProjectIndex{ID: id, Name: fmt.Sprintf("project%d", i), FolderPath: fmt.Sprintf("/project%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	workflows := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for i, id := range workflows {
		if err := repo.CreateWorkflow(ctx, &database.WorkflowIndex{ID: id, ProjectID: &projects[i/2], Name: fmt.Sprintf("workflow%d", i), FolderPath: fmt.Sprintf("/workflow%d", i), Version: 1}); err != nil {
			t.Fatal(err)
		}
	}
	ids := make([]uuid.UUID, 60)
	started := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for i := range ids {
		ids[i] = uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1))
		status := database.ExecutionStatusCompleted
		if i%2 == 0 {
			status = database.ExecutionStatusRunning
		}
		if err := repo.CreateExecution(ctx, &database.ExecutionIndex{ID: ids[i], WorkflowID: workflows[i%3], Status: status, StartedAt: started}); err != nil {
			t.Fatal(err)
		}
	}
	exec := &stubExecutor{listFn: repo.ListExecutions, hydrateFn: func(_ context.Context, e *database.ExecutionIndex) (*basexecution.Execution, error) {
		return &basexecution.Execution{ExecutionId: e.ID.String()}, nil
	}}
	client, stop := newTestService(t, exec, nil)
	defer stop()
	running := basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
	for _, tc := range []struct {
		name              string
		limit, offset     int32
		workflow, project *string
		status            *basbase.ExecutionStatus
		count, total      int
		more              bool
		first             string
	}{
		{name: "default", count: 50, total: 60, more: true, first: ids[59].String()},
		{name: "all", limit: 100, count: 60, total: 60, first: ids[59].String()},
		{name: "project", limit: 100, project: pstr(projects[0].String()), count: 40, total: 40, first: ids[58].String()},
		{name: "status and workflow", limit: 2, workflow: pstr(workflows[0].String()), status: &running, count: 2, total: 10, more: true, first: ids[54].String()},
		{name: "final full page", limit: 2, offset: 8, workflow: pstr(workflows[0].String()), status: &running, count: 2, total: 10, first: ids[6].String()},
		{name: "past end", limit: 2, offset: 10, workflow: pstr(workflows[0].String()), status: &running, total: 10},
		{name: "disjoint filters", limit: 2, workflow: pstr(workflows[0].String()), project: pstr(projects[1].String()), status: &running},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &basapi.ListExecutionsRequest{WorkflowId: tc.workflow, ProjectId: tc.project, Status: tc.status, Offset: &tc.offset}
			if tc.limit != 0 {
				req.Limit = &tc.limit
			}
			resp, err := client.ListExecutions(ctx, connect.NewRequest(req))
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Msg.Executions) != tc.count || int(resp.Msg.Total) != tc.total || resp.Msg.HasMore != tc.more {
				t.Fatalf("page count=%d total=%d more=%t; want %d/%d/%t", len(resp.Msg.Executions), resp.Msg.Total, resp.Msg.HasMore, tc.count, tc.total, tc.more)
			}
			if tc.first != "" && resp.Msg.Executions[0].ExecutionId != tc.first {
				t.Fatalf("equal-time order first=%s want=%s", resp.Msg.Executions[0].ExecutionId, tc.first)
			}
		})
	}
	zero, negative, large := int32(0), int32(-1), int32(101)
	unknown := basbase.ExecutionStatus(999)
	for _, req := range []*basapi.ListExecutionsRequest{{Limit: &zero}, {Limit: &negative}, {Limit: &large}, {Offset: &negative}, {Status: &unknown}} {
		_, err := client.ListExecutions(ctx, connect.NewRequest(req))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("invalid query %v: %v", req, err)
		}
	}
}
