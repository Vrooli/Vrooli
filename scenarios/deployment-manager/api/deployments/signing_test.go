package deployments

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"deployment-manager/codesigning"

	"github.com/gorilla/mux"
)

// mockSigningValidator is a test mock for SigningValidator
type mockSigningValidator struct {
	config       *codesigning.SigningConfig
	configErr    error
	validateRes  *codesigning.ValidationResult
	prereqRes    *codesigning.ValidationResult
}

func (m *mockSigningValidator) GetSigningConfig(ctx context.Context, profileID string) (*codesigning.SigningConfig, error) {
	return m.config, m.configErr
}

func (m *mockSigningValidator) ValidateConfig(config *codesigning.SigningConfig) *codesigning.ValidationResult {
	if m.validateRes != nil {
		return m.validateRes
	}
	return &codesigning.ValidationResult{Valid: true}
}

func (m *mockSigningValidator) CheckPrerequisites(ctx context.Context, config *codesigning.SigningConfig) *codesigning.ValidationResult {
	if m.prereqRes != nil {
		return m.prereqRes
	}
	return &codesigning.ValidationResult{Valid: true}
}

func TestDeploy_WithoutSigningValidator(t *testing.T) {
	h := NewHandler(nil)

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestDeploy_SigningDisabled(t *testing.T) {
	mock := &mockSigningValidator{
		config: &codesigning.SigningConfig{Enabled: false},
	}
	h := NewHandlerWithSigning(nil, mock)

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestDeploy_SigningEnabled_Valid(t *testing.T) {
	mock := &mockSigningValidator{
		config: &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{
				CertificateSource: "store",
				CertificateThumbprint: "ABC123",
			},
		},
		validateRes: &codesigning.ValidationResult{Valid: true},
		prereqRes:   &codesigning.ValidationResult{Valid: true},
	}
	h := NewHandlerWithSigning(nil, mock)

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestDeploy_SigningValidationFails(t *testing.T) {
	mock := &mockSigningValidator{
		config: &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{},
		},
		validateRes: &codesigning.ValidationResult{
			Valid: false,
			Errors: []codesigning.ValidationError{
				{
					Platform: "windows",
					Message:  "certificate_source is required",
				},
			},
		},
	}
	h := NewHandlerWithSigning(nil, mock)

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	// Should return 424 Failed Dependency for signing errors
	if rec.Code != 424 {
		t.Errorf("expected status 424, got %d", rec.Code)
	}
}

func TestDeploy_SigningPrerequisitesFail(t *testing.T) {
	mock := &mockSigningValidator{
		config: &codesigning.SigningConfig{
			Enabled: true,
			Windows: &codesigning.WindowsSigningConfig{
				CertificateSource: "store",
				CertificateThumbprint: "ABC123",
			},
		},
		validateRes: &codesigning.ValidationResult{Valid: true},
		prereqRes: &codesigning.ValidationResult{
			Valid: false,
			Errors: []codesigning.ValidationError{
				{
					Platform:    "windows",
					Message:     "signtool not found",
					Remediation: "Install Windows SDK",
				},
			},
		},
	}
	h := NewHandlerWithSigning(nil, mock)

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	// Should return 424 Failed Dependency for signing errors
	if rec.Code != 424 {
		t.Errorf("expected status 424, got %d", rec.Code)
	}
}

func TestDeploy_NoSigningConfig(t *testing.T) {
	mock := &mockSigningValidator{
		config:    nil,
		configErr: errors.New("not found"),
	}
	h := NewHandlerWithSigning(nil, mock)

	req := httptest.NewRequest("POST", "/api/v1/deploy/profile-test", nil)
	req = mux.SetURLVars(req, map[string]string{"profile_id": "profile-test"})
	rec := httptest.NewRecorder()

	h.Deploy(rec, req)

	// Should succeed - no signing config means no signing required
	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestBuildSigningRemediation(t *testing.T) {
	tests := []struct {
		name     string
		result   *codesigning.ValidationResult
		expected string
	}{
		{
			name: "with remediations",
			result: &codesigning.ValidationResult{
				Errors: []codesigning.ValidationError{
					{Remediation: "Install signtool"},
					{Remediation: "Configure certificate"},
				},
			},
			expected: "Install signtool; Configure certificate",
		},
		{
			name: "empty remediations",
			result: &codesigning.ValidationResult{
				Errors: []codesigning.ValidationError{
					{Message: "error without remediation"},
				},
			},
			expected: "Install required signing tools or configure valid certificates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSigningRemediation(tt.result)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
