// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// DOC: docs/internal/SEAMS.md#http-handler-seam
// [REQ:REQ-P1-002b] Filtering and Sorting - Focused tests for filtering behavior
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

// =============================================================================
// Status Filter Tests
// [REQ:REQ-P1-002b] Tests for status filtering capability
// =============================================================================

func TestFiltering_StatusFilter(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "filter_by_pending_status",
			url:  "/api/v1/tasks?status=pending",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusCompleted).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("3").WithStatus(tasks.StatusPending).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name: "filter_by_completed_status",
			url:  "/api/v1/tasks?status=completed",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusCompleted).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "filter_by_in_progress_status",
			url:  "/api/v1/tasks?status=in_progress",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusInProgress).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusPending).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "filter_no_matches",
			url:  "/api/v1/tasks?status=archived",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusCompleted).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
			category:   "edge_case",
		},
		{
			name:       "filter_invalid_status_rejected",
			url:        "/api/v1/tasks?status=invalid",
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
			router := setupFilteringRouter(repo)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var response ListResponse
				testutil.AssertJSON(t, rec, &response)

				if data, ok := response.Data.([]interface{}); ok {
					if len(data) != tc.wantCount {
						t.Errorf("expected %d tasks, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// =============================================================================
// Priority Filter Tests
// [REQ:REQ-P1-002b] Tests for priority filtering capability
// =============================================================================

func TestFiltering_PriorityFilter(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "filter_by_high_priority",
			url:  "/api/v1/tasks?priority=3",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithPriority(tasks.PriorityHigh).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithPriority(tasks.PriorityLow).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("3").WithPriority(tasks.PriorityHigh).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name: "filter_by_medium_priority",
			url:  "/api/v1/tasks?priority=2",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithPriority(tasks.PriorityMedium).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithPriority(tasks.PriorityLow).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "filter_by_low_priority",
			url:  "/api/v1/tasks?priority=1",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithPriority(tasks.PriorityLow).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithPriority(tasks.PriorityHigh).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name:       "filter_invalid_priority_text_rejected",
			url:        "/api/v1/tasks?priority=abc",
			setupMock:  func(m *mocks.MockTaskRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:       "filter_invalid_priority_value_rejected",
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
			router := setupFilteringRouter(repo)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var response ListResponse
				testutil.AssertJSON(t, rec, &response)

				if data, ok := response.Data.([]interface{}); ok {
					if len(data) != tc.wantCount {
						t.Errorf("expected %d tasks, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// =============================================================================
// Project Filter Tests
// [REQ:REQ-P1-002b] Tests for project-based filtering capability
// =============================================================================

func TestFiltering_ProjectFilter(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "filter_by_project_id",
			url:  "/api/v1/tasks?project_id=proj-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithProjectID("proj-123").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithProjectID("proj-456").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("3").WithProjectID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name: "filter_project_no_matches",
			url:  "/api/v1/tasks?project_id=nonexistent",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithProjectID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
			category:   "edge_case",
		},
		{
			name: "filter_tasks_without_project",
			url:  "/api/v1/tasks?project_id=",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithProjectID("").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithProjectID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2, // Empty project_id filter returns all
			category:   "edge_case",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupFilteringRouter(repo)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var response ListResponse
				testutil.AssertJSON(t, rec, &response)

				if data, ok := response.Data.([]interface{}); ok {
					if len(data) != tc.wantCount {
						t.Errorf("expected %d tasks, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// =============================================================================
// Combined Filter Tests
// [REQ:REQ-P1-002b] Tests for combined filtering capability
// =============================================================================

func TestFiltering_CombinedFilters(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "filter_by_status_and_priority",
			url:  "/api/v1/tasks?status=pending&priority=3",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).WithPriority(tasks.PriorityHigh).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusPending).WithPriority(tasks.PriorityLow).Build())
				m.WithTask(testutil.NewTaskFactory().WithID("3").WithStatus(tasks.StatusCompleted).WithPriority(tasks.PriorityHigh).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "filter_by_status_and_project",
			url:  "/api/v1/tasks?status=pending&project_id=proj-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).WithProjectID("proj-123").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusPending).WithProjectID("proj-456").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("3").WithStatus(tasks.StatusCompleted).WithProjectID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "filter_all_parameters",
			url:  "/api/v1/tasks?status=pending&priority=2&project_id=proj-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("1").WithStatus(tasks.StatusPending).WithPriority(tasks.PriorityMedium).WithProjectID("proj-123").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("2").WithStatus(tasks.StatusPending).WithPriority(tasks.PriorityHigh).WithProjectID("proj-123").Build())
				m.WithTask(testutil.NewTaskFactory().WithID("3").WithStatus(tasks.StatusCompleted).WithPriority(tasks.PriorityMedium).WithProjectID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := setupFilteringRouter(repo)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var response ListResponse
				testutil.AssertJSON(t, rec, &response)

				if data, ok := response.Data.([]interface{}); ok {
					if len(data) != tc.wantCount {
						t.Errorf("expected %d tasks, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

// setupFilteringRouter creates a test router for filtering tests.
func setupFilteringRouter(repo *mocks.MockTaskRepository) *mux.Router {
	r := mux.NewRouter()
	cfg := handlers.PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
	}
	h := handlers.NewTaskHandler(repo, cfg)
	h.RegisterRoutes(r)
	return r
}
