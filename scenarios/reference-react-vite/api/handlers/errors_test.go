// DOC: docs/internal/ERROR_SEMANTICS.md
// [REQ:RRV-API-ERR] Error Semantics - Tests for error response format and recovery hints
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"reference-react-vite/api/handlers"
	"reference-react-vite/api/internal/mocks"
	"reference-react-vite/api/internal/testutil"
)

// APIError represents the expected error response structure.
type APIError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Recovery  string                 `json:"recovery,omitempty"`
	Retryable bool                   `json:"retryable"`
	RequestID string                 `json:"request_id,omitempty"`
}

// =============================================================================
// Error Response Format Tests
// =============================================================================

func TestErrorResponse_Format(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		setupMock      func(*mocks.MockTaskRepository)
		wantStatus     int
		wantCode       string
		wantRetryable  bool
		wantRecovery   bool // Should have recovery hint
		wantRequestID  bool // Should have request ID
	}{
		{
			name:          "validation_error_has_recovery_hint",
			method:        http.MethodPost,
			path:          "/api/v1/tasks",
			body:          `{"title": ""}`,
			setupMock:     func(m *mocks.MockTaskRepository) {},
			wantStatus:    http.StatusUnprocessableEntity,
			wantCode:      "VALIDATION_ERROR",
			wantRetryable: false,
			wantRecovery:  true,
			wantRequestID: true,
		},
		{
			name:          "not_found_error_has_recovery_hint",
			method:        http.MethodGet,
			path:          "/api/v1/tasks/nonexistent",
			body:          "",
			setupMock:     func(m *mocks.MockTaskRepository) {},
			wantStatus:    http.StatusNotFound,
			wantCode:      "NOT_FOUND",
			wantRetryable: false,
			wantRecovery:  true,
			wantRequestID: true,
		},
		{
			name:          "bad_request_has_recovery_hint",
			method:        http.MethodPost,
			path:          "/api/v1/tasks",
			body:          `{invalid json`,
			setupMock:     func(m *mocks.MockTaskRepository) {},
			wantStatus:    http.StatusBadRequest,
			wantCode:      "BAD_REQUEST",
			wantRetryable: false,
			wantRecovery:  true,
			wantRequestID: true,
		},
		{
			name:   "internal_error_is_retryable",
			method: http.MethodGet,
			path:   "/api/v1/tasks",
			body:   "",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithListError(testutil.ErrDatabase)
			},
			wantStatus:    http.StatusInternalServerError,
			wantCode:      "INTERNAL_ERROR",
			wantRetryable: true,
			wantRecovery:  true,
			wantRequestID: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := mux.NewRouter()
			h := handlers.NewTaskHandler(repo, handlers.PaginationConfig{
				DefaultLimit: 20,
				MaxLimit:     100,
			})
			h.RegisterRoutes(router)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			var apiErr APIError
			if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			// Check error code
			if apiErr.Code != tc.wantCode {
				t.Errorf("expected code %q, got %q", tc.wantCode, apiErr.Code)
			}

			// Check retryable flag
			if apiErr.Retryable != tc.wantRetryable {
				t.Errorf("expected retryable=%v, got %v", tc.wantRetryable, apiErr.Retryable)
			}

			// Check recovery hint presence
			if tc.wantRecovery && apiErr.Recovery == "" {
				t.Error("expected recovery hint, got empty string")
			}

			// Check request ID presence
			if tc.wantRequestID && apiErr.RequestID == "" {
				t.Error("expected request_id, got empty string")
			}

			// Check message is not empty
			if apiErr.Message == "" {
				t.Error("expected non-empty message")
			}
		})
	}
}

// =============================================================================
// Recovery Hint Content Tests
// =============================================================================

func TestErrorResponse_RecoveryHints(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		body             string
		setupMock        func(*mocks.MockTaskRepository)
		wantCode         string
		recoveryContains string // Substring expected in recovery hint
	}{
		{
			name:             "validation_error_mentions_details",
			method:           http.MethodPost,
			path:             "/api/v1/tasks",
			body:             `{"title": ""}`,
			setupMock:        func(m *mocks.MockTaskRepository) {},
			wantCode:         "VALIDATION_ERROR",
			recoveryContains: "details",
		},
		{
			name:             "not_found_suggests_list_endpoint",
			method:           http.MethodGet,
			path:             "/api/v1/tasks/nonexistent",
			body:             "",
			setupMock:        func(m *mocks.MockTaskRepository) {},
			wantCode:         "NOT_FOUND",
			recoveryContains: "list",
		},
		{
			name:             "bad_request_mentions_api_docs",
			method:           http.MethodPost,
			path:             "/api/v1/tasks",
			body:             `{invalid`,
			setupMock:        func(m *mocks.MockTaskRepository) {},
			wantCode:         "BAD_REQUEST",
			recoveryContains: "format",
		},
		{
			name:   "internal_error_suggests_retry",
			method: http.MethodGet,
			path:   "/api/v1/tasks",
			body:   "",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithListError(testutil.ErrDatabase)
			},
			wantCode:         "INTERNAL_ERROR",
			recoveryContains: "retry",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := mux.NewRouter()
			h := handlers.NewTaskHandler(repo, handlers.PaginationConfig{
				DefaultLimit: 20,
				MaxLimit:     100,
			})
			h.RegisterRoutes(router)

			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			var apiErr APIError
			if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if apiErr.Code != tc.wantCode {
				t.Errorf("expected code %q, got %q", tc.wantCode, apiErr.Code)
			}

			if !strings.Contains(strings.ToLower(apiErr.Recovery), tc.recoveryContains) {
				t.Errorf("expected recovery to contain %q, got %q", tc.recoveryContains, apiErr.Recovery)
			}
		})
	}
}

// =============================================================================
// Request ID Propagation Tests
// =============================================================================

func TestErrorResponse_RequestIDPropagation(t *testing.T) {
	// ARRANGE
	repo := mocks.NewMockTaskRepository()
	router := mux.NewRouter()
	h := handlers.NewTaskHandler(repo, handlers.PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
	})
	h.RegisterRoutes(router)

	t.Run("uses_provided_request_id", func(t *testing.T) {
		providedID := "client-request-123"
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent", nil)
		req.Header.Set("X-Request-ID", providedID)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var apiErr APIError
		if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}

		if apiErr.RequestID != providedID {
			t.Errorf("expected request_id %q, got %q", providedID, apiErr.RequestID)
		}
	})

	t.Run("generates_request_id_if_not_provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var apiErr APIError
		if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}

		if apiErr.RequestID == "" {
			t.Error("expected generated request_id, got empty string")
		}
	})
}

// =============================================================================
// Typed Error Detection Tests
// =============================================================================

func TestTypedErrorDetection(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		setupMock  func(*mocks.MockTaskRepository)
		wantStatus int
		wantCode   string
	}{
		{
			name:   "delete_nonexistent_returns_not_found",
			taskID: "nonexistent",
			setupMock: func(m *mocks.MockTaskRepository) {
				// No tasks in the mock
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:   "delete_existing_succeeds",
			taskID: "task-123",
			setupMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			wantStatus: http.StatusNoContent,
			wantCode:   "", // No error
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockTaskRepository()
			tc.setupMock(repo)
			router := mux.NewRouter()
			h := handlers.NewTaskHandler(repo, handlers.PaginationConfig{
				DefaultLimit: 20,
				MaxLimit:     100,
			})
			h.RegisterRoutes(router)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+tc.taskID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantCode != "" {
				var apiErr APIError
				if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if apiErr.Code != tc.wantCode {
					t.Errorf("expected code %q, got %q", tc.wantCode, apiErr.Code)
				}
			}
		})
	}
}

// =============================================================================
// Validation Error Details Tests
// =============================================================================

func TestValidationErrorDetails(t *testing.T) {
	repo := mocks.NewMockTaskRepository()
	router := mux.NewRouter()
	h := handlers.NewTaskHandler(repo, handlers.PaginationConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
	})
	h.RegisterRoutes(router)

	t.Run("invalid_status_includes_details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?status=invalid_status", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusUnprocessableEntity)

		var apiErr APIError
		if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}

		if apiErr.Code != "VALIDATION_ERROR" {
			t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Code)
		}

		// Should have details about the invalid status
		if apiErr.Details == nil {
			t.Error("expected details to be present for validation error")
		}
		if _, ok := apiErr.Details["status"]; !ok {
			t.Error("expected details to contain 'status' field")
		}
	})

	t.Run("invalid_priority_includes_details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?priority=invalid", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		testutil.AssertStatus(t, rec, http.StatusUnprocessableEntity)

		var apiErr APIError
		if err := json.NewDecoder(rec.Body).Decode(&apiErr); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}

		if apiErr.Code != "VALIDATION_ERROR" {
			t.Errorf("expected code VALIDATION_ERROR, got %q", apiErr.Code)
		}

		// Should have details about the invalid priority
		if apiErr.Details == nil {
			t.Error("expected details to be present for validation error")
		}
	})
}
