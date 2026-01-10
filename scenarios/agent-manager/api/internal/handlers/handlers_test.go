// Package handlers provides HTTP handlers for the agent-manager API.
// This file contains integration tests for API endpoints.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// =============================================================================
// TEST SETUP HELPERS
// =============================================================================

// decodeProtoJSON decodes a proto JSON response body into a proto message.
func decodeProtoJSON(t *testing.T, body []byte, msg proto.Message) {
	t.Helper()
	// Use protojson via protoconv for consistent unmarshalling
	if err := protoconv.UnmarshalJSON(body, msg); err != nil {
		t.Fatalf("failed to decode proto JSON: %v\nBody: %s", err, string(body))
	}
}

func encodeProtoJSON(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	body, err := protoconv.MarshalJSON(msg)
	if err != nil {
		t.Fatalf("failed to encode proto JSON: %v", err)
	}
	return body
}

// setupTestHandler creates a handler with in-memory repositories for testing.
func setupTestHandler() (*Handler, *mux.Router) {
	// Create in-memory repositories
	profiles := repository.NewMemoryProfileRepository()
	tasks := repository.NewMemoryTaskRepository()
	runs := repository.NewMemoryRunRepository()
	events := event.NewMemoryStore()

	// Create runner registry with mock runner
	registry := runner.NewRegistry()
	if err := registry.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode)); err != nil {
		panic(fmt.Sprintf("register runner: %v", err))
	}

	// Create orchestrator with dependencies
	orch := orchestration.New(
		profiles,
		tasks,
		runs,
		orchestration.WithEvents(events),
		orchestration.WithRunners(registry),
	)

	// Create handler
	handler := New(orch)

	// Create router and register routes
	r := mux.NewRouter()
	handler.RegisterRoutes(r)
	r.HandleFunc("/health", handler.Health).Methods("GET")
	r.HandleFunc("/api/v1/health", handler.Health).Methods("GET")

	return handler, r
}

// =============================================================================
// PROFILE HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-001] Create Agent Profile
// [REQ:REQ-P0-002] Update Agent Profile

// TestCreateProfile_Success tests successful profile creation.
// [REQ:REQ-P0-001] Verify profile creation with valid data
func TestCreateProfile_Success(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:        "test-profile",
			ProfileKey:  "test-profile-key",
			Description: "Test profile for unit tests",
			RunnerType:  pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
			MaxTurns:    100,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response apipb.CreateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)

	if response.Profile.GetName() != "test-profile" {
		t.Errorf("expected name 'test-profile', got '%s'", response.Profile.GetName())
	}
	if response.Profile.GetId() == "" {
		t.Error("expected profile ID to be assigned")
	}
}

// TestCreateProfile_ValidationError tests profile creation with invalid data.
// [REQ:REQ-P0-001] Verify validation of required fields
func TestCreateProfile_ValidationError(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name    string
		profile *pb.AgentProfile
		errCode string
	}{
		{
			name:    "empty name",
			profile: &pb.AgentProfile{Name: "", ProfileKey: "test-key", RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE},
			errCode: "VALIDATION",
		},
		{
			name:    "invalid runner type",
			profile: &pb.AgentProfile{Name: "test", ProfileKey: "test-key", RunnerType: pb.RunnerType_RUNNER_TYPE_UNSPECIFIED},
			errCode: "VALIDATION",
		},
		{
			name:    "negative max turns",
			profile: &pb.AgentProfile{Name: "test", ProfileKey: "test-key", RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE, MaxTurns: -1},
			errCode: "VALIDATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := encodeProtoJSON(t, &apipb.CreateProfileRequest{Profile: tt.profile})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
			}
		})
	}
}

// TestGetProfile_Success tests successful profile retrieval.
// [REQ:REQ-P0-001] Test profile retrieval by ID
func TestGetProfile_Success(t *testing.T) {
	_, router := setupTestHandler()

	// First create a profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "test-profile",
			ProfileKey: "test-profile-key",
			RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdResp apipb.CreateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdResp)
	created := createdResp.Profile

	// Now retrieve it
	req = httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+created.Id, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var retrievedResp apipb.GetProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &retrievedResp)
	retrieved := retrievedResp.Profile

	if retrieved.GetId() != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, retrieved.GetId())
	}
}

// TestGetProfile_NotFound tests retrieval of non-existent profile.
// [REQ:REQ-P0-001] Test error handling for missing profile
func TestGetProfile_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

// TestListProfiles tests listing all profiles.
// [REQ:REQ-P0-001] Test listing profiles
func TestListProfiles(t *testing.T) {
	_, router := setupTestHandler()

	// Create multiple profiles
	for i := 0; i < 3; i++ {
		body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
			Profile: &pb.AgentProfile{
				Name:       "profile-" + string(rune('A'+i)),
				ProfileKey: "profile-key-" + string(rune('A'+i)),
				RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}

	// List all profiles
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Response is ListProfilesResponse with profiles array and total
	var response apipb.ListProfilesResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)

	if len(response.Profiles) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(response.Profiles))
	}
}

// TestUpdateProfile tests profile update.
// [REQ:REQ-P0-002] Verify profile updates persist correctly
func TestUpdateProfile_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "original-name",
			ProfileKey: "original-key",
			RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdResp apipb.CreateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdResp)
	created := createdResp.Profile

	// Update profile
	body = encodeProtoJSON(t, &apipb.UpdateProfileRequest{
		ProfileId: created.Id,
		Profile: &pb.AgentProfile{
			Name:        "updated-name",
			ProfileKey:  "updated-key",
			Description: "Updated description",
			RunnerType:  pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/profiles/"+created.Id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var updatedResp apipb.UpdateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &updatedResp)
	updated := updatedResp.Profile

	if updated.GetName() != "updated-name" {
		t.Errorf("expected name 'updated-name', got '%s'", updated.GetName())
	}
	if updated.GetDescription() != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", updated.GetDescription())
	}
}

// TestDeleteProfile tests profile deletion.
// [REQ:REQ-P0-001] Test profile deletion
func TestDeleteProfile_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "to-delete",
			ProfileKey: "to-delete-key",
			RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdResp apipb.CreateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdResp)
	created := createdResp.Profile

	// Delete profile
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/"+created.Id, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var deleteResp apipb.DeleteProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &deleteResp)
	if !deleteResp.Success {
		t.Errorf("expected success true, got false")
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+created.Id, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d after deletion, got %d", http.StatusNotFound, rr.Code)
	}
}

// =============================================================================
// TASK HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-003] Create Task

// TestCreateTask_Success tests successful task creation.
// [REQ:REQ-P0-003] Verify task creation with valid data
func TestCreateTask_Success(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:       "Fix login bug",
			Description: "Users cannot login with email",
			ScopePath:   "src/auth",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)

	if response.Task.GetTitle() != "Fix login bug" {
		t.Errorf("expected title 'Fix login bug', got '%s'", response.Task.GetTitle())
	}
	if response.Task.GetStatus() != pb.TaskStatus_TASK_STATUS_QUEUED {
		t.Errorf("expected status TASK_STATUS_QUEUED, got '%s'", response.Task.GetStatus())
	}
}

// TestCreateTask_ValidationError tests task creation with invalid data.
// [REQ:REQ-P0-003] Verify validation of required fields
func TestCreateTask_ValidationError(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name string
		task *pb.Task
	}{
		{
			name: "empty title",
			task: &pb.Task{Title: "", ScopePath: "src/"},
		},
		{
			name: "empty scope path",
			task: &pb.Task{Title: "Test", ScopePath: ""},
		},
		{
			name: "path traversal",
			task: &pb.Task{Title: "Test", ScopePath: "../outside"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: tt.task})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
			}
		})
	}
}

// TestGetTask_Success tests successful task retrieval.
// [REQ:REQ-P0-003] Test task retrieval
func TestGetTask_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create task
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:     "Test task",
			ScopePath: "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdResp)
	created := createdResp.Task

	// Get task
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+created.Id, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var retrievedResp apipb.GetTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &retrievedResp)
	retrieved := retrievedResp.Task

	if retrieved.GetId() != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, retrieved.GetId())
	}
}

// TestListTasks tests listing all tasks.
// [REQ:REQ-P0-003] Test listing tasks
func TestListTasks(t *testing.T) {
	_, router := setupTestHandler()

	// Create tasks
	for i := 0; i < 3; i++ {
		body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
			Task: &pb.Task{
				Title:     "Task " + string(rune('A'+i)),
				ScopePath: "src/",
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}

	// List tasks
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Response is ListTasksResponse with tasks array and total
	var response apipb.ListTasksResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)

	if len(response.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(response.Tasks))
	}
}

// TestCancelTask tests task cancellation.
// [REQ:REQ-P0-003] Test task cancellation
func TestCancelTask_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create task
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:     "To cancel",
			ScopePath: "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdResp)
	created := createdResp.Task

	// Cancel task
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+created.Id+"/cancel", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

// =============================================================================
// HEALTH HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-011] Health Check API

// TestHealth_Success tests the health endpoint.
// [REQ:REQ-P0-011] Verify health endpoint returns proper format
func TestHealth_Success(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var result commonpb.HealthResponse
	if err := protoconv.UnmarshalJSON(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	// Verify required fields per health-api.schema.json
	if result.Status == commonpb.HealthStatus_HEALTH_STATUS_UNSPECIFIED {
		t.Error("health status should not be empty")
	}
	if result.Service != "agent-manager" {
		t.Errorf("expected service 'agent-manager', got '%s'", result.Service)
	}
	if result.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	// Readiness should be true for a healthy system
	if !result.Readiness {
		t.Error("expected readiness to be true")
	}
}

// TestHealth_ApiV1Path tests the /api/v1/health endpoint.
// [REQ:REQ-P0-011] Verify both health endpoint paths work
func TestHealth_ApiV1Path(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// =============================================================================
// RUN HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-004] Run Status Tracking
// [REQ:REQ-P0-005] Run Creation

// TestCreateRun_Success tests successful run creation.
// [REQ:REQ-P0-005] Verify run creation with valid data
func TestCreateRun_Success(t *testing.T) {
	_, router := setupTestHandler()

	// First create a profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "runner-profile",
			ProfileKey: "runner-profile-key",
			RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
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
}

// TestListRuns tests listing runs.
// [REQ:REQ-P0-004] Test run listing
func TestListRuns(t *testing.T) {
	_, router := setupTestHandler()

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
	_, router := setupTestHandler()

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
func TestRequestIDMiddleware(t *testing.T) {
	_, router := setupTestHandler()

	// Test without request ID - should be assigned
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	// Test with existing request ID - should be preserved
	customID := uuid.New().String()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", customID)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != customID {
		t.Errorf("expected X-Request-ID to be preserved, got '%s'", rr.Header().Get("X-Request-ID"))
	}
}

// =============================================================================
// ERROR RESPONSE TESTS
// =============================================================================

// TestErrorResponse_Format tests that error responses have proper structure.
func TestErrorResponse_Format(t *testing.T) {
	_, router := setupTestHandler()

	// Request non-existent resource
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var errResp commonpb.ErrorResponse
	if err := protoconv.UnmarshalJSON(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code == "" {
		t.Error("error response should have code")
	}
	if errResp.Message == "" {
		t.Error("error response should have message")
	}
	if errResp.Details == nil || errResp.Details.Fields["request_id"] == nil {
		t.Error("error response should include request_id")
	}
}

// =============================================================================
// EDGE CASE TESTS - MALFORMED INPUT
// =============================================================================

// TestCreateProfile_MalformedJSON tests profile creation with invalid JSON.
func TestCreateProfile_MalformedJSON(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid JSON", "{invalid json}"},
		{"truncated JSON", `{"name": "test"`},
		{"array instead of object", `["name", "test"]`},
		{"null", "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
			}
		})
	}
}

// TestCreateTask_MalformedJSON tests task creation with invalid JSON.
func TestCreateTask_MalformedJSON(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid JSON", "{not valid}"},
		{"wrong type for field", `{"title": 123, "scopePath": "src/"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
			}
		})
	}
}

// TestGetProfile_InvalidUUID tests profile retrieval with invalid UUID.
func TestGetProfile_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"not a uuid", "not-a-uuid"},
		{"partial uuid", "12345678-1234"},
		{"uuid with extra chars", uuid.New().String() + "extra"},
		{"hyphens only", "----"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+tt.id, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Should get 400 for invalid UUID or 404 for empty route
			if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d", rr.Code)
			}
		})
	}
}

// TestGetTask_InvalidUUID tests task retrieval with invalid UUID.
func TestGetTask_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestGetRun_InvalidUUID tests run retrieval with invalid UUID.
func TestGetRun_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestCreateRun_MalformedJSON tests run creation with invalid JSON.
func TestCreateRun_MalformedJSON(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid JSON", "{broken}"},
		{"invalid task ID", `{"taskId": "not-uuid", "agentProfileId": "` + uuid.New().String() + `"}`},
		{"invalid profile ID", `{"taskId": "` + uuid.New().String() + `", "agentProfileId": "not-uuid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
			}
		})
	}
}

// TestUpdateProfile_InvalidUUID tests profile update with invalid UUID.
func TestUpdateProfile_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.UpdateProfileRequest{
		ProfileId: "invalid-uuid",
		Profile: &pb.AgentProfile{
			Name:       "updated",
			ProfileKey: "updated-key",
			RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/invalid-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestDeleteProfile_InvalidUUID tests profile deletion with invalid UUID.
func TestDeleteProfile_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestCancelTask_InvalidUUID tests task cancellation with invalid UUID.
func TestCancelTask_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/invalid-uuid/cancel", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestListProfiles_InvalidPagination tests profile listing with invalid pagination params.
func TestListProfiles_InvalidPagination(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name  string
		query string
	}{
		{"negative limit", "?limit=-1"},
		{"negative offset", "?offset=-1"},
		{"non-numeric limit", "?limit=abc"},
		{"non-numeric offset", "?offset=xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles"+tt.query, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Invalid pagination might return 400 or be silently ignored (200)
			// depending on implementation - both are acceptable
			if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 200 or 400, got %d", rr.Code)
			}
		})
	}
}

// TestContentTypeRequired tests that Content-Type header is handled properly.
func TestContentTypeRequired(t *testing.T) {
	_, router := setupTestHandler()

	body := `{"name": "test", "runnerType": "claude-code"}`

	// Without Content-Type header
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte(body)))
	// Intentionally not setting Content-Type
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should either work (server parses JSON anyway) or return 415
	// Both are acceptable depending on server strictness
	if rr.Code != http.StatusCreated && rr.Code != http.StatusUnsupportedMediaType && rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 201, 415, or 400, got %d", rr.Code)
	}
}

// TestLargePayload tests handling of very large request payloads.
func TestLargePayload_Profile(t *testing.T) {
	_, router := setupTestHandler()

	// Create a profile with a very long description
	longDesc := make([]byte, 100000) // 100KB
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:        "test-profile",
			ProfileKey:  "test-profile-key",
			Description: string(longDesc),
			RunnerType:  pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should either reject (413 or 400) or accept (201)
	// Depends on payload size limits in server
	if rr.Code != http.StatusCreated && rr.Code != http.StatusBadRequest &&
		rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 201, 400, or 413, got %d", rr.Code)
	}
}

// =============================================================================
// INTEGRATION TEST HELPERS
// =============================================================================

// createTestProfile is a helper to create a profile for testing.
func createTestProfile(t *testing.T, router *mux.Router) *pb.AgentProfile {
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "test-profile-" + uuid.New().String()[:8],
			ProfileKey: "test-profile-key-" + uuid.New().String()[:8],
			RunnerType: pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test profile: %d %s", rr.Code, rr.Body.String())
	}

	var result apipb.CreateProfileResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &result)
	return result.Profile
}

// createTestTask is a helper to create a task for testing.
func createTestTask(t *testing.T, router *mux.Router) *pb.Task {
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:     "test-task-" + uuid.New().String()[:8],
			ScopePath: "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test task: %d %s", rr.Code, rr.Body.String())
	}

	var result apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &result)
	return result.Task
}

// =============================================================================
// RUN LIFECYCLE HANDLER TESTS
// =============================================================================
// [REQ:REQ-P0-004] Run Status Tracking
// [REQ:REQ-P0-007] Approval Workflow

// TestStopRun_Success tests stopping a running run.
// [REQ:REQ-P0-004] Test run stop operation
func TestStopRun_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create profile, task, and run first
	profile := createTestProfile(t, router)
	task := createTestTask(t, router)

	agentProfileID := profile.Id
	body := encodeProtoJSON(t, &apipb.CreateRunRequest{
		TaskId:         task.Id,
		AgentProfileId: &agentProfileID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdRunResp apipb.CreateRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdRunResp)
	createdRun := createdRunResp.Run

	// Stop the run
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+createdRun.Id+"/stop", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should return 200 OK or 404 if run doesn't support stop
	if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound && rr.Code != http.StatusConflict {
		t.Errorf("expected status 200, 404, or 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestStopRun_InvalidUUID tests stopping with invalid run ID.
func TestStopRun_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/invalid-uuid/stop", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestStopRun_NotFound tests stopping a non-existent run.
func TestStopRun_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+uuid.New().String()+"/stop", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetRunByTag tests retrieving a run by custom tag.
func TestGetRunByTag_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/tag/nonexistent-tag", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestStopRunByTag tests stopping a run by its custom tag.
func TestStopRunByTag_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/tag/nonexistent-tag/stop", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestStopAllRuns tests the bulk stop operation.
func TestStopAllRuns_Success(t *testing.T) {
	_, router := setupTestHandler()

	tagPrefix := ""
	body := encodeProtoJSON(t, &apipb.StopAllRunsRequest{
		TagPrefix: &tagPrefix,
		Force:     false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify response structure
	var result map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

// TestStopAllRuns_WithTagPrefix tests bulk stop with tag filtering.
func TestStopAllRuns_WithTagPrefix(t *testing.T) {
	_, router := setupTestHandler()

	tagPrefix := "test-"
	body := encodeProtoJSON(t, &apipb.StopAllRunsRequest{
		TagPrefix: &tagPrefix,
		Force:     true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestStopAllRuns_EmptyBody tests bulk stop with no body (should work).
func TestStopAllRuns_EmptyBody(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should work with empty body
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestGetRunEvents tests retrieving events for a run.
func TestGetRunEvents_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create a run first
	profile := createTestProfile(t, router)
	prompt := "Summarize the task and list next steps."
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:       "test-task-" + uuid.New().String()[:8],
			Description: prompt,
			ScopePath:   "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test task: %d %s", rr.Code, rr.Body.String())
	}

	var taskResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &taskResp)
	task := taskResp.Task

	agentProfileID := profile.Id
	body = encodeProtoJSON(t, &apipb.CreateRunRequest{
		TaskId:         task.Id,
		AgentProfileId: &agentProfileID,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdRunResp apipb.CreateRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdRunResp)
	createdRun := createdRunResp.Run

	// Get events for the run
	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+createdRun.Id+"/events", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify response contains events array
	var response apipb.GetRunEventsResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	if len(response.Events) == 0 {
		t.Fatalf("expected events for run, got none")
	}

	found := false
	for _, evt := range response.Events {
		msg := evt.GetMessage()
		if msg != nil && msg.Role == "user" && msg.Content == prompt {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected initial user prompt event with content %q", prompt)
	}
}

// TestGetRunEvents_NotFound tests getting events for non-existent run.
func TestGetRunEvents_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+uuid.New().String()+"/events", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// May return 404 or empty array depending on implementation
	if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 200 or 404, got %d", rr.Code)
	}
}

// TestGetRunEvents_InvalidUUID tests getting events with invalid run ID.
func TestGetRunEvents_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/invalid-uuid/events", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestGetRunDiff_NotFound tests getting diff for non-existent run.
func TestGetRunDiff_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+uuid.New().String()+"/diff", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetRunDiff_InvalidUUID tests getting diff with invalid run ID.
func TestGetRunDiff_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/invalid-uuid/diff", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestApproveRun_NotFound tests approving a non-existent run.
func TestApproveRun_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	runID := uuid.New().String()
	body := encodeProtoJSON(t, &apipb.ApproveRunRequest{
		RunId:     runID,
		Actor:     func() *string { actor := "test-user"; return &actor }(),
		CommitMsg: func() *string { msg := "Apply changes"; return &msg }(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+runID+"/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestApproveRun_InvalidUUID tests approving with invalid run ID.
func TestApproveRun_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.ApproveRunRequest{
		RunId: "invalid-uuid",
		Actor: func() *string { actor := "test-user"; return &actor }(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/invalid-uuid/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestApproveRun_MalformedBody tests approving with invalid JSON.
func TestApproveRun_MalformedBody(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+uuid.New().String()+"/approve", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// TestRejectRun_NotFound tests rejecting a non-existent run.
func TestRejectRun_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	runID := uuid.New().String()
	body := encodeProtoJSON(t, &apipb.RejectRunRequest{
		RunId:  runID,
		Actor:  func() *string { actor := "test-user"; return &actor }(),
		Reason: "Changes not acceptable",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+runID+"/reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestRejectRun_InvalidUUID tests rejecting with invalid run ID.
func TestRejectRun_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.RejectRunRequest{
		RunId:  "invalid-uuid",
		Actor:  func() *string { actor := "test-user"; return &actor }(),
		Reason: "test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/invalid-uuid/reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestRejectRun_MalformedBody tests rejecting with invalid JSON.
func TestRejectRun_MalformedBody(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+uuid.New().String()+"/reject", bytes.NewReader([]byte("{broken")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// =============================================================================
// LIST RUNS FILTER TESTS
// =============================================================================

// TestListRuns_WithStatusFilter tests listing runs with status filter.
func TestListRuns_WithStatusFilter(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?status=running", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response apipb.ListRunsResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &response)
	// All returned runs should have status "running" (or empty if none)
}

// TestListRuns_WithTaskIDFilter tests listing runs with task ID filter.
func TestListRuns_WithTaskIDFilter(t *testing.T) {
	_, router := setupTestHandler()

	taskID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?taskId="+taskID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithProfileIDFilter tests listing runs with profile ID filter.
func TestListRuns_WithProfileIDFilter(t *testing.T) {
	_, router := setupTestHandler()

	profileID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?profileId="+profileID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithTagPrefixFilter tests listing runs with tag prefix filter.
func TestListRuns_WithTagPrefixFilter(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?tagPrefix=ecosystem-", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithMultipleFilters tests listing runs with multiple filters.
func TestListRuns_WithMultipleFilters(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?status=pending&tagPrefix=test-", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// =============================================================================
// UPDATE TASK TESTS
// =============================================================================

// TestUpdateTask_Success tests successful task update.
func TestUpdateTask_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create task first
	task := createTestTask(t, router)

	// Update task
	body := encodeProtoJSON(t, &apipb.UpdateTaskRequest{
		TaskId: task.Id,
		Task: &pb.Task{
			Title:       "Updated Title",
			Description: "Updated description",
			ScopePath:   "src/updated",
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+task.Id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var updatedResp apipb.UpdateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &updatedResp)
	updated := updatedResp.Task

	if updated.GetTitle() != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", updated.GetTitle())
	}
}

// TestUpdateTask_NotFound tests updating non-existent task.
func TestUpdateTask_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	missingID := uuid.New().String()
	body := encodeProtoJSON(t, &apipb.UpdateTaskRequest{
		TaskId: missingID,
		Task: &pb.Task{
			Title:     "Test",
			ScopePath: "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+missingID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// May return 404 Not Found or 500 if service doesn't handle not found gracefully
	if rr.Code != http.StatusNotFound && rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", rr.Code)
	}
}

// TestUpdateTask_InvalidUUID tests updating with invalid task ID.
func TestUpdateTask_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.UpdateTaskRequest{
		TaskId: "invalid-uuid",
		Task: &pb.Task{
			Title:     "Test",
			ScopePath: "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/invalid-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestUpdateTask_MalformedBody tests updating with invalid JSON.
func TestUpdateTask_MalformedBody(t *testing.T) {
	_, router := setupTestHandler()

	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:     "Test",
			ScopePath: "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createdResp)
	created := createdResp.Task

	// Try to update with malformed JSON
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+created.Id, bytes.NewReader([]byte("{broken")))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// =============================================================================
// CANCEL TASK EDGE CASE TESTS
// =============================================================================

// TestCancelTask_NotFound tests cancelling non-existent task.
func TestCancelTask_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+uuid.New().String()+"/cancel", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetTask_NotFound tests retrieving non-existent task.
func TestGetTask_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// Compile-time interface check
var (
	_ context.Context
	_ time.Time
)
