package accel_test

import (
	"testing"

	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Feature: the same selection code chooses a backend on every platform
//
//	As the cross-platform programme
//	I want an Apple Silicon or AMD host to select and report a backend through
//	the same path a CUDA host uses
//	So that "accelerated" stops meaning "NVIDIA on Linux".
//
// These are SELECTION tests over recorded facts. They are not placement proof:
// no live macOS or Windows host was available, and live placement verification
// on those platforms is recorded as unknown.

// appleSiliconHost is the fact set an Apple Silicon machine publishes.
func appleSiliconHost() hostinventory.Snapshot {
	return hostinventory.Snapshot{
		OS: "darwin", Arch: "arm64",
		RuntimeTools:  map[string]hostinventory.Tool{hostinventory.ToolSystemProfiler: {Present: true}},
		ProbeStatuses: map[string]string{"nvidia_gpu": "not_present", "rocm": "unsupported", "vulkan": "no_devices"},
		GPUs:          []hostinventory.GPU{{Index: 0, Name: "Apple M3 Pro", VRAMBytes: 12 << 30, Source: hostinventory.SourceSystemProfiler}},
	}
}

// amdROCmHost is the fact set an AMD machine with a loaded compute driver
// publishes.
func amdROCmHost() hostinventory.Snapshot {
	return hostinventory.Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools:    map[string]hostinventory.Tool{hostinventory.ToolROCmSMI: {Present: true}},
		ProbeStatuses:   map[string]string{"nvidia_gpu": "not_present", "rocm": "ok"},
		Devices:         []hostinventory.Device{{ID: "0000:03:00.0", Class: hostinventory.DeviceClassGraphics, Vendor: "Advanced Micro Devices, Inc. [AMD/ATI]"}},
		ROCmDeviceNodes: []string{"/dev/kfd"},
		GPUs:            []hostinventory.GPU{{Index: 0, Name: "Radeon RX 7900 XTX", VRAMBytes: 24 << 30, Source: hostinventory.SourceROCmSMI}},
	}
}

// Scenario: the same declaration selects a different backend per platform.
func TestBackendSelectionAcrossPlatforms(t *testing.T) {
	// Given a resource that declares cuda, metal and a cpu floor — ollama's
	// declaration after the fleet migration
	spec := accel.Spec{
		Resource: "ollama",
		Backends: []accel.Backend{accel.BackendCUDA, accel.BackendMetal, accel.BackendCPU},
		Require:  accel.RequirePreferred,
	}

	cases := []struct {
		scenario     string
		snapshot     hostinventory.Snapshot
		wantSelected accel.Backend
		wantDrift    bool
	}{
		{
			scenario:     "Given a CUDA host, Then cuda is selected, unchanged from before the cross-platform work",
			snapshot:     hostWithCUDA(),
			wantSelected: accel.BackendCUDA,
		},
		{
			scenario:     "Given an Apple Silicon host, Then metal is selected through the same code path",
			snapshot:     appleSiliconHost(),
			wantSelected: accel.BackendMetal,
			wantDrift:    true,
		},
		{
			scenario:     "Given a host with neither, Then it falls back to the cpu floor and says so",
			snapshot:     hostWithNoAccelerator(),
			wantSelected: accel.BackendCPU,
			wantDrift:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When readiness is evaluated over those facts
			result, err := accel.ReadinessFromSnapshot(tc.snapshot, spec)
			if err != nil {
				t.Fatalf("ReadinessFromSnapshot() = %v, want nil", err)
			}

			// Then the platform's own backend is selected
			if result.Selected != tc.wantSelected {
				t.Fatalf("Selected = %q, want %q (verdicts: %+v)", result.Selected, tc.wantSelected, result.Considered)
			}
			// And falling below the first choice is reported rather than hidden
			if result.Drift != tc.wantDrift {
				t.Fatalf("Drift = %v, want %v", result.Drift, tc.wantDrift)
			}
			// And every declared backend has a verdict with a reason
			for _, verdict := range result.Considered {
				if verdict.Reason == "" {
					t.Fatalf("backend %q has no verdict reason", verdict.Backend)
				}
			}
		})
	}
}

// Scenario: an AMD host reaches rocm, and a resource declaring it selects it.
func TestROCmHostSelectsROCm(t *testing.T) {
	// Given a resource declaring rocm with a cpu floor
	spec := accel.Spec{
		Resource: "fixture",
		Backends: []accel.Backend{accel.BackendROCm, accel.BackendCPU},
		Require:  accel.RequirePreferred,
	}

	// When readiness is evaluated over an AMD host's facts
	result, err := accel.ReadinessFromSnapshot(amdROCmHost(), spec)
	if err != nil {
		t.Fatalf("ReadinessFromSnapshot() = %v, want nil", err)
	}

	// Then rocm is selected with no drift
	if result.Selected != accel.BackendROCm || result.Drift {
		t.Fatalf("Selected = %q Drift = %v, want rocm with no drift", result.Selected, result.Drift)
	}

	// And a CUDA-only resource on that same host falls back and says why
	cudaOnly := accel.Spec{Resource: "fixture", Backends: []accel.Backend{accel.BackendCUDA, accel.BackendCPU}}
	fallback, err := accel.ReadinessFromSnapshot(amdROCmHost(), cudaOnly)
	if err != nil {
		t.Fatalf("ReadinessFromSnapshot() = %v, want nil", err)
	}
	if fallback.Selected != accel.BackendCPU || !fallback.Drift {
		t.Fatalf("Selected = %q Drift = %v, want the cpu floor with drift", fallback.Selected, fallback.Drift)
	}
}

// Scenario: the host publishes a single-valued backend fact for predicates.
//
// accel.backends is a comma-joined list, which an acquisition predicate cannot
// match against. accel.backend names the one a target selects on.
func TestHostPublishesASelectableBackendFact(t *testing.T) {
	cases := []struct {
		scenario string
		snapshot hostinventory.Snapshot
		want     string
	}{
		{scenario: "Given a CUDA host, Then accel.backend is cuda", snapshot: hostWithCUDA(), want: "cuda"},
		{scenario: "Given an Apple Silicon host, Then accel.backend is metal", snapshot: appleSiliconHost(), want: "metal"},
		{scenario: "Given an AMD host, Then accel.backend is rocm", snapshot: amdROCmHost(), want: "rocm"},
		{scenario: "Given a host with no accelerator, Then accel.backend is cpu", snapshot: hostWithNoAccelerator(), want: "cpu"},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When the facts are projected
			got := tc.snapshot.AcceleratorFacts()[hostinventory.FactAccelBackend]

			// Then a target can select on exactly one value
			if got != tc.want {
				t.Fatalf("accel.backend = %q, want %q", got, tc.want)
			}
		})
	}
}
