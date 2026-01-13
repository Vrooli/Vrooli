package generation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// mockService implements parts of DefaultService for testing
type mockService struct {
	vrooliRoot  string
	templateDir string
	analyzer    ScenarioAnalyzer
}

func (m *mockService) GetAnalyzer() ScenarioAnalyzer {
	return m.analyzer
}

func (m *mockService) StandardOutputPath(appName string) string {
	return filepath.Join(m.vrooliRoot, "scenarios", appName, "platforms", "electron")
}

func (m *mockService) ScenarioRoot(appName string) string {
	return filepath.Join(m.vrooliRoot, "scenarios", appName)
}

func (m *mockService) QueueBuild(config *DesktopConfig, metadata *ScenarioMetadata, quick bool) *BuildStatus {
	return &BuildStatus{
		BuildID:    "test-build-123",
		Status:     "building",
		OutputPath: config.OutputPath,
	}
}

// mockAnalyzer implements ScenarioAnalyzer for testing
type mockAnalyzer struct {
	metadata *ScenarioMetadata
	err      error
}

func (m *mockAnalyzer) AnalyzeScenario(name string) (*ScenarioMetadata, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metadata, nil
}

func (m *mockAnalyzer) CreateDesktopConfigFromMetadata(metadata *ScenarioMetadata, templateType string) (*DesktopConfig, error) {
	return &DesktopConfig{
		AppName:        metadata.Name,
		AppDisplayName: metadata.DisplayName,
		AppDescription: metadata.Description,
		TemplateType:   templateType,
	}, nil
}

func (m *mockAnalyzer) ValidateScenarioForDesktop(metadata *ScenarioMetadata) error {
	return nil
}

// mockConfigValidator implements ConfigValidator for testing
type mockConfigValidator struct {
	validateErr error
}

func (m *mockConfigValidator) Validate(config *DesktopConfig) error {
	return m.validateErr
}

func (m *mockConfigValidator) ValidateAndPrepareBundle(config *DesktopConfig) error {
	return m.validateErr
}

// mockConfigLoader implements ConfigLoader for testing
type mockConfigLoader struct {
	config *ConnectionConfig
	err    error
}

func (m *mockConfigLoader) Load(scenarioPath string) (*ConnectionConfig, error) {
	return m.config, m.err
}

// mockConfigPersister implements ConfigPersister for testing
type mockConfigPersister struct {
	called bool
}

func (m *mockConfigPersister) Save(scenarioPath string, config *DesktopConfig) error {
	m.called = true
	return nil
}

func (m *mockConfigPersister) Load(scenarioPath string) (*ConnectionConfig, error) {
	return nil, nil
}

// mockRecordDeleter implements RecordDeleter for testing
type mockRecordDeleter struct {
	deletedCount int
}

func (m *mockRecordDeleter) DeleteByScenario(scenarioName string) int {
	return m.deletedCount
}

// Minimal real service for handler creation
func newTestService(vrooliRoot string) *DefaultService {
	return &DefaultService{
		vrooliRoot: vrooliRoot,
	}
}

func TestNewHandler(t *testing.T) {
	svc := newTestService("/tmp")
	h := NewHandler(svc)

	if h == nil {
		t.Fatal("expected handler to be created")
	}
	if h.service != svc {
		t.Error("expected service to be set")
	}
	if h.logger == nil {
		t.Error("expected default logger to be set")
	}
}

func TestHandlerOptions(t *testing.T) {
	svc := newTestService("/tmp")

	validator := &mockConfigValidator{}
	persister := &mockConfigPersister{}
	loader := &mockConfigLoader{}
	deleter := &mockRecordDeleter{deletedCount: 5}

	h := NewHandler(svc,
		WithConfigValidator(validator),
		WithConfigPersister(persister),
		WithConfigLoader(loader),
		WithRecordDeleter(deleter),
	)

	if h.configValidator == nil {
		t.Error("expected config validator to be set")
	}
	if h.configPersister == nil {
		t.Error("expected config persister to be set")
	}
	if h.configLoader == nil {
		t.Error("expected config loader to be set")
	}
	if h.recordDeleter == nil {
		t.Error("expected record deleter to be set")
	}
}

func TestRegisterRoutes(t *testing.T) {
	svc := newTestService("/tmp")
	h := NewHandler(svc)
	router := mux.NewRouter()

	h.RegisterRoutes(router)

	// Verify routes are registered by checking if they match
	req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate", nil)
	match := &mux.RouteMatch{}
	if !router.Match(req, match) {
		t.Error("expected /api/v1/desktop/generate route to be registered")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate/quick", nil)
	if !router.Match(req, match) {
		t.Error("expected /api/v1/desktop/generate/quick route to be registered")
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/desktop/delete/test-scenario", nil)
	if !router.Match(req, match) {
		t.Error("expected /api/v1/desktop/delete/{scenario_name} route to be registered")
	}
}

func TestGenerateHandler(t *testing.T) {
	svc := newTestService("/tmp")
	h := NewHandler(svc)

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate", bytes.NewBufferString("not json"))
		rr := httptest.NewRecorder()

		h.GenerateHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("valid config without validator", func(t *testing.T) {
		config := &DesktopConfig{
			AppName:    "test-app",
			OutputPath: "/tmp/output",
		}
		body, _ := json.Marshal(config)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.GenerateHandler(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rr.Code)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		validator := &mockConfigValidator{validateErr: os.ErrNotExist}
		h2 := NewHandler(svc, WithConfigValidator(validator))

		config := &DesktopConfig{AppName: "test-app"}
		body, _ := json.Marshal(config)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h2.GenerateHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("with config persister", func(t *testing.T) {
		persister := &mockConfigPersister{}
		h2 := NewHandler(svc, WithConfigPersister(persister))

		config := &DesktopConfig{
			AppName:    "test-app",
			OutputPath: "/tmp/output",
		}
		body, _ := json.Marshal(config)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h2.GenerateHandler(rr, req)

		if !persister.called {
			t.Error("expected config persister to be called")
		}
	})
}

func TestQuickGenerateHandler(t *testing.T) {
	svc := newTestService("/tmp")

	t.Run("invalid JSON", func(t *testing.T) {
		h := NewHandler(svc)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate/quick", bytes.NewBufferString("not json"))
		rr := httptest.NewRecorder()

		h.QuickGenerateHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("missing scenario name", func(t *testing.T) {
		h := NewHandler(svc)
		request := QuickGenerateRequest{}
		body, _ := json.Marshal(request)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate/quick", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.QuickGenerateHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("invalid template type", func(t *testing.T) {
		h := NewHandler(svc)
		request := QuickGenerateRequest{
			ScenarioName: "test",
			TemplateType: "invalid-type",
		}
		body, _ := json.Marshal(request)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate/quick", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.QuickGenerateHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("no analyzer configured", func(t *testing.T) {
		h := NewHandler(svc)
		request := QuickGenerateRequest{
			ScenarioName: "test",
		}
		body, _ := json.Marshal(request)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/generate/quick", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		h.QuickGenerateHandler(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})
}

func TestDeleteHandler(t *testing.T) {
	tmpDir := t.TempDir()
	svc := newTestService(tmpDir)
	h := NewHandler(svc, WithRecordDeleter(&mockRecordDeleter{deletedCount: 3}))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/desktop/delete/{scenario_name}", h.DeleteHandler).Methods("DELETE")

	t.Run("missing scenario name", func(t *testing.T) {
		// Create a request without the mux vars
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/desktop/delete/", nil)
		rr := httptest.NewRecorder()

		h.DeleteHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	// Note: Path traversal attacks are also handled by isSafeScenarioName, tested directly below

	t.Run("scenario not found (already deleted)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/desktop/delete/nonexistent", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("successful deletion", func(t *testing.T) {
		// Create the directory to delete
		scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario", "platforms", "electron")
		if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
			t.Fatalf("failed to create scenario dir: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/desktop/delete/test-scenario", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		// Verify directory was deleted
		if _, err := os.Stat(scenarioDir); !os.IsNotExist(err) {
			t.Error("expected directory to be deleted")
		}
	})
}

func TestIsValidTemplateType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"universal", "universal", true},
		{"basic", "basic", true},
		{"advanced", "advanced", true},
		{"multi_window", "multi_window", true},
		{"kiosk", "kiosk", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
		{"random", "random123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTemplateType(tt.input)
			if result != tt.expected {
				t.Errorf("isValidTemplateType(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDecodeDesktopConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		body := []byte(`{"app_name": "test", "version": "1.0.0"}`)
		config, err := decodeDesktopConfig(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.AppName != "test" {
			t.Errorf("expected app_name 'test', got %q", config.AppName)
		}
		if config.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %q", config.Version)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := []byte(`not valid json`)
		_, err := decodeDesktopConfig(body)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestMergeQuickGenerateConfig(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		result := mergeQuickGenerateConfig(nil, QuickGenerateRequest{}, nil, "/tmp")
		if result != nil {
			t.Error("expected nil result for nil config")
		}
	})

	t.Run("request values take priority", func(t *testing.T) {
		config := &DesktopConfig{}
		request := QuickGenerateRequest{
			ProxyURL:       "http://request.proxy",
			DeploymentMode: "bundled",
			Platforms:      []string{"linux"},
		}

		result := mergeQuickGenerateConfig(config, request, nil, "/tmp/output")

		if result.ProxyURL != "http://request.proxy" {
			t.Errorf("expected proxy URL from request, got %q", result.ProxyURL)
		}
		if result.DeploymentMode != "bundled" {
			t.Errorf("expected deployment mode from request, got %q", result.DeploymentMode)
		}
		if len(result.Platforms) != 1 || result.Platforms[0] != "linux" {
			t.Errorf("expected platforms from request, got %v", result.Platforms)
		}
	})

	t.Run("saved config values used when request empty", func(t *testing.T) {
		config := &DesktopConfig{}
		savedConfig := &ConnectionConfig{
			ProxyURL:         "http://saved.proxy",
			DeploymentMode:   "proxy",
			VrooliBinaryPath: "/usr/local/bin/vrooli",
			AppDisplayName:   "Saved App",
		}

		result := mergeQuickGenerateConfig(config, QuickGenerateRequest{}, savedConfig, "/tmp/output")

		if result.ProxyURL != "http://saved.proxy" {
			t.Errorf("expected proxy URL from saved config, got %q", result.ProxyURL)
		}
		if result.DeploymentMode != "proxy" {
			t.Errorf("expected deployment mode from saved config, got %q", result.DeploymentMode)
		}
		if result.AppDisplayName != "Saved App" {
			t.Errorf("expected app display name from saved config, got %q", result.AppDisplayName)
		}
	})

	t.Run("default output path when empty", func(t *testing.T) {
		config := &DesktopConfig{}
		result := mergeQuickGenerateConfig(config, QuickGenerateRequest{}, nil, "/default/path")
		if result.OutputPath != "/default/path" {
			t.Errorf("expected default output path, got %q", result.OutputPath)
		}
	})

	t.Run("default location mode", func(t *testing.T) {
		config := &DesktopConfig{}
		result := mergeQuickGenerateConfig(config, QuickGenerateRequest{}, nil, "/tmp")
		if result.LocationMode != "proper" {
			t.Errorf("expected default location mode 'proper', got %q", result.LocationMode)
		}
	})
}

func TestChooseProxyURL(t *testing.T) {
	t.Run("request proxy_url first priority", func(t *testing.T) {
		request := QuickGenerateRequest{ProxyURL: "http://request.proxy"}
		result := chooseProxyURL(request, &ConnectionConfig{ProxyURL: "http://saved.proxy"})
		if result != "http://request.proxy" {
			t.Errorf("expected request proxy URL, got %q", result)
		}
	})

	t.Run("saved config second priority", func(t *testing.T) {
		request := QuickGenerateRequest{}
		result := chooseProxyURL(request, &ConnectionConfig{ProxyURL: "http://saved.proxy"})
		if result != "http://saved.proxy" {
			t.Errorf("expected saved proxy URL, got %q", result)
		}
	})

	t.Run("legacy server_url third priority", func(t *testing.T) {
		request := QuickGenerateRequest{LegacyServerURL: "http://legacy.server"}
		result := chooseProxyURL(request, nil)
		if result != "http://legacy.server" {
			t.Errorf("expected legacy server URL, got %q", result)
		}
	})

	t.Run("legacy api_url fourth priority", func(t *testing.T) {
		request := QuickGenerateRequest{LegacyAPIURL: "http://legacy.api"}
		result := chooseProxyURL(request, nil)
		if result != "http://legacy.api" {
			t.Errorf("expected legacy API URL, got %q", result)
		}
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		result := chooseProxyURL(QuickGenerateRequest{}, nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestChooseVrooliBinary(t *testing.T) {
	t.Run("request value first priority", func(t *testing.T) {
		result := chooseVrooliBinary(QuickGenerateRequest{VrooliBinary: "/request/vrooli"}, nil, "/existing")
		if result != "/request/vrooli" {
			t.Errorf("expected request vrooli binary, got %q", result)
		}
	})

	t.Run("saved config second priority", func(t *testing.T) {
		result := chooseVrooliBinary(QuickGenerateRequest{}, &ConnectionConfig{VrooliBinaryPath: "/saved/vrooli"}, "/existing")
		if result != "/saved/vrooli" {
			t.Errorf("expected saved vrooli binary, got %q", result)
		}
	})

	t.Run("existing value used when nothing else set", func(t *testing.T) {
		result := chooseVrooliBinary(QuickGenerateRequest{}, nil, "/existing/vrooli")
		if result != "/existing/vrooli" {
			t.Errorf("expected existing vrooli binary, got %q", result)
		}
	})
}

func TestChooseDeploymentMode(t *testing.T) {
	t.Run("request value first priority", func(t *testing.T) {
		result := chooseDeploymentMode(QuickGenerateRequest{DeploymentMode: "bundled"}, nil, "proxy")
		if result != "bundled" {
			t.Errorf("expected request mode, got %q", result)
		}
	})

	t.Run("saved config second priority", func(t *testing.T) {
		result := chooseDeploymentMode(QuickGenerateRequest{}, &ConnectionConfig{DeploymentMode: "bundled"}, "proxy")
		if result != "bundled" {
			t.Errorf("expected saved mode, got %q", result)
		}
	})

	t.Run("existing value used when nothing else set", func(t *testing.T) {
		result := chooseDeploymentMode(QuickGenerateRequest{}, nil, "proxy")
		if result != "proxy" {
			t.Errorf("expected existing mode, got %q", result)
		}
	})
}

func TestChooseBundleManifestPath(t *testing.T) {
	t.Run("request value first priority", func(t *testing.T) {
		result := chooseBundleManifestPath(QuickGenerateRequest{BundleManifest: "/request/manifest.json"}, nil)
		if result != "/request/manifest.json" {
			t.Errorf("expected request manifest path, got %q", result)
		}
	})

	t.Run("saved config second priority", func(t *testing.T) {
		result := chooseBundleManifestPath(QuickGenerateRequest{}, &ConnectionConfig{BundleManifestPath: "/saved/manifest.json"})
		if result != "/saved/manifest.json" {
			t.Errorf("expected saved manifest path, got %q", result)
		}
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		result := chooseBundleManifestPath(QuickGenerateRequest{}, nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestChoosePlatforms(t *testing.T) {
	t.Run("requested platforms first priority", func(t *testing.T) {
		result := choosePlatforms([]string{"linux", "win"}, []string{"mac"})
		if len(result) != 2 || result[0] != "linux" {
			t.Errorf("expected requested platforms, got %v", result)
		}
	})

	t.Run("existing platforms second priority", func(t *testing.T) {
		result := choosePlatforms(nil, []string{"mac", "linux"})
		if len(result) != 2 || result[0] != "mac" {
			t.Errorf("expected existing platforms, got %v", result)
		}
	})

	t.Run("nil when nothing set", func(t *testing.T) {
		result := choosePlatforms(nil, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestChooseAutoManage(t *testing.T) {
	boolTrue := true
	boolFalse := false

	t.Run("request auto_manage_vrooli first priority", func(t *testing.T) {
		result := chooseAutoManage(QuickGenerateRequest{AutoManageVrooli: &boolTrue}, nil, false)
		if !result {
			t.Error("expected true from request")
		}
	})

	t.Run("request legacy_auto_manage second priority", func(t *testing.T) {
		result := chooseAutoManage(QuickGenerateRequest{LegacyAutoManage: &boolFalse}, nil, true)
		if result {
			t.Error("expected false from legacy request")
		}
	})

	t.Run("saved config third priority", func(t *testing.T) {
		result := chooseAutoManage(QuickGenerateRequest{}, &ConnectionConfig{AutoManageVrooli: true}, false)
		if !result {
			t.Error("expected true from saved config")
		}
	})

	t.Run("existing value used when nothing else set", func(t *testing.T) {
		result := chooseAutoManage(QuickGenerateRequest{}, nil, true)
		if !result {
			t.Error("expected existing value true")
		}
	})
}

func TestIsSafeScenarioName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple name", "my-scenario", true},
		{"with underscores", "my_scenario_123", true},
		{"path traversal ..", "../etc", false},
		{"path traversal forward slash", "path/to/scenario", false},
		{"path traversal backslash", "path\\to\\scenario", false},
		{"hidden path traversal", "scenario/../etc", false},
		{"empty string", "", true},
		{"alphanumeric", "scenario123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeScenarioName(tt.input)
			if result != tt.expected {
				t.Errorf("isSafeScenarioName(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
