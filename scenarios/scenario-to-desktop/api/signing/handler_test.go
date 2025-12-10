package signing

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

	"scenario-to-desktop-api/signing/mocks"
	"scenario-to-desktop-api/signing/types"
)

// mockValidator implements signing.Validator for testing.
type mockValidator struct {
	result *types.ValidationResult
}

func (m *mockValidator) ValidateConfig(config *types.SigningConfig) *types.ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

func (m *mockValidator) ValidateForPlatform(config *types.SigningConfig, platform string) *types.ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

// mockPrereqChecker implements PrerequisiteChecker for testing.
type mockPrereqChecker struct {
	result *types.ValidationResult
	tools  []types.ToolDetectionResult
	err    error
}

func (m *mockPrereqChecker) CheckPrerequisites(ctx context.Context, config *types.SigningConfig) *types.ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

func (m *mockPrereqChecker) CheckPlatformPrerequisites(ctx context.Context, config *types.SigningConfig, platform string) *types.ValidationResult {
	if m.result != nil {
		return m.result
	}
	return NewValidationResult()
}

func (m *mockPrereqChecker) DetectTools(ctx context.Context) ([]types.ToolDetectionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tools, nil
}

// mockDetector implements CertificateDiscoverer for testing.
type mockDetector struct {
	certs []types.DiscoveredCertificate
	err   error
}

func (m *mockDetector) DiscoverCertificates(ctx context.Context, platform string) ([]types.DiscoveredCertificate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.certs, nil
}

// testHandler creates a handler with mock dependencies for testing.
type testHandler struct {
	handler       *Handler
	repo          *mocks.MockRepository
	validator     *mockValidator
	prereqChecker *mockPrereqChecker
	detector      *mockDetector
}

func newTestHandler() *testHandler {
	repo := mocks.NewMockRepository()
	validator := &mockValidator{}
	prereqChecker := &mockPrereqChecker{}
	detector := &mockDetector{}

	h := &Handler{
		repo:          repo,
		validator:     validator,
		prereqChecker: prereqChecker,
		detector:      detector,
		generator:     nil, // Not needed for most tests
	}

	return &testHandler{
		handler:       h,
		repo:          repo,
		validator:     validator,
		prereqChecker: prereqChecker,
		detector:      detector,
	}
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

func TestHandler_GetConfig_Success(t *testing.T) {
	th := newTestHandler()

	// Add a config to the mock repo
	config := &types.SigningConfig{
		SchemaVersion: "1.0",
		Enabled:       true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	}
	th.repo.AddConfig("test-scenario", config)

	rr := makeRequest(t, th.handler.GetConfig, "GET", "/api/v1/signing/test-scenario", nil, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.SigningConfigResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-scenario", response.Scenario)
	assert.NotNil(t, response.Config)
	assert.True(t, response.Config.Enabled)
	assert.NotNil(t, response.Config.Windows)
	assert.Equal(t, "./cert.pfx", response.Config.Windows.CertificateFile)
}

func TestHandler_GetConfig_NoConfig(t *testing.T) {
	th := newTestHandler()

	rr := makeRequest(t, th.handler.GetConfig, "GET", "/api/v1/signing/nonexistent", nil, map[string]string{"scenario": "nonexistent"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.SigningConfigResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "nonexistent", response.Scenario)
	assert.Nil(t, response.Config)
}

func TestHandler_PutConfig_Success(t *testing.T) {
	th := newTestHandler()

	config := types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource:      types.CertSourceFile,
			CertificateFile:        "./cert.pfx",
			CertificatePasswordEnv: "WIN_CERT_PASSWORD",
			TimestampServer:        types.DefaultTimestampServerDigiCert,
			SignAlgorithm:          types.SignAlgorithmSHA256,
		},
	}

	rr := makeRequest(t, th.handler.PutConfig, "PUT", "/api/v1/signing/test-scenario", config, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.SigningConfigResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-scenario", response.Scenario)
	assert.NotNil(t, response.Config)

	// Verify it was saved
	savedConfig, err := th.repo.Get(context.Background(), "test-scenario")
	require.NoError(t, err)
	assert.NotNil(t, savedConfig)
	assert.True(t, savedConfig.Enabled)
}

func TestHandler_PutConfig_InvalidJSON(t *testing.T) {
	th := newTestHandler()

	req := httptest.NewRequest("PUT", "/api/v1/signing/test-scenario", bytes.NewReader([]byte("not json")))
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rr := httptest.NewRecorder()

	th.handler.PutConfig(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid JSON")
}

func TestHandler_PutConfig_ValidationFails(t *testing.T) {
	th := newTestHandler()

	// Configure validator to fail
	th.validator.result = &types.ValidationResult{
		Valid: false,
		Errors: []types.ValidationError{
			{
				Code:    "TEST_ERROR",
				Message: "Test validation error",
			},
		},
	}

	config := types.SigningConfig{
		Enabled: true,
	}

	rr := makeRequest(t, th.handler.PutConfig, "PUT", "/api/v1/signing/test-scenario", config, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "validation failed")
}

func TestHandler_PatchPlatformConfig_Windows(t *testing.T) {
	th := newTestHandler()

	winConfig := types.WindowsSigningConfig{
		CertificateSource: types.CertSourceFile,
		CertificateFile:   "./cert.pfx",
	}

	rr := makeRequest(t, th.handler.PatchPlatformConfig, "PATCH", "/api/v1/signing/test-scenario/windows", winConfig, map[string]string{
		"scenario": "test-scenario",
		"platform": "windows",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify Windows config was saved
	savedConfig, _ := th.repo.Get(context.Background(), "test-scenario")
	assert.NotNil(t, savedConfig)
	assert.NotNil(t, savedConfig.Windows)
	assert.Equal(t, "./cert.pfx", savedConfig.Windows.CertificateFile)
}

func TestHandler_PatchPlatformConfig_MacOS(t *testing.T) {
	th := newTestHandler()

	macConfig := types.MacOSSigningConfig{
		Identity:        "Developer ID Application: Test (TEAMID)",
		TeamID:          "TEAMID1234",
		HardenedRuntime: true,
		Notarize:        true,
	}

	rr := makeRequest(t, th.handler.PatchPlatformConfig, "PATCH", "/api/v1/signing/test-scenario/macos", macConfig, map[string]string{
		"scenario": "test-scenario",
		"platform": "macos",
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_PatchPlatformConfig_Linux(t *testing.T) {
	th := newTestHandler()

	linuxConfig := types.LinuxSigningConfig{
		GPGKeyID: "ABC123DEF456",
	}

	rr := makeRequest(t, th.handler.PatchPlatformConfig, "PATCH", "/api/v1/signing/test-scenario/linux", linuxConfig, map[string]string{
		"scenario": "test-scenario",
		"platform": "linux",
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandler_PatchPlatformConfig_InvalidPlatform(t *testing.T) {
	th := newTestHandler()

	rr := makeRequest(t, th.handler.PatchPlatformConfig, "PATCH", "/api/v1/signing/test-scenario/invalid", nil, map[string]string{
		"scenario": "test-scenario",
		"platform": "invalid",
	})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid platform")
}

func TestHandler_DeleteConfig_Success(t *testing.T) {
	th := newTestHandler()

	// Add a config first
	th.repo.AddConfig("test-scenario", &types.SigningConfig{Enabled: true})

	rr := makeRequest(t, th.handler.DeleteConfig, "DELETE", "/api/v1/signing/test-scenario", nil, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "deleted", response["status"])

	// Verify it was deleted
	exists, _ := th.repo.Exists(context.Background(), "test-scenario")
	assert.False(t, exists)
}

func TestHandler_DeletePlatformConfig_Success(t *testing.T) {
	th := newTestHandler()

	// Add a config with all platforms
	th.repo.AddConfig("test-scenario", &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{CertificateFile: "./cert.pfx"},
		MacOS:   &types.MacOSSigningConfig{Identity: "Test"},
		Linux:   &types.LinuxSigningConfig{GPGKeyID: "ABC123"},
	})

	rr := makeRequest(t, th.handler.DeletePlatformConfig, "DELETE", "/api/v1/signing/test-scenario/windows", nil, map[string]string{
		"scenario": "test-scenario",
		"platform": "windows",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify Windows config was deleted but others remain
	savedConfig, _ := th.repo.Get(context.Background(), "test-scenario")
	assert.Nil(t, savedConfig.Windows)
	assert.NotNil(t, savedConfig.MacOS)
	assert.NotNil(t, savedConfig.Linux)
}

func TestHandler_ValidateConfig_Success(t *testing.T) {
	th := newTestHandler()

	th.repo.AddConfig("test-scenario", &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	})

	rr := makeRequest(t, th.handler.ValidateConfig, "POST", "/api/v1/signing/test-scenario/validate", nil, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.ValidationResult
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Valid)
}

func TestHandler_ValidateConfig_NoConfig(t *testing.T) {
	th := newTestHandler()

	rr := makeRequest(t, th.handler.ValidateConfig, "POST", "/api/v1/signing/nonexistent/validate", nil, map[string]string{"scenario": "nonexistent"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["valid"])
}

func TestHandler_CheckReady_NotEnabled(t *testing.T) {
	th := newTestHandler()

	th.repo.AddConfig("test-scenario", &types.SigningConfig{Enabled: false})

	rr := makeRequest(t, th.handler.CheckReady, "GET", "/api/v1/signing/test-scenario/ready", nil, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.ReadinessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Ready)
	assert.Contains(t, response.Issues, "Signing is not enabled for this scenario")
}

func TestHandler_CheckReady_NoConfig(t *testing.T) {
	th := newTestHandler()

	rr := makeRequest(t, th.handler.CheckReady, "GET", "/api/v1/signing/nonexistent/ready", nil, map[string]string{"scenario": "nonexistent"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.ReadinessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Ready)
}

func TestHandler_CheckReady_WithPlatforms(t *testing.T) {
	th := newTestHandler()

	th.repo.AddConfig("test-scenario", &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	})

	rr := makeRequest(t, th.handler.CheckReady, "GET", "/api/v1/signing/test-scenario/ready", nil, map[string]string{"scenario": "test-scenario"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.ReadinessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-scenario", response.Scenario)
	assert.True(t, response.Ready) // At least one platform is configured
}

func TestHandler_GetPrerequisites_Success(t *testing.T) {
	th := newTestHandler()

	th.prereqChecker.tools = []types.ToolDetectionResult{
		{Platform: types.PlatformLinux, Tool: "gpg", Installed: true, Path: "/usr/bin/gpg", Version: "2.2.27"},
		{Platform: types.PlatformMacOS, Tool: "codesign", Installed: true, Path: "/usr/bin/codesign"},
	}

	rr := makeRequest(t, th.handler.GetPrerequisites, "GET", "/api/v1/signing/prerequisites", nil, nil)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	tools, ok := response["tools"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tools, 2)
}

func TestHandler_DiscoverCertificates_Success(t *testing.T) {
	th := newTestHandler()

	th.detector.certs = []types.DiscoveredCertificate{
		{
			ID:       "ABC123",
			Name:     "Test Certificate",
			Platform: types.PlatformWindows,
		},
	}

	rr := makeRequest(t, th.handler.DiscoverCertificates, "GET", "/api/v1/signing/discover/windows", nil, map[string]string{"platform": "windows"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "windows", response["platform"])

	certs, ok := response["certificates"].([]interface{})
	require.True(t, ok)
	assert.Len(t, certs, 1)
}

func TestHandler_RegisterRoutes(t *testing.T) {
	handler := NewHandler()
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Verify routes are registered by checking that requests don't 404
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/signing/test-scenario"},
		{"PUT", "/api/v1/signing/test-scenario"},
		{"DELETE", "/api/v1/signing/test-scenario"},
		{"PATCH", "/api/v1/signing/test-scenario/windows"},
		{"DELETE", "/api/v1/signing/test-scenario/windows"},
		{"POST", "/api/v1/signing/test-scenario/validate"},
		{"GET", "/api/v1/signing/test-scenario/ready"},
		{"GET", "/api/v1/signing/prerequisites"},
		{"GET", "/api/v1/signing/discover/windows"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			// Should not be 404 (route exists)
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "Route %s %s should be registered", route.method, route.path)
		})
	}
}
