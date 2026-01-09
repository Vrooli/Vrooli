package gpu

import (
	"testing"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
)

func TestApplier_Apply(t *testing.T) {
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
			status := Status{Available: tt.gpuAvail, Method: "test", Reason: "test"}
			applier := NewApplier(status, telemetry.NopRecorder{})

			var gpu *manifest.GPURequirements
			if tt.requirement != "" {
				gpu = &manifest.GPURequirements{Requirement: tt.requirement}
			}

			svc := manifest.Service{ID: "test-svc", GPU: gpu}
			env := map[string]string{}

			err := applier.Apply(env, svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// BUNDLE_GPU_AVAILABLE should always be set
			if env["BUNDLE_GPU_AVAILABLE"] != BoolToString(tt.gpuAvail) {
				t.Errorf("BUNDLE_GPU_AVAILABLE = %q, want %q", env["BUNDLE_GPU_AVAILABLE"], BoolToString(tt.gpuAvail))
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
		got := BoolToString(tt.input)
		if got != tt.want {
			t.Errorf("BoolToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
