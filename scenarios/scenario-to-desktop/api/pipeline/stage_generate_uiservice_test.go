package pipeline

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/generation"
)

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
