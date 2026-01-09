package errors

import (
	"errors"
	"testing"
)

// [REQ:LIFECYCLE-GRACEFUL-DEGRADATION] Tests for structured error types
func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		contains string
	}{
		{
			name: "simple error without cause",
			err: &AppError{
				Code:    ErrCodeInvalidInput,
				Message: "Invalid input",
			},
			contains: "INVALID_INPUT",
		},
		{
			name: "error with cause",
			err: &AppError{
				Code:    ErrCodeDatabaseError,
				Message: "Database error",
				Cause:   errors.New("connection refused"),
			},
			contains: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			if !containsString(errStr, tt.contains) {
				t.Errorf("Error() = %q, want to contain %q", errStr, tt.contains)
			}
		})
	}
}

// [REQ:LIFECYCLE-GRACEFUL-DEGRADATION] Tests for error unwrapping
func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	appErr := &AppError{
		Code:    ErrCodeInternal,
		Message: "Something went wrong",
		Cause:   cause,
	}

	unwrapped := appErr.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

// [REQ:LIFECYCLE-GRACEFUL-DEGRADATION] Tests for error response serialization
func TestAppError_ToResponse(t *testing.T) {
	appErr := &AppError{
		Code:        ErrCodeScenarioNotFound,
		Message:     "Scenario not found",
		Details:     "scenario-123",
		Recoverable: false,
		Suggestion:  "Check the scenario ID",
	}

	resp := appErr.ToResponse()

	if resp["success"] != false {
		t.Error("Expected success=false")
	}
	if resp["error_code"] != ErrCodeScenarioNotFound {
		t.Errorf("Expected error_code=%s, got %v", ErrCodeScenarioNotFound, resp["error_code"])
	}
	if resp["message"] != "Scenario not found" {
		t.Errorf("Expected message='Scenario not found', got %v", resp["message"])
	}
	if resp["details"] != "scenario-123" {
		t.Errorf("Expected details='scenario-123', got %v", resp["details"])
	}
	if resp["recoverable"] != false {
		t.Error("Expected recoverable=false")
	}
	if resp["suggestion"] != "Check the scenario ID" {
		t.Errorf("Expected suggestion='Check the scenario ID', got %v", resp["suggestion"])
	}
}

// [REQ:LIFECYCLE-GRACEFUL-DEGRADATION] Tests for error constructors
func TestErrorConstructors(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError("email", "Invalid email format")
		if err.Code != ErrCodeInvalidInput {
			t.Errorf("Expected code %s, got %s", ErrCodeInvalidInput, err.Code)
		}
		if !err.Recoverable {
			t.Error("Validation errors should be recoverable")
		}
	})

	t.Run("NewMissingFieldError", func(t *testing.T) {
		err := NewMissingFieldError("name")
		if err.Code != ErrCodeMissingRequired {
			t.Errorf("Expected code %s, got %s", ErrCodeMissingRequired, err.Code)
		}
		if !containsString(err.Message, "name") {
			t.Error("Error message should contain field name")
		}
	})

	t.Run("NewNotFoundError_scenario", func(t *testing.T) {
		err := NewNotFoundError("scenario", "test-scenario")
		if err.Code != ErrCodeScenarioNotFound {
			t.Errorf("Expected code %s, got %s", ErrCodeScenarioNotFound, err.Code)
		}
		if !containsString(err.Message, "test-scenario") {
			t.Error("Error message should contain resource identifier")
		}
	})

	t.Run("NewNotFoundError_template", func(t *testing.T) {
		err := NewNotFoundError("template", "my-template")
		if err.Code != ErrCodeTemplateNotFound {
			t.Errorf("Expected code %s, got %s", ErrCodeTemplateNotFound, err.Code)
		}
	})

	t.Run("NewConflictError", func(t *testing.T) {
		err := NewConflictError("scenario-123", "Already exists")
		if err.Code != ErrCodeConflict {
			t.Errorf("Expected code %s, got %s", ErrCodeConflict, err.Code)
		}
		if !err.Recoverable {
			t.Error("Conflict errors should be recoverable")
		}
	})

	t.Run("NewDatabaseError", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := NewDatabaseError("insert", cause)
		if err.Code != ErrCodeDatabaseError {
			t.Errorf("Expected code %s, got %s", ErrCodeDatabaseError, err.Code)
		}
		if err.Cause != cause {
			t.Error("Expected cause to be preserved")
		}
	})

	t.Run("NewLifecycleError", func(t *testing.T) {
		cause := errors.New("process failed")
		err := NewLifecycleError("start", "my-scenario", cause)
		if err.Code != ErrCodeLifecycleError {
			t.Errorf("Expected code %s, got %s", ErrCodeLifecycleError, err.Code)
		}
		if !containsString(err.Message, "start") {
			t.Error("Error message should contain operation")
		}
		if !containsString(err.Details, "my-scenario") {
			t.Error("Details should contain scenario ID")
		}
	})

	t.Run("NewGenerationError", func(t *testing.T) {
		cause := errors.New("template parsing failed")
		err := NewGenerationError("saas-template", "my-landing", cause)
		if err.Code != ErrCodeGenerationError {
			t.Errorf("Expected code %s, got %s", ErrCodeGenerationError, err.Code)
		}
		if !containsString(err.Details, "saas-template") {
			t.Error("Details should contain template ID")
		}
		if !containsString(err.Details, "my-landing") {
			t.Error("Details should contain slug")
		}
	})

	t.Run("NewIssueTrackerError", func(t *testing.T) {
		cause := errors.New("network error")
		err := NewIssueTrackerError("create issue", cause)
		if err.Code != ErrCodeIssueTrackerError {
			t.Errorf("Expected code %s, got %s", ErrCodeIssueTrackerError, err.Code)
		}
		if !err.Recoverable {
			t.Error("Issue tracker errors should be recoverable")
		}
	})

	t.Run("NewFileSystemError", func(t *testing.T) {
		cause := errors.New("permission denied")
		err := NewFileSystemError("write", "/path/to/file", cause)
		if err.Code != ErrCodeFileSystemError {
			t.Errorf("Expected code %s, got %s", ErrCodeFileSystemError, err.Code)
		}
		if err.Recoverable {
			t.Error("File system errors should not be recoverable by default")
		}
	})

	t.Run("NewInternalError", func(t *testing.T) {
		cause := errors.New("unexpected nil")
		err := NewInternalError("Something went wrong", cause)
		if err.Code != ErrCodeInternal {
			t.Errorf("Expected code %s, got %s", ErrCodeInternal, err.Code)
		}
		if err.Recoverable {
			t.Error("Internal errors should not be recoverable")
		}
	})
}

// [REQ:LIFECYCLE-GRACEFUL-DEGRADATION] Tests for error type checks
func TestErrorTypeChecks(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		scenarioErr := NewNotFoundError("scenario", "test")
		templateErr := NewNotFoundError("template", "test")
		validationErr := NewValidationError("field", "invalid")

		if !IsNotFound(scenarioErr) {
			t.Error("Expected scenario not found to be IsNotFound")
		}
		if !IsNotFound(templateErr) {
			t.Error("Expected template not found to be IsNotFound")
		}
		if IsNotFound(validationErr) {
			t.Error("Expected validation error to not be IsNotFound")
		}
		if IsNotFound(errors.New("regular error")) {
			t.Error("Expected regular error to not be IsNotFound")
		}
	})

	t.Run("IsValidation", func(t *testing.T) {
		validationErr := NewValidationError("field", "invalid")
		missingErr := NewMissingFieldError("name")
		notFoundErr := NewNotFoundError("scenario", "test")

		if !IsValidation(validationErr) {
			t.Error("Expected validation error to be IsValidation")
		}
		if !IsValidation(missingErr) {
			t.Error("Expected missing field error to be IsValidation")
		}
		if IsValidation(notFoundErr) {
			t.Error("Expected not found error to not be IsValidation")
		}
	})

	t.Run("IsExternalDependency", func(t *testing.T) {
		dbErr := NewDatabaseError("query", nil)
		issueErr := NewIssueTrackerError("create", nil)
		svcErr := NewExternalServiceError("api", nil)
		cmdErr := NewCommandError("vrooli", "output", nil)
		validationErr := NewValidationError("field", "invalid")

		if !IsExternalDependency(dbErr) {
			t.Error("Expected database error to be external dependency")
		}
		if !IsExternalDependency(issueErr) {
			t.Error("Expected issue tracker error to be external dependency")
		}
		if !IsExternalDependency(svcErr) {
			t.Error("Expected external service error to be external dependency")
		}
		if !IsExternalDependency(cmdErr) {
			t.Error("Expected command error to be external dependency")
		}
		if IsExternalDependency(validationErr) {
			t.Error("Expected validation error to not be external dependency")
		}
	})
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
