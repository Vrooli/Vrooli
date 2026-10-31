package gpu

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
)

// mockCommandRunner is a test double for infra.CommandRunner.
type mockCommandRunner struct {
	lookPathFunc func(cmd string) (string, error)
	outputFunc   func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func (m *mockCommandRunner) LookPath(cmd string) (string, error) {
	if m.lookPathFunc != nil {
		return m.lookPathFunc(cmd)
	}
	return "", errors.New("not found")
}

func (m *mockCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.outputFunc != nil {
		return m.outputFunc(ctx, name, args...)
	}
	return nil, errors.New("command failed")
}

func (m *mockCommandRunner) Run(_ context.Context, _ string, _ []string) error {
	return nil
}

// mockEnvReader is a test double for infra.EnvReader.
type mockEnvReader struct {
	env map[string]string
}

func (m *mockEnvReader) Getenv(key string) string {
	if m.env != nil {
		return m.env[key]
	}
	return ""
}

func (m *mockEnvReader) Environ() []string {
	out := make([]string, 0, len(m.env))
	for k, v := range m.env {
		out = append(out, k+"="+v)
	}
	return out
}

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

func TestRealDetector_NvidiaSmi(t *testing.T) {
	tests := []struct {
		name       string
		lookPath   func(string) (string, error)
		output     func(context.Context, string, ...string) ([]byte, error)
		wantAvail  bool
		wantMethod string
	}{
		{
			name: "nvidia-smi found and returns GPU",
			lookPath: func(cmd string) (string, error) {
				if cmd == "nvidia-smi" {
					return "/usr/bin/nvidia-smi", nil
				}
				return "", errors.New("not found")
			},
			output: func(_ context.Context, name string, args ...string) ([]byte, error) {
				if name == "nvidia-smi" {
					return []byte("0, NVIDIA GeForce RTX 3080, GPU-1, 555.42, 0, 0, 10240, 0, 40, N/A, 50, 100, 1000, 5000\n"), nil
				}
				return nil, errors.New("command failed")
			},
			wantAvail:  true,
			wantMethod: "nvidia-smi",
		},
		{
			name: "nvidia-smi not found falls back to probe",
			lookPath: func(_ string) (string, error) {
				return "", errors.New("not found")
			},
			wantAvail:  false,
			wantMethod: "probe",
		},
		{
			name: "nvidia-smi returns empty output",
			lookPath: func(cmd string) (string, error) {
				if cmd == "nvidia-smi" {
					return "/usr/bin/nvidia-smi", nil
				}
				return "", errors.New("not found")
			},
			output: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte("   \n"), nil // Only whitespace
			},
			wantAvail:  false,
			wantMethod: "probe",
		},
		{
			name: "nvidia-smi command fails",
			lookPath: func(cmd string) (string, error) {
				if cmd == "nvidia-smi" {
					return "/usr/bin/nvidia-smi", nil
				}
				return "", errors.New("not found")
			},
			output: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, errors.New("nvidia-smi failed")
			},
			wantAvail:  false,
			wantMethod: "probe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdRunner := &mockCommandRunner{
				lookPathFunc: tt.lookPath,
				outputFunc:   tt.output,
			}
			envReader := &mockEnvReader{env: map[string]string{}}

			detector := NewDetector(cmdRunner, envReader)
			status := detector.Detect()

			if status.Available != tt.wantAvail {
				t.Errorf("Detect().Available = %v, want %v", status.Available, tt.wantAvail)
			}
			if status.Method != tt.wantMethod {
				t.Errorf("Detect().Method = %q, want %q", status.Method, tt.wantMethod)
			}
		})
	}
}

func TestStatusFromSnapshot(t *testing.T) {
	tests := []struct {
		name       string
		snapshot   hostinventory.Snapshot
		wantAvail  bool
		wantMethod string
		wantReason string
	}{
		{
			name:       "no GPU",
			snapshot:   hostinventory.Snapshot{},
			wantAvail:  false,
			wantMethod: "probe",
			wantReason: "no GPU detected",
		},
		{
			name: "NVIDIA GPU",
			snapshot: hostinventory.Snapshot{
				GPUs: []hostinventory.GPU{{Name: "NVIDIA GeForce RTX 3080", Source: "nvidia-smi"}},
			},
			wantAvail:  true,
			wantMethod: "nvidia-smi",
			wantReason: "nvidia gpu detected",
		},
		{
			name: "Darwin GPU",
			snapshot: hostinventory.Snapshot{
				GPUs: []hostinventory.GPU{{Name: "Apple M3 Pro", Source: "system_profiler"}},
			},
			wantAvail:  true,
			wantMethod: "system_profiler",
			wantReason: "GPU reported by system_profiler",
		},
		{
			name: "Windows GPU",
			snapshot: hostinventory.Snapshot{
				GPUs: []hostinventory.GPU{{Name: "AMD Radeon RX 6800", Source: "wmic"}},
			},
			wantAvail:  true,
			wantMethod: "wmic",
			wantReason: "GPU reported by wmic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := statusFromSnapshot(tt.snapshot)

			if status.Available != tt.wantAvail {
				t.Errorf("statusFromSnapshot().Available = %v, want %v", status.Available, tt.wantAvail)
			}
			if status.Method != tt.wantMethod {
				t.Errorf("statusFromSnapshot().Method = %q, want %q", status.Method, tt.wantMethod)
			}
			if status.Reason != tt.wantReason {
				t.Errorf("statusFromSnapshot().Reason = %q, want %q", status.Reason, tt.wantReason)
			}
		})
	}
}
