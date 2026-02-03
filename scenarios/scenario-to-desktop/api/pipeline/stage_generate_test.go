package pipeline

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
)

// trackingBuildStore tracks Get/Update calls for test verification.
type trackingBuildStore struct {
	mu          sync.RWMutex
	statuses    map[string]*generation.BuildStatus
	getCalls    int
	updateCalls int
}

func newTrackingBuildStore() *trackingBuildStore {
	return &trackingBuildStore{
		statuses: make(map[string]*generation.BuildStatus),
	}
}

func (s *trackingBuildStore) Create(buildID string) *generation.BuildStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := &generation.BuildStatus{
		BuildID: buildID,
		Status:  "building",
	}
	s.statuses[buildID] = status
	return status
}

func (s *trackingBuildStore) Get(buildID string) (*generation.BuildStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	status, ok := s.statuses[buildID]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent external modification
	copy := *status
	return &copy, true
}

func (s *trackingBuildStore) Update(buildID string, fn func(*generation.BuildStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCalls++
	if status, ok := s.statuses[buildID]; ok {
		fn(status)
	}
}

func (s *trackingBuildStore) GetCalls() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getCalls
}

func TestWaitForGeneration_PollsStoreNotPointer(t *testing.T) {
	// Setup: mock store that updates status asynchronously
	store := newTrackingBuildStore()

	// Create initial "building" status
	buildID := "test-build-123"
	store.Create(buildID)

	stage := NewGenerateStage(
		WithGenerateBuildStore(store),
		WithGenerateTimeProvider(NewRealTimeProvider()),
	)

	// Simulate async completion after 600ms (after first poll at 500ms)
	go func() {
		time.Sleep(600 * time.Millisecond)
		store.Update(buildID, func(status *generation.BuildStatus) {
			status.Status = "ready"
			status.OutputPath = "/output/path"
		})
	}()

	// Create stale pointer (simulates what QueueBuild returns)
	stalePointer := &generation.BuildStatus{
		BuildID: buildID,
		Status:  "building", // Will never change
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should succeed by polling the store, not the stale pointer
	path, err := stage.waitForGeneration(ctx, buildID, stalePointer)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if path != "/output/path" {
		t.Errorf("expected /output/path, got %s", path)
	}
	// Should have at least polled once before completion (the store's Get is called each poll)
	if store.GetCalls() < 1 {
		t.Errorf("expected at least 1 Get() call (polling), got %d", store.GetCalls())
	}
}

func TestWaitForGeneration_HandlesFailure(t *testing.T) {
	store := newTrackingBuildStore()
	buildID := "build-fail"
	store.Create(buildID)

	stage := NewGenerateStage(WithGenerateBuildStore(store))

	go func() {
		time.Sleep(50 * time.Millisecond)
		store.Update(buildID, func(status *generation.BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = []string{"template error"}
		})
	}()

	ctx := context.Background()
	_, err := stage.waitForGeneration(ctx, buildID, &generation.BuildStatus{Status: "building"})

	if err == nil || !strings.Contains(err.Error(), "template error") {
		t.Errorf("expected failure with error message, got: %v", err)
	}
}

func TestWaitForGeneration_RespectsContext(t *testing.T) {
	store := newTrackingBuildStore()
	buildID := "build-ctx"
	store.Create(buildID)

	stage := NewGenerateStage(WithGenerateBuildStore(store))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel() // Cancel before completion
	}()

	_, err := stage.waitForGeneration(ctx, buildID, &generation.BuildStatus{Status: "building"})

	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected cancellation error, got: %v", err)
	}
}

func TestWaitForGeneration_ImmediateSuccess(t *testing.T) {
	store := newTrackingBuildStore()
	stage := NewGenerateStage(WithGenerateBuildStore(store))

	// If the initial status is already "ready", should return immediately
	initialStatus := &generation.BuildStatus{
		BuildID:    "immediate-123",
		Status:     "ready",
		OutputPath: "/immediate/path",
	}

	ctx := context.Background()
	path, err := stage.waitForGeneration(ctx, "immediate-123", initialStatus)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if path != "/immediate/path" {
		t.Errorf("expected /immediate/path, got %s", path)
	}
	// Should not have polled the store since initial status was ready
	if store.GetCalls() != 0 {
		t.Errorf("expected 0 Get() calls for immediate success, got %d", store.GetCalls())
	}
}

func TestWaitForGeneration_BuildNotFound(t *testing.T) {
	store := newTrackingBuildStore()
	// Don't create any status - simulate missing build

	stage := NewGenerateStage(WithGenerateBuildStore(store))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := stage.waitForGeneration(ctx, "nonexistent-build", &generation.BuildStatus{Status: "building"})

	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestWaitForGeneration_NilStore(t *testing.T) {
	// Without a buildStore, it should fall back to checking the initialStatus pointer
	// This won't work for async updates, but maintains backward compatibility
	stage := NewGenerateStage() // No buildStore

	initialStatus := &generation.BuildStatus{
		BuildID:    "no-store-build",
		Status:     "ready",
		OutputPath: "/no-store/path",
	}

	ctx := context.Background()
	path, err := stage.waitForGeneration(ctx, "no-store-build", initialStatus)
	if err != nil {
		t.Fatalf("expected success with nil store and ready status, got error: %v", err)
	}
	if path != "/no-store/path" {
		t.Errorf("expected /no-store/path, got %s", path)
	}
}

func TestGenerateStage_WithBuildStore(t *testing.T) {
	store := newTrackingBuildStore()
	stage := NewGenerateStage(
		WithGenerateBuildStore(store),
	)

	if stage.buildStore == nil {
		t.Error("expected buildStore to be set")
	}
}

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
// This is critical for bundled mode to work correctly - the token path mismatch
// was causing desktop apps to timeout waiting for a non-existent token file.
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

// TestExtractBundleUIServiceConfig tests the helper function directly.
func TestExtractBundleUIServiceConfig(t *testing.T) {
	tests := []struct {
		name         string
		content      map[string]interface{}
		wantSvcID    string
		wantPortName string
	}{
		{
			name:         "nil content",
			content:      nil,
			wantSvcID:    "",
			wantPortName: "",
		},
		{
			name:         "empty content",
			content:      map[string]interface{}{},
			wantSvcID:    "",
			wantPortName: "",
		},
		{
			name: "no services section",
			content: map[string]interface{}{
				"app": map[string]interface{}{"name": "test"},
			},
			wantSvcID:    "",
			wantPortName: "",
		},
		{
			name: "services is wrong type",
			content: map[string]interface{}{
				"services": "not an array",
			},
			wantSvcID:    "",
			wantPortName: "",
		},
		{
			name: "UI service with ports",
			content: map[string]interface{}{
				"services": []interface{}{
					map[string]interface{}{
						"type": "ui-bundle",
						"id":   "ui",
						"ports": map[string]interface{}{
							"requested": []interface{}{
								map[string]interface{}{
									"name": "ui",
								},
							},
						},
					},
				},
			},
			wantSvcID:    "ui",
			wantPortName: "ui",
		},
		{
			name: "UI service without ports defaults port name",
			content: map[string]interface{}{
				"services": []interface{}{
					map[string]interface{}{
						"type": "ui",
						"id":   "frontend",
					},
				},
			},
			wantSvcID:    "frontend",
			wantPortName: "ui",
		},
		{
			name: "multiple services returns first UI service",
			content: map[string]interface{}{
				"services": []interface{}{
					map[string]interface{}{
						"type": "api",
						"id":   "api-service",
					},
					map[string]interface{}{
						"type": "ui-bundle",
						"id":   "first-ui",
						"ports": map[string]interface{}{
							"requested": []interface{}{
								map[string]interface{}{
									"name": "http",
								},
							},
						},
					},
					map[string]interface{}{
						"type": "ui",
						"id":   "second-ui",
					},
				},
			},
			wantSvcID:    "first-ui",
			wantPortName: "http",
		},
		{
			name: "non-UI services are skipped",
			content: map[string]interface{}{
				"services": []interface{}{
					map[string]interface{}{
						"type": "api",
						"id":   "api-service",
					},
					map[string]interface{}{
						"type": "worker",
						"id":   "background-worker",
					},
				},
			},
			wantSvcID:    "",
			wantPortName: "",
		},
		{
			name: "service with empty id is skipped",
			content: map[string]interface{}{
				"services": []interface{}{
					map[string]interface{}{
						"type": "ui",
						"id":   "",
					},
					map[string]interface{}{
						"type": "ui-bundle",
						"id":   "valid-ui",
					},
				},
			},
			wantSvcID:    "valid-ui",
			wantPortName: "ui",
		},
		{
			name: "frontend type is recognized",
			content: map[string]interface{}{
				"services": []interface{}{
					map[string]interface{}{
						"type": "frontend",
						"id":   "webapp",
						"ports": map[string]interface{}{
							"requested": []interface{}{
								map[string]interface{}{
									"name": "web",
								},
							},
						},
					},
				},
			},
			wantSvcID:    "webapp",
			wantPortName: "web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcID, portName := extractBundleUIServiceConfig(tt.content)

			if svcID != tt.wantSvcID {
				t.Errorf("serviceID = %q, want %q", svcID, tt.wantSvcID)
			}

			if portName != tt.wantPortName {
				t.Errorf("portName = %q, want %q", portName, tt.wantPortName)
			}
		})
	}
}

// TestGenerateStage_BundledMode_ExtractsBundleUIService verifies that UI service
// configuration from bundle.json is correctly extracted and passed to the desktop config.
func TestGenerateStage_BundledMode_ExtractsBundleUIService(t *testing.T) {
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

	// ManifestContent with UI service configuration
	manifestContent := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"type": "ui-bundle",
				"id":   "ui",
				"ports": map[string]interface{}{
					"requested": []interface{}{
						map[string]interface{}{
							"name": "ui",
						},
					},
				},
			},
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

	// Critical assertion: BundleUISvcID and BundleUIPortName should be extracted
	if svc.lastConfig.BundleUISvcID != "ui" {
		t.Errorf("BundleUISvcID = %q, want %q", svc.lastConfig.BundleUISvcID, "ui")
	}

	if svc.lastConfig.BundleUIPortName != "ui" {
		t.Errorf("BundleUIPortName = %q, want %q", svc.lastConfig.BundleUIPortName, "ui")
	}

	// Verify logs mention UI service extraction
	found := false
	for _, log := range result.Logs {
		if strings.Contains(log, "Bundle UI service") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry about Bundle UI service extraction")
	}
}

// TestGenerateStage_EmptyDeploymentMode_UsesPipelineDefault verifies that when
// the pipeline config doesn't explicitly set a deployment mode, the pipeline's
// default ("bundled") is used.
//
// IMPORTANT: "bundled" is the default because it creates fully self-contained
// desktop applications that work offline without any external dependencies.
// This is the most common deployment mode for production desktop apps.
func TestGenerateStage_EmptyDeploymentMode_UsesPipelineDefault(t *testing.T) {
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
			ScenarioName: "test-app",
			// DeploymentMode is intentionally empty - should use pipeline default "bundled"
			Platforms: []string{"linux"},
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

	// When DeploymentMode is not explicitly set, the pipeline's default "bundled" should be used
	if svc.lastConfig.DeploymentMode != DeploymentModeBundled {
		t.Errorf("DeploymentMode should use pipeline default 'bundled' when not explicitly set, got %q",
			svc.lastConfig.DeploymentMode)
	}
}
