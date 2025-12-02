package bundleruntime

import (
	"testing"
)

// Note: Unit tests for gpu.Applier have been moved to gpu/applier_test.go.
// Unit tests for gpu.RealDetector are in gpu/detector_test.go.
// This file contains integration tests for Supervisor GPU accessor methods.

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
