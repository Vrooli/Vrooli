// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store - HTTP handler tests
package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"development-toolchain-validator/domain/skill"
	"development-toolchain-validator/internal/mocks"
	"development-toolchain-validator/internal/testutil"
)

// setupSkillTestRouter creates a router with the skill handler registered.
func setupSkillTestRouter(repo *mocks.MockSkillRepository) *mux.Router {
	service := skill.NewService(repo)
	handler := NewSkillHandler(service)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return router
}

// TestSkillHandler_List tests the List endpoint.
func TestSkillHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setupMock  func(*mocks.MockSkillRepository)
		wantStatus int
		category   string
	}{
		{
			name: "list_all_connections",
			path: "/api/v1/connections",
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "conn-1",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
				m.WithConnection(&skill.Connection{
					ID:          "conn-2",
					ReferenceID: "ref-456",
					SkillID:     "cli-steer",
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "empty_list",
			path:       "/api/v1/connections",
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusOK,
			category:   "boundary",
		},
		{
			name: "filter_by_reference_id",
			path: "/api/v1/connections?reference_id=ref-123",
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "conn-1",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
				m.WithConnection(&skill.Connection{
					ID:          "conn-2",
					ReferenceID: "ref-456",
					SkillID:     "cli-steer",
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
			testutil.AssertContentType(t, rec, "application/json")
		})
	}
}

// TestSkillHandler_Connect tests the Connect endpoint.
func TestSkillHandler_Connect(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setupMock  func(*mocks.MockSkillRepository)
		wantStatus int
		category   string
	}{
		{
			name: "connect_success",
			body: `{"reference_id": "ref-123", "skill_id": "api-steer", "skill_version": "v1.0"}`,
			setupMock: func(m *mocks.MockSkillRepository) {
				// Empty - no existing connection
			},
			wantStatus: http.StatusCreated,
			category:   "happy_path",
		},
		{
			name:       "invalid_json",
			body:       `{invalid}`,
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusBadRequest,
			category:   "error",
		},
		{
			name:       "invalid_skill_id",
			body:       `{"reference_id": "ref-123", "skill_id": "123-invalid"}`,
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusBadRequest,
			category:   "validation",
		},
		{
			name:       "missing_reference_id",
			body:       `{"skill_id": "api-steer"}`,
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusBadRequest,
			category:   "validation",
		},
		{
			name: "connection_already_exists",
			body: `{"reference_id": "ref-123", "skill_id": "api-steer"}`,
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "existing-conn",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
			},
			wantStatus: http.StatusConflict,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/connections", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_Connect_DryRun tests dry-run mode for Connect.
func TestSkillHandler_Connect_DryRun(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	router := setupSkillTestRouter(repo)

	body := `{"reference_id": "ref-123", "skill_id": "api-steer", "skill_version": "v1.0"}`
	req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/connections", strings.NewReader(body))
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusOK)
	// Verify no connection was created
	if repo.ConnectCallCount() != 0 {
		t.Error("expected no connection to be created during dry-run")
	}
}

// TestSkillHandler_GetByID tests the GetByID endpoint.
func TestSkillHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setupMock  func(*mocks.MockSkillRepository)
		wantStatus int
		category   string
	}{
		{
			name: "get_existing",
			id:   "conn-123",
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "conn-123",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "not_found",
			id:         "nonexistent",
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodGet, "/api/v1/connections/"+tc.id, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_Disconnect tests the Disconnect endpoint.
func TestSkillHandler_Disconnect(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setupMock  func(*mocks.MockSkillRepository)
		wantStatus int
		category   string
	}{
		{
			name: "disconnect_success",
			id:   "conn-123",
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "conn-123",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
			},
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:       "not_found",
			id:         "nonexistent",
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodDelete, "/api/v1/connections/"+tc.id, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_Disconnect_DryRun tests dry-run mode for Disconnect.
func TestSkillHandler_Disconnect_DryRun(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	repo.WithConnection(&skill.Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	})
	router := setupSkillTestRouter(repo)

	req := testutil.MakeRequest(t, http.MethodDelete, "/api/v1/connections/conn-123", nil)
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusOK)
	// Verify connection was not deleted
	if repo.DisconnectCallCount() != 0 {
		t.Error("expected no disconnection during dry-run")
	}
}

// TestSkillHandler_CheckDrift tests the CheckDrift endpoint.
func TestSkillHandler_CheckDrift(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		setupMock  func(*mocks.MockSkillRepository)
		wantStatus int
		category   string
	}{
		{
			name: "no_drift",
			id:   "conn-123",
			body: `{"current_version": "v1.0", "current_hash": "hash123"}`,
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:               "conn-123",
					ReferenceID:      "ref-123",
					SkillID:          "api-steer",
					SkillVersion:     "v1.0",
					SkillContentHash: "hash123",
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name: "drift_detected",
			id:   "conn-123",
			body: `{"current_version": "v2.0", "current_hash": "hash456"}`,
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:               "conn-123",
					ReferenceID:      "ref-123",
					SkillID:          "api-steer",
					SkillVersion:     "v1.0",
					SkillContentHash: "hash123",
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "connection_not_found",
			id:         "nonexistent",
			body:       `{"current_version": "v1.0", "current_hash": "hash123"}`,
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/connections/"+tc.id+"/drift", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_Update tests the Update endpoint.
func TestSkillHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		setupMock  func(*mocks.MockSkillRepository)
		wantStatus int
		category   string
	}{
		{
			name: "update_success",
			id:   "conn-123",
			body: `{"skill_version": "v2.0", "skill_content_hash": "newhash"}`,
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:               "conn-123",
					ReferenceID:      "ref-123",
					SkillID:          "api-steer",
					SkillVersion:     "v1.0",
					SkillContentHash: "oldhash",
					ConnectedAt:      time.Now(),
					UpdatedAt:        time.Now(),
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:       "not_found",
			id:         "nonexistent",
			body:       `{"skill_version": "v2.0"}`,
			setupMock:  func(m *mocks.MockSkillRepository) {},
			wantStatus: http.StatusNotFound,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/connections/"+tc.id, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_GetByReferenceAndSkill tests the GetByReferenceAndSkill endpoint.
func TestSkillHandler_GetByReferenceAndSkill(t *testing.T) {
	tests := []struct {
		name        string
		referenceID string
		skillID     string
		setupMock   func(*mocks.MockSkillRepository)
		wantStatus  int
		category    string
	}{
		{
			name:        "get_existing",
			referenceID: "ref-123",
			skillID:     "api-steer",
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "conn-123",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:        "not_found",
			referenceID: "ref-999",
			skillID:     "nonexistent",
			setupMock:   func(m *mocks.MockSkillRepository) {},
			wantStatus:  http.StatusNotFound,
			category:    "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodGet, "/api/v1/references/"+tc.referenceID+"/connections/"+tc.skillID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_DisconnectByReferenceAndSkill tests disconnection by reference+skill.
func TestSkillHandler_DisconnectByReferenceAndSkill(t *testing.T) {
	tests := []struct {
		name        string
		referenceID string
		skillID     string
		setupMock   func(*mocks.MockSkillRepository)
		wantStatus  int
		category    string
	}{
		{
			name:        "disconnect_existing",
			referenceID: "ref-123",
			skillID:     "api-steer",
			setupMock: func(m *mocks.MockSkillRepository) {
				m.WithConnection(&skill.Connection{
					ID:          "conn-123",
					ReferenceID: "ref-123",
					SkillID:     "api-steer",
				})
			},
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:        "not_found",
			referenceID: "ref-999",
			skillID:     "nonexistent",
			setupMock:   func(m *mocks.MockSkillRepository) {},
			wantStatus:  http.StatusNotFound,
			category:    "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockSkillRepository()
			tc.setupMock(repo)
			router := setupSkillTestRouter(repo)

			req := testutil.MakeRequest(t, http.MethodDelete, "/api/v1/references/"+tc.referenceID+"/connections/"+tc.skillID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// TestSkillHandler_DisconnectByReferenceAndSkill_DryRun tests dry-run mode for disconnect by reference+skill.
func TestSkillHandler_DisconnectByReferenceAndSkill_DryRun(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	repo.WithConnection(&skill.Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	})
	router := setupSkillTestRouter(repo)

	req := testutil.MakeRequest(t, http.MethodDelete, "/api/v1/references/ref-123/connections/api-steer", nil)
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusOK)
	// Verify connection was not deleted
	if repo.DisconnectCallCount() != 0 {
		t.Error("expected no disconnection during dry-run")
	}
}

// TestSkillHandler_Update_DryRun tests dry-run mode for Update.
func TestSkillHandler_Update_DryRun(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	repo.WithConnection(&skill.Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "oldhash",
		ConnectedAt:      time.Now(),
		UpdatedAt:        time.Now(),
	})
	router := setupSkillTestRouter(repo)

	body := `{"skill_version": "v2.0", "skill_content_hash": "newhash"}`
	req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/connections/conn-123", strings.NewReader(body))
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusOK)
	// Verify update was not persisted
	if repo.UpdateCallCount() != 0 {
		t.Error("expected no update during dry-run")
	}
}

// TestSkillHandler_Update_DryRun_NotFound tests dry-run for non-existent connection.
func TestSkillHandler_Update_DryRun_NotFound(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	router := setupSkillTestRouter(repo)

	body := `{"skill_version": "v2.0"}`
	req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/connections/nonexistent", strings.NewReader(body))
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusNotFound)
}

// TestSkillHandler_Update_InvalidJSON tests update with invalid JSON.
func TestSkillHandler_Update_InvalidJSON(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	repo.WithConnection(&skill.Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	})
	router := setupSkillTestRouter(repo)

	req := testutil.MakeRequest(t, http.MethodPatch, "/api/v1/connections/conn-123", strings.NewReader("{invalid}"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusBadRequest)
}

// TestSkillHandler_CheckDrift_InvalidJSON tests drift check with invalid JSON.
func TestSkillHandler_CheckDrift_InvalidJSON(t *testing.T) {
	repo := mocks.NewMockSkillRepository()
	repo.WithConnection(&skill.Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	})
	router := setupSkillTestRouter(repo)

	req := testutil.MakeRequest(t, http.MethodPost, "/api/v1/connections/conn-123/drift", strings.NewReader("{invalid}"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusBadRequest)
}
