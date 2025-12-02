package gpu

import (
	"testing"

	"scenario-to-desktop-runtime/infra"
)

func TestRealDetector_EnvOverride(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		wantAvail  bool
		wantMethod string
	}{
		{"true enables GPU", "true", true, "env_override"},
		{"1 enables GPU", "1", true, "env_override"},
		{"yes enables GPU", "yes", true, "env_override"},
		{"on enables GPU", "on", true, "env_override"},
		{"false disables GPU", "false", false, "env_override"},
		{"0 disables GPU", "0", false, "env_override"},
		{"no disables GPU", "no", false, "env_override"},
		{"off disables GPU", "off", false, "env_override"},
		{"TRUE case insensitive", "TRUE", true, "env_override"},
		{"False case insensitive", "False", false, "env_override"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BUNDLE_GPU_AVAILABLE", tt.envValue)

			detector := NewDetector(infra.RealCommandRunner{}, infra.RealEnvReader{})
			status := detector.Detect()
			if status.Available != tt.wantAvail {
				t.Errorf("RealDetector.Detect().Available = %v, want %v", status.Available, tt.wantAvail)
			}
			if status.Method != tt.wantMethod {
				t.Errorf("RealDetector.Detect().Method = %q, want %q", status.Method, tt.wantMethod)
			}
		})
	}
}

func TestRealDetector_NoOverride(t *testing.T) {
	t.Setenv("BUNDLE_GPU_AVAILABLE", "")

	detector := NewDetector(infra.RealCommandRunner{}, infra.RealEnvReader{})
	status := detector.Detect()

	// We can't predict the result, but we can verify the structure is valid
	if status.Method == "" {
		t.Error("RealDetector.Detect() returned empty method")
	}
	if status.Reason == "" {
		t.Error("RealDetector.Detect() returned empty reason")
	}
}
