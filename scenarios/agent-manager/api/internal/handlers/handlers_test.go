// Package handlers provides HTTP handlers for the agent-manager API.
// This file contains integration tests for API endpoints.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// =============================================================================
// TEST SETUP HELPERS
// =============================================================================

// setupTestHandler creates a handler with in-memory repositories for testing.
func setupTestHandler() (*Handler, *mux.Router) {
	// Create in-memory repositories
	profiles := repository.NewMemoryProfileRepository()
	tasks := repository.NewMemoryTaskRepository()
	runs := repository.NewMemoryRunRepository()
	events := event.NewMemoryStore()

	// Create runner registry with mock runner
	registry := runner.NewRegistry()
	registry.Register(runner.NewMockRunner(domain.RunnerTypeClaudeCode))

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

	profile := map[string]interface{}{
		"name":        "test-profile",
		"description": "Test profile for unit tests",
		"runnerType":  "claude-code",
		"maxTurns":    100,
	}
	body, _ := json.Marshal(profile)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var result domain.AgentProfile
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Name != "test-profile" {
		t.Errorf("expected name 'test-profile', got '%s'", result.Name)
	}
	if result.ID == uuid.Nil {
		t.Error("expected profile ID to be assigned")
	}
}

// TestCreateProfile_ValidationError tests profile creation with invalid data.
// [REQ:REQ-P0-001] Verify validation of required fields
func TestCreateProfile_ValidationError(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name    string
		profile map[string]interface{}
		errCode string
	}{
		{
			name:    "empty name",
			profile: map[string]interface{}{"name": "", "runnerType": "claude-code"},
			errCode: "VALIDATION",
		},
		{
			name:    "invalid runner type",
			profile: map[string]interface{}{"name": "test", "runnerType": "invalid"},
			errCode: "VALIDATION",
		},
		{
			name:    "negative max turns",
			profile: map[string]interface{}{"name": "test", "runnerType": "claude-code", "maxTurns": -1},
			errCode: "VALIDATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.profile)
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
	profile := map[string]interface{}{
		"name":       "test-profile",
		"runnerType": "claude-code",
	}
	body, _ := json.Marshal(profile)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&created)

	// Now retrieve it
	req = httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+created.ID.String(), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var retrieved domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&retrieved)

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
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
		profile := map[string]interface{}{
			"name":       "profile-" + string(rune('A'+i)),
			"runnerType": "claude-code",
		}
		body, _ := json.Marshal(profile)
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

	var profiles []*domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&profiles)

	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(profiles))
	}
}

// TestUpdateProfile tests profile update.
// [REQ:REQ-P0-002] Verify profile updates persist correctly
func TestUpdateProfile_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create profile
	profile := map[string]interface{}{
		"name":       "original-name",
		"runnerType": "claude-code",
	}
	body, _ := json.Marshal(profile)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&created)

	// Update profile
	updateData := map[string]interface{}{
		"name":        "updated-name",
		"description": "Updated description",
		"runnerType":  "claude-code",
	}
	body, _ = json.Marshal(updateData)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/profiles/"+created.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var updated domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&updated)

	if updated.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got '%s'", updated.Name)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", updated.Description)
	}
}

// TestDeleteProfile tests profile deletion.
// [REQ:REQ-P0-001] Test profile deletion
func TestDeleteProfile_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create profile
	profile := map[string]interface{}{
		"name":       "to-delete",
		"runnerType": "claude-code",
	}
	body, _ := json.Marshal(profile)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&created)

	// Delete profile
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/"+created.ID.String(), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+created.ID.String(), nil)
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

	task := map[string]interface{}{
		"title":       "Fix login bug",
		"description": "Users cannot login with email",
		"scopePath":   "src/auth",
	}
	body, _ := json.Marshal(task)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var result domain.Task
	json.NewDecoder(rr.Body).Decode(&result)

	if result.Title != "Fix login bug" {
		t.Errorf("expected title 'Fix login bug', got '%s'", result.Title)
	}
	if result.Status != domain.TaskStatusQueued {
		t.Errorf("expected status 'queued', got '%s'", result.Status)
	}
}

// TestCreateTask_ValidationError tests task creation with invalid data.
// [REQ:REQ-P0-003] Verify validation of required fields
func TestCreateTask_ValidationError(t *testing.T) {
	_, router := setupTestHandler()

	tests := []struct {
		name string
		task map[string]interface{}
	}{
		{
			name: "empty title",
			task: map[string]interface{}{"title": "", "scopePath": "src/"},
		},
		{
			name: "empty scope path",
			task: map[string]interface{}{"title": "Test", "scopePath": ""},
		},
		{
			name: "path traversal",
			task: map[string]interface{}{"title": "Test", "scopePath": "../outside"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.task)
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
	task := map[string]interface{}{
		"title":     "Test task",
		"scopePath": "src/",
	}
	body, _ := json.Marshal(task)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created domain.Task
	json.NewDecoder(rr.Body).Decode(&created)

	// Get task
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+created.ID.String(), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestListTasks tests listing all tasks.
// [REQ:REQ-P0-003] Test listing tasks
func TestListTasks(t *testing.T) {
	_, router := setupTestHandler()

	// Create tasks
	for i := 0; i < 3; i++ {
		task := map[string]interface{}{
			"title":     "Task " + string(rune('A'+i)),
			"scopePath": "src/",
		}
		body, _ := json.Marshal(task)
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

	var tasks []*domain.Task
	json.NewDecoder(rr.Body).Decode(&tasks)

	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

// TestCancelTask tests task cancellation.
// [REQ:REQ-P0-003] Test task cancellation
func TestCancelTask_Success(t *testing.T) {
	_, router := setupTestHandler()

	// Create task
	task := map[string]interface{}{
		"title":     "To cancel",
		"scopePath": "src/",
	}
	body, _ := json.Marshal(task)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var created domain.Task
	json.NewDecoder(rr.Body).Decode(&created)

	// Cancel task
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+created.ID.String()+"/cancel", nil)
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

	var result orchestration.HealthStatus
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	// Verify required fields per health-api.schema.json
	if result.Status == "" {
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
	profile := map[string]interface{}{
		"name":       "runner-profile",
		"runnerType": "claude-code",
	}
	body, _ := json.Marshal(profile)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdProfile domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&createdProfile)

	// Create a task
	task := map[string]interface{}{
		"title":     "Test task for run",
		"scopePath": "src/",
	}
	body, _ = json.Marshal(task)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var createdTask domain.Task
	json.NewDecoder(rr.Body).Decode(&createdTask)

	// Create a run
	runReq := map[string]interface{}{
		"taskId":         createdTask.ID.String(),
		"agentProfileId": createdProfile.ID.String(),
	}
	body, _ = json.Marshal(runReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var createdRun domain.Run
	json.NewDecoder(rr.Body).Decode(&createdRun)

	if createdRun.TaskID != createdTask.ID {
		t.Errorf("expected task ID %s, got %s", createdTask.ID, createdRun.TaskID)
	}
	if createdRun.AgentProfileID != createdProfile.ID {
		t.Errorf("expected profile ID %s, got %s", createdProfile.ID, createdRun.AgentProfileID)
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

	var statuses []*orchestration.RunnerStatus
	json.NewDecoder(rr.Body).Decode(&statuses)

	// Should have at least one runner (the mock)
	if len(statuses) == 0 {
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

	var errResp domain.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code == "" {
		t.Error("error response should have code")
	}
	if errResp.Message == "" {
		t.Error("error response should have message")
	}
	if errResp.RequestID == "" {
		t.Error("error response should have requestId")
	}
}

// =============================================================================
// INTEGRATION TEST HELPERS
// =============================================================================

// createTestProfile is a helper to create a profile for testing.
func createTestProfile(t *testing.T, router *mux.Router) *domain.AgentProfile {
	profile := map[string]interface{}{
		"name":       "test-profile-" + uuid.New().String()[:8],
		"runnerType": "claude-code",
	}
	body, _ := json.Marshal(profile)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test profile: %d %s", rr.Code, rr.Body.String())
	}

	var result domain.AgentProfile
	json.NewDecoder(rr.Body).Decode(&result)
	return &result
}

// createTestTask is a helper to create a task for testing.
func createTestTask(t *testing.T, router *mux.Router) *domain.Task {
	task := map[string]interface{}{
		"title":     "test-task-" + uuid.New().String()[:8],
		"scopePath": "src/",
	}
	body, _ := json.Marshal(task)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test task: %d %s", rr.Code, rr.Body.String())
	}

	var result domain.Task
	json.NewDecoder(rr.Body).Decode(&result)
	return &result
}

// Compile-time interface check
var _ context.Context
var _ time.Time
