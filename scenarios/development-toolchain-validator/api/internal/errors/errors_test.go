package errors_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	apierrors "development-toolchain-validator/internal/errors"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *apierrors.Error
		contains string
	}{
		{
			name:     "simple error",
			err:      apierrors.Validation("test_code", "Test message"),
			contains: "test_code: Test message",
		},
		{
			name:     "error with cause",
			err:      apierrors.Internal("Failed").WithCause(errors.New("underlying")),
			contains: "underlying",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !containsSubstring(got, tt.contains) {
				t.Errorf("Error() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := apierrors.Internal("wrapper").WithCause(cause)

	if !errors.Is(err, cause) {
		t.Error("Unwrap() should allow errors.Is to find the cause")
	}
}

func TestError_WithDetails(t *testing.T) {
	err := apierrors.Validation("test", "message").
		WithDetails("key1", "value1").
		WithDetails("key2", 42)

	if err.Details["key1"] != "value1" {
		t.Errorf("Details[key1] = %v, want value1", err.Details["key1"])
	}
	if err.Details["key2"] != 42 {
		t.Errorf("Details[key2] = %v, want 42", err.Details["key2"])
	}
}

func TestError_ToHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    *apierrors.Error
		status int
	}{
		{
			name:   "validation error",
			err:    apierrors.Validation("test", "message"),
			status: http.StatusBadRequest,
		},
		{
			name:   "not found error",
			err:    apierrors.NotFound("resource", "123"),
			status: http.StatusNotFound,
		},
		{
			name:   "conflict error",
			err:    apierrors.Conflict("test", "message"),
			status: http.StatusConflict,
		},
		{
			name:   "database error (transient)",
			err:    apierrors.Database("db failed", true),
			status: http.StatusServiceUnavailable,
		},
		{
			name:   "database error (permanent)",
			err:    apierrors.Database("db failed", false),
			status: http.StatusInternalServerError,
		},
		{
			name:   "internal error",
			err:    apierrors.Internal("oops"),
			status: http.StatusInternalServerError,
		},
		{
			name:   "dependency error (transient)",
			err:    apierrors.Dependency("external-api", "timeout", true),
			status: http.StatusServiceUnavailable,
		},
		{
			name:   "dependency error (permanent)",
			err:    apierrors.Dependency("external-api", "invalid response", false),
			status: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.ToHTTPStatus(); got != tt.status {
				t.Errorf("ToHTTPStatus() = %d, want %d", got, tt.status)
			}
		})
	}
}

func TestError_ToJSON(t *testing.T) {
	err := apierrors.Validation("invalid_input", "Input is invalid").
		WithDetails("field", "name")

	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("ToJSON() error = %v", jsonErr)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonBytes, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	errObj, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["code"] != "invalid_input" {
		t.Errorf("code = %v, want invalid_input", errObj["code"])
	}
	if errObj["category"] != "validation" {
		t.Errorf("category = %v, want validation", errObj["category"])
	}
	if errObj["recovery"] == nil || errObj["recovery"] == "" {
		t.Error("Expected recovery guidance in response")
	}
}

func TestError_IsTransient(t *testing.T) {
	transient := apierrors.Database("temp failure", true)
	if !transient.IsTransient() {
		t.Error("Expected transient error to return true")
	}

	permanent := apierrors.Database("perm failure", false)
	if permanent.IsTransient() {
		t.Error("Expected permanent error to return false")
	}
}

// Domain-specific error constructor tests

func TestInvalidSlug(t *testing.T) {
	err := apierrors.InvalidSlug("bad slug!", 2, 100)

	if err.Category != apierrors.CategoryValidation {
		t.Errorf("Category = %v, want validation", err.Category)
	}
	if err.Code != "invalid_slug" {
		t.Errorf("Code = %v, want invalid_slug", err.Code)
	}
	if err.Details["provided"] != "bad slug!" {
		t.Errorf("Details[provided] = %v, want 'bad slug!'", err.Details["provided"])
	}
	if err.Details["min_length"] != 2 {
		t.Errorf("Details[min_length] = %v, want 2", err.Details["min_length"])
	}
}

func TestSlugExists(t *testing.T) {
	err := apierrors.SlugExists("existing-slug")

	if err.Category != apierrors.CategoryConflict {
		t.Errorf("Category = %v, want conflict", err.Category)
	}
	if err.Code != "slug_exists" {
		t.Errorf("Code = %v, want slug_exists", err.Code)
	}
}

func TestPathNotExists(t *testing.T) {
	err := apierrors.PathNotExists("/nonexistent/path")

	if err.Category != apierrors.CategoryValidation {
		t.Errorf("Category = %v, want validation", err.Category)
	}
	if err.Code != "path_not_exists" {
		t.Errorf("Code = %v, want path_not_exists", err.Code)
	}
}

func TestReferenceNotFound(t *testing.T) {
	err := apierrors.ReferenceNotFound("abc-123")

	if err.Category != apierrors.CategoryNotFound {
		t.Errorf("Category = %v, want not_found", err.Category)
	}
	if err.Details["resource"] != "reference" {
		t.Errorf("Details[resource] = %v, want reference", err.Details["resource"])
	}
}

func TestInvalidRequestBody(t *testing.T) {
	err := apierrors.InvalidRequestBody("unexpected EOF")

	if err.Category != apierrors.CategoryValidation {
		t.Errorf("Category = %v, want validation", err.Category)
	}
	if err.Code != "invalid_request_body" {
		t.Errorf("Code = %v, want invalid_request_body", err.Code)
	}
}

// Recovery path tests

func TestRecoveryPaths(t *testing.T) {
	tests := []struct {
		name     string
		err      *apierrors.Error
		hasHint  bool
		contains string
	}{
		{
			name:     "validation has user-actionable recovery",
			err:      apierrors.Validation("test", "message"),
			hasHint:  true,
			contains: "Check",
		},
		{
			name:     "not found suggests refresh",
			err:      apierrors.NotFound("item", "123"),
			hasHint:  true,
			contains: "Check",
		},
		{
			name:     "transient db suggests retry",
			err:      apierrors.Database("failed", true),
			hasHint:  true,
			contains: "try again",
		},
		{
			name:     "permanent db suggests support",
			err:      apierrors.Database("failed", false),
			hasHint:  true,
			contains: "support",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hasHint && tt.err.Recovery == "" {
				t.Error("Expected recovery hint but got empty string")
			}
			if tt.contains != "" && !containsSubstring(tt.err.Recovery, tt.contains) {
				t.Errorf("Recovery = %q, want to contain %q", tt.err.Recovery, tt.contains)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
