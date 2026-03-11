// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// DOC: docs/internal/SEAMS.md#http-handler-seam
// [REQ:RRV-API-002] Projects API - Unit tests for project HTTP handlers
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/projects"
	"reference-react-vite/api/handlers"
	"reference-react-vite/api/internal/mocks"
	"reference-react-vite/api/internal/testutil"
)

// defaultProjectPaginationCfg is used in tests.
var defaultProjectPaginationCfg = handlers.PaginationConfig{
	DefaultLimit: 20,
	MaxLimit:     100,
}

// setupProjectRouter creates a test router with the project handler registered.
func setupProjectRouter(repo *mocks.MockProjectRepository) *mux.Router {
	r := mux.NewRouter()
	h := handlers.NewProjectHandler(repo, defaultProjectPaginationCfg)
	h.RegisterRoutes(r)
	return r
}

// =============================================================================
// ProjectHandler.List Tests
// =============================================================================

func TestProjectHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMock  func(*mocks.MockProjectRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "list_empty",
			url:  "/api/v1/projects",
			setupMock: func(m *mocks.MockProjectRepository) {
				// No projects added
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
			category:   "happy_path",
		},
		{
			name: "list_with_projects",
			url:  "/api/v1/projects",
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("1").WithName("Project 1").Build())
				m.WithProject(testutil.NewProjectFactory().WithID("2").WithName("Project 2").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name: "list_with_status_filter",
			url:  "/api/v1/projects?status=active",
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("1").WithStatus(projects.StatusActive).Build())
				m.WithProject(testutil.NewProjectFactory().WithID("2").WithStatus(projects.StatusPaused).Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "list_with_pagination",
			url:  "/api/v1/projects?limit=1&offset=1",
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("1").Build())
				m.WithProject(testutil.NewProjectFactory().WithID("2").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name:       "list_invalid_status",
			url:        "/api/v1/projects?status=invalid",
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockProjectRepository()
			tc.setupMock(repo)
			router := setupProjectRouter(repo)

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
						t.Errorf("expected %d projects, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// =============================================================================
// ProjectHandler.Create Tests
// =============================================================================

func TestProjectHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(*mocks.MockProjectRepository)
		wantStatus int
		category   string
	}{
		{
			name: "create_valid_project",
			body: map[string]interface{}{
				"name":        "New Project",
				"description": "Project description",
				"color":       "#ff0000",
			},
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
		{
			name: "create_minimal_project",
			body: map[string]interface{}{
				"name": "Minimal Project",
			},
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
		{
			name:       "create_missing_name",
			body:       map[string]interface{}{},
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name: "create_empty_name",
			body: map[string]interface{}{
				"name": "",
			},
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name: "create_invalid_color",
			body: map[string]interface{}{
				"name":  "Project",
				"color": "invalid",
			},
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:       "create_invalid_json",
			body:       "not json",
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusBadRequest,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockProjectRepository()
			tc.setupMock(repo)
			router := setupProjectRouter(repo)

			req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/projects", tc.body)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusCreated {
				var project projects.Project
				testutil.AssertJSON(t, rec, &project)
				if project.ID == "" {
					t.Error("expected non-empty project ID")
				}
				if project.Status != projects.StatusActive {
					t.Errorf("expected status %q, got %q", projects.StatusActive, project.Status)
				}
			}
		})
	}
}

// =============================================================================
// ProjectHandler.Get Tests
// =============================================================================

func TestProjectHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		projectID  string
		setupMock  func(*mocks.MockProjectRepository)
		wantStatus int
		category   string
	}{
		{
			name:      "get_existing_project",
			projectID: "proj-123",
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").WithName("Test Project").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "get_nonexistent_project",
			projectID:  "nonexistent",
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockProjectRepository()
			tc.setupMock(repo)
			router := setupProjectRouter(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+tc.projectID, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var project projects.Project
				testutil.AssertJSON(t, rec, &project)
				if project.ID != tc.projectID {
					t.Errorf("expected ID %q, got %q", tc.projectID, project.ID)
				}
			}
		})
	}
}

// =============================================================================
// ProjectHandler.Update Tests
// =============================================================================

func TestProjectHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		projectID  string
		body       interface{}
		setupMock  func(*mocks.MockProjectRepository)
		wantStatus int
		category   string
	}{
		{
			name:      "update_name",
			projectID: "proj-123",
			body: map[string]interface{}{
				"name": "Updated Name",
			},
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:      "update_status",
			projectID: "proj-123",
			body: map[string]interface{}{
				"status": "paused",
			},
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:      "update_color",
			projectID: "proj-123",
			body: map[string]interface{}{
				"color": "#00ff00",
			},
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:      "update_nonexistent_project",
			projectID: "nonexistent",
			body: map[string]interface{}{
				"name": "Updated",
			},
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
		{
			name:      "update_with_empty_name",
			projectID: "proj-123",
			body: map[string]interface{}{
				"name": "",
			},
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:      "update_with_invalid_status",
			projectID: "proj-123",
			body: map[string]interface{}{
				"status": "invalid",
			},
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
		{
			name:      "update_with_invalid_color",
			projectID: "proj-123",
			body: map[string]interface{}{
				"color": "bad-color",
			},
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockProjectRepository()
			tc.setupMock(repo)
			router := setupProjectRouter(repo)

			req := testutil.MakeJSONRequest(t, http.MethodPatch, "/api/v1/projects/"+tc.projectID, tc.body)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// =============================================================================
// ProjectHandler.Delete Tests
// =============================================================================

func TestProjectHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		projectID  string
		setupMock  func(*mocks.MockProjectRepository)
		wantStatus int
		category   string
	}{
		{
			name:      "delete_existing_project",
			projectID: "proj-123",
			setupMock: func(m *mocks.MockProjectRepository) {
				m.WithProject(testutil.NewProjectFactory().WithID("proj-123").Build())
			},
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:       "delete_nonexistent_project",
			projectID:  "nonexistent",
			setupMock:  func(m *mocks.MockProjectRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockProjectRepository()
			tc.setupMock(repo)
			router := setupProjectRouter(repo)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+tc.projectID, nil)
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
