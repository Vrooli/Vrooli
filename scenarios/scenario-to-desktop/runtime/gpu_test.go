package bundleruntime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

func TestDetectGPU_EnvOverride(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		wantAvail bool
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

			status := detectGPU()
			if status.Available != tt.wantAvail {
				t.Errorf("detectGPU().Available = %v, want %v", status.Available, tt.wantAvail)
			}
			if status.Method != tt.wantMethod {
				t.Errorf("detectGPU().Method = %q, want %q", status.Method, tt.wantMethod)
			}
		})
	}
}

func TestApplyGPURequirement(t *testing.T) {
	tmp := t.TempDir()
	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")

	tests := []struct {
		name        string
		gpuAvail    bool
		requirement string
		wantErr     bool
		wantMode    string
	}{
		{"no requirement with gpu", true, "", false, ""},
		{"no requirement without gpu", false, "", false, ""},
		{"required with gpu", true, "required", false, "gpu"},
		{"required without gpu", false, "required", true, ""},
		{"optional_with_cpu_fallback with gpu", true, "optional_with_cpu_fallback", false, "gpu"},
		{"optional_with_cpu_fallback without gpu", false, "optional_with_cpu_fallback", false, "cpu"},
		{"optional_but_warn with gpu", true, "optional_but_warn", false, "gpu"},
		{"optional_but_warn without gpu", false, "optional_but_warn", false, "cpu"},
		{"unknown requirement", true, "unknown", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Supervisor{
				gpuStatus:     gpuStatus{Available: tt.gpuAvail, Method: "test", Reason: "test"},
				telemetryPath: telemetryPath,
			}

			var gpu *manifest.GPURequirements
			if tt.requirement != "" {
				gpu = &manifest.GPURequirements{Requirement: tt.requirement}
			}

			svc := manifest.Service{ID: "test-svc", GPU: gpu}
			env := map[string]string{}

			err := s.applyGPURequirement(env, svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("applyGPURequirement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// BUNDLE_GPU_AVAILABLE should always be set
			if env["BUNDLE_GPU_AVAILABLE"] != boolToString(tt.gpuAvail) {
				t.Errorf("BUNDLE_GPU_AVAILABLE = %q, want %q", env["BUNDLE_GPU_AVAILABLE"], boolToString(tt.gpuAvail))
			}

			// BUNDLE_GPU_MODE should be set when there's a requirement and no error
			if !tt.wantErr && tt.wantMode != "" {
				if env["BUNDLE_GPU_MODE"] != tt.wantMode {
					t.Errorf("BUNDLE_GPU_MODE = %q, want %q", env["BUNDLE_GPU_MODE"], tt.wantMode)
				}
			}
		})
	}
}

func TestBoolToString(t *testing.T) {
	tests := []struct {
		input bool
		want  string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		got := boolToString(tt.input)
		if got != tt.want {
			t.Errorf("boolToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGPUStatus(t *testing.T) {
	s := &Supervisor{
		gpuStatus: gpuStatus{
			Available: true,
			Method:    "nvidia-smi",
			Reason:    "nvidia gpu detected",
		},
	}

	got := s.GPUStatus()
	if got.Available != true {
		t.Errorf("GPUStatus().Available = %v, want true", got.Available)
	}
	if got.Method != "nvidia-smi" {
		t.Errorf("GPUStatus().Method = %q, want %q", got.Method, "nvidia-smi")
	}
	if got.Reason != "nvidia gpu detected" {
		t.Errorf("GPUStatus().Reason = %q, want %q", got.Reason, "nvidia gpu detected")
	}
}

func TestRunCommandWithTimeout(t *testing.T) {
	// Test with a simple command that should succeed
	t.Run("successful command", func(t *testing.T) {
		out, err := runCommandWithTimeout(5*time.Second, "echo", "hello")
		if err != nil {
			t.Errorf("runCommandWithTimeout() error = %v", err)
		}
		if len(out) == 0 {
			t.Error("runCommandWithTimeout() returned empty output")
		}
	})

	// Test with a non-existent command
	t.Run("command not found", func(t *testing.T) {
		_, err := runCommandWithTimeout(1*time.Second, "nonexistent-command-12345")
		if err == nil {
			t.Error("runCommandWithTimeout() expected error for non-existent command")
		}
	})
}

func TestDetectGPU_NoOverride(t *testing.T) {
	// Ensure no override is set
	os.Unsetenv("BUNDLE_GPU_AVAILABLE")

	status := detectGPU()

	// We can't predict the result, but we can verify the structure is valid
	if status.Method == "" {
		t.Error("detectGPU() returned empty method")
	}
	if status.Reason == "" {
		t.Error("detectGPU() returned empty reason")
	}
}
