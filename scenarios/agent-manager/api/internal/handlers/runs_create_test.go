package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestCreateRun_Success(t *testing.T) {
	_, router := setupTestHandler(t)

	// First create a profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "runner-profile",
			ProfileKey: "runner-profile-key", RoleRef: "code.default",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdProfileResp apipb.CreateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdProfileResp)
	createdProfile := createdProfileResp.Profile

	// Create a task
	body = encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:     "Test task for run",
			ScopePath: "src/",
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdTaskResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdTaskResp)
	createdTask := createdTaskResp.Task

	// Create a run
	agentProfileID := createdProfile.Id
	body = encodeProtoJSON(t, &apipb.CreateRunRequest{
		TaskId:         createdTask.Id,
		AgentProfileId: &agentProfileID,
		InlineConfig: &pb.RunConfigOverrides{ResultSpec: &pb.ResultSpec{
			Version: "result-spec/v1", Kind: pb.ResultSpecKind_RESULT_SPEC_KIND_CLASSIFICATION,
			ClassificationValues: []string{"complete", "blocked"},
		}, Model: func() *string { model := "model-override"; return &model }()},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var createdRunResp apipb.CreateRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdRunResp)
	createdRun := createdRunResp.Run

	if createdRun.GetTaskId() != createdTask.Id {
		t.Errorf("expected task ID %s, got %s", createdTask.Id, createdRun.GetTaskId())
	}
	if createdRun.AgentProfileId == nil || *createdRun.AgentProfileId != createdProfile.Id {
		t.Errorf("expected profile ID %s, got %v", createdProfile.Id, createdRun.AgentProfileId)
	}
	if createdRun.ResolvedConfig == nil || createdRun.ResolvedConfig.ResultSpec == nil || createdRun.ResolvedConfig.ResultSpec.SchemaDigest == "" {
		t.Fatalf("expected normalized result spec in create response, got %+v", createdRun.ResolvedConfig)
	}
	if len(createdRun.ResolvedConfig.ResultSpec.ClassificationValues) != 0 || len(createdRun.ResolvedConfig.ResultSpec.Schema) == 0 {
		t.Fatalf("result spec was not canonicalized: %+v", createdRun.ResolvedConfig.ResultSpec)
	}
	if got := createdRun.GetResolvedConfig().GetModel(); got != "model-override" {
		t.Fatalf("expected inline model override to be resolved, got %q", got)
	}
}

func TestPartialApproveRunRejectsInvalidRequestsBeforeCallingService(t *testing.T) {
	_, router := setupTestHandler(t)
	runID := uuid.New()
	otherRunID := uuid.New()
	fileID := uuid.New()

	tests := []struct {
		name string
		path string
		body []byte
	}{
		{
			name: "invalid URL run ID",
			path: "/api/v1/runs/not-a-uuid/partial-approve",
			body: encodeProtoJSON(t, &apipb.PartialApproveRunRequest{FileIds: []string{fileID.String()}}),
		},
		{
			name: "malformed JSON",
			path: "/api/v1/runs/" + runID.String() + "/partial-approve",
			body: []byte(`{`),
		},
		{
			name: "mismatched run ID",
			path: "/api/v1/runs/" + runID.String() + "/partial-approve",
			body: encodeProtoJSON(t, &apipb.PartialApproveRunRequest{RunId: otherRunID.String(), FileIds: []string{fileID.String()}}),
		},
		{
			name: "no selected files",
			path: "/api/v1/runs/" + runID.String() + "/partial-approve",
			body: encodeProtoJSON(t, &apipb.PartialApproveRunRequest{RunId: runID.String()}),
		},
		{
			name: "invalid file ID",
			path: "/api/v1/runs/" + runID.String() + "/partial-approve",
			body: encodeProtoJSON(t, &apipb.PartialApproveRunRequest{RunId: runID.String(), FileIds: []string{"not-a-uuid"}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(tt.body)))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestSyncRunFromSandboxRejectsInvalidRequestsBeforeCallingService(t *testing.T) {
	_, router := setupTestHandler(t)
	runID := uuid.New()
	otherRunID := uuid.New()

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "invalid URL run ID", path: "/api/v1/runs/not-a-uuid/sandbox-sync", body: `{"status":"approved"}`},
		{name: "malformed JSON", path: "/api/v1/runs/" + runID.String() + "/sandbox-sync", body: `{`},
		{name: "mismatched run ID", path: "/api/v1/runs/" + runID.String() + "/sandbox-sync", body: `{"runId":"` + otherRunID.String() + `","status":"approved"}`},
		{name: "missing status", path: "/api/v1/runs/" + runID.String() + "/sandbox-sync", body: `{}`},
		{name: "invalid sandbox ID", path: "/api/v1/runs/" + runID.String() + "/sandbox-sync", body: `{"status":"approved","sandboxId":"not-a-uuid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestSyncRunFromSandboxUpdatesApprovalState(t *testing.T) {
	_, router := setupTestHandler(t)

	t.Run("fully approved", func(t *testing.T) {
		run := createRunnableTestRun(t, router)
		rr := httptest.NewRecorder()
		body := `{"runId":"` + run.GetId() + `","status":"approved","actor":"reviewer"}`
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.GetId()+"/sandbox-sync", strings.NewReader(body)))
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response["status"] != "synced" || response["runStatus"] != string(domain.RunStatusComplete) || response["approvalState"] != string(domain.ApprovalStateApproved) {
			t.Fatalf("unexpected response=%v", response)
		}
	})

	t.Run("partially approved", func(t *testing.T) {
		run := createRunnableTestRun(t, router)
		rr := httptest.NewRecorder()
		body := `{"status":"approved","isPartial":true}`
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.GetId()+"/sandbox-sync", strings.NewReader(body)))
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response["approvalState"] != string(domain.ApprovalStatePartiallyApproved) {
			t.Fatalf("unexpected response=%v", response)
		}
	})

	t.Run("unsupported status", func(t *testing.T) {
		run := createRunnableTestRun(t, router)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.GetId()+"/sandbox-sync", strings.NewReader(`{"status":"unknown"}`)))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestDeleteRunMessageDeletesFirstMessageThroughHTTPContract(t *testing.T) {
	_, router, repos, eventStore := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "message delete", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	message := domain.NewMessageEvent(run.ID, "assistant", "first durable message")
	if err := eventStore.Append(ctx, run.ID, message); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	path := "/api/v1/runs/" + run.ID.String() + "/messages/" + message.ID.String() + "/delete"
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("delete message status=%d body=%s", rr.Code, rr.Body.String())
	}
	var response pb.DeleteRunMessageResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	if !response.GetSuccess() {
		t.Fatalf("delete response=%+v", &response)
	}
	events, err := eventStore.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[1].EventType != domain.EventTypeMessageDeleted {
		t.Fatalf("events=%+v", events)
	}

	for _, badPath := range []string{
		"/api/v1/runs/not-a-uuid/messages/" + message.ID.String() + "/delete",
		"/api/v1/runs/" + run.ID.String() + "/messages/not-a-uuid/delete",
	} {
		invalid := httptest.NewRecorder()
		router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, badPath, nil))
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", badPath, invalid.Code, invalid.Body.String())
		}
	}
}

func TestGetAuditTranscriptServesBoundedPersistedEvidence(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "audit transcript", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "transcript.log")
	if err := os.WriteFile(transcript, []byte("abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, TranscriptPath: transcript}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}

	type transcriptResponse struct {
		RunID       string `json:"runId"`
		MaxBytes    int    `json:"maxBytes"`
		Truncated   bool   `json:"truncated"`
		ContentHash string `json:"contentHash"`
		Content     string `json:"content"`
	}
	bounded := httptest.NewRecorder()
	router.ServeHTTP(bounded, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID.String()+"/audit-transcript?maxBytes=4", nil))
	if bounded.Code != http.StatusOK {
		t.Fatalf("bounded transcript status=%d body=%s", bounded.Code, bounded.Body.String())
	}
	var response transcriptResponse
	if err := json.Unmarshal(bounded.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.RunID != run.ID.String() || response.MaxBytes != 4 || !response.Truncated || response.Content != "abcd" || !strings.HasPrefix(response.ContentHash, "sha256:") {
		t.Fatalf("bounded transcript=%+v", response)
	}

	for _, path := range []string{
		"/api/v1/runs/not-a-uuid/audit-transcript",
		"/api/v1/runs/" + run.ID.String() + "/audit-transcript?maxBytes=0",
		"/api/v1/runs/" + run.ID.String() + "/audit-transcript?maxBytes=65537",
	} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}

	noTranscript := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted}
	if err := repos.Runs.Create(ctx, noTranscript); err != nil {
		t.Fatal(err)
	}
	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+noTranscript.ID.String()+"/audit-transcript", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing transcript status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func TestDeletePendingRunIsRejectedWithLifecycleGuidance(t *testing.T) {
	release := make(chan struct{})
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.ExecuteFunc = func(ctx context.Context, _ runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		select {
		case <-release:
			return &runner.ExecuteResult{ExitCode: 0}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	t.Cleanup(func() { close(release) })
	_, router := setupTestHandlerWithRunner(t, mock)
	profileBody := encodeProtoJSON(t, &apipb.CreateProfileRequest{Profile: &pb.AgentProfile{
		Name: "delete-run", ProfileKey: "delete-run", RoleRef: "code.default",
		SandboxConfig: &pb.SandboxConfig{Mode: pb.SandboxMode_SANDBOX_MODE_OFF},
	}})
	profileRR := httptest.NewRecorder()
	router.ServeHTTP(profileRR, httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(profileBody)))
	var profile apipb.CreateProfileResponse
	decodeProtoJSON(t, profileRR.Body.Bytes(), &profile)
	taskBody := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &pb.Task{Title: "delete-run", ScopePath: "src/delete"}})
	taskRR := httptest.NewRecorder()
	router.ServeHTTP(taskRR, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody)))
	var task apipb.CreateTaskResponse
	decodeProtoJSON(t, taskRR.Body.Bytes(), &task)
	profileID := profile.Profile.GetId()
	runBody := encodeProtoJSON(t, &apipb.CreateRunRequest{TaskId: task.Task.GetId(), AgentProfileId: &profileID})
	runRR := httptest.NewRecorder()
	router.ServeHTTP(runRR, httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(runBody)))
	if runRR.Code != http.StatusCreated {
		t.Fatalf("run status=%d body=%s", runRR.Code, runRR.Body.String())
	}
	var run apipb.CreateRunResponse
	decodeProtoJSON(t, runRR.Body.Bytes(), &run)
	deleteRR := httptest.NewRecorder()
	router.ServeHTTP(deleteRR, httptest.NewRequest(http.MethodDelete, "/api/v1/runs/"+run.Run.GetId(), nil))
	if deleteRR.Code != http.StatusConflict {
		t.Fatalf("delete pending status=%d body=%s", deleteRR.Code, deleteRR.Body.String())
	}
}

func TestCreateRun_InvalidNestedEnumIncludesParseDetail(t *testing.T) {
	_, router := setupTestHandler(t)

	body := []byte(`{
		"task_id":"11111111-1111-1111-1111-111111111111",
		"profile_ref":{
			"profile_key":"probe",
			"defaults":{
				"name":"Probe",
				"profile_key":"probe",
				"role_ref":"code.default",
				"sandbox_config":{"mode":"off"}
			}
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "enum field mode") || !strings.Contains(rr.Body.String(), "off") {
		t.Fatalf("expected parse detail to identify nested enum value, got: %s", rr.Body.String())
	}
}

// TestListRuns tests listing runs.
// [REQ:REQ-P0-004] Test run listing
func TestListRuns(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestGetRunnerStatus tests the runner status endpoint.
// [REQ:REQ-P0-006] Test runner status
func TestGetRunnerStatus(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runners", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response apipb.GetRunnerStatusResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)

	// Should have at least one runner (the mock)
	if len(response.Runners) == 0 {
		t.Error("expected at least one runner status")
	}
}

// =============================================================================
// REQUEST ID MIDDLEWARE TESTS
// =============================================================================

// TestRequestIDMiddleware tests that request IDs are properly assigned.
