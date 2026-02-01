package smoketest_test

import (
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
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
