package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/api-core/discovery"
	httpx "github.com/vrooli/api-core/servertest"

	bas_base "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bas_telemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bas_execution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"

	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// stubWorkflows / stubExecutions stand in for the generated Connect clients
// so we can assert request shape and return canned responses without spinning
// up Connect handlers.

type stubWorkflows struct {
	gotReq *bas_execution.ExecuteAdhocRequest
	resp   *bas_execution.ExecuteAdhocResponse
	err    error
}

func (s *stubWorkflows) ExecuteAdhocWorkflow(_ context.Context, req *connect.Request[bas_execution.ExecuteAdhocRequest]) (*connect.Response[bas_execution.ExecuteAdhocResponse], error) {
	s.gotReq = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.resp), nil
}

type stubExecutions struct {
	getExec      *bas_execution.Execution
	getExecCalls []string
	screenshots  *bas_execution.GetScreenshotsResponse
	videos       *basapi.GetExecutionVideosResponse
	err          error
}

func (s *stubExecutions) GetExecution(_ context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
	s.getExecCalls = append(s.getExecCalls, req.Msg.GetExecutionId())
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&basapi.GetExecutionResponse{Execution: s.getExec}), nil
}

func (s *stubExecutions) GetExecutionScreenshots(_ context.Context, req *connect.Request[basapi.GetExecutionScreenshotsRequest]) (*connect.Response[bas_execution.GetScreenshotsResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.screenshots), nil
}

func (s *stubExecutions) GetExecutionRecordedVideos(_ context.Context, req *connect.Request[basapi.GetExecutionArtifactsRequest]) (*connect.Response[basapi.GetExecutionVideosResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.videos), nil
}

func newStubClient(t *testing.T, wf *stubWorkflows, ex *stubExecutions) *BrowserAutomationClient {
	t.Helper()
	return &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver("http://stub"),
			serviceName: "browser-automation-studio",
		},
		workflowsFactory:  func(string) basWorkflowsAPI { return wf },
		executionsFactory: func(string) basExecutionsAPI { return ex },
	}
}

// minimalWorkflowJSON is a small but valid WorkflowDefinitionV2 JSON document
// (proto-JSON shape). The client must accept it and forward a typed
// WorkflowDefinitionV2 to BAS.
const minimalWorkflowJSON = `{
  "metadata": {"name": "smoke", "execution_mode": "observer"},
  "nodes": [
    {"id": "wait", "action": {"type": "ACTION_TYPE_WAIT", "wait": {"duration_ms": 1}}}
  ]
}`

func TestBASClient_ExecuteAdhocWorkflow_ParsesAndForwards(t *testing.T) {
	t.Parallel()

	wf := &stubWorkflows{
		resp: &bas_execution.ExecuteAdhocResponse{
			ExecutionId: "exec-123",
			Status:      bas_base.ExecutionStatus_EXECUTION_STATUS_RUNNING,
		},
	}
	client := newStubClient(t, wf, &stubExecutions{})

	resp, err := client.ExecuteAdhocWorkflow(context.Background(), BASExecuteAdhocRequest{
		FlowDefinition: json.RawMessage(minimalWorkflowJSON),
		Parameters:     map[string]interface{}{"project_root": "/repo/scenario"},
	}, true)
	if err != nil {
		t.Fatalf("ExecuteAdhocWorkflow returned error: %v", err)
	}
	if resp.ExecutionID != "exec-123" {
		t.Errorf("expected ExecutionID exec-123, got %s", resp.ExecutionID)
	}
	if resp.Status != "EXECUTION_STATUS_RUNNING" {
		t.Errorf("expected status string EXECUTION_STATUS_RUNNING, got %s", resp.Status)
	}
	if wf.gotReq == nil {
		t.Fatal("expected ExecuteAdhocRequest to be sent")
	}
	if got := wf.gotReq.GetOptions().GetRequiresVideo(); !got {
		t.Errorf("expected RequiresVideo=true; got false")
	}
	if got := wf.gotReq.GetParameters().GetProjectRoot(); got != "/repo/scenario" {
		t.Errorf("expected ProjectRoot=/repo/scenario; got %q", got)
	}
	if got := wf.gotReq.GetFlowDefinition().GetMetadata().GetName(); got != "smoke" {
		t.Errorf("expected parsed workflow name 'smoke'; got %q", got)
	}
}

func TestBASClient_ExecuteAdhocWorkflow_InvalidJSON(t *testing.T) {
	t.Parallel()

	client := newStubClient(t, &stubWorkflows{}, &stubExecutions{})
	_, err := client.ExecuteAdhocWorkflow(context.Background(), BASExecuteAdhocRequest{
		FlowDefinition: json.RawMessage(`not json`),
	}, false)
	if err == nil {
		t.Fatal("expected parse error for invalid JSON")
	}
}

func TestBASClient_GetExecutionStatus_MapsEnumToString(t *testing.T) {
	t.Parallel()

	exec := &stubExecutions{
		getExec: &bas_execution.Execution{
			ExecutionId: "exec-poll",
			Status:      bas_base.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		},
	}
	client := newStubClient(t, &stubWorkflows{}, exec)

	detail, err := client.GetExecutionStatus(context.Background(), "exec-poll")
	if err != nil {
		t.Fatalf("GetExecutionStatus: %v", err)
	}
	if detail.Status != "EXECUTION_STATUS_COMPLETED" {
		t.Errorf("expected COMPLETED status string; got %s", detail.Status)
	}
	if len(exec.getExecCalls) != 1 || exec.getExecCalls[0] != "exec-poll" {
		t.Errorf("expected one GetExecution call for exec-poll; got %v", exec.getExecCalls)
	}
}

func TestBASClient_PollExecutionCompletion_Terminates(t *testing.T) {
	t.Parallel()

	exec := &stubExecutions{
		getExec: &bas_execution.Execution{
			ExecutionId: "exec-poll",
			Status:      bas_base.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		},
	}
	client := newStubClient(t, &stubWorkflows{}, exec)

	detail, err := client.PollExecutionCompletion(context.Background(), "exec-poll", time.Millisecond)
	if err != nil {
		t.Fatalf("PollExecutionCompletion returned error: %v", err)
	}
	if detail.Status != "EXECUTION_STATUS_COMPLETED" {
		t.Errorf("expected terminal status; got %s", detail.Status)
	}
}

func TestBASClient_PollExecutionCompletion_PropagatesFailure(t *testing.T) {
	t.Parallel()

	failure := "step timed out"
	exec := &stubExecutions{
		getExec: &bas_execution.Execution{
			ExecutionId: "exec-fail",
			Status:      bas_base.ExecutionStatus_EXECUTION_STATUS_FAILED,
			Error:       &failure,
		},
	}
	client := newStubClient(t, &stubWorkflows{}, exec)

	detail, err := client.PollExecutionCompletion(context.Background(), "exec-fail", time.Millisecond)
	if err != nil {
		t.Fatalf("PollExecutionCompletion returned error: %v", err)
	}
	if detail.Status != "EXECUTION_STATUS_FAILED" {
		t.Errorf("expected FAILED; got %s", detail.Status)
	}
	if detail.Error != "step timed out" {
		t.Errorf("expected error 'step timed out'; got %s", detail.Error)
	}
}

func TestBASClient_GetScreenshots_MapsLegacyShape(t *testing.T) {
	t.Parallel()

	ts := timestamppb.New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	exec := &stubExecutions{
		screenshots: &bas_execution.GetScreenshotsResponse{
			ExecutionId: "exec-456",
			Total:       2,
			Screenshots: []*bas_execution.ExecutionScreenshot{
				{
					StepIndex: 0,
					StepLabel: stringPtr("Navigate"),
					Timestamp: ts,
					Screenshot: &bas_telemetry.TimelineScreenshot{
						ArtifactId:  "ss-1",
						Url:         "/api/v1/screenshots/artifacts/ss-1.png",
						ContentType: "image/png",
						Width:       100,
						Height:      50,
					},
				},
				{StepIndex: 1, StepLabel: stringPtr("Click")},
			},
		},
	}
	client := newStubClient(t, &stubWorkflows{}, exec)

	resp, err := client.GetScreenshots(context.Background(), "exec-456")
	if err != nil {
		t.Fatalf("GetScreenshots: %v", err)
	}
	if resp.Total != 2 || len(resp.Screenshots) != 2 {
		t.Fatalf("expected 2 screenshots; got total=%d len=%d", resp.Total, len(resp.Screenshots))
	}
	if got := resp.Screenshots[0].Screenshot.Url; got != "/api/v1/screenshots/artifacts/ss-1.png" {
		t.Errorf("expected Url passthrough; got %s", got)
	}
	if got := resp.Screenshots[0].StepLabel; got != "Navigate" {
		t.Errorf("expected StepLabel=Navigate; got %s", got)
	}
}

func TestBASClient_GetRecordedVideos_MapsLegacyShape(t *testing.T) {
	t.Parallel()

	size := int64(1024)
	exec := &stubExecutions{
		videos: &basapi.GetExecutionVideosResponse{
			ExecutionId: "exec-789",
			Videos: []*basapi.ExecutionFileArtifact{
				{ArtifactId: "vid-1", StorageUrl: "/storage/vid-1.webm", ContentType: "video/webm", SizeBytes: &size},
			},
		},
	}
	client := newStubClient(t, &stubWorkflows{}, exec)

	resp, err := client.GetRecordedVideos(context.Background(), "exec-789")
	if err != nil {
		t.Fatalf("GetRecordedVideos: %v", err)
	}
	if resp.ExecutionID != "exec-789" || len(resp.Videos) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Videos[0].StorageURL != "/storage/vid-1.webm" || resp.Videos[0].SizeBytes != 1024 {
		t.Errorf("video mapping wrong: %+v", resp.Videos[0])
	}
}

func TestBASClient_RPC_ErrorPropagation(t *testing.T) {
	t.Parallel()

	exec := &stubExecutions{err: errors.New("connection refused")}
	client := newStubClient(t, &stubWorkflows{}, exec)

	if _, err := client.GetExecutionStatus(context.Background(), "exec-x"); err == nil {
		t.Fatal("expected error to propagate")
	}
}

// -----------------------------------------------------------------------------
// Existing raw-HTTP helper tests retained: GetScreenshotData and GetVideoData
// continue to fetch bytes directly from BAS's asset proxy.
// -----------------------------------------------------------------------------

func TestBASClient_GetScreenshotData(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/screenshots/artifacts/ss-1.png", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	data, contentType, err := client.GetScreenshotData(context.Background(), "/api/v1/screenshots/artifacts/ss-1.png")
	if err != nil {
		t.Fatalf("GetScreenshotData returned error: %v", err)
	}
	if contentType != "image/png" {
		t.Errorf("expected content-type image/png, got %s", contentType)
	}
	if len(data) != len(pngBytes) {
		t.Errorf("expected %d bytes, got %d", len(pngBytes), len(data))
	}
}

func TestBASClient_GetScreenshotData_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/screenshots/artifacts/ss-1.png", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	if _, _, err := client.GetScreenshotData(context.Background(), "/api/v1/screenshots/artifacts/ss-1.png"); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestBASClient_GetVideoData(t *testing.T) {
	t.Parallel()

	mp4Bytes := []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}
	mux := http.NewServeMux()
	mux.HandleFunc("/storage/vid-1.webm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "video/webm")
		_, _ = w.Write(mp4Bytes)
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &BrowserAutomationClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "browser-automation-studio",
		},
	}

	data, contentType, err := client.GetVideoData(context.Background(), "/storage/vid-1.webm")
	if err != nil {
		t.Fatalf("GetVideoData returned error: %v", err)
	}
	if contentType != "video/webm" {
		t.Errorf("expected content-type video/webm, got %s", contentType)
	}
	if len(data) != len(mp4Bytes) {
		t.Errorf("expected %d bytes, got %d", len(mp4Bytes), len(data))
	}
}

func stringPtr(s string) *string { return &s }
