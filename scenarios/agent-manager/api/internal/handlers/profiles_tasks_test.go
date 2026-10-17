package handlers

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestCreateProfile_Success(t *testing.T) {
	_, router := setupTestHandler(t)

	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:        "test-profile",
			ProfileKey:  "test-profile-key",
			Description: "Test profile for unit tests",

			MaxTurns: 100, RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

	tests := []struct {
		name    string
		profile *pb.AgentProfile
		errCode string
	}{
		{
			name:    "empty name",
			profile: &pb.AgentProfile{Name: "", ProfileKey: "test-key", RoleRef: "code.default"},
			errCode: "VALIDATION",
		},
		{
			name:    "missing role reference",
			profile: &pb.AgentProfile{Name: "test", ProfileKey: "test-key"},
			errCode: "VALIDATION",
		},
		{
			name:    "negative max turns",
			profile: &pb.AgentProfile{Name: "test", ProfileKey: "test-key", MaxTurns: -1, RoleRef: "code.default"},
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
	_, router := setupTestHandler(t)

	// First create a profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "test-profile",
			ProfileKey: "test-profile-key", RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

	// Create multiple profiles
	for i := 0; i < 3; i++ {
		body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
			Profile: &pb.AgentProfile{
				Name:       "profile-" + string(rune('A'+i)),
				ProfileKey: "profile-key-" + string(rune('A'+i)), RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

	// Create profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "original-name",
			ProfileKey: "original-key", RoleRef: "code.default",
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
			Description: "Updated description", RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

	// Create profile
	body := encodeProtoJSON(t, &apipb.CreateProfileRequest{
		Profile: &pb.AgentProfile{
			Name:       "to-delete",
			ProfileKey: "to-delete-key", RoleRef: "code.default",
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
	_, router := setupTestHandler(t)

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

func TestTaskCancellationAndDeletionLifecycle(t *testing.T) {
	_, router := setupTestHandler(t)
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &pb.Task{Title: "cleanup", ScopePath: "src/cleanup"}})
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created apipb.CreateTaskResponse
	decodeProtoJSON(t, create.Body.Bytes(), &created)

	deleteBeforeCancel := httptest.NewRecorder()
	router.ServeHTTP(deleteBeforeCancel, httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+created.Task.GetId(), nil))
	if deleteBeforeCancel.Code != http.StatusConflict {
		t.Fatalf("uncancelled delete status=%d body=%s", deleteBeforeCancel.Code, deleteBeforeCancel.Body.String())
	}
	cancel := httptest.NewRecorder()
	router.ServeHTTP(cancel, httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+created.Task.GetId()+"/cancel", nil))
	if cancel.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", cancel.Code, cancel.Body.String())
	}
	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+created.Task.GetId(), nil))
	if remove.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", remove.Code, remove.Body.String())
	}
	var deleted apipb.DeleteTaskResponse
	decodeProtoJSON(t, remove.Body.Bytes(), &deleted)
	if !deleted.Success {
		t.Fatalf("delete response=%+v", &deleted)
	}
}

func TestListTasksAppliesQueryFiltersAndRejectsInvalidValues(t *testing.T) {
	_, router := setupTestHandler(t)
	for _, task := range []*pb.Task{{Title: "matching", ScopePath: "src/match"}, {Title: "other", ScopePath: "docs/other"}} {
		body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: task})
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body)))
		if rr.Code != http.StatusCreated {
			t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/tasks?status=TASK_STATUS_QUEUED&scopePrefix=src/&limit=10&offset=0", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var response apipb.ListTasksResponse
	decodeProtoJSON(t, list.Body.Bytes(), &response)
	if response.GetTotal() != 1 || response.GetTasks()[0].GetTitle() != "matching" {
		t.Fatalf("list response=%+v", &response)
	}
	for _, path := range []string{"/api/v1/tasks?limit=invalid", "/api/v1/tasks?status=unknown"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}
}

// TestCreateTask_ValidationError tests task creation with invalid data.
// [REQ:REQ-P0-003] Verify validation of required fields
func TestCreateTask_ValidationError(t *testing.T) {
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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
	_, router := setupTestHandler(t)

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

// This qualifies the real REST router, SQLite query and protobuf projection.
// Setup writes precede the request; an owner repository read verifies no GET
// mutation. More than the usual first page proves omitted pagination is unbounded.
func TestListProfilesUnpaginatedReadPreservesOwnerExecutionContract(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	var target *domain.AgentProfile
	for i := 0; i < 40; i++ {
		profile := &domain.AgentProfile{ID: uuid.New(), Name: fmt.Sprintf("profile-%02d", i), ProfileKey: fmt.Sprintf("owner/profile-%02d", i), RoleRef: "code.default", CreatedAt: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 10, 10, 0, 0, i, 0, time.UTC)}
		if i == 0 {
			profile.OwnerScenario = "prompt-manager"
			profile.SourcePath = ".vrooli/agent-manager/heartbeat.json"
			profile.SourceHash = "original-hash"
			profile.LastAppliedHash = "original-hash"
			profile.MaxTurns = 50
			profile.Timeout = 600 * time.Second
			profile.MaxWeightedTokens = 12345
			profile.LocalOverride = true
			profile.SandboxConfig = &domain.SandboxConfig{Mode: domain.SandboxModeProtected}
			target = profile
		}
		if err := repos.Profiles.Create(ctx, profile); err != nil {
			t.Fatal(err)
		}
	}
	before, err := repos.Profiles.List(ctx, repository.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read=%d %s", response.Code, response.Body)
	}
	var result apipb.ListProfilesResponse
	decodeProtoJSON(t, response.Body.Bytes(), &result)
	if len(result.Profiles) != 40 || result.Total != 40 {
		t.Fatalf("implicit page truncation: len=%d total=%d", len(result.Profiles), result.Total)
	}
	var found *pb.AgentProfile
	for _, profile := range result.Profiles {
		if profile.ProfileKey == target.ProfileKey {
			if found != nil {
				t.Fatal("duplicate exact key")
			}
			found = profile
		}
	}
	if found == nil || found.Id != target.ID.String() || found.OwnerScenario != "prompt-manager" || found.SourceHash != "original-hash" || found.LastAppliedHash != "original-hash" || !found.LocalOverride || found.Timeout.AsDuration() != 600*time.Second || found.MaxTurns != 50 || found.MaxWeightedTokens != 12345 || found.SandboxConfig.GetMode() != pb.SandboxMode_SANDBOX_MODE_PROTECTED {
		t.Fatalf("owner data lost: %v", found)
	}
	after, err := repos.Profiles.List(ctx, repository.ListFilter{})
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("profile GET changed owner store: %v", err)
	}
	duplicate := *target
	duplicate.ID = uuid.New()
	duplicate.Name = "new-name"
	if err := repos.Profiles.Create(ctx, &duplicate); err == nil {
		t.Fatal("owner schema permits duplicate profile key")
	}
}
