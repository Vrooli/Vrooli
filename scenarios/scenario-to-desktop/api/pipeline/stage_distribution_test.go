package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/distribution"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

// =============================================================================
// Mock distribution service
// =============================================================================

type mockDistributionService struct {
	distributeResp *distribution.DistributeResponse
	distributeErr  error
	lastRequest    *distribution.DistributeRequest
}

func (m *mockDistributionService) Distribute(ctx context.Context, req *distribution.DistributeRequest) (*distribution.DistributeResponse, error) {
	m.lastRequest = req
	if m.distributeErr != nil {
		return nil, m.distributeErr
	}
	if m.distributeResp == nil {
		return &distribution.DistributeResponse{
			DistributionID: "dist-123",
			Status:         distribution.StatusRunning,
		}, nil
	}
	return m.distributeResp, nil
}

func (m *mockDistributionService) GetDistributionStatus(distributionID string) (*distribution.DistributionStatus, bool) {
	return nil, false
}

func (m *mockDistributionService) ListDistributions() []*distribution.DistributionStatus {
	return nil
}

func (m *mockDistributionService) CancelDistribution(distributionID string) bool {
	return false
}

func (m *mockDistributionService) ValidateTargets(ctx context.Context, targetNames []string) *distribution.ValidationResult {
	return nil
}

func (m *mockDistributionService) CheckCredentials(ctx context.Context, req *distribution.CheckCredentialsRequest) *distribution.CheckCredentialsResponse {
	return nil
}

// =============================================================================
// Mock distribution store
// =============================================================================

type mockDistributionStore struct {
	status *distribution.DistributionStatus
}

func (m *mockDistributionStore) Save(status *distribution.DistributionStatus) {}

func (m *mockDistributionStore) Get(distributionID string) (*distribution.DistributionStatus, bool) {
	if m.status == nil {
		return nil, false
	}
	return m.status, true
}

func (m *mockDistributionStore) Update(distributionID string, fn func(*distribution.DistributionStatus)) bool {
	return false
}

func (m *mockDistributionStore) List() []*distribution.DistributionStatus {
	return nil
}

func (m *mockDistributionStore) Delete(distributionID string) bool {
	return false
}

// =============================================================================
// Mock manifest generator dependencies
// =============================================================================

type mockManifestProvider struct {
	name             string
	validateErr      error
	publishConfig    map[string]interface{}
	publishConfigErr error
	manifestResult   *updates.ManifestResult
	manifestErr      error
	requiresUpload   bool
}

func (m *mockManifestProvider) Name() string    { return m.name }
func (m *mockManifestProvider) Validate() error { return m.validateErr }
func (m *mockManifestProvider) GetPublishConfig(channel string) (map[string]interface{}, error) {
	return m.publishConfig, m.publishConfigErr
}

func (m *mockManifestProvider) GenerateManifest(ctx context.Context, req *updates.ManifestRequest) (*updates.ManifestResult, error) {
	return m.manifestResult, m.manifestErr
}
func (m *mockManifestProvider) RequiresManifestUpload() bool { return m.requiresUpload }

type mockManifestProviderFactory struct {
	provider Provider
	warnings []updates.ValidationWarning
	err      error
}

func (m *mockManifestProviderFactory) Create(config *generation.UpdateConfig) (updates.Provider, []updates.ValidationWarning, error) {
	return m.provider, m.warnings, m.err
}

// Provider interface adapter to satisfy updates.Provider
type Provider = updates.Provider

// =============================================================================
// Tests
// =============================================================================

func TestDistributionStage_Execute_WithManifestGenerator(t *testing.T) {
	// Arrange
	manifestResult := &updates.ManifestResult{
		Manifests: map[string][]byte{
			"latest.yml": []byte("version: 1.0.0"),
		},
		ManifestPaths: map[string]string{
			"latest.yml": "/tmp/output/latest.yml",
		},
		Warnings: []updates.ValidationWarning{},
	}

	provider := &mockManifestProvider{
		name:           "generic",
		requiresUpload: true,
		publishConfig:  map[string]interface{}{"url": "https://updates.example.com/stable"},
		manifestResult: manifestResult,
	}

	factory := &mockManifestProviderFactory{provider: provider}
	manifestGen := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	completedStatus := &distribution.DistributionStatus{
		DistributionID: "dist-123",
		Status:         distribution.StatusCompleted,
		Targets:        map[string]*distribution.TargetDistribution{},
	}

	svc := &mockDistributionService{}
	store := &mockDistributionStore{status: completedStatus}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionStore(store),
		WithDistributionTimeProvider(mockTime),
		WithUpdateManifestGenerator(manifestGen),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
			Version:      "1.0.0",
			UpdateConfig: &generation.UpdateConfig{
				Provider: "generic",
				Channel:  "stable",
				Generic: &generation.GenericUpdateConfig{
					URL: "https://updates.example.com",
				},
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert
	if result.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, result.Status)
	}

	// Verify manifest file was added to artifacts
	if svc.lastRequest == nil {
		t.Fatal("expected distribute request to be made")
	}

	// Check that manifest artifact was included
	if _, ok := svc.lastRequest.Artifacts["manifest:latest.yml"]; !ok {
		t.Error("expected manifest:latest.yml in artifacts")
	}

	// Check logs mention manifest generation
	hasManifestLog := false
	for _, log := range result.Logs {
		if log == "Generated update manifest: latest.yml" {
			hasManifestLog = true
			break
		}
	}
	if !hasManifestLog {
		t.Error("expected log about generated manifest")
	}
}

func TestDistributionStage_Execute_ManifestGenerationFailure(t *testing.T) {
	// Arrange: manifest generation fails
	provider := &mockManifestProvider{
		name:           "generic",
		requiresUpload: true,
		publishConfig:  map[string]interface{}{"url": "https://updates.example.com/stable"},
		manifestErr:    errors.New("hash calculation failed"),
	}

	factory := &mockManifestProviderFactory{provider: provider}
	manifestGen := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	completedStatus := &distribution.DistributionStatus{
		DistributionID: "dist-123",
		Status:         distribution.StatusCompleted,
		Targets:        map[string]*distribution.TargetDistribution{},
	}

	svc := &mockDistributionService{}
	store := &mockDistributionStore{status: completedStatus}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionStore(store),
		WithDistributionTimeProvider(mockTime),
		WithUpdateManifestGenerator(manifestGen),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
			Version:      "1.0.0",
			UpdateConfig: &generation.UpdateConfig{
				Provider: "generic",
				Generic:  &generation.GenericUpdateConfig{URL: "https://example.com"},
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert: distribution should still succeed (manifest failure is non-fatal)
	if result.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s (manifest failure should be non-fatal)", StatusCompleted, result.Status)
	}

	// Check warning was logged
	hasWarning := false
	for _, log := range result.Logs {
		if len(log) > 7 && log[:7] == "WARNING" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected WARNING log for manifest generation failure")
	}
}

func TestDistributionStage_Execute_ManifestWarnings(t *testing.T) {
	// Arrange: provider returns warnings
	manifestResult := &updates.ManifestResult{
		Manifests:     map[string][]byte{},
		ManifestPaths: map[string]string{},
		Warnings: []updates.ValidationWarning{
			{Code: "EMPTY_ARTIFACTS", Message: "No artifacts provided"},
		},
	}

	provider := &mockManifestProvider{
		name:           "generic",
		requiresUpload: true,
		publishConfig:  map[string]interface{}{"url": "https://updates.example.com/stable"},
		manifestResult: manifestResult,
	}

	factory := &mockManifestProviderFactory{provider: provider}
	manifestGen := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	completedStatus := &distribution.DistributionStatus{
		DistributionID: "dist-123",
		Status:         distribution.StatusCompleted,
		Targets:        map[string]*distribution.TargetDistribution{},
	}

	svc := &mockDistributionService{}
	store := &mockDistributionStore{status: completedStatus}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionStore(store),
		WithDistributionTimeProvider(mockTime),
		WithUpdateManifestGenerator(manifestGen),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
			Version:      "1.0.0",
			UpdateConfig: &generation.UpdateConfig{
				Provider: "generic",
				Generic:  &generation.GenericUpdateConfig{URL: "https://example.com"},
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert
	if result.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, result.Status)
	}

	// Check warnings were logged
	hasWarningLog := false
	for _, log := range result.Logs {
		if len(log) > 25 && log[:25] == "Update manifest warning: " {
			hasWarningLog = true
			break
		}
	}
	if !hasWarningLog {
		t.Error("expected warning log from manifest generation")
	}
}

func TestDistributionStage_Execute_NoManifestGenerator(t *testing.T) {
	// Arrange: no manifest generator configured
	completedStatus := &distribution.DistributionStatus{
		DistributionID: "dist-123",
		Status:         distribution.StatusCompleted,
		Targets:        map[string]*distribution.TargetDistribution{},
	}

	svc := &mockDistributionService{}
	store := &mockDistributionStore{status: completedStatus}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionStore(store),
		WithDistributionTimeProvider(mockTime),
		// No WithUpdateManifestGenerator
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
			Version:      "1.0.0",
			UpdateConfig: &generation.UpdateConfig{
				Provider: "generic",
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert
	if result.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, result.Status)
	}

	// Should not have manifest in artifacts
	if svc.lastRequest != nil {
		if _, ok := svc.lastRequest.Artifacts["manifest:latest.yml"]; ok {
			t.Error("did not expect manifest artifact without generator")
		}
	}
}

func TestDistributionStage_Execute_NoUpdateConfig(t *testing.T) {
	// Arrange: no update config
	manifestGen := updates.NewManifestGenerator()

	completedStatus := &distribution.DistributionStatus{
		DistributionID: "dist-123",
		Status:         distribution.StatusCompleted,
		Targets:        map[string]*distribution.TargetDistribution{},
	}

	svc := &mockDistributionService{}
	store := &mockDistributionStore{status: completedStatus}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionStore(store),
		WithDistributionTimeProvider(mockTime),
		WithUpdateManifestGenerator(manifestGen),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
			Version:      "1.0.0",
			// No UpdateConfig
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert
	if result.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, result.Status)
	}

	// Should not have manifest in artifacts
	if svc.lastRequest != nil {
		for key := range svc.lastRequest.Artifacts {
			if len(key) > 9 && key[:9] == "manifest:" {
				t.Errorf("did not expect manifest artifact without update config, found %s", key)
			}
		}
	}
}

func TestDistributionStage_Execute_GitHubProviderNoManifestUpload(t *testing.T) {
	// Arrange: GitHub provider doesn't require manifest upload
	provider := &mockManifestProvider{
		name:           "github",
		requiresUpload: false,
	}

	factory := &mockManifestProviderFactory{provider: provider}
	manifestGen := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	completedStatus := &distribution.DistributionStatus{
		DistributionID: "dist-123",
		Status:         distribution.StatusCompleted,
		Targets:        map[string]*distribution.TargetDistribution{},
	}

	svc := &mockDistributionService{}
	store := &mockDistributionStore{status: completedStatus}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionStore(store),
		WithDistributionTimeProvider(mockTime),
		WithUpdateManifestGenerator(manifestGen),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
			Version:      "1.0.0",
			UpdateConfig: &generation.UpdateConfig{
				Provider: "github",
			},
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert
	if result.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, result.Status)
	}

	// Should not have manifest in artifacts (GitHub handles manifests internally)
	if svc.lastRequest != nil {
		for key := range svc.lastRequest.Artifacts {
			if len(key) > 9 && key[:9] == "manifest:" {
				t.Errorf("did not expect manifest artifact for GitHub provider, found %s", key)
			}
		}
	}
}

func TestDistributionStage_Execute_NoArtifacts(t *testing.T) {
	// Arrange
	svc := &mockDistributionService{}
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionService(svc),
		WithDistributionTimeProvider(mockTime),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert: should skip when no artifacts
	if result.Status != StatusSkipped {
		t.Errorf("expected status %s, got %s", StatusSkipped, result.Status)
	}
}

func TestDistributionStage_Execute_NoService(t *testing.T) {
	// Arrange
	mockTime := &mockTimeProvider{now: time.Now().Unix()}

	stage := NewDistributionStage(
		WithDistributionTimeProvider(mockTime),
		// No service configured
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"win": {Status: "ready", Artifact: "/tmp/app.exe"},
			},
		},
	}

	// Act
	result := stage.Execute(context.Background(), input)

	// Assert
	if result.Status != StatusFailed {
		t.Errorf("expected status %s, got %s", StatusFailed, result.Status)
	}
}
