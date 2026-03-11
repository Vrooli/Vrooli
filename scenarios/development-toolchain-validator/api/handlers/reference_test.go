// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - HTTP handler tests
package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/internal/config"
	"development-toolchain-validator/internal/mocks"
	"development-toolchain-validator/internal/testutil"
)

// setupTestRouter creates a router with the reference handler registered.
func setupTestRouter(repo *mocks.MockRepository) *mux.Router {
	service := reference.NewService(repo)
	handler := NewReferenceHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return router
}

// TestReferenceHandler_List tests the List endpoint.
func TestReferenceHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		wantCount  int
		category   string
	}{
		{
			name: "list_all_references",
			path: "/api/v1/references",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().WithID("1").Build())
				m.WithReference(testutil.NewReferenceFactory().WithID("2").WithSlug("second").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name:       "empty_list",
			path:       "/api/v1/references",
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusOK,
			wantCount:  0,
			category:   "boundary",
		},
		{
			name: "filter_by_template",
			path: "/api/v1/references?template=react-vite",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("1").
					WithTemplate("react-vite").
					Build())
				m.WithReference(testutil.NewReferenceFactory().
					WithID("2").
					WithSlug("go-scenario").
					WithTemplate("go-api").
					Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name: "repository_error",
			path: "/api/v1/references",
			setupMock: func(m *mocks.MockRepository) {
				m.WithListError(reference.ErrNotFound)
			},
			wantStatus: http.StatusInternalServerError,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
			testutil.AssertContentType(t, rec, "application/json")

			if tc.wantStatus == http.StatusOK && tc.wantCount >= 0 {
				var response struct {
					References []interface{} `json:"references"`
					Count      int           `json:"count"`
				}
				testutil.AssertJSON(t, rec, &response)
				if response.Count != tc.wantCount {
					t.Errorf("expected count %d, got %d", tc.wantCount, response.Count)
				}
			}
		})
	}
}

// TestReferenceHandler_Create tests the Create endpoint.
func TestReferenceHandler_Create(t *testing.T) {
	// Create a temporary directory for path validation
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		body       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		wantErr    string
		category   string
	}{
		{
			name: "valid_input",
			body: `{"slug":"test-scenario","name":"Test Scenario","template":"react-vite","path":"` + tempDir + `"}`,
			setupMock: func(m *mocks.MockRepository) {},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
		{
			name:       "invalid_json",
			body:       `{invalid json}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    "Invalid request body",
			category:   "error",
		},
		{
			name:       "invalid_slug_format",
			body:       `{"slug":"Invalid-Slug","name":"Test","template":"react-vite","path":"` + tempDir + `"}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    "lowercase", // Improved error message explains slug constraints
			category:   "error",
		},
		{
			name: "duplicate_slug",
			body: `{"slug":"existing-slug","name":"Test","template":"react-vite","path":"` + tempDir + `"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithSlug("existing-slug").
					Build())
			},
			wantStatus: http.StatusConflict,
			wantErr:    "already exists",
			category:   "error",
		},
		{
			name:       "nonexistent_path",
			body:       `{"slug":"test-scenario","name":"Test","template":"react-vite","path":"/nonexistent/path"}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    "path does not exist",
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/references", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
			testutil.AssertContentType(t, rec, "application/json")

			if tc.wantErr != "" {
				var response struct {
					Error string `json:"error"`
				}
				testutil.AssertJSON(t, rec, &response)
				if !strings.Contains(response.Error, tc.wantErr) {
					t.Errorf("expected error containing %q, got %q", tc.wantErr, response.Error)
				}
			}
		})
	}
}

// TestReferenceHandler_GetByID tests the GetByID endpoint.
func TestReferenceHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		category   string
	}{
		{
			name: "existing_reference",
			id:   "ref-123",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "nonexistent_reference",
			id:         "nonexistent",
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodGet, "/api/v1/references/"+tc.id, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestReferenceHandler_GetBySlug tests the GetBySlug endpoint.
func TestReferenceHandler_GetBySlug(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		category   string
	}{
		{
			name: "existing_reference",
			slug: "test-reference",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithSlug("test-reference").
					Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "nonexistent_slug",
			slug:       "nonexistent-slug",
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodGet, "/api/v1/references/by-slug/"+tc.slug, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestReferenceHandler_Update tests the Update endpoint.
func TestReferenceHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		category   string
	}{
		{
			name: "update_name",
			id:   "ref-123",
			body: `{"name":"Updated Name"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "nonexistent_reference",
			id:         "nonexistent",
			body:       `{"name":"Updated"}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
		{
			name: "invalid_json",
			id:   "ref-123",
			body: `{invalid}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusBadRequest,
			category:   "error",
		},
		{
			name: "invalid_path",
			id:   "ref-123",
			body: `{"path":"/nonexistent/path"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusBadRequest,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/references/"+tc.id, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestReferenceHandler_Delete tests the Delete endpoint.
func TestReferenceHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		category   string
	}{
		{
			name: "delete_existing",
			id:   "ref-123",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:       "delete_nonexistent",
			id:         "nonexistent",
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodDelete, "/api/v1/references/"+tc.id, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestWriteJSON tests the JSON serialization helper.
func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
		category   string
	}{
		{
			name:       "success_response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "created_response",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeJSON(rec, tc.status, tc.data)

			testutil.AssertStatus(t, rec, tc.wantStatus)
			testutil.AssertContentType(t, rec, "application/json")
		})
	}
}

// TestWriteError tests the error response helper.
func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "test error message")

	testutil.AssertStatus(t, rec, http.StatusBadRequest)
	testutil.AssertContentType(t, rec, "application/json")

	var response struct {
		Error string `json:"error"`
	}
	testutil.AssertJSON(t, rec, &response)
	if response.Error != "test error message" {
		t.Errorf("expected error %q, got %q", "test error message", response.Error)
	}
}

// TestNewReferenceHandlerWithConfig tests the config-based constructor.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - custom config constructor
func TestNewReferenceHandlerWithConfig(t *testing.T) {
	repo := mocks.NewMockRepository()
	service := reference.NewService(repo)

	customCfg := config.ValidationConfig{
		SlugMinLength: 5,
		SlugMaxLength: 50,
	}

	handler := NewReferenceHandlerWithConfig(service, customCfg)
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}
	if handler.config.SlugMinLength != 5 {
		t.Errorf("expected SlugMinLength 5, got %d", handler.config.SlugMinLength)
	}
	if handler.config.SlugMaxLength != 50 {
		t.Errorf("expected SlugMaxLength 50, got %d", handler.config.SlugMaxLength)
	}
}

// setupTestRouterWithConfig creates a router with a custom config.
func setupTestRouterWithConfig(repo *mocks.MockRepository, cfg config.ValidationConfig) *mux.Router {
	service := reference.NewService(repo)
	handler := NewReferenceHandlerWithConfig(service, cfg)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return router
}

// TestReferenceHandler_Create_DryRun tests dry-run mode for Create.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - dry-run validation
func TestReferenceHandler_Create_DryRun(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		body       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		wantDryRun bool
		category   string
	}{
		{
			name:       "dry_run_valid_input",
			body:       `{"slug":"test-scenario","name":"Test Scenario","template":"react-vite","path":"` + tempDir + `"}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusOK,
			wantDryRun: true,
			category:   "happy_path",
		},
		{
			name:       "dry_run_invalid_slug",
			body:       `{"slug":"Bad-Slug","name":"Test","template":"react-vite","path":"` + tempDir + `"}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusBadRequest,
			category:   "validation",
		},
		{
			name: "dry_run_duplicate_slug",
			body: `{"slug":"existing","name":"Test","template":"react-vite","path":"` + tempDir + `"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().WithSlug("existing").Build())
			},
			wantStatus: http.StatusConflict,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/references", strings.NewReader(tc.body))
			req.Header.Set("X-Dry-Run", "true")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
			testutil.AssertContentType(t, rec, "application/json")

			if tc.wantDryRun {
				var response struct {
					DryRun  bool   `json:"dry_run"`
					Success bool   `json:"success"`
					Data    struct {
						ID   string `json:"id"`
						Slug string `json:"slug"`
					} `json:"data"`
				}
				testutil.AssertJSON(t, rec, &response)
				if !response.DryRun {
					t.Error("expected dry_run to be true")
				}
				if !response.Success {
					t.Error("expected success to be true")
				}
				if response.Data.ID == "" {
					t.Error("expected data.id to be non-empty")
				}
			}
		})
	}
}

// TestReferenceHandler_Update_DryRun tests dry-run mode for Update.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - dry-run validation
func TestReferenceHandler_Update_DryRun(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		id         string
		body       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		wantDryRun bool
		category   string
	}{
		{
			name: "dry_run_update_name",
			id:   "ref-123",
			body: `{"name":"Updated Name"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					WithName("Original Name").
					Build())
			},
			wantStatus: http.StatusOK,
			wantDryRun: true,
			category:   "happy_path",
		},
		{
			name: "dry_run_update_all_fields",
			id:   "ref-123",
			body: `{"name":"New Name","template":"go-api","path":"` + tempDir + `","description":"New desc"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusOK,
			wantDryRun: true,
			category:   "happy_path",
		},
		{
			name:       "dry_run_not_found",
			id:         "nonexistent",
			body:       `{"name":"Updated"}`,
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
		{
			name: "dry_run_invalid_path",
			id:   "ref-123",
			body: `{"path":"/nonexistent/path"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusBadRequest,
			category:   "validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/references/"+tc.id, strings.NewReader(tc.body))
			req.Header.Set("X-Dry-Run", "true")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantDryRun {
				var response struct {
					DryRun  bool `json:"dry_run"`
					Success bool `json:"success"`
					Data    struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"data"`
				}
				testutil.AssertJSON(t, rec, &response)
				if !response.DryRun {
					t.Error("expected dry_run to be true")
				}
				if !response.Success {
					t.Error("expected success to be true")
				}
			}
		})
	}
}

// TestReferenceHandler_Delete_DryRun tests dry-run mode for Delete.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - dry-run validation
func TestReferenceHandler_Delete_DryRun(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		wantDryRun bool
		category   string
	}{
		{
			name: "dry_run_delete_existing",
			id:   "ref-123",
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusOK,
			wantDryRun: true,
			category:   "happy_path",
		},
		{
			name:       "dry_run_delete_nonexistent",
			id:         "nonexistent",
			setupMock:  func(m *mocks.MockRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodDelete, "/api/v1/references/"+tc.id, nil)
			req.Header.Set("X-Dry-Run", "true")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantDryRun {
				var response struct {
					DryRun  bool   `json:"dry_run"`
					Success bool   `json:"success"`
					Deleted string `json:"deleted"`
				}
				testutil.AssertJSON(t, rec, &response)
				if !response.DryRun {
					t.Error("expected dry_run to be true")
				}
				if !response.Success {
					t.Error("expected success to be true")
				}
				if response.Deleted != tc.id {
					t.Errorf("expected deleted %q, got %q", tc.id, response.Deleted)
				}
			}
		})
	}
}

// TestReferenceHandler_Create_WithCustomConfig tests error message formatting with custom config.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - custom config for error messages
func TestReferenceHandler_Create_WithCustomConfig(t *testing.T) {
	// Custom config affects error message formatting (e.g., slug length constraints in error messages)
	// The actual validation happens at the service layer with service config
	customCfg := config.ValidationConfig{
		SlugMinLength: 5,
		SlugMaxLength: 10,
	}

	repo := mocks.NewMockRepository()
	router := setupTestRouterWithConfig(repo, customCfg)

	// Test that config is properly used - create a reference with invalid slug format
	// (not length-based, since that's service-level, but format-based)
	body := `{"slug":"INVALID-CAPS","name":"Test","template":"react-vite","path":"/tmp"}`
	req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/references", strings.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Invalid slug format should fail validation
	testutil.AssertStatus(t, rec, http.StatusBadRequest)

	var response struct {
		Error string `json:"error"`
	}
	testutil.AssertJSON(t, rec, &response)
	// Error message should mention slug constraints
	if !strings.Contains(response.Error, "lowercase") && !strings.Contains(response.Error, "slug") {
		t.Errorf("expected error message about slug, got: %s", response.Error)
	}
}

// TestReferenceHandler_Update_PartialFields tests Update with various field combinations.
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - partial update coverage
func TestReferenceHandler_Update_PartialFields(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		id         string
		body       string
		setupMock  func(*mocks.MockRepository)
		wantStatus int
		category   string
	}{
		{
			name: "update_template_only",
			id:   "ref-123",
			body: `{"template":"go-api"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					WithTemplate("react-vite").
					Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name: "update_description_only",
			id:   "ref-123",
			body: `{"description":"New description here"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name: "update_path_valid",
			id:   "ref-123",
			body: `{"path":"` + tempDir + `"}`,
			setupMock: func(m *mocks.MockRepository) {
				m.WithReference(testutil.NewReferenceFactory().
					WithID("ref-123").
					Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockRepository()
			tc.setupMock(repo)
			router := setupTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/references/"+tc.id, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}
