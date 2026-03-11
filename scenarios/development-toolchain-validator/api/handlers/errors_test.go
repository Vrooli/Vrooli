// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// [REQ:REQ-P0-002] Reference Scenario API Endpoints - Error mapping tests
package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"development-toolchain-validator/domain/reference"
	apierrors "development-toolchain-validator/internal/errors"
)

// TestMapDomainError tests domain error mapping to API errors.
func TestMapDomainError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		cfg           ErrorMappingConfig
		wantStatus    int
		wantCategory  apierrors.Category
		wantInMessage string   // Optional: expected string in message
		wantDetailKey string   // Optional: expected key in details
	}{
		{
			name: "reference_not_found",
			err:  reference.ErrNotFound,
			cfg: ErrorMappingConfig{
				ResourceID: "ref-123",
			},
			wantStatus:    http.StatusNotFound,
			wantCategory:  apierrors.CategoryNotFound,
			wantInMessage: "not found",
			wantDetailKey: "id",
		},
		{
			name: "invalid_slug",
			err:  reference.ErrInvalidSlug,
			cfg: ErrorMappingConfig{
				Slug:       "Bad-Slug",
				SlugMinLen: 2,
				SlugMaxLen: 50,
			},
			wantStatus:    http.StatusBadRequest,
			wantCategory:  apierrors.CategoryValidation,
			wantInMessage: "lowercase",
			wantDetailKey: "provided",
		},
		{
			name: "slug_exists",
			err:  reference.ErrSlugExists,
			cfg: ErrorMappingConfig{
				Slug: "existing-slug",
			},
			wantStatus:    http.StatusConflict,
			wantCategory:  apierrors.CategoryConflict,
			wantInMessage: "already exists",
		},
		{
			name: "path_not_exists",
			err:  reference.ErrPathNotExists,
			cfg: ErrorMappingConfig{
				Path: "/nonexistent/path",
			},
			wantStatus:    http.StatusBadRequest,
			wantCategory:  apierrors.CategoryValidation,
			wantInMessage: "path does not exist",
		},
		{
			name:          "unknown_error",
			err:           errors.New("some unexpected error"),
			cfg:           ErrorMappingConfig{},
			wantStatus:    http.StatusInternalServerError,
			wantCategory:  apierrors.CategoryInternal,
			wantInMessage: "unexpected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := MapDomainError(tc.err, tc.cfg)

			if apiErr.ToHTTPStatus() != tc.wantStatus {
				t.Errorf("MapDomainError() status = %d, want %d", apiErr.ToHTTPStatus(), tc.wantStatus)
			}

			if apiErr.Category != tc.wantCategory {
				t.Errorf("MapDomainError() category = %v, want %v", apiErr.Category, tc.wantCategory)
			}

			if tc.wantInMessage != "" && !strings.Contains(apiErr.Message, tc.wantInMessage) {
				t.Errorf("MapDomainError() message %q should contain %q", apiErr.Message, tc.wantInMessage)
			}

			if tc.wantDetailKey != "" && apiErr.Details != nil {
				if _, ok := apiErr.Details[tc.wantDetailKey]; !ok {
					t.Errorf("MapDomainError() details should contain key %q, got: %v", tc.wantDetailKey, apiErr.Details)
				}
			}
		})
	}
}

// TestWriteStructuredError tests structured error response writing.
func TestWriteStructuredError(t *testing.T) {
	tests := []struct {
		name           string
		err            *apierrors.Error
		wantStatus     int
		wantErrorField string
	}{
		{
			name:           "not_found_error",
			err:            apierrors.ReferenceNotFound("ref-123"),
			wantStatus:     http.StatusNotFound,
			wantErrorField: "ref-123",
		},
		{
			name:           "validation_error",
			err:            apierrors.InvalidSlug("bad", 2, 50),
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "bad",
		},
		{
			name:           "conflict_error",
			err:            apierrors.SlugExists("taken"),
			wantStatus:     http.StatusConflict,
			wantErrorField: "taken",
		},
		{
			name:           "internal_error",
			err:            apierrors.Internal("database failure"),
			wantStatus:     http.StatusInternalServerError,
			wantErrorField: "database failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			WriteStructuredError(rec, tc.err)

			if rec.Code != tc.wantStatus {
				t.Errorf("WriteStructuredError() status = %d, want %d", rec.Code, tc.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("WriteStructuredError() Content-Type = %q, want %q", contentType, "application/json")
			}

			body := rec.Body.String()
			if !strings.Contains(body, tc.wantErrorField) {
				t.Errorf("WriteStructuredError() body %q should contain %q", body, tc.wantErrorField)
			}

			// Verify the code field is present
			if !strings.Contains(body, `"code":`) {
				t.Errorf("WriteStructuredError() body %q should contain code field", body)
			}
		})
	}
}

// TestHandleCreateError tests the create error handler.
func TestHandleCreateError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		cfg        ErrorMappingConfig
		wantStatus int
	}{
		{
			name:       "not_found",
			err:        reference.ErrNotFound,
			cfg:        ErrorMappingConfig{ResourceID: "ref-1"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid_slug",
			err:        reference.ErrInvalidSlug,
			cfg:        ErrorMappingConfig{Slug: "Bad"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "slug_conflict",
			err:        reference.ErrSlugExists,
			cfg:        ErrorMappingConfig{Slug: "taken"},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := HandleCreateError(tc.err, tc.cfg)

			if status != tc.wantStatus {
				t.Errorf("HandleCreateError() status = %d, want %d", status, tc.wantStatus)
			}

			if msg == "" {
				t.Error("HandleCreateError() message should not be empty")
			}
		})
	}
}

// TestHandleGetError tests the get error handler.
func TestHandleGetError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		resourceID string
		wantStatus int
	}{
		{
			name:       "not_found",
			err:        reference.ErrNotFound,
			resourceID: "ref-123",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown_error",
			err:        errors.New("some error"),
			resourceID: "ref-456",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := HandleGetError(tc.err, tc.resourceID)

			if status != tc.wantStatus {
				t.Errorf("HandleGetError() status = %d, want %d", status, tc.wantStatus)
			}

			if msg == "" {
				t.Error("HandleGetError() message should not be empty")
			}
		})
	}
}

// TestErrorResponseStructure tests the error response JSON structure.
func TestErrorResponseStructure(t *testing.T) {
	err := apierrors.InvalidSlug("bad", 2, 50).
		WithDetails("hint", "use lowercase")

	rec := httptest.NewRecorder()
	WriteStructuredError(rec, err)

	body := rec.Body.String()

	// Verify key expected fields are present
	expectedFields := []string{
		`"error":`,    // Primary error message
		`"code":`,     // Machine-readable code
		`"category":`, // Error category
		`"details":`,  // Additional context
	}

	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Response should contain %s, got: %s", field, body)
		}
	}
}

// TestWriteErrorCompat tests the legacy error format helper.
func TestWriteErrorCompat(t *testing.T) {
	rec := httptest.NewRecorder()
	writeErrorCompat(rec, http.StatusBadRequest, "test error")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("writeErrorCompat() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"error":"test error"`) {
		t.Errorf("writeErrorCompat() body = %q, want error field", body)
	}

	// Ensure no structured fields are present
	if strings.Contains(body, `"category":`) {
		t.Error("writeErrorCompat() should not include structured error fields")
	}
}

// TestErrorSeverityLogging tests that different severities don't panic.
func TestErrorSeverityLogging(t *testing.T) {
	severities := []apierrors.Severity{
		apierrors.SeverityLow,
		apierrors.SeverityMedium,
		apierrors.SeverityHigh,
		apierrors.SeverityCritical,
	}

	for _, sev := range severities {
		t.Run(string(sev), func(t *testing.T) {
			err := &apierrors.Error{
				Category: apierrors.CategoryInternal,
				Code:     "test_error",
				Message:  "test message",
				Severity: sev,
			}

			// This should not panic
			rec := httptest.NewRecorder()
			WriteStructuredError(rec, err)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("WriteStructuredError() status = %d, want %d", rec.Code, http.StatusInternalServerError)
			}
		})
	}
}

// TestErrorWithCause tests error cause wrapping.
func TestErrorWithCause(t *testing.T) {
	cause := errors.New("underlying database error")
	err := apierrors.Internal("failed to save").WithCause(cause)

	rec := httptest.NewRecorder()
	WriteStructuredError(rec, err)

	// Cause should NOT be exposed in the response body
	body := rec.Body.String()
	if strings.Contains(body, "underlying database error") {
		t.Error("Response should not contain underlying error details")
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("WriteStructuredError() status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
