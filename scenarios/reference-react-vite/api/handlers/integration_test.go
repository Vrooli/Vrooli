// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#integration-tests
// DOC: docs/internal/SEAMS.md#http-handler-seam
// [REQ:REQ-P1-006b] API Integration Tests - End-to-end API handler testing with httptest
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/tasks"
	"reference-react-vite/api/handlers"
	"reference-react-vite/api/internal/mocks"
	"reference-react-vite/api/internal/testutil"
)

// =============================================================================
// Full CRUD Integration Tests
// [REQ:REQ-P1-006b] Tests full request-response cycle through API handlers
// =============================================================================

func TestIntegration_TaskCRUDWorkflow(t *testing.T) {
	// ARRANGE - Set up router with mock repository
	repo := mocks.NewMockTaskRepository()
	router := setupIntegrationRouter(repo)

	var createdTaskID string

	// ==========================================================================
	// Step 1: Create a task
	// ==========================================================================
	t.Run("create_task", func(t *testing.T) {
		body := map[string]interface{}{
			"title":       "Integration Test Task",
			"description": "Testing full CRUD workflow",
			"priority":    2,
		}

		req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/tasks", body)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusCreated)

		var task tasks.Task
		testutil.AssertJSON(t, rec, &task)

		if task.ID == "" {
			t.Fatal("expected non-empty task ID")
		}
		if task.Title != "Integration Test Task" {
			t.Errorf("expected title %q, got %q", "Integration Test Task", task.Title)
		}
		if task.Status != tasks.StatusPending {
			t.Errorf("expected status %q, got %q", tasks.StatusPending, task.Status)
		}

		createdTaskID = task.ID
	})

	// ==========================================================================
	// Step 2: List tasks (verify task appears)
	// ==========================================================================
	t.Run("list_tasks_after_create", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusOK)

		var response ListResponse
		testutil.AssertJSON(t, rec, &response)

		data, ok := response.Data.([]interface{})
		if !ok {
			t.Fatal("expected data to be array")
		}
		if len(data) < 1 {
			t.Error("expected at least 1 task in list")
		}
	})

	// ==========================================================================
	// Step 3: Get created task by ID
	// ==========================================================================
	t.Run("get_task_by_id", func(t *testing.T) {
		if createdTaskID == "" {
			t.Skip("no task was created")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+createdTaskID, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusOK)

		var task tasks.Task
		testutil.AssertJSON(t, rec, &task)

		if task.ID != createdTaskID {
			t.Errorf("expected task ID %q, got %q", createdTaskID, task.ID)
		}
	})

	// ==========================================================================
	// Step 4: Update the task
	// ==========================================================================
	t.Run("update_task", func(t *testing.T) {
		if createdTaskID == "" {
			t.Skip("no task was created")
		}

		body := map[string]interface{}{
			"title":  "Updated Integration Task",
			"status": "in_progress",
		}

		req := testutil.MakeJSONRequest(t, http.MethodPatch, "/api/v1/tasks/"+createdTaskID, body)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusOK)

		var task tasks.Task
		testutil.AssertJSON(t, rec, &task)

		if task.Title != "Updated Integration Task" {
			t.Errorf("expected title %q, got %q", "Updated Integration Task", task.Title)
		}
	})

	// ==========================================================================
	// Step 5: Delete the task
	// ==========================================================================
	t.Run("delete_task", func(t *testing.T) {
		if createdTaskID == "" {
			t.Skip("no task was created")
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+createdTaskID, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusNoContent)
	})
}

// =============================================================================
// Error Response Consistency Tests
// [REQ:REQ-P1-006b] Tests error response structure consistency
// =============================================================================

func TestIntegration_ErrorResponseConsistency(t *testing.T) {
	repo := mocks.NewMockTaskRepository()
	router := setupIntegrationRouter(repo)

	tests := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		wantStatus int
		category   string
	}{
		{
			name:       "not_found_error",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/nonexistent",
			wantStatus: http.StatusNotFound,
			category:   "error_handling",
		},
		{
			name:       "validation_error",
			method:     http.MethodPost,
			path:       "/api/v1/tasks",
			body:       map[string]interface{}{},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error_handling",
		},
		{
			name:       "bad_request_error",
			method:     http.MethodPost,
			path:       "/api/v1/tasks",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			category:   "error_handling",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != nil {
				req = testutil.MakeJSONRequest(t, tc.method, tc.path, tc.body)
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)

			// Verify error response has consistent structure
			if rec.Code >= 400 {
				var errResp map[string]interface{}
				testutil.AssertJSON(t, rec, &errResp)

				// Check for required error response fields
				if _, ok := errResp["error"]; !ok {
					if _, ok := errResp["message"]; !ok {
						t.Error("expected error response to have 'error' or 'message' field")
					}
				}
			}
		})
	}
}

// =============================================================================
// Pagination Integration Tests
// [REQ:REQ-P1-006b] Tests pagination works correctly across API
// =============================================================================

func TestIntegration_PaginationBehavior(t *testing.T) {
	repo := mocks.NewMockTaskRepository()

	// Add multiple tasks for pagination testing
	for i := 0; i < 25; i++ {
		task := testutil.NewTaskFactory().
			WithID(string(rune('a' + i))).
			WithTitle("Task " + string(rune('A'+i))).
			Build()
		repo.WithTask(task)
	}

	router := setupIntegrationRouter(repo)

	tests := []struct {
		name            string
		url             string
		wantCount       int
		checkPagination func(t *testing.T, resp ListResponse)
		category        string
	}{
		{
			name:      "default_pagination",
			url:       "/api/v1/tasks",
			wantCount: 20, // Default limit
			checkPagination: func(t *testing.T, resp ListResponse) {
				if resp.Pagination.Limit != 20 {
					t.Errorf("expected limit 20, got %d", resp.Pagination.Limit)
				}
				if resp.Pagination.Total != 25 {
					t.Errorf("expected total 25, got %d", resp.Pagination.Total)
				}
			},
			category: "pagination",
		},
		{
			name:      "custom_limit",
			url:       "/api/v1/tasks?limit=5",
			wantCount: 5,
			checkPagination: func(t *testing.T, resp ListResponse) {
				if resp.Pagination.Limit != 5 {
					t.Errorf("expected limit 5, got %d", resp.Pagination.Limit)
				}
			},
			category: "pagination",
		},
		{
			name:      "offset_pagination",
			url:       "/api/v1/tasks?limit=10&offset=20",
			wantCount: 5, // Only 5 remaining after offset 20
			checkPagination: func(t *testing.T, resp ListResponse) {
				if resp.Pagination.Offset != 20 {
					t.Errorf("expected offset 20, got %d", resp.Pagination.Offset)
				}
			},
			category: "pagination",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, http.StatusOK)

			var response ListResponse
			testutil.AssertJSON(t, rec, &response)

			if data, ok := response.Data.([]interface{}); ok {
				if len(data) != tc.wantCount {
					t.Errorf("expected %d items, got %d", tc.wantCount, len(data))
				}
			}

			if tc.checkPagination != nil {
				tc.checkPagination(t, response)
			}
		})
	}
}

// =============================================================================
// Concurrent Request Tests
// [REQ:REQ-P1-006b] Tests API handles concurrent requests correctly
// =============================================================================

func TestIntegration_ConcurrentRequests(t *testing.T) {
	repo := mocks.NewMockTaskRepository()
	router := setupIntegrationRouter(repo)

	// Create some initial tasks
	for i := 0; i < 5; i++ {
		task := testutil.NewTaskFactory().
			WithID(string(rune('a' + i))).
			WithTitle("Concurrent Task " + string(rune('A'+i))).
			Build()
		repo.WithTask(task)
	}

	// Run concurrent list requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent request failed with status %d", rec.Code)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// Repository Method Coverage Test
// [REQ:REQ-P1-006b] Tests repository methods are called correctly
// =============================================================================

func TestIntegration_RepositoryMethodCalls(t *testing.T) {
	repo := mocks.NewMockTaskRepository()
	router := setupIntegrationRouter(repo)

	// Test that Create is called
	t.Run("create_calls_repository", func(t *testing.T) {
		initialCount := repo.CreateCallCount()

		body := map[string]interface{}{
			"title": "Repo Test Task",
		}

		req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/tasks", body)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if repo.CreateCallCount() != initialCount+1 {
			t.Error("expected Create to be called")
		}
	})

	// Test that FindByID is called
	t.Run("get_calls_repository", func(t *testing.T) {
		task := testutil.NewTaskFactory().WithID("find-test").Build()
		repo.WithTask(task)
		initialCount := repo.FindCallCount()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/find-test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if repo.FindCallCount() <= initialCount {
			t.Error("expected FindByID to be called")
		}
	})

	// Test that Delete is called
	t.Run("delete_calls_repository", func(t *testing.T) {
		task := testutil.NewTaskFactory().WithID("delete-test").Build()
		repo.WithTask(task)
		initialCount := repo.DeleteCallCount()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/delete-test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if repo.DeleteCallCount() != initialCount+1 {
			t.Error("expected Delete to be called")
		}
	})
}

// =============================================================================
// Helpers
// =============================================================================

// setupIntegrationRouter creates a router for integration tests.
func setupIntegrationRouter(repo *mocks.MockTaskRepository) *mux.Router {
	r := mux.NewRouter()
	cfg := handlers.PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
	}
	h := handlers.NewTaskHandler(repo, cfg)
	h.RegisterRoutes(r)
	return r
}

// Ensure context is used (fixes import warning)
var _ = context.Background
