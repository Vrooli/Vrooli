package accel_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Feature: readiness picks a backend from what the host actually reports
//
//	As a resource start path
//	I want the first declared backend the host can reach
//	So that a GPU-preferring resource still starts on a CPU-only host, and a
//	GPU-only resource fails loudly instead of serving from the wrong device.

// hostWithCUDA is a snapshot from a host with a working NVIDIA device.
func hostWithCUDA() hostinventory.Snapshot {
	return hostinventory.Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools:  map[string]hostinventory.Tool{hostinventory.ToolNvidiaSMI: {Present: true}},
		ProbeStatuses: map[string]string{"nvidia_gpu": "ok"},
		GPUs: []hostinventory.GPU{
			{Index: 0, Name: "NVIDIA RTX", CUDAComputeCapability: "8.9", VRAMBytes: 16 << 30, Source: hostinventory.SourceNvidiaSMI},
		},
	}
}

// hostWithNoAccelerator is the snapshot every test in CI runs against.
func hostWithNoAccelerator() hostinventory.Snapshot {
	return hostinventory.Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools:  map[string]hostinventory.Tool{},
		ProbeStatuses: map[string]string{"nvidia_gpu": "not_present", "rocm": "no_devices", "vulkan": "no_devices"},
	}
}

// Scenario: a preferred resource falls back to the CPU floor without an error.
func TestReadinessFallsBackToCPUForAPreferredResource(t *testing.T) {
	// Given a resource that prefers CUDA and declares a CPU floor
	spec := accel.Spec{
		Resource: "ollama",
		Backends: []accel.Backend{accel.BackendCUDA, accel.BackendCPU},
		Require:  accel.RequirePreferred,
	}
	// And a host with no accelerator at all
	source := accel.StaticFactSource{Inventory: hostWithNoAccelerator()}

	// When readiness is evaluated
	result, err := accel.Readiness(context.Background(), source, spec)
	// Then it selects the CPU floor and does not fail
	if err != nil {
		t.Fatalf("Readiness() = %v, want nil; falling back is a state, not a failure", err)
	}
	if result.Selected != accel.BackendCPU {
		t.Fatalf("Selected = %q, want %q", result.Selected, accel.BackendCPU)
	}
	// And it reports the drift, so the fallback is never silent
	if !result.Drift {
		t.Fatal("Drift = false, want true; the resource is running below its declared backend")
	}
	if result.Declared != accel.BackendCUDA {
		t.Fatalf("Declared = %q, want %q", result.Declared, accel.BackendCUDA)
	}
	// And it says why CUDA was skipped, in the host's own words
	if len(result.Considered) == 0 || !strings.Contains(result.Considered[0].Reason, "not installed") {
		t.Fatalf("Considered = %+v, want the cuda verdict to name what the host reported", result.Considered)
	}
}

// Scenario: a required resource fails loudly with a repair command.
func TestReadinessRefusesARequiredResourceWithNoBackend(t *testing.T) {
	// Given a resource that requires CUDA
	spec := accel.Spec{
		Resource: "kyutai-stt",
		Backends: []accel.Backend{accel.BackendCUDA, accel.BackendCPU},
		Require:  accel.RequireRequired,
	}
	// And a host with no accelerator
	source := accel.StaticFactSource{Inventory: hostWithNoAccelerator()}

	// When readiness is evaluated
	_, err := accel.Readiness(context.Background(), source, spec)

	// Then it fails with the typed error
	if !errors.Is(err, accel.ErrNoBackendReady) {
		t.Fatalf("Readiness() = %v, want ErrNoBackendReady", err)
	}
	// And the message names the resource and a non-empty repair command
	var typed *accel.NoBackendReadyError
	if !errors.As(err, &typed) {
		t.Fatalf("Readiness() error = %T, want *accel.NoBackendReadyError", err)
	}
	if typed.Remediation == "" {
		t.Fatal("Remediation is empty; an operator must be told what to run")
	}
	if !strings.Contains(err.Error(), "kyutai-stt") || !strings.Contains(err.Error(), typed.Remediation) {
		t.Fatalf("error = %q, want it to name the resource and the repair command", err)
	}
}

// Scenario: an opportunistic resource never reports drift.
func TestReadinessTreatsFallbackAsNormalWhenRequireIsNone(t *testing.T) {
	// Given a resource whose accelerator use is opportunistic
	spec := accel.Spec{
		Resource: "sherpa-onnx",
		Backends: []accel.Backend{accel.BackendCUDA, accel.BackendCPU},
		Require:  accel.RequireNone,
	}
	source := accel.StaticFactSource{Inventory: hostWithNoAccelerator()}

	// When readiness is evaluated
	result, err := accel.Readiness(context.Background(), source, spec)
	// Then the CPU is selected and falling back is not drift
	if err != nil {
		t.Fatalf("Readiness() = %v, want nil", err)
	}
	if result.Selected != accel.BackendCPU || result.Drift {
		t.Fatalf("Selected = %q Drift = %v, want cpu and no drift", result.Selected, result.Drift)
	}
}

// Scenario: the first reachable declared backend wins, in declared order.
func TestReadinessSelectsTheFirstReachableDeclaredBackend(t *testing.T) {
	cases := []struct {
		scenario string
		declared []accel.Backend
		want     accel.Backend
		wantDrif bool
	}{
		{
			scenario: "Given cuda is declared first and reachable, Then cuda is selected",
			declared: []accel.Backend{accel.BackendCUDA, accel.BackendCPU},
			want:     accel.BackendCUDA,
		},
		{
			scenario: "Given rocm is declared first but unreachable, Then cuda is selected and drift is reported",
			declared: []accel.Backend{accel.BackendROCm, accel.BackendCUDA, accel.BackendCPU},
			want:     accel.BackendCUDA,
			wantDrif: true,
		},
		{
			scenario: "Given only cpu is declared, Then cpu is selected with no drift",
			declared: []accel.Backend{accel.BackendCPU},
			want:     accel.BackendCPU,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a host that can reach CUDA
			source := accel.StaticFactSource{Inventory: hostWithCUDA()}
			spec := accel.Spec{Resource: "whisper", Backends: tc.declared}

			// When readiness is evaluated
			result, err := accel.Readiness(context.Background(), source, spec)
			// Then the first reachable declared backend is selected
			if err != nil {
				t.Fatalf("Readiness() = %v, want nil", err)
			}
			if result.Selected != tc.want {
				t.Fatalf("Selected = %q, want %q", result.Selected, tc.want)
			}
			if result.Drift != tc.wantDrif {
				t.Fatalf("Drift = %v, want %v", result.Drift, tc.wantDrif)
			}
			// And every declared backend has a verdict, so nothing is skipped silently
			if len(result.Considered) < len(tc.declared) {
				t.Fatalf("Considered = %d verdicts, want at least %d", len(result.Considered), len(tc.declared))
			}
		})
	}
}

// Scenario: a fact-source failure is an error, never a silent CPU fallback.
func TestReadinessSurfacesAFactSourceFailure(t *testing.T) {
	// Given a host whose inventory cannot be read
	source := accel.StaticFactSource{Err: errors.New("collector timed out")}
	spec := accel.Spec{Resource: "reranker", Backends: []accel.Backend{accel.BackendCUDA, accel.BackendCPU}}

	// When readiness is evaluated
	_, err := accel.Readiness(context.Background(), source, spec)

	// Then the failure surfaces rather than being read as "no accelerator"
	if err == nil || !strings.Contains(err.Error(), "collector timed out") {
		t.Fatalf("Readiness() = %v, want the underlying collector failure", err)
	}
}

// Scenario: the unreachable reason names the platform, not a generic message.
func TestReadinessExplainsUnreachableBackendsPerPlatform(t *testing.T) {
	cases := []struct {
		scenario   string
		snapshot   hostinventory.Snapshot
		backend    accel.Backend
		wantReason string
	}{
		{
			scenario:   "Given metal on linux, Then the reason names the platform",
			snapshot:   hostWithNoAccelerator(),
			backend:    accel.BackendMetal,
			wantReason: "only reachable on darwin",
		},
		{
			scenario: "Given rocm on darwin, Then the reason says the interface is Linux-only",
			snapshot: hostinventory.Snapshot{
				OS: "darwin", Arch: "arm64",
				RuntimeTools:  map[string]hostinventory.Tool{},
				ProbeStatuses: map[string]string{"rocm": "unsupported"},
			},
			backend:    accel.BackendROCm,
			wantReason: "Linux-only",
		},
		{
			scenario:   "Given vulkan with no ICD manifest, Then the reason names the manifest",
			snapshot:   hostWithNoAccelerator(),
			backend:    accel.BackendVulkan,
			wantReason: "installable client driver manifest",
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a resource declaring that backend with a cpu floor
			spec := accel.Spec{Resource: "fixture", Backends: []accel.Backend{tc.backend, accel.BackendCPU}}

			// When readiness is evaluated
			result, err := accel.ReadinessFromSnapshot(tc.snapshot, spec)
			if err != nil {
				t.Fatalf("ReadinessFromSnapshot() = %v, want nil", err)
			}

			// Then the verdict explains the platform reality
			if len(result.Considered) == 0 || !strings.Contains(result.Considered[0].Reason, tc.wantReason) {
				t.Fatalf("reason = %q, want it to contain %q", result.Considered[0].Reason, tc.wantReason)
			}
		})
	}
}

// Scenario: the backend vocabulary is closed at the parse boundary.
func TestParseBackendRejectsAnythingOutsideTheClosedSet(t *testing.T) {
	// Given a declared backend list containing a typo
	// When it is parsed
	_, err := accel.ParseBackends([]string{"cuda", "metall"})

	// Then the whole list is rejected and the message names the closed set
	if err == nil || !strings.Contains(err.Error(), "metall") {
		t.Fatalf("ParseBackends() = %v, want a rejection naming metall", err)
	}

	// And a valid list parses in declared order
	parsed, err := accel.ParseBackends([]string{"cuda", "cpu"})
	if err != nil {
		t.Fatalf("ParseBackends() = %v, want nil", err)
	}
	if len(parsed) != 2 || parsed[0] != accel.BackendCUDA || parsed[1] != accel.BackendCPU {
		t.Fatalf("ParseBackends() = %v, want [cuda cpu]", parsed)
	}
}
