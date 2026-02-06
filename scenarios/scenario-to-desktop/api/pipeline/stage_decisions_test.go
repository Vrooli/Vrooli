package pipeline

import (
	"testing"
)

func TestShouldSkipPreflight(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			name:     "skip_preflight true returns true",
			config:   &Config{SkipPreflight: true},
			expected: true,
		},
		{
			name:     "deployment_mode proxy returns true",
			config:   &Config{DeploymentMode: DeploymentModeProxy},
			expected: true,
		},
		{
			name:     "deployment_mode external-server returns true",
			config:   &Config{DeploymentMode: DeploymentModeExternalServer},
			expected: true,
		},
		{
			name:     "deployment_mode cloud-api returns true",
			config:   &Config{DeploymentMode: DeploymentModeCloudAPI},
			expected: true,
		},
		{
			name:     "deployment_mode bundled returns false",
			config:   &Config{DeploymentMode: DeploymentModeBundled},
			expected: false,
		},
		{
			name:     "default deployment mode returns false (bundled)",
			config:   &Config{},
			expected: false, // Default is bundled, which requires preflight
		},
		{
			name:     "skip_preflight with proxy returns true",
			config:   &Config{SkipPreflight: true, DeploymentMode: DeploymentModeProxy},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipPreflight(tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSkipPreflight() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipBundle(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			name:     "deployment_mode proxy returns true",
			config:   &Config{DeploymentMode: DeploymentModeProxy},
			expected: true,
		},
		{
			name:     "deployment_mode external-server returns true",
			config:   &Config{DeploymentMode: DeploymentModeExternalServer},
			expected: true,
		},
		{
			name:     "deployment_mode cloud-api returns true",
			config:   &Config{DeploymentMode: DeploymentModeCloudAPI},
			expected: true,
		},
		{
			name:     "deployment_mode bundled returns false",
			config:   &Config{DeploymentMode: DeploymentModeBundled},
			expected: false,
		},
		{
			name:     "default deployment mode returns false (bundled)",
			config:   &Config{},
			expected: false, // Default is bundled, which requires bundle stage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipBundle(tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSkipBundle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipSmokeTest(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			name:     "skip_smoke_test true returns true",
			config:   &Config{SkipSmokeTest: true},
			expected: true,
		},
		{
			name:     "skip_smoke_test false returns false",
			config:   &Config{SkipSmokeTest: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipSmokeTest(tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSkipSmokeTest() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipDeploy(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config returns true",
			config:   nil,
			expected: true,
		},
		{
			name:     "nil deploy config returns true",
			config:   &Config{},
			expected: true,
		},
		{
			name:     "deploy config present returns false",
			config:   &Config{DeployConfig: &DeployConfig{AppKey: "my-app"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipDeploy(tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSkipDeploy() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateCanResume(t *testing.T) {
	tests := []struct {
		name        string
		status      *Status
		expectError bool
	}{
		{
			name:        "nil status returns error",
			status:      nil,
			expectError: true,
		},
		{
			name:        "running status returns error",
			status:      &Status{PipelineID: "test", Status: StatusRunning},
			expectError: true,
		},
		{
			name:        "failed status returns error",
			status:      &Status{PipelineID: "test", Status: StatusFailed},
			expectError: true,
		},
		{
			name:        "completed without stopped_after_stage returns error",
			status:      &Status{PipelineID: "test", Status: StatusCompleted},
			expectError: true,
		},
		{
			name:        "completed with stopped_after_stage returns nil",
			status:      &Status{PipelineID: "test", Status: StatusCompleted, StoppedAfterStage: StageBuild},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCanResume(tt.status)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateCanResume() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestShouldStopAfterStage(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		stageName string
		expected  bool
	}{
		{
			name:      "nil config returns false",
			config:    nil,
			stageName: StageBuild,
			expected:  false,
		},
		{
			name:      "matching stage returns true",
			config:    &Config{StopAfterStage: StageBuild},
			stageName: StageBuild,
			expected:  true,
		},
		{
			name:      "non-matching stage returns false",
			config:    &Config{StopAfterStage: StageBuild},
			stageName: StageGenerate,
			expected:  false,
		},
		{
			name:      "empty stop_after_stage returns false",
			config:    &Config{},
			stageName: StageBuild,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldStopAfterStage(tt.config, tt.stageName)
			if result != tt.expected {
				t.Errorf("ShouldStopAfterStage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipSigning(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config returns true",
			config:   nil,
			expected: true,
		},
		{
			name:     "sign false returns true",
			config:   &Config{Sign: false},
			expected: true,
		},
		{
			name:     "sign true returns false",
			config:   &Config{Sign: true},
			expected: false,
		},
		{
			name:     "default config returns true (signing is opt-in)",
			config:   &Config{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipSigning(tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSkipSigning() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipGeneration(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			name:     "no resume returns false",
			config:   &Config{},
			expected: false,
		},
		{
			name:     "resume from bundle returns false",
			config:   &Config{ResumeFromStage: StageBundle},
			expected: false,
		},
		{
			name:     "resume from preflight returns false",
			config:   &Config{ResumeFromStage: StagePreflight},
			expected: false,
		},
		{
			name:     "resume from generate returns false",
			config:   &Config{ResumeFromStage: StageGenerate},
			expected: false,
		},
		{
			name:     "resume from build returns true",
			config:   &Config{ResumeFromStage: StageBuild},
			expected: true,
		},
		{
			name:     "resume from smoketest returns true",
			config:   &Config{ResumeFromStage: StageSmokeTest},
			expected: true,
		},
		{
			name:     "resume from deploy returns true",
			config:   &Config{ResumeFromStage: StageDeploy},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipGeneration(tt.config)
			if result != tt.expected {
				t.Errorf("ShouldSkipGeneration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsBuildComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "building returns false",
			status:   BuildStatusBuilding,
			expected: false,
		},
		{
			name:     "ready returns true",
			status:   BuildStatusReady,
			expected: true,
		},
		{
			name:     "partial returns true",
			status:   BuildStatusPartial,
			expected: true,
		},
		{
			name:     "failed returns true",
			status:   BuildStatusFailed,
			expected: true,
		},
		{
			name:     "skipped returns false",
			status:   BuildStatusSkipped,
			expected: false,
		},
		{
			name:     "empty returns false",
			status:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuildComplete(tt.status)
			if result != tt.expected {
				t.Errorf("IsBuildComplete() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsBuildFailed(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "failed returns true",
			status:   BuildStatusFailed,
			expected: true,
		},
		{
			name:     "ready returns false",
			status:   BuildStatusReady,
			expected: false,
		},
		{
			name:     "partial returns false",
			status:   BuildStatusPartial,
			expected: false,
		},
		{
			name:     "building returns false",
			status:   BuildStatusBuilding,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuildFailed(tt.status)
			if result != tt.expected {
				t.Errorf("IsBuildFailed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsBuildSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "ready returns true",
			status:   BuildStatusReady,
			expected: true,
		},
		{
			name:     "partial returns true",
			status:   BuildStatusPartial,
			expected: true,
		},
		{
			name:     "failed returns false",
			status:   BuildStatusFailed,
			expected: false,
		},
		{
			name:     "building returns false",
			status:   BuildStatusBuilding,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuildSuccessful(tt.status)
			if result != tt.expected {
				t.Errorf("IsBuildSuccessful() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipPlatform(t *testing.T) {
	tests := []struct {
		name          string
		platform      string
		wineInstalled bool
		expected      bool
	}{
		{
			name:          "windows without wine returns true",
			platform:      "win",
			wineInstalled: false,
			expected:      true,
		},
		{
			name:          "windows with wine returns false",
			platform:      "win",
			wineInstalled: true,
			expected:      false,
		},
		{
			name:          "linux without wine returns false",
			platform:      "linux",
			wineInstalled: false,
			expected:      false,
		},
		{
			name:          "mac without wine returns false",
			platform:      "mac",
			wineInstalled: false,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipPlatform(tt.platform, tt.wineInstalled)
			if result != tt.expected {
				t.Errorf("ShouldSkipPlatform() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCanRunStagesInParallel(t *testing.T) {
	// Currently all stages run sequentially
	tests := []struct {
		name     string
		stageA   string
		stageB   string
		expected bool
	}{
		{
			name:     "bundle and preflight cannot run in parallel",
			stageA:   StageBundle,
			stageB:   StagePreflight,
			expected: false,
		},
		{
			name:     "generate and build cannot run in parallel",
			stageA:   StageGenerate,
			stageB:   StageBuild,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanRunStagesInParallel(tt.stageA, tt.stageB)
			if result != tt.expected {
				t.Errorf("CanRunStagesInParallel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsThinClientMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected bool
	}{
		{
			name:     "external-server is thin client",
			mode:     DeploymentModeExternalServer,
			expected: true,
		},
		{
			name:     "cloud-api is thin client",
			mode:     DeploymentModeCloudAPI,
			expected: true,
		},
		{
			name:     "proxy is thin client",
			mode:     DeploymentModeProxy,
			expected: true,
		},
		{
			name:     "bundled is not thin client",
			mode:     DeploymentModeBundled,
			expected: false,
		},
		{
			name:     "empty is not thin client",
			mode:     "",
			expected: false,
		},
		{
			name:     "unknown mode is not thin client",
			mode:     "unknown",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsThinClientMode(tt.mode)
			if result != tt.expected {
				t.Errorf("IsThinClientMode(%q) = %v, want %v", tt.mode, result, tt.expected)
			}
		})
	}
}
