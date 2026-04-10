// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// DOC: docs/internal/SEAMS.md#http-handler-seam
// [REQ:RRV-API-001] Tasks API - Unit tests for task HTTP handlers
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/tasks"
	"reference-react-vite/api/handlers"
	"reference-react-vite/api/internal/mocks"
	"reference-react-vite/api/internal/testutil"
)

// defaultPaginationCfg is used in tests.
var defaultPaginationCfg = handlers.PaginationConfig{
	DefaultLimit: 20,
	MaxLimit:     100,
}

// setupTaskRouter creates a test router with the task handler registered.
func setupTaskRouter(repo *mocks.MockTaskRepository) *mux.Router {
	r := mux.NewRouter()
	h := handlers.NewTaskHandler(repo, defaultPaginationCfg)
	h.RegisterRoutes(r)
	return r
}

// =============================================================================
// TaskHandler.List Tests
// =============================================================================

func TestTaskHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "list_empty",
			url:  "/api/v1/tasks",
			setupMock: func(m *mocks.MockTaskRepository) {
				// No tasks added
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
			category:   "happy_path",
		},
		{
			name: "list_with_tasks",
			url:  "/api/v1/tasks",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithTitle("Task 1").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithTitle("Task 2").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name: "list_with_status_filter",
			url:  "/api/v1/tasks?status=pending",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusCompleted).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "list_with_priority_filter",
			url:  "/api/v1/tasks?priority=3",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithPriority(tasks.PriorityHigh).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithPriority(tasks.PriorityLow).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "list_with_project_filter",
			url:  "/api/v1/tasks?project_id=proj-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithProjectID("proj-123").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithProjectID("proj-456").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "list_with_pagination",
			url:  "/api/v1/tasks?limit=1&offset=1",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name:       "list_invalid_status",
			url:        "/api/v1/tasks?status=invalid",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:       "list_invalid_priority",
			url:        "/api/v1/tasks?priority=abc",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:       "list_invalid_priority_value",
			url:        "/api/v1/tasks?priority=5",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupTaskRouter(repo)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var response ListResponse
				testutil.AssertJSON(t, rec, &response)

				// Cast data to []interface{} to count items
				if data, ok := response.Data.([]interface{}); ok {
					if len(data) != tc.wantCount {
						t.Errorf("expected %d tasks, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// ListResponse matches the handlers.ListResponse structure for testing.
type ListResponse struct {
	Data       interface{} `json:"data"`
	Pagination struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"pagination"`
}

// =============================================================================
// TaskHandler.Create Tests
// =============================================================================

func TestTaskHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		category   string
	}{
		{
			name: "create_valid_task",
			body: map[string]interface{}{
				"title":       "New Task",
				"description": "Task description",
				"priority":    2,
			},
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
		{
			name: "create_minimal_task",
			body: map[string]interface{}{
				"title": "Minimal Task",
			},
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
		{
			name:       "create_missing_title",
			body:       map[string]interface{}{},
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name: "create_empty_title",
			body: map[string]interface{}{
				"title": "",
			},
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name: "create_invalid_priority",
			body: map[string]interface{}{
				"title":    "Task",
				"priority": 5,
			},
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:       "create_invalid_json",
			body:       "not json",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusBadRequest,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupTaskRouter(repo)

			req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/tasks", tc.body)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusCreated {
				var task tasks.Task
				testutil.AssertJSON(t, rec, &task)
				if task.ID == "" {
					t.Error("expected non-empty task ID")
				}
				if task.Status != tasks.StatusPending {
					t.Errorf("expected status %q, got %q", tasks.StatusPending, task.Status)
				}
			}
		})
	}
}

// =============================================================================
// TaskHandler.Get Tests
// =============================================================================

func TestTaskHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		category   string
	}{
		{
			name:   "get_existing_task",
			taskID: "task-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").WithTitle("Test Task").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "get_nonexistent_task",
			taskID:     "nonexistent",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupTaskRouter(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+tc.taskID, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var task tasks.Task
				testutil.AssertJSON(t, rec, &task)
				if task.ID != tc.taskID {
					t.Errorf("expected ID %q, got %q", tc.taskID, task.ID)
				}
			}
		})
	}
}

// =============================================================================
// TaskHandler.Update Tests
// =============================================================================

func TestTaskHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		body       interface{}
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		category   string
	}{
		{
			name:   "update_title",
			taskID: "task-123",
			body: map[string]interface{}{
				"title": "Updated Title",
			},
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:   "update_status",
			taskID: "task-123",
			body: map[string]interface{}{
				"status": "completed",
			},
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:   "update_nonexistent_task",
			taskID: "nonexistent",
			body: map[string]interface{}{
				"title": "Updated",
			},
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
		{
			name:   "update_with_empty_title",
			taskID: "task-123",
			body: map[string]interface{}{
				"title": "",
			},
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:   "update_with_invalid_status",
			taskID: "task-123",
			body: map[string]interface{}{
				"status": "invalid",
			},
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupTaskRouter(repo)

			req := testutil.MakeJSONRequest(t, http.MethodPatch, "/api/v1/tasks/"+tc.taskID, tc.body)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// =============================================================================
// TaskHandler.Delete Tests
// =============================================================================

func TestTaskHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		category   string
	}{
		{
			name:   "delete_existing_task",
			taskID: "task-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:       "delete_nonexistent_task",
			taskID:     "nonexistent",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupTaskRouter(repo)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+tc.taskID, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			// Verify delete was called
			if tc.wantStatus == http.StatusNoContent {
				if repo.DeleteCallCount() != 1 {
					t.Errorf("expected 1 delete call, got %d", repo.DeleteCallCount())
				}
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkTaskHandler_List(b *testing.B) {
	repo := mocks.NewMockTaskRepository()
	for i := 0; i < 100; i++ {
		repo.WithTask(testutil.NewTaskFactory().WithID(string(rune(i))).Build())
	}
	router := setupTaskRouter(repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}
}
