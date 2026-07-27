package pipeline

import (
	"context"
	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Bundled mode tests
// =============================================================================

// mockAnalyzer is a test analyzer that returns fixed metadata.
type mockAnalyzer struct {
	metadata *generation.ScenarioMetadata
	err      error
}

func (m *mockAnalyzer) AnalyzeScenario(name string) (*generation.ScenarioMetadata, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metadata, nil
}

func (m *mockAnalyzer) ValidateScenarioForDesktop(name string) error {
	return nil
}

func (m *mockAnalyzer) CreateDesktopConfigFromMetadata(metadata *generation.ScenarioMetadata, templateType string) (*generation.DesktopConfig, error) {
	return &generation.DesktopConfig{
		AppName:        metadata.Name,
		AppDisplayName: metadata.DisplayName,
		ScenarioPath:   metadata.UIDistPath, // This is the key field - set to source ui/dist by default
		ScenarioName:   metadata.Name,
		DeploymentMode: "external-server",
		Framework:      "electron",
		TemplateType:   templateType,
	}, nil
}

// capturingService captures the config passed to QueueBuild.
type capturingService struct {
	lastConfig *generation.DesktopConfig
}

func (s *capturingService) Generate(buildID string, config *generation.DesktopConfig) {
	// No-op for tests
}

func (s *capturingService) QueueBuild(config *generation.DesktopConfig, metadata *generation.ScenarioMetadata, includeMetadata bool) *generation.BuildStatus {
	s.lastConfig = config
	return &generation.BuildStatus{
		BuildID:    "test-build",
		Status:     "ready",
		OutputPath: "/test/output",
	}
}

// TestGenerateStage_BundledMode_SetsScenarioPathToBundle verifies that for bundled
// deployment mode, the ScenarioPath is set to "bundle" instead of the source ui/dist path.
// This ensures extraResources in package.json points to the bundled assets.
func TestGenerateStage_BundledMode_SetsScenarioPathToBundle(t *testing.T) {
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	analyzer := &mockAnalyzer{
		metadata: &generation.ScenarioMetadata{
			Name:        "test-app",
			DisplayName: "Test App",
			Version:     "1.0.0",
			HasUI:       true,
			UIDistPath:  "/home/user/Vrooli/scenarios/test-app/ui/dist", // Source path
		},
	}

	svc := &capturingService{}

	stage := NewGenerateStage(
		WithScenarioAnalyzer(analyzer),
		WithGenerateService(svc),
		WithGenerateTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-app",
			DeploymentMode: "bundled", // Bundled mode
			Platforms:      []string{"linux"},
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:    "/home/user/Vrooli/scenarios/test-app/platforms/electron/bundle",
			ManifestPath: "/home/user/Vrooli/scenarios/test-app/platforms/electron/bundle/bundle.json",
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	if result.Status != StatusCompleted {
		t.Fatalf("expected status %q, got %q (error: %s)", StatusCompleted, result.Status, result.Error)
	}

	// Critical assertion: ScenarioPath should be "bundle" for bundled mode
	if svc.lastConfig == nil {
		t.Fatal("service was not called")
	}

	if svc.lastConfig.ScenarioPath != "bundle" {
		t.Errorf("ScenarioPath should be 'bundle' for bundled mode:\n  got:  %s\n  want: bundle\n  (source path was): %s",
			svc.lastConfig.ScenarioPath,
			analyzer.metadata.UIDistPath)
	}

	// Also verify deployment mode was set
	if svc.lastConfig.DeploymentMode != "bundled" {
		t.Errorf("DeploymentMode should be 'bundled', got %s", svc.lastConfig.DeploymentMode)
	}
}

// TestGenerateStage_NonBundledMode_PreservesSourcePath verifies that for non-bundled
// deployment modes, the ScenarioPath remains the source ui/dist path.
func TestGenerateStage_NonBundledMode_PreservesSourcePath(t *testing.T) {
	mockTime := &mockTimeProvider{now: time.Now().Unix()}
	sourceUIPath := "/home/user/Vrooli/scenarios/test-app/ui/dist"

	analyzer := &mockAnalyzer{
		metadata: &generation.ScenarioMetadata{
			Name:        "test-app",
			DisplayName: "Test App",
			Version:     "1.0.0",
			HasUI:       true,
			UIDistPath:  sourceUIPath,
		},
	}

	svc := &capturingService{}

	stage := NewGenerateStage(
		WithScenarioAnalyzer(analyzer),
		WithGenerateService(svc),
		WithGenerateTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-app",
			DeploymentMode: "external-server", // Non-bundled mode
			Platforms:      []string{"linux"},
		},
		// No BundleResult for non-bundled mode
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	if result.Status != StatusCompleted {
		t.Fatalf("expected status %q, got %q (error: %s)", StatusCompleted, result.Status, result.Error)
	}

	// For non-bundled mode, ScenarioPath should remain the source path
	if svc.lastConfig == nil {
		t.Fatal("service was not called")
	}

	if svc.lastConfig.ScenarioPath != sourceUIPath {
		t.Errorf("ScenarioPath should be source path for non-bundled mode:\n  got:  %s\n  want: %s",
			svc.lastConfig.ScenarioPath,
			sourceUIPath)
	}
}

// TestGenerateStage_BundledMode_ExtractsBundleIPC verifies that IPC configuration
// from bundle.json is correctly extracted and passed to the desktop config.
func TestGenerateStage_BundledMode_ExtractsBundleIPC(t *testing.T) {
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	analyzer := &mockAnalyzer{
		metadata: &generation.ScenarioMetadata{
			Name:        "test-app",
			DisplayName: "Test App",
			Version:     "1.0.0",
			HasUI:       true,
			UIDistPath:  "/home/user/Vrooli/scenarios/test-app/ui/dist",
		},
	}

	svc := &capturingService{}

	stage := NewGenerateStage(
		WithScenarioAnalyzer(analyzer),
		WithGenerateService(svc),
		WithGenerateTimeProvider(mockTime),
	)

	// ManifestContent with IPC configuration - note auth_token_path uses underscore
	manifestContent := map[string]interface{}{
		"ipc": map[string]interface{}{
			"host":            "127.0.0.1",
			"port":            float64(39200), // JSON numbers are float64
			"auth_token_path": "runtime/auth_token",
		},
	}

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-app",
			DeploymentMode: "bundled",
			Platforms:      []string{"linux"},
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:       "/home/user/Vrooli/scenarios/test-app/platforms/electron/bundle",
			ManifestPath:    "/home/user/Vrooli/scenarios/test-app/platforms/electron/bundle/bundle.json",
			ManifestContent: manifestContent,
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	if result.Status != StatusCompleted {
		t.Fatalf("expected status %q, got %q (error: %s)", StatusCompleted, result.Status, result.Error)
	}

	if svc.lastConfig == nil {
		t.Fatal("service was not called")
	}

	// Critical assertion: BundleIPC should be extracted from manifest
	if svc.lastConfig.BundleIPC == nil {
		t.Fatal("BundleIPC should be extracted from ManifestContent, got nil")
	}

	if svc.lastConfig.BundleIPC.Host != "127.0.0.1" {
		t.Errorf("BundleIPC.Host = %q, want %q", svc.lastConfig.BundleIPC.Host, "127.0.0.1")
	}

	if svc.lastConfig.BundleIPC.Port != 39200 {
		t.Errorf("BundleIPC.Port = %d, want %d", svc.lastConfig.BundleIPC.Port, 39200)
	}

	// This is the critical test case - the token path must match exactly
	if svc.lastConfig.BundleIPC.AuthTokenRel != "runtime/auth_token" {
		t.Errorf("BundleIPC.AuthTokenRel = %q, want %q (token path must match bundle.json exactly!)",
			svc.lastConfig.BundleIPC.AuthTokenRel, "runtime/auth_token")
	}

	// Verify logs mention IPC extraction
	found := false
	for _, log := range result.Logs {
		if strings.Contains(log, "Bundle IPC") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry about Bundle IPC extraction")
	}
}

// TestGenerateStage_BundledMode_NilManifestContent verifies graceful handling
// when ManifestContent is nil (shouldn't crash, BundleIPC will be nil).
func TestGenerateStage_BundledMode_NilManifestContent(t *testing.T) {
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	analyzer := &mockAnalyzer{
		metadata: &generation.ScenarioMetadata{
			Name:        "test-app",
			DisplayName: "Test App",
			Version:     "1.0.0",
			HasUI:       true,
			UIDistPath:  "/home/user/Vrooli/scenarios/test-app/ui/dist",
		},
	}

	svc := &capturingService{}

	stage := NewGenerateStage(
		WithScenarioAnalyzer(analyzer),
		WithGenerateService(svc),
		WithGenerateTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:   "test-app",
			DeploymentMode: "bundled",
			Platforms:      []string{"linux"},
		},
		BundleResult: &bundle.PackageResult{
			BundleDir:       "/home/user/Vrooli/scenarios/test-app/platforms/electron/bundle",
			ManifestPath:    "/home/user/Vrooli/scenarios/test-app/platforms/electron/bundle/bundle.json",
			ManifestContent: nil, // No manifest content
		},
		Logger: &mockLogger{},
	}

	result := stage.Execute(context.Background(), input)

	if result.Status != StatusCompleted {
		t.Fatalf("expected status %q, got %q (error: %s)", StatusCompleted, result.Status, result.Error)
	}

	// Should complete without error, BundleIPC will be nil (template uses defaults)
	if svc.lastConfig == nil {
		t.Fatal("service was not called")
	}
}

// TestExtractBundleIPCConfig tests the helper function directly.
func TestExtractBundleIPCConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  map[string]interface{}
		wantNil  bool
		wantHost string
		wantPort int
		wantPath string
	}{
		{
			name:    "nil content",
			content: nil,
			wantNil: true,
		},
		{
			name:    "empty content",
			content: map[string]interface{}{},
			wantNil: true,
		},
		{
			name: "no ipc section",
			content: map[string]interface{}{
				"app": map[string]interface{}{"name": "test"},
			},
			wantNil: true,
		},
		{
			name: "ipc is wrong type",
			content: map[string]interface{}{
				"ipc": "not a map",
			},
			wantNil: true,
		},
		{
			name: "full ipc config",
			content: map[string]interface{}{
				"ipc": map[string]interface{}{
					"host":            "10.0.0.1",
					"port":            float64(12345),
					"auth_token_path": "custom/token/path",
				},
			},
			wantNil:  false,
			wantHost: "10.0.0.1",
			wantPort: 12345,
			wantPath: "custom/token/path",
		},
		{
			name: "partial ipc config - only host",
			content: map[string]interface{}{
				"ipc": map[string]interface{}{
					"host": "192.168.1.1",
				},
			},
			wantNil:  false,
			wantHost: "192.168.1.1",
			wantPort: 0,
			wantPath: "",
		},
		{
			name: "partial ipc config - only port",
			content: map[string]interface{}{
				"ipc": map[string]interface{}{
					"port": float64(9999),
				},
			},
			wantNil:  false,
			wantHost: "",
			wantPort: 9999,
			wantPath: "",
		},
		{
			name: "underscore vs hyphen in token path",
			content: map[string]interface{}{
				"ipc": map[string]interface{}{
					"auth_token_path": "runtime/auth_token", // underscore - common convention
				},
			},
			wantNil:  false,
			wantPath: "runtime/auth_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBundleIPCConfig(tt.content)

			if tt.wantNil {
				if result != nil {
					t.Errorf("extractBundleIPCConfig() = %+v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("extractBundleIPCConfig() = nil, want non-nil")
			}

			if result.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", result.Host, tt.wantHost)
			}

			if result.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", result.Port, tt.wantPort)
			}

			if result.AuthTokenRel != tt.wantPath {
				t.Errorf("AuthTokenRel = %q, want %q", result.AuthTokenRel, tt.wantPath)
			}
		})
	}
}
