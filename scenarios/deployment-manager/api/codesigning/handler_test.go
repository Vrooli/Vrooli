package codesigning

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	configs     map[string]*SigningConfig
	getError    error
	saveError   error
	deleteError error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		configs: make(map[string]*SigningConfig),
	}
}

func (m *mockRepository) AddConfig(profileID string, config *SigningConfig) *mockRepository {
	m.configs[profileID] = config
	return m
}

func (m *mockRepository) Get(ctx context.Context, profileID string) (*SigningConfig, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.configs[profileID], nil
}

func (m *mockRepository) Save(ctx context.Context, profileID string, config *SigningConfig) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.configs[profileID] = config
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, profileID string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	delete(m.configs, profileID)
	return nil
}

func (m *mockRepository) GetForPlatform(ctx context.Context, profileID string, platform string) (interface{}, error) {
	config, err := m.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	switch platform {
	case PlatformWindows:
		return config.Windows, nil
	case PlatformMacOS:
		return config.MacOS, nil
	case PlatformLinux:
		return config.Linux, nil
	default:
		return nil, nil
	}
}

func (m *mockRepository) SaveForPlatform(ctx context.Context, profileID string, platform string, platformConfig interface{}) error {
	if m.saveError != nil {
		return m.saveError
	}
	existing := m.configs[profileID]
	if existing == nil {
		existing = &SigningConfig{Enabled: true}
		m.configs[profileID] = existing
	}
	switch platform {
	case PlatformWindows:
		if c, ok := platformConfig.(*WindowsSigningConfig); ok {
			existing.Windows = c
		}
	case PlatformMacOS:
		if c, ok := platformConfig.(*MacOSSigningConfig); ok {
			existing.MacOS = c
		}
	case PlatformLinux:
		if c, ok := platformConfig.(*LinuxSigningConfig); ok {
			existing.Linux = c
		}
	}
	return nil
}

// testLogger is a no-op logger for tests.
func testLogger(msg string, fields map[string]interface{}) {}

// mockValidator is a simple mock for validation.
type mockValidator struct {
	result *ValidationResult
}

func (m *mockValidator) ValidateConfig(config *SigningConfig) *ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

func (m *mockValidator) ValidateForPlatform(config *SigningConfig, platform string) *ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

// setupTestHandler creates a handler with mock dependencies.
func setupTestHandler(t *testing.T) (*Handler, *mockRepository) {
	repo := newMockRepository()
	validator := &mockValidator{}
	checker := &mockPrereqChecker{}
	handler := NewHandler(repo, validator, checker, testLogger)
	return handler, repo
}

// makeRequest creates and executes a test HTTP request.
func makeRequest(t *testing.T, handler http.HandlerFunc, method, path string, body interface{}, vars map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(bodyBytes)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	// Set route variables
	if vars != nil {
		req = mux.SetURLVars(req, vars)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestHandler_GetSigning_Success(t *testing.T) {
	handler, repo := setupTestHandler(t)

	// Add a config to the mock repo
	config := &SigningConfig{
		Enabled: true,
		Windows: &WindowsSigningConfig{
			CertificateSource: CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	}
	repo.AddConfig("profile-123", config)

	rr := makeRequest(t, handler.GetSigning, "GET", "/api/v1/profiles/profile-123/signing", nil, map[string]string{"id": "profile-123"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response SigningConfig
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Enabled)
	assert.NotNil(t, response.Windows)
	assert.Equal(t, "./cert.pfx", response.Windows.CertificateFile)
}

func TestHandler_GetSigning_NotFound(t *testing.T) {
	handler, repo := setupTestHandler(t)
	repo.getError = ErrProfileNotFound

	rr := makeRequest(t, handler.GetSigning, "GET", "/api/v1/profiles/nonexistent/signing", nil, map[string]string{"id": "nonexistent"})

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandler_GetSigning_NoConfig(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// Profile exists but has no signing config
	rr := makeRequest(t, handler.GetSigning, "GET", "/api/v1/profiles/profile-123/signing", nil, map[string]string{"id": "profile-123"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response SigningConfig
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Enabled) // Default is disabled
}

func TestHandler_SetSigning_Success(t *testing.T) {
	handler, _ := setupTestHandler(t)

	config := SigningConfig{
		Enabled: true,
		Windows: &WindowsSigningConfig{
			CertificateSource:      CertSourceFile,
			CertificateFile:        "./cert.pfx",
			CertificatePasswordEnv: "WIN_CERT_PASSWORD",
			TimestampServer:        DefaultTimestampServerDigiCert,
			SignAlgorithm:          SignAlgorithmSHA256,
		},
	}

	rr := makeRequest(t, handler.SetSigning, "PUT", "/api/v1/profiles/profile-123/signing", config, map[string]string{"id": "profile-123"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "updated", response["status"])
}

func TestHandler_SetSigning_InvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("PUT", "/api/v1/profiles/profile-123/signing", bytes.NewReader([]byte("not json")))
	req = mux.SetURLVars(req, map[string]string{"id": "profile-123"})
	rr := httptest.NewRecorder()

	handler.SetSigning(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid JSON")
}

func TestHandler_SetSigning_ProfileNotFound(t *testing.T) {
	handler, repo := setupTestHandler(t)
	repo.saveError = ErrProfileNotFound

	config := SigningConfig{Enabled: true}
	rr := makeRequest(t, handler.SetSigning, "PUT", "/api/v1/profiles/nonexistent/signing", config, map[string]string{"id": "nonexistent"})

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandler_SetPlatformSigning_Windows(t *testing.T) {
	handler, _ := setupTestHandler(t)

	winConfig := WindowsSigningConfig{
		CertificateSource: CertSourceFile,
		CertificateFile:   "./cert.pfx",
	}

	rr := makeRequest(t, handler.SetPlatformSigning, "PATCH", "/api/v1/profiles/profile-123/signing/windows", winConfig, map[string]string{
		"id":       "profile-123",
		"platform": "windows",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "updated", response["status"])
	assert.Equal(t, "windows", response["platform"])
}

func TestHandler_SetPlatformSigning_MacOS(t *testing.T) {
	handler, _ := setupTestHandler(t)

	macConfig := MacOSSigningConfig{
		Identity:        "Developer ID Application: Test (TEAMID)",
		TeamID:          "TEAMID1234",
		HardenedRuntime: true,
		Notarize:        true,
	}

	rr := makeRequest(t, handler.SetPlatformSigning, "PATCH", "/api/v1/profiles/profile-123/signing/macos", macConfig, map[string]string{
		"id":       "profile-123",
		"platform": "macos",
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_SetPlatformSigning_InvalidPlatform(t *testing.T) {
	handler, _ := setupTestHandler(t)

	rr := makeRequest(t, handler.SetPlatformSigning, "PATCH", "/api/v1/profiles/profile-123/signing/invalid", nil, map[string]string{
		"id":       "profile-123",
		"platform": "invalid",
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid platform")
}

func TestHandler_DeleteSigning_Success(t *testing.T) {
	handler, repo := setupTestHandler(t)

	repo.AddConfig("profile-123", &SigningConfig{Enabled: true})

	rr := makeRequest(t, handler.DeleteSigning, "DELETE", "/api/v1/profiles/profile-123/signing", nil, map[string]string{"id": "profile-123"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "deleted", response["status"])
}

func TestHandler_DeleteSigning_ProfileNotFound(t *testing.T) {
	handler, repo := setupTestHandler(t)
	repo.deleteError = ErrProfileNotFound

	rr := makeRequest(t, handler.DeleteSigning, "DELETE", "/api/v1/profiles/nonexistent/signing", nil, map[string]string{"id": "nonexistent"})

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandler_ValidateSigning_Disabled(t *testing.T) {
	handler, repo := setupTestHandler(t)

	repo.AddConfig("profile-123", &SigningConfig{Enabled: false})

	rr := makeRequest(t, handler.ValidateSigning, "POST", "/api/v1/profiles/profile-123/signing/validate", nil, map[string]string{"id": "profile-123"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["valid"])
	assert.Contains(t, response["message"], "disabled")
}

func TestHandler_ValidateSigning_NoConfig(t *testing.T) {
	handler, _ := setupTestHandler(t)

	rr := makeRequest(t, handler.ValidateSigning, "POST", "/api/v1/profiles/profile-123/signing/validate", nil, map[string]string{"id": "profile-123"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["valid"])
}

func TestHandler_CheckPrerequisites(t *testing.T) {
	// Create handler with mock checker that returns tools
	repo := newMockRepository()
	validator := &mockValidator{}
	checker := &mockPrereqChecker{
		tools: []ToolDetectionResult{
			{Platform: PlatformLinux, Tool: "gpg", Installed: true, Path: "/usr/bin/gpg", Version: "2.2.27"},
		},
	}
	handler := NewHandler(repo, validator, checker, testLogger)

	rr := makeRequest(t, handler.CheckPrerequisites, "GET", "/api/v1/signing/prerequisites", nil, nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	tools, ok := response["tools"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tools, 1)
}

func TestHandler_MissingProfileID(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// Test each endpoint without profile ID
	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
	}{
		{"GetSigning", handler.GetSigning, "GET"},
		{"SetSigning", handler.SetSigning, "PUT"},
		{"DeleteSigning", handler.DeleteSigning, "DELETE"},
		{"ValidateSigning", handler.ValidateSigning, "POST"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := makeRequest(t, tc.handler, tc.method, "/api/v1/profiles//signing", nil, map[string]string{"id": ""})
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "profile_id is required")
		})
	}
}

// mockPrereqChecker is a simple mock for prerequisite checking.
type mockPrereqChecker struct {
	result *ValidationResult
	tools  []ToolDetectionResult
	err    error
}

func (m *mockPrereqChecker) CheckPrerequisites(ctx context.Context, config *SigningConfig) *ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

func (m *mockPrereqChecker) CheckPlatformPrerequisites(ctx context.Context, config *SigningConfig, platform string) *ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

func (m *mockPrereqChecker) DetectTools(ctx context.Context) ([]ToolDetectionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tools, nil
}
