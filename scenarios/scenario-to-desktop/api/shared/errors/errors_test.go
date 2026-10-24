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

func TestStageErrorConstructorsProvideActionableStructuredRecovery(t *testing.T) {
	cause := errors.New("underlying failure")
	context := map[string]string{"pipeline_id": "pipeline-1"}
	constructors := []struct {
		name string
		err  *DomainError
	}{
		{"scenario unbundleable with alternatives", ErrScenarioUnbundleable("demo", "postgres", "not embeddable", []string{"sqlite"})},
		{"scenario unbundleable without alternatives", ErrScenarioUnbundleable("demo", "postgres", "not embeddable", nil)},
		{"template missing", ErrTemplateNotFound("react")},
		{"pipeline not resumable", ErrPipelineNotResumable("pipeline-1", "terminal")},
		{"pipeline orchestrator missing", ErrPipelineOrchestratorNotConfigured()},
		{"pipeline stage invalid", ErrPipelineInvalidStage("ship")},
		{"pipeline scenario required", ErrPipelineScenarioRequired()},
		{"bundle manifest missing", ErrBundleManifestNotFound("/bundle.json")},
		{"bundle manifest generation", ErrBundleManifestGeneration(cause)},
		{"bundle packaging", ErrBundlePackagingFailed(cause, "/bundle")},
		{"bundle service missing", ErrBundleServiceNotConfigured()},
		{"preflight service missing", ErrPreflightServiceNotConfigured()},
		{"preflight validation with findings", ErrPreflightValidationFailed(cause, []string{"port unavailable"})},
		{"preflight validation without findings", ErrPreflightValidationFailed(cause, nil)},
		{"preflight bundle missing", ErrPreflightBundleNotAvailable()},
		{"preflight timeout", ErrPreflightTimeout("30s")},
		{"analyzer missing", ErrGenerateAnalyzerNotConfigured()},
		{"generation service missing", ErrGenerateServiceNotConfigured()},
		{"scenario analysis", ErrScenarioAnalysisFailed(cause, "demo")},
		{"scenario validation", ErrScenarioValidationFailed(cause, "demo")},
		{"desktop config", ErrDesktopConfigFailed(cause)},
		{"generation timeout", ErrGenerationTimeout("build-1", "30s")},
		{"generation failure", ErrGenerationFailed(cause)},
		{"build service missing", ErrBuildServiceNotConfigured()},
		{"build store missing", ErrBuildStoreNotConfigured()},
		{"build desktop path missing", ErrBuildDesktopPathMissing()},
		{"build start", ErrBuildStartFailed(cause, "linux")},
		{"build timeout", ErrBuildTimedOut("build-1", "30s")},
		{"deploy failure", ErrDeployFailed(cause, "remote")},
		{"deploy timeout", ErrDeployTimeout("deploy-1", "30s")},
		{"smoke artifact missing", ErrSmokeTestArtifactNotFound("/app")},
		{"smoke execution", ErrSmokeTestExecutionFailed(cause, context)},
		{"smoke timeout", ErrSmokeTestTimeout("30s", context)},
		{"smoke validation", ErrSmokeTestValidationFailed(context)},
		{"smoke telemetry", ErrSmokeTestTelemetryFailed(cause, context)},
		{"smoke storage", ErrSmokeTestStoreFailed(cause)},
		{"smoke cancelled", ErrSmokeTestCancelled()},
	}
	for _, platform := range []string{"linux", "mac", "win", "other"} {
		constructors = append(constructors,
			struct {
				name string
				err  *DomainError
			}{"build platform " + platform, ErrBuildPlatformFailed(cause, platform, "last output")},
			struct {
				name string
				err  *DomainError
			}{"smoke platform " + platform, ErrSmokeTestPlatformError(cause, platform)},
		)
	}
	for _, tt := range constructors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil || tt.err.Domain == "" || tt.err.Code == "" || tt.err.Message == "" {
				t.Fatalf("constructor returned incomplete error: %#v", tt.err)
			}
			if tt.err.Recovery != "" && tt.err.RecoveryHint == "" {
				t.Fatalf("constructor omitted recovery guidance: %#v", tt.err)
			}
		})
	}
}

// assertDomainError is a helper that checks code, domain, detail key/value, and cause on a DomainError.
func assertDomainError(t *testing.T, err *DomainError, wantCode ErrorCode, wantDomain string, detailKey string, detailVal interface{}, wantCause error) {
	t.Helper()
	if wantCode != "" && err.Code != wantCode {
		t.Errorf("expected code %q, got %q", wantCode, err.Code)
	}
	if wantDomain != "" && err.Domain != wantDomain {
		t.Errorf("expected domain %q, got %q", wantDomain, err.Domain)
	}
	if detailKey != "" && err.Details[detailKey] != detailVal {
		t.Errorf("expected detail %q=%v, got %v", detailKey, detailVal, err.Details[detailKey])
	}
	if wantCause != nil && err.Cause != wantCause {
		t.Errorf("expected cause to be set")
	}
}

func TestDomainSpecificConstructors(t *testing.T) {
	jsonParseErr := errors.New("json parse error")
	npmErr := errors.New("npm error")
	validationErr := errors.New("validation error")

	tests := []struct {
		name       string
		err        *DomainError
		wantCode   ErrorCode
		wantDomain string
		detailKey  string
		detailVal  interface{}
		wantCause  error
	}{
		{"ErrBundleNotFound", ErrBundleNotFound("/path/to/bundle"), CodeBundleNotFound, "bundle", "bundle_path", "/path/to/bundle", nil},
		{"ErrBundleManifest", ErrBundleManifest(jsonParseErr), CodeBundleManifestError, "", "", nil, jsonParseErr},
		{"ErrBuildNotFound", ErrBuildNotFound("build-123"), "", "", "build_id", "build-123", nil},
		{"ErrBuildFailed", ErrBuildFailed(npmErr, "linux"), "", "", "platform", "linux", nil},
		{"ErrWrapperNotFound", ErrWrapperNotFound("my-scenario"), "", "generation", "", nil, nil},
		{"ErrScenarioNotFound", ErrScenarioNotFound("my-scenario"), CodeScenarioNotFound, "", "", nil, nil},
		{"ErrSessionNotFound", ErrSessionNotFound("session-123"), "", "preflight", "", nil, nil},
		{"ErrSessionExpired", ErrSessionExpired("session-123"), CodeSessionExpired, "", "", nil, nil},
		{"ErrJobNotFound", ErrJobNotFound("job-123"), "", "", "job_id", "job-123", nil},
		{"ErrPreflightFailed", ErrPreflightFailed(validationErr), CodePreflightFailed, "", "", nil, nil},
		{"ErrSmokeTestNotFound", ErrSmokeTestNotFound("test-123"), "", "smoketest", "", nil, nil},
		{"ErrArtifactNotFound", ErrArtifactNotFound("/path/to/artifact"), "", "", "artifact_path", "/path/to/artifact", nil},
		{"ErrPipelineNotFound", ErrPipelineNotFound("pipeline-123"), "", "pipeline", "", nil, nil},
		{"ErrPipelineCancelled", ErrPipelineCancelled("pipeline-123"), CodePipelineCancelled, "", "", nil, nil},
		{"ErrCertificateNotFound", ErrCertificateNotFound("cert-123"), "", "signing", "", nil, nil},
		{"ErrCertificateExpired", ErrCertificateExpired("cert-123", "2024-01-01"), "", "", "expires_at", "2024-01-01", nil},
		{"ErrWineNotInstalled", ErrWineNotInstalled(), CodeWineNotInstalled, "system", "", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDomainError(t, tt.err, tt.wantCode, tt.wantDomain, tt.detailKey, tt.detailVal, tt.wantCause)
		})
	}
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

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		// Domain errors with retry recovery
		{"CodeInternal", New(CodeInternal, "internal error"), true},
		{"CodeTimeout", New(CodeTimeout, "timeout"), true},
		{"CodeUnavailable", New(CodeUnavailable, "unavailable"), true},
		{"CodeBuildFailed", New(CodeBuildFailed, "build failed"), true},

		// Domain errors that shouldn't be retried
		{"CodeNotFound", New(CodeNotFound, "not found"), false},
		{"CodeBadRequest", New(CodeBadRequest, "bad request"), false},
		{"CodeValidation", New(CodeValidation, "validation error"), false},
		{"CodeUnauthorized", New(CodeUnauthorized, "unauthorized"), false},
		{"CodeForbidden", New(CodeForbidden, "forbidden"), false},
		{"CodePipelineCancelled", New(CodePipelineCancelled, "cancelled"), false},

		// Standard errors - transient detection
		{"standard timeout error", errors.New("connection timeout"), true},
		{"standard unavailable error", errors.New("service unavailable"), true},
		{"standard try again error", errors.New("try again later"), true},
		{"standard error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ShouldRetry(tt.err) != tt.expected {
				t.Errorf("expected ShouldRetry to return %v for %v", tt.expected, tt.err)
			}
		})
	}
}

func TestIsUserError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		// 4xx errors (user errors)
		{"CodeNotFound", New(CodeNotFound, "not found"), true},
		{"CodeBadRequest", New(CodeBadRequest, "bad request"), true},
		{"CodeValidation", New(CodeValidation, "validation error"), true},
		{"CodeUnauthorized", New(CodeUnauthorized, "unauthorized"), true},
		{"CodeForbidden", New(CodeForbidden, "forbidden"), true},
		{"CodeConflict", New(CodeConflict, "conflict"), true},

		// 5xx errors (server errors)
		{"CodeInternal", New(CodeInternal, "internal error"), false},
		{"CodeUnavailable", New(CodeUnavailable, "unavailable"), false},
		{"CodeTimeout", New(CodeTimeout, "timeout"), false},

		// Standard errors
		{"standard error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsUserError(tt.err) != tt.expected {
				t.Errorf("expected IsUserError to return %v", tt.expected)
			}
		})
	}
}

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		// Transient domain errors
		{"CodeInternal", New(CodeInternal, "internal error"), true},
		{"CodeTimeout", New(CodeTimeout, "timeout"), true},
		{"CodeUnavailable", New(CodeUnavailable, "unavailable"), true},
		{"CodePreflightTimeout", New(CodePreflightTimeout, "preflight timeout"), true},
		{"CodeBundleServiceTimeout", New(CodeBundleServiceTimeout, "bundle timeout"), true},
		{"CodeServiceHealthError", New(CodeServiceHealthError, "health error"), true},
		{"CodeSystemResourceError", New(CodeSystemResourceError, "resource error"), true},
		{"CodeProcessKillTimeout", New(CodeProcessKillTimeout, "kill timeout"), true},
		{"CodeKeychainError", New(CodeKeychainError, "keychain error"), true},

		// Non-transient domain errors
		{"CodeNotFound", New(CodeNotFound, "not found"), false},
		{"CodeBadRequest", New(CodeBadRequest, "bad request"), false},
		{"CodeValidation", New(CodeValidation, "validation error"), false},
		{"CodeBuildFailed", New(CodeBuildFailed, "build failed"), false},

		// Standard errors - transient detection
		{"standard timeout error", errors.New("connection timeout"), true},
		{"standard unavailable error", errors.New("service unavailable"), true},
		{"standard connection refused", errors.New("connection refused"), true},
		{"standard connection reset", errors.New("connection reset by peer"), true},
		{"standard error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsTransient(tt.err) != tt.expected {
				t.Errorf("expected IsTransient to return %v for %v", tt.expected, tt.err)
			}
		})
	}
}
