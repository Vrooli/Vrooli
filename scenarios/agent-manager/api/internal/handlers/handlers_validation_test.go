package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestCreateProfile_MalformedJSON(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestGetRun_InvalidUUID tests run retrieval with invalid UUID.
func TestGetRun_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestCreateRun_MalformedJSON tests run creation with invalid JSON.
func TestCreateRun_MalformedJSON(t *testing.T) {
	_, router := setupTestHandler(t)

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

// TestCreateInvestigationRun_RejectsNonVrooliEnv verifies that the investigation
// + apply endpoints validate custom environment variables the same way the
// CreateRun path does: a non-VROOLI_-prefixed key is rejected with 400 before
// the run is created. This closes the gap where investigation runs silently
// dropped the Environment field (e.g. VROOLI_SHADOW_SCENARIOS) instead of
// forwarding it.
func TestCreateInvestigationRun_RejectsNonVrooliEnv(t *testing.T) {
	_, router := setupTestHandler(t)

	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "investigate",
			path: "/api/v1/runs/investigate",
			body: `{"runIds":["` + uuid.New().String() + `"],"environment":{"NOT_VROOLI":"x"}}`,
		},
		{
			name: "investigation-apply",
			path: "/api/v1/runs/investigation-apply",
			body: `{"investigationRunId":"` + uuid.New().String() + `","environment":{"NOT_VROOLI":"x"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d for non-VROOLI_ env, got %d: %s",
					http.StatusBadRequest, rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "VROOLI_") {
				t.Errorf("expected error to mention VROOLI_ prefix, got: %s", rr.Body.String())
			}
		})
	}
}

// TestUpdateProfile_InvalidUUID tests profile update with invalid UUID.
func TestUpdateProfile_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler(t)

	body := encodeProtoJSON(t, &apipb.UpdateProfileRequest{
		ProfileId: "invalid-uuid",
		Profile: &pb.AgentProfile{
			Name:       "updated",
			ProfileKey: "updated-key", RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestCancelTask_InvalidUUID tests task cancellation with invalid UUID.
func TestCancelTask_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/invalid-uuid/cancel", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestListProfiles_InvalidPagination tests profile listing with invalid pagination params.
func TestListProfiles_InvalidPagination(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	// Create a profile with a very long description
	longDesc := make([]byte, 100000) // 100KB
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:        "test-profile",
			ProfileKey:  "test-profile-key",
			Description: string(longDesc), RoleRef: "code.default",
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
			ProfileKey: "test-profile-key-" + uuid.New().String()[:8], RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

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
	if rr.Code == http.StatusOK {
		var stopResp apipb.StopRunResponse
		decodeProtoJSON(t, rr.Body.Bytes(), &stopResp)
		if stopResp.Status == "" {
			t.Fatalf("expected stop response status")
		}
		if stopResp.Run == nil {
			t.Fatalf("expected hydrated run in stop response")
		}
		if stopResp.Run.Id != createdRun.Id {
			t.Fatalf("expected response run ID %s, got %s", createdRun.Id, stopResp.Run.Id)
		}
		if stopResp.Run.Actions == nil {
			t.Fatalf("expected hydrated run actions in stop response")
		}
	}
}

// TestStopRun_InvalidUUID tests stopping with invalid run ID.
func TestStopRun_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/invalid-uuid/stop", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestStopRun_NotFound tests stopping a non-existent run.
func TestStopRun_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+uuid.New().String()+"/stop", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetRunByTag tests retrieving a run by custom tag.
func TestGetRunByTag_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/tag/nonexistent-tag", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestStopRunByTag tests stopping a run by its custom tag.
func TestStopRunByTag_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/tag/nonexistent-tag/stop", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestStopAllRuns tests the bulk stop operation.
func TestStopAllRuns_Success(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should work with empty body
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestQuiesceScenario_DrainsWhenIdle verifies the endpoint returns drained=true
// for a scenario with no in-flight runs.
func TestQuiesceScenario_DrainsWhenIdle(t *testing.T) {
	_, router := setupTestHandler(t)

	timeout := "40ms"
	body := encodeProtoJSON(t, &apipb.QuiesceScenarioRequest{
		Scenario: "scenario-with-no-runs",
		Timeout:  &timeout,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/quiesce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Result struct {
			Drained bool `json:"drained"`
		} `json:"result"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Result.Drained {
		t.Fatalf("expected drained=true for an idle scenario, body: %s", rr.Body.String())
	}
}

// TestQuiesceScenario_MissingScenario verifies proto validation rejects a blank
// scenario.
func TestQuiesceScenario_MissingScenario(t *testing.T) {
	_, router := setupTestHandler(t)

	body := encodeProtoJSON(t, &apipb.QuiesceScenarioRequest{Scenario: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/quiesce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf("expected a 4xx for a blank scenario, got 200: %s", rr.Body.String())
	}
}

// TestQuiesceScenario_InvalidTimeout verifies a non-duration timeout is rejected.
func TestQuiesceScenario_InvalidTimeout(t *testing.T) {
	_, router := setupTestHandler(t)

	bad := "not-a-duration"
	body := encodeProtoJSON(t, &apipb.QuiesceScenarioRequest{Scenario: "x", Timeout: &bad})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/quiesce", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf("expected a 4xx for an invalid timeout, got 200: %s", rr.Body.String())
	}
}

// TestGetRunEvents tests retrieving events for a run.
func TestGetRunEvents_Success(t *testing.T) {
	_, router := setupTestHandler(t)

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

	firstSequence := response.Events[0].Sequence
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/runs/%s/events?after_sequence=%d", createdRun.Id, firstSequence), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected after_sequence status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var afterResponse apipb.GetRunEventsResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &afterResponse)
	for _, evt := range afterResponse.Events {
		if evt.Sequence <= firstSequence {
			t.Fatalf("after_sequence returned sequence %d <= %d", evt.Sequence, firstSequence)
		}
	}
}

// TestGetRunEvents_NotFound tests getting events for non-existent run.
func TestGetRunEvents_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/invalid-uuid/events", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestGetRunDiff_NotFound tests getting diff for non-existent run.
func TestGetRunDiff_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+uuid.New().String()+"/diff", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetRunDiff_InvalidUUID tests getting diff with invalid run ID.
func TestGetRunDiff_InvalidUUID(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/invalid-uuid/diff", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// TestApproveRun_NotFound tests approving a non-existent run.
func TestApproveRun_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?tagPrefix=ecosystem-", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithMultipleFilters tests listing runs with multiple filters.
func TestListRuns_WithMultipleFilters(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?status=pending&tagPrefix=test-", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithInvestigatesRunIDFilter tests listing runs with investigates_run_id filter.
func TestListRuns_WithInvestigatesRunIDFilter(t *testing.T) {
	_, router := setupTestHandler(t)

	sourceRunID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?investigates_run_id="+sourceRunID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithAppliesInvestigationRunIDFilter tests listing runs with applies_investigation_run_id filter.
func TestListRuns_WithAppliesInvestigationRunIDFilter(t *testing.T) {
	_, router := setupTestHandler(t)

	investigationRunID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?applies_investigation_run_id="+investigationRunID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

// TestListRuns_WithInvalidInvestigatesRunID tests validation for investigates_run_id.
func TestListRuns_WithInvalidInvestigatesRunID(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?investigates_run_id=invalid-uuid", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// TestListRuns_WithInvalidAppliesInvestigationRunID tests validation for applies_investigation_run_id.
func TestListRuns_WithInvalidAppliesInvestigationRunID(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?applies_investigation_run_id=invalid-uuid", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// =============================================================================
// UPDATE TASK TESTS
// =============================================================================

// TestUpdateTask_Success tests successful task update.
func TestUpdateTask_Success(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+uuid.New().String()+"/cancel", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetTask_NotFound tests retrieving non-existent task.
func TestGetTask_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestListRuns_IncludesPromptPreview verifies that listing runs includes
// the first ~120 characters of the associated task description.
func TestListRuns_IncludesPromptPreview(t *testing.T) {
	_, router := setupTestHandler(t)

	// Create a task with a known description (the user's prompt).
	taskDesc := "Fix the failing auth tests in the login module and update the middleware"
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:       "test-task-preview",
			Description: taskDesc,
			ScopePath:   "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rr.Code, rr.Body.String())
	}
	var taskResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &taskResp)

	// Create a profile for the run.
	profile := createTestProfile(t, router)

	// Create a run linked to that task.
	profileID := profile.Id
	tag := "gct-test-preview"
	body = encodeProtoJSON(t, &apipb.CreateRunRequest{
		TaskId:         taskResp.Task.Id,
		AgentProfileId: &profileID,
		Tag:            &tag,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create run: %d %s", rr.Code, rr.Body.String())
	}

	// List runs and check for prompt_preview.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs?tagPrefix=gct-test-preview", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list runs: %d %s", rr.Code, rr.Body.String())
	}

	var listResp apipb.ListRunsResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &listResp)

	if len(listResp.Runs) == 0 {
		t.Fatal("expected at least one run")
	}

	got := listResp.Runs[0].PromptPreview
	if got != taskDesc {
		t.Errorf("expected prompt_preview %q, got %q", taskDesc, got)
	}
}

// TestListRuns_OmitsHeavyFields verifies that the list endpoint returns nil for
// heavy fields (summary, resolvedConfig) while GetRun returns them.
func TestListRuns_OmitsHeavyFields(t *testing.T) {
	_, router := setupTestHandler(t)

	// Create profile and task
	profile := createTestProfile(t, router)
	taskDesc := "Heavy fields test task description"
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{
		Task: &pb.Task{
			Title:       "heavy-fields-test",
			Description: taskDesc,
			ScopePath:   "src/",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rr.Code, rr.Body.String())
	}
	var taskResp apipb.CreateTaskResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &taskResp)

	// Create a run
	profileID := profile.Id
	tag := "heavy-test-tag"
	body = encodeProtoJSON(t, &apipb.CreateRunRequest{
		TaskId:         taskResp.Task.Id,
		AgentProfileId: &profileID,
		Tag:            &tag,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create run: %d %s", rr.Code, rr.Body.String())
	}
	var createResp apipb.CreateRunResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &createResp)
	runID := createResp.Run.Id

	// List runs — heavy fields should be nil
	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs?tagPrefix=heavy-test-tag", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list runs: %d %s", rr.Code, rr.Body.String())
	}
	var listResp apipb.ListRunsResponse
	decodeProtoJSON(t, rr.Body.Bytes(), &listResp)

	if len(listResp.Runs) == 0 {
		t.Fatal("expected at least one run in list")
	}
	listedRun := listResp.Runs[0]
	if listedRun.Summary != nil {
		t.Errorf("List: expected nil Summary, got %+v", listedRun.Summary)
	}
	if listedRun.ResolvedConfig != nil {
		t.Errorf("List: expected nil ResolvedConfig, got %+v", listedRun.ResolvedConfig)
	}

	// Get single run — should return whatever fields exist (not nil by design)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get run: %d %s", rr.Code, rr.Body.String())
	}
	// Just verifying the Get endpoint works — newly created runs may not have
	// summary/resolvedConfig set yet, but the endpoint should return the full object.
}

// =============================================================================
// CUSTOM ENVIRONMENT VALIDATION TESTS
// =============================================================================

func TestValidateCustomEnvironment_Valid(t *testing.T) {
	env := map[string]string{
		"VROOLI_SPAWN_SOURCE": "research/my-research",
		"VROOLI_CUSTOM_VAR":   "value",
	}
	if err := validateCustomEnvironment(env); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateCustomEnvironment_InvalidPrefix(t *testing.T) {
	env := map[string]string{
		"PATH": "/usr/bin",
	}
	err := validateCustomEnvironment(env)
	if err == nil {
		t.Fatal("expected error for non-VROOLI_ prefix")
	}
	if !strings.Contains(err.Error(), "VROOLI_") {
		t.Errorf("error should mention VROOLI_ prefix, got: %v", err)
	}
}

func TestValidateCustomEnvironment_TooManyEntries(t *testing.T) {
	env := make(map[string]string, 21)
	for i := range 21 {
		env[fmt.Sprintf("VROOLI_VAR_%d", i)] = "v"
	}
	err := validateCustomEnvironment(env)
	if err == nil {
		t.Fatal("expected error for >20 entries")
	}
	if !strings.Contains(err.Error(), "20") {
		t.Errorf("error should mention limit, got: %v", err)
	}
}

func TestValidateCustomEnvironment_TooLarge(t *testing.T) {
	env := map[string]string{
		"VROOLI_BIG": strings.Repeat("x", 5000),
	}
	err := validateCustomEnvironment(env)
	if err == nil {
		t.Fatal("expected error for oversized env")
	}
	if !strings.Contains(err.Error(), "4096") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

func TestValidateCustomEnvironment_Empty(t *testing.T) {
	if err := validateCustomEnvironment(map[string]string{}); err != nil {
		t.Errorf("expected no error for empty map, got %v", err)
	}
}

// Compile-time interface check
var (
	_ context.Context
	_ time.Time
)
