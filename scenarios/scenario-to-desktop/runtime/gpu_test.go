package bundleruntime

import (
	"context"
	"path/filepath"
	"testing"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
)

// Note: Unit tests for gpu.RealDetector have been moved to gpu/detector_test.go.
// This file contains integration tests for Supervisor GPU methods.

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
				gpuStatus:     GPUStatus{Available: tt.gpuAvail, Method: "test", Reason: "test"},
				telemetryPath: telemetryPath,
				fs:            RealFileSystem{},
				clock:         RealClock{},
				telemetry:     telemetry.NewFileRecorder(telemetryPath, RealClock{}, RealFileSystem{}),
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
		gpuStatus: GPUStatus{
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

func TestCommandRunnerOutput(t *testing.T) {
	runner := RealCommandRunner{}
	ctx := context.Background()

	t.Run("successful command", func(t *testing.T) {
		out, err := runner.Output(ctx, "echo", "hello")
		if err != nil {
			t.Errorf("Output() error = %v", err)
		}
		if len(out) == 0 {
			t.Error("Output() returned empty output")
		}
	})

	t.Run("command not found", func(t *testing.T) {
		_, err := runner.Output(ctx, "nonexistent-command-12345")
		if err == nil {
			t.Error("Output() expected error for non-existent command")
		}
	})
}
