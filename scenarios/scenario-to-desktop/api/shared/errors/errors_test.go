package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound, "resource not found")
	if err.Code != CodeNotFound {
		t.Errorf("expected code %q, got %q", CodeNotFound, err.Code)
	}
	if err.Message != "resource not found" {
		t.Errorf("expected message %q, got %q", "resource not found", err.Message)
	}
}

func TestNewf(t *testing.T) {
	err := Newf(CodeNotFound, "user %s not found", "123")
	if err.Message != "user 123 not found" {
		t.Errorf("expected message %q, got %q", "user 123 not found", err.Message)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(CodeInternal, cause, "operation failed")
	if err.Cause != cause {
		t.Errorf("expected cause to be set")
	}
	if !errors.Is(err, cause) {
		t.Errorf("expected errors.Is to find cause")
	}
}

func TestWrapf(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrapf(CodeInternal, cause, "operation %s failed", "test")
	if err.Message != "operation test failed" {
		t.Errorf("expected formatted message, got %q", err.Message)
	}
}

func TestDomainError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := New(CodeNotFound, "not found")
		expected := "[NOT_FOUND] not found"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("db error")
		err := Wrap(CodeInternal, cause, "query failed")
		if err.Error() != "[INTERNAL_ERROR] query failed: db error" {
			t.Errorf("unexpected error string: %q", err.Error())
		}
	})
}

func TestDomainError_Unwrap(t *testing.T) {
	cause := errors.New("original")
	err := Wrap(CodeInternal, cause, "wrapped")

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("expected unwrapped to be original cause")
	}
}

func TestDomainError_WithCause(t *testing.T) {
	original := New(CodeNotFound, "not found")
	cause := errors.New("db error")
	withCause := original.WithCause(cause)

	if withCause.Cause != cause {
		t.Errorf("expected cause to be set")
	}
	// Original should be unchanged
	if original.Cause != nil {
		t.Errorf("expected original to be unchanged")
	}
}

func TestDomainError_WithDetail(t *testing.T) {
	err := New(CodeNotFound, "user not found")
	withDetail := err.WithDetail("user_id", "123")

	if withDetail.Details["user_id"] != "123" {
		t.Errorf("expected detail to be set")
	}
	// Original should be unchanged
	if err.Details != nil {
		t.Errorf("expected original to have no details")
	}
}

func TestDomainError_WithDetails(t *testing.T) {
	err := New(CodeValidation, "validation failed").
		WithDetail("field1", "error1")
	withMore := err.WithDetails(map[string]interface{}{
		"field2": "error2",
		"field3": "error3",
	})

	if withMore.Details["field1"] != "error1" {
		t.Errorf("expected field1 to be preserved")
	}
	if withMore.Details["field2"] != "error2" {
		t.Errorf("expected field2 to be added")
	}
	if withMore.Details["field3"] != "error3" {
		t.Errorf("expected field3 to be added")
	}
}

func TestDomainError_WithMessage(t *testing.T) {
	err := New(CodeNotFound, "original message")
	withMsg := err.WithMessage("new message")

	if withMsg.Message != "new message" {
		t.Errorf("expected new message, got %q", withMsg.Message)
	}
	if err.Message != "original message" {
		t.Errorf("expected original to be unchanged")
	}
}

func TestDomainError_WithMessagef(t *testing.T) {
	err := New(CodeNotFound, "original")
	withMsg := err.WithMessagef("user %d not found", 42)

	if withMsg.Message != "user 42 not found" {
		t.Errorf("expected formatted message, got %q", withMsg.Message)
	}
}

func TestDomainError_InDomain(t *testing.T) {
	err := New(CodeNotFound, "not found")
	inDomain := err.InDomain("users")

	if inDomain.Domain != "users" {
		t.Errorf("expected domain %q, got %q", "users", inDomain.Domain)
	}
	if err.Domain != "" {
		t.Errorf("expected original domain to be empty")
	}
}

func TestDomainError_HTTPStatus(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected int
	}{
		{CodeNotFound, http.StatusNotFound},
		{CodeBadRequest, http.StatusBadRequest},
		{CodeInternal, http.StatusInternalServerError},
		{CodeUnauthorized, http.StatusUnauthorized},
		{CodeForbidden, http.StatusForbidden},
		{CodeConflict, http.StatusConflict},
		{CodeTimeout, http.StatusGatewayTimeout},
		{CodeValidation, http.StatusUnprocessableEntity},
		{CodeUnavailable, http.StatusServiceUnavailable},
		{CodeNotImplemented, http.StatusNotImplemented},
		{CodeBuildNotFound, http.StatusNotFound},
		{CodeBuildInProgress, http.StatusConflict},
		{CodeBuildFailed, http.StatusInternalServerError},
		{CodePipelineNotFound, http.StatusNotFound},
		{CodePipelineCancelled, http.StatusConflict},
		{CodeSessionExpired, http.StatusGone},
		{CodeDependencyError, http.StatusFailedDependency},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test")
			if err.HTTPStatus() != tt.expected {
				t.Errorf("expected status %d, got %d", tt.expected, err.HTTPStatus())
			}
		})
	}

	t.Run("unknown code defaults to 500", func(t *testing.T) {
		err := New("UNKNOWN_CODE", "test")
		if err.HTTPStatus() != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", err.HTTPStatus())
		}
	})
}

func TestIsDomainError(t *testing.T) {
	t.Run("with domain error", func(t *testing.T) {
		err := New(CodeNotFound, "not found")
		de, ok := IsDomainError(err)
		if !ok {
			t.Errorf("expected IsDomainError to return true")
		}
		if de.Code != CodeNotFound {
			t.Errorf("expected code to match")
		}
	})

	t.Run("with wrapped domain error", func(t *testing.T) {
		inner := New(CodeNotFound, "not found")
		wrapped := Wrap(CodeInternal, inner, "wrapped")
		de, ok := IsDomainError(wrapped)
		if !ok {
			t.Errorf("expected IsDomainError to return true")
		}
		if de.Code != CodeInternal {
			t.Errorf("expected outer error code")
		}
	})

	t.Run("with standard error", func(t *testing.T) {
		err := errors.New("standard error")
		de, ok := IsDomainError(err)
		if ok {
			t.Errorf("expected IsDomainError to return false")
		}
		if de != nil {
			t.Errorf("expected nil domain error")
		}
	})
}

func TestGetHTTPStatus(t *testing.T) {
	t.Run("domain error", func(t *testing.T) {
		err := New(CodeNotFound, "not found")
		if GetHTTPStatus(err) != http.StatusNotFound {
			t.Errorf("expected 404")
		}
	})

	t.Run("standard error", func(t *testing.T) {
		err := errors.New("standard error")
		if GetHTTPStatus(err) != http.StatusInternalServerError {
			t.Errorf("expected 500")
		}
	})
}

func TestConvenienceConstructors(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		err := ErrNotFound("user")
		if err.Code != CodeNotFound {
			t.Errorf("expected NOT_FOUND code")
		}
		if err.Message != "user not found" {
			t.Errorf("expected 'user not found', got %q", err.Message)
		}
	})

	t.Run("ErrBadRequest", func(t *testing.T) {
		err := ErrBadRequest("invalid input")
		if err.Code != CodeBadRequest {
			t.Errorf("expected BAD_REQUEST code")
		}
	})

	t.Run("ErrValidation", func(t *testing.T) {
		err := ErrValidation("invalid fields", map[string]interface{}{
			"email": "invalid format",
		})
		if err.Code != CodeValidation {
			t.Errorf("expected VALIDATION_ERROR code")
		}
		if err.Details["email"] != "invalid format" {
			t.Errorf("expected details to be set")
		}
	})

	t.Run("ErrInternal", func(t *testing.T) {
		err := ErrInternal("something went wrong")
		if err.Code != CodeInternal {
			t.Errorf("expected INTERNAL_ERROR code")
		}
	})

	t.Run("ErrInternalf", func(t *testing.T) {
		err := ErrInternalf("error in %s", "module")
		if err.Message != "error in module" {
			t.Errorf("expected formatted message")
		}
	})

	t.Run("ErrTimeout", func(t *testing.T) {
		err := ErrTimeout("database query")
		if err.Code != CodeTimeout {
			t.Errorf("expected TIMEOUT code")
		}
		if err.Message != "database query timed out" {
			t.Errorf("expected timeout message")
		}
	})

	t.Run("ErrUnavailable", func(t *testing.T) {
		err := ErrUnavailable("auth service")
		if err.Code != CodeUnavailable {
			t.Errorf("expected SERVICE_UNAVAILABLE code")
		}
	})
}

func TestDomainSpecificConstructors(t *testing.T) {
	t.Run("ErrBundleNotFound", func(t *testing.T) {
		err := ErrBundleNotFound("/path/to/bundle")
		if err.Code != CodeBundleNotFound {
			t.Errorf("expected BUNDLE_NOT_FOUND code")
		}
		if err.Domain != "bundle" {
			t.Errorf("expected bundle domain")
		}
		if err.Details["bundle_path"] != "/path/to/bundle" {
			t.Errorf("expected bundle_path detail")
		}
	})

	t.Run("ErrBundleManifest", func(t *testing.T) {
		cause := errors.New("json parse error")
		err := ErrBundleManifest(cause)
		if err.Code != CodeBundleManifestError {
			t.Errorf("expected BUNDLE_MANIFEST_ERROR code")
		}
		if err.Cause != cause {
			t.Errorf("expected cause to be set")
		}
	})

	t.Run("ErrBuildNotFound", func(t *testing.T) {
		err := ErrBuildNotFound("build-123")
		if err.Details["build_id"] != "build-123" {
			t.Errorf("expected build_id detail")
		}
	})

	t.Run("ErrBuildFailed", func(t *testing.T) {
		cause := errors.New("npm error")
		err := ErrBuildFailed(cause, "linux")
		if err.Details["platform"] != "linux" {
			t.Errorf("expected platform detail")
		}
	})

	t.Run("ErrWrapperNotFound", func(t *testing.T) {
		err := ErrWrapperNotFound("my-scenario")
		if err.Domain != "generation" {
			t.Errorf("expected generation domain")
		}
	})

	t.Run("ErrScenarioNotFound", func(t *testing.T) {
		err := ErrScenarioNotFound("my-scenario")
		if err.Code != CodeScenarioNotFound {
			t.Errorf("expected SCENARIO_NOT_FOUND code")
		}
	})

	t.Run("ErrSessionNotFound", func(t *testing.T) {
		err := ErrSessionNotFound("session-123")
		if err.Domain != "preflight" {
			t.Errorf("expected preflight domain")
		}
	})

	t.Run("ErrSessionExpired", func(t *testing.T) {
		err := ErrSessionExpired("session-123")
		if err.Code != CodeSessionExpired {
			t.Errorf("expected SESSION_EXPIRED code")
		}
	})

	t.Run("ErrJobNotFound", func(t *testing.T) {
		err := ErrJobNotFound("job-123")
		if err.Details["job_id"] != "job-123" {
			t.Errorf("expected job_id detail")
		}
	})

	t.Run("ErrPreflightFailed", func(t *testing.T) {
		cause := errors.New("validation error")
		err := ErrPreflightFailed(cause)
		if err.Code != CodePreflightFailed {
			t.Errorf("expected PREFLIGHT_FAILED code")
		}
	})

	t.Run("ErrSmokeTestNotFound", func(t *testing.T) {
		err := ErrSmokeTestNotFound("test-123")
		if err.Domain != "smoketest" {
			t.Errorf("expected smoketest domain")
		}
	})

	t.Run("ErrArtifactNotFound", func(t *testing.T) {
		err := ErrArtifactNotFound("/path/to/artifact")
		if err.Details["artifact_path"] != "/path/to/artifact" {
			t.Errorf("expected artifact_path detail")
		}
	})

	t.Run("ErrPipelineNotFound", func(t *testing.T) {
		err := ErrPipelineNotFound("pipeline-123")
		if err.Domain != "pipeline" {
			t.Errorf("expected pipeline domain")
		}
	})

	t.Run("ErrPipelineCancelled", func(t *testing.T) {
		err := ErrPipelineCancelled("pipeline-123")
		if err.Code != CodePipelineCancelled {
			t.Errorf("expected PIPELINE_CANCELLED code")
		}
	})

	t.Run("ErrCertificateNotFound", func(t *testing.T) {
		err := ErrCertificateNotFound("cert-123")
		if err.Domain != "signing" {
			t.Errorf("expected signing domain")
		}
	})

	t.Run("ErrCertificateExpired", func(t *testing.T) {
		err := ErrCertificateExpired("cert-123", "2024-01-01")
		if err.Details["expires_at"] != "2024-01-01" {
			t.Errorf("expected expires_at detail")
		}
	})

	t.Run("ErrWineNotInstalled", func(t *testing.T) {
		err := ErrWineNotInstalled()
		if err.Code != CodeWineNotInstalled {
			t.Errorf("expected WINE_NOT_INSTALLED code")
		}
		if err.Domain != "system" {
			t.Errorf("expected system domain")
		}
	})
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"CodeNotFound", New(CodeNotFound, "not found"), true},
		{"CodeBundleNotFound", New(CodeBundleNotFound, "bundle not found"), true},
		{"CodeBuildNotFound", New(CodeBuildNotFound, "build not found"), true},
		{"CodeWrapperNotFound", New(CodeWrapperNotFound, "wrapper not found"), true},
		{"CodeTemplateNotFound", New(CodeTemplateNotFound, "template not found"), true},
		{"CodeScenarioNotFound", New(CodeScenarioNotFound, "scenario not found"), true},
		{"CodeSessionNotFound", New(CodeSessionNotFound, "session not found"), true},
		{"CodeJobNotFound", New(CodeJobNotFound, "job not found"), true},
		{"CodeSmokeTestNotFound", New(CodeSmokeTestNotFound, "smoke test not found"), true},
		{"CodeArtifactNotFound", New(CodeArtifactNotFound, "artifact not found"), true},
		{"CodePipelineNotFound", New(CodePipelineNotFound, "pipeline not found"), true},
		{"CodeCertificateNotFound", New(CodeCertificateNotFound, "certificate not found"), true},
		{"CodeInternal", New(CodeInternal, "internal error"), false},
		{"standard error", errors.New("not a domain error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsNotFound(tt.err) != tt.expected {
				t.Errorf("expected IsNotFound to return %v", tt.expected)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"CodeTimeout", New(CodeTimeout, "timeout"), true},
		{"CodePreflightTimeout", New(CodePreflightTimeout, "preflight timeout"), true},
		{"CodeBundleServiceTimeout", New(CodeBundleServiceTimeout, "bundle timeout"), true},
		{"CodeProcessKillTimeout", New(CodeProcessKillTimeout, "process kill timeout"), true},
		{"CodeInternal", New(CodeInternal, "internal error"), false},
		{"standard error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsTimeout(tt.err) != tt.expected {
				t.Errorf("expected IsTimeout to return %v", tt.expected)
			}
		})
	}
}

func TestIsValidation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"CodeValidation", New(CodeValidation, "validation error"), true},
		{"CodeBadRequest", New(CodeBadRequest, "bad request"), true},
		{"CodeConfigInvalid", New(CodeConfigInvalid, "config invalid"), true},
		{"CodeInternal", New(CodeInternal, "internal error"), false},
		{"standard error", errors.New("validation error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsValidation(tt.err) != tt.expected {
				t.Errorf("expected IsValidation to return %v", tt.expected)
			}
		})
	}
}

func TestIsConflict(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"CodeConflict", New(CodeConflict, "conflict"), true},
		{"CodeBuildInProgress", New(CodeBuildInProgress, "build in progress"), true},
		{"CodePipelineCancelled", New(CodePipelineCancelled, "pipeline cancelled"), true},
		{"CodeInternal", New(CodeInternal, "internal error"), false},
		{"standard error", errors.New("conflict"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsConflict(tt.err) != tt.expected {
				t.Errorf("expected IsConflict to return %v", tt.expected)
			}
		})
	}
}

