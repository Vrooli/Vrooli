package smoketest_test

import (
	"scenario-to-desktop-api/smoketest"
	"testing"
	"time"
)

func TestDefaultConfig_Values(t *testing.T) {
	config := smoketest.DefaultConfig()

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"TimeoutSeconds", config.TimeoutSeconds, 30},
		{"TelemetryPathMarker", config.TelemetryPathMarker, "[Desktop App] Telemetry initialized at "},
		{"SuccessMarker", config.SuccessMarker, "SMOKE_TEST_RESULT=passed"},
		{"UploadSuccessMarker", config.UploadSuccessMarker, "SMOKE_TEST_UPLOAD=ok"},
		{"UploadErrorMarker", config.UploadErrorMarker, "SMOKE_TEST_UPLOAD=error"},
		{"MaxTelemetryEvents", config.MaxTelemetryEvents, 500},
		{"XvfbCommand", config.XvfbCommand, "xvfb-run"},
		{"TelemetryFileName", config.TelemetryFileName, "deployment-telemetry.jsonl"},
		{"InitMarker", config.InitMarker, "SMOKE_TEST_INIT=started"},
		{"ReadyMarker", config.ReadyMarker, "SMOKE_TEST_READY=true"},
		{"ExitMarker", config.ExitMarker, "SMOKE_TEST_EXIT=clean"},
		{"MaxOutputBytes", config.MaxOutputBytes, 10 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestConfig_Timeout(t *testing.T) {
	tests := []struct {
		name           string
		timeoutSeconds int
		wantDuration   time.Duration
	}{
		{
			name:           "default 30 seconds",
			timeoutSeconds: 30,
			wantDuration:   30 * time.Second,
		},
		{
			name:           "60 seconds",
			timeoutSeconds: 60,
			wantDuration:   60 * time.Second,
		},
		{
			name:           "1 second",
			timeoutSeconds: 1,
			wantDuration:   1 * time.Second,
		},
		{
			name:           "zero seconds",
			timeoutSeconds: 0,
			wantDuration:   0,
		},
		{
			name:           "300 seconds (5 minutes)",
			timeoutSeconds: 300,
			wantDuration:   5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := smoketest.Config{
				TimeoutSeconds: tt.timeoutSeconds,
			}

			got := config.Timeout()
			if got != tt.wantDuration {
				t.Errorf("Timeout() = %v, want %v", got, tt.wantDuration)
			}
		})
	}
}

func TestConfig_TimeoutMS(t *testing.T) {
	tests := []struct {
		name           string
		timeoutSeconds int
		wantMS         int
	}{
		{
			name:           "default 30 seconds",
			timeoutSeconds: 30,
			wantMS:         30000,
		},
		{
			name:           "60 seconds",
			timeoutSeconds: 60,
			wantMS:         60000,
		},
		{
			name:           "1 second",
			timeoutSeconds: 1,
			wantMS:         1000,
		},
		{
			name:           "zero seconds",
			timeoutSeconds: 0,
			wantMS:         0,
		},
		{
			name:           "120 seconds",
			timeoutSeconds: 120,
			wantMS:         120000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := smoketest.Config{
				TimeoutSeconds: tt.timeoutSeconds,
			}

			got := config.TimeoutMS()
			if got != tt.wantMS {
				t.Errorf("TimeoutMS() = %d, want %d", got, tt.wantMS)
			}
		})
	}
}

func TestConfig_DefaultTimeout(t *testing.T) {
	config := smoketest.DefaultConfig()

	// Verify Timeout() returns correct duration for default config
	expectedDuration := 30 * time.Second
	if config.Timeout() != expectedDuration {
		t.Errorf("DefaultConfig Timeout() = %v, want %v", config.Timeout(), expectedDuration)
	}

	// Verify TimeoutMS() returns correct milliseconds for default config
	expectedMS := 30000
	if config.TimeoutMS() != expectedMS {
		t.Errorf("DefaultConfig TimeoutMS() = %d, want %d", config.TimeoutMS(), expectedMS)
	}
}

func TestConfig_TimeoutForDeploymentMode(t *testing.T) {
	tests := []struct {
		name           string
		config         smoketest.Config
		deploymentMode string
		wantDuration   time.Duration
	}{
		{
			name: "bundled mode uses BundledModeTimeoutSeconds",
			config: smoketest.Config{
				TimeoutSeconds:            30,
				BundledModeTimeoutSeconds: 60,
			},
			deploymentMode: "bundled",
			wantDuration:   60 * time.Second,
		},
		{
			name: "external-server mode uses ExternalServerModeTimeoutSeconds",
			config: smoketest.Config{
				TimeoutSeconds:                   30,
				ExternalServerModeTimeoutSeconds: 20,
			},
			deploymentMode: "external-server",
			wantDuration:   20 * time.Second,
		},
		{
			name: "cloud-api mode uses ExternalServerModeTimeoutSeconds",
			config: smoketest.Config{
				TimeoutSeconds:                   30,
				ExternalServerModeTimeoutSeconds: 25,
			},
			deploymentMode: "cloud-api",
			wantDuration:   25 * time.Second,
		},
		{
			name: "unknown mode falls back to default TimeoutSeconds",
			config: smoketest.Config{
				TimeoutSeconds:            45,
				BundledModeTimeoutSeconds: 60,
			},
			deploymentMode: "unknown",
			wantDuration:   45 * time.Second,
		},
		{
			name: "bundled mode falls back to default when BundledModeTimeoutSeconds is 0",
			config: smoketest.Config{
				TimeoutSeconds:            30,
				BundledModeTimeoutSeconds: 0,
			},
			deploymentMode: "bundled",
			wantDuration:   30 * time.Second,
		},
		{
			name: "external-server falls back to default when ExternalServerModeTimeoutSeconds is 0",
			config: smoketest.Config{
				TimeoutSeconds:                   30,
				ExternalServerModeTimeoutSeconds: 0,
			},
			deploymentMode: "external-server",
			wantDuration:   30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.TimeoutForDeploymentMode(tt.deploymentMode)
			if got != tt.wantDuration {
				t.Errorf("TimeoutForDeploymentMode(%q) = %v, want %v", tt.deploymentMode, got, tt.wantDuration)
			}
		})
	}
}

func TestConfig_TimeoutMSForDeploymentMode(t *testing.T) {
	config := smoketest.Config{
		TimeoutSeconds:                   30,
		BundledModeTimeoutSeconds:        60,
		ExternalServerModeTimeoutSeconds: 20,
	}

	tests := []struct {
		deploymentMode string
		wantMS         int
	}{
		{"bundled", 60000},
		{"external-server", 20000},
		{"cloud-api", 20000},
		{"unknown", 30000},
	}

	for _, tt := range tests {
		t.Run(tt.deploymentMode, func(t *testing.T) {
			got := config.TimeoutMSForDeploymentMode(tt.deploymentMode)
			if got != tt.wantMS {
				t.Errorf("TimeoutMSForDeploymentMode(%q) = %d, want %d", tt.deploymentMode, got, tt.wantMS)
			}
		})
	}
}

func TestDefaultConfig_DeploymentModeTimeouts(t *testing.T) {
	config := smoketest.DefaultConfig()

	// Verify bundled mode gets longer timeout
	bundledTimeout := config.TimeoutForDeploymentMode("bundled")
	if bundledTimeout != 60*time.Second {
		t.Errorf("Bundled mode timeout = %v, want 60s", bundledTimeout)
	}

	// Verify external-server mode uses default
	externalTimeout := config.TimeoutForDeploymentMode("external-server")
	if externalTimeout != 30*time.Second {
		t.Errorf("External-server mode timeout = %v, want 30s", externalTimeout)
	}
}

func TestDefaultConfig_GranularLifecycleMarkers(t *testing.T) {
	config := smoketest.DefaultConfig()
	markers := config.GranularLifecycleMarkers

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"BundleResolving", markers.BundleResolving, "SMOKE_TEST_STAGE=bundle_resolving"},
		{"RuntimeStarting", markers.RuntimeStarting, "SMOKE_TEST_STAGE=runtime_starting"},
		{"RuntimeHealthz", markers.RuntimeHealthz, "SMOKE_TEST_STAGE=runtime_healthz"},
		{"RuntimeReadyz", markers.RuntimeReadyz, "SMOKE_TEST_STAGE=runtime_readyz"},
		{"RuntimePorts", markers.RuntimePorts, "SMOKE_TEST_STAGE=runtime_ports"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}
