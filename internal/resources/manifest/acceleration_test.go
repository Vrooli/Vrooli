package manifest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/capacity"
)

// Feature: one accelerator declaration
//
//	As the control plane
//	I want a resource to declare its accelerator backends exactly once
//	So that "does this resource use the GPU" has one answer instead of two
//	contradictory ones.

func vramClaim() *capacity.ResourceClaimSpec {
	return &capacity.ResourceClaimSpec{
		ResourceKind:   capacity.ResourceKindVRAM,
		PreferredBytes: 4 << 30,
		FloorBytes:     1 << 30,
		Priority:       "service",
		YieldWhenIdle:  true,
		Profile: &capacity.DegradeProfile{
			Steps: []capacity.DegradeStep{
				{Label: "full", AmountBytes: 4 << 30},
				{Label: "reduced", AmountBytes: 1 << 30},
			},
			Apply: capacity.DegradeApply{Verb: "capacity", Argv: []string{"degrade", "--to", "{label}"}},
		},
	}
}

// Scenario: a well-formed acceleration block is accepted.
func TestAccelerationValidateAcceptsWellFormedBlock(t *testing.T) {
	// Given a resource that prefers CUDA and falls back to CPU
	spec := AccelerationSpec{
		Backends: []string{BackendCUDA, BackendCPU},
		Require:  RequirePreferred,
		Backend: map[string]BackendConfig{
			BackendCUDA: {MinCompute: "8.9", Env: map[string]string{"DEVICE": "cuda"}},
			BackendCPU:  {Env: map[string]string{"DEVICE": "cpu"}},
		},
		Claim: vramClaim(),
	}

	// When the contract validates it
	err := spec.Validate()
	// Then it is accepted
	if err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	// And the resource reports that it declares acceleration
	if !spec.DeclaresAcceleration() {
		t.Fatal("DeclaresAcceleration() = false, want true")
	}
	// And an absent require resolves to preferred
	spec.Require = ""
	if got := spec.EffectiveRequire(); got != RequirePreferred {
		t.Fatalf("EffectiveRequire() = %q, want %q", got, RequirePreferred)
	}
}

// Scenario: the backend vocabulary is closed.
func TestAccelerationValidateRejectsContractViolations(t *testing.T) {
	cases := []accelerationCase{
		{
			scenario: "Given backends names a value outside the closed set, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{"tpu"},
				Backend:  map[string]BackendConfig{"tpu": {}},
			},
			wantErr: `entry "tpu" is invalid`,
		},
		{
			scenario: "Given backends is empty, Then it is rejected",
			spec:     AccelerationSpec{Backend: map[string]BackendConfig{BackendCPU: {}}},
			wantErr:  "must list at least one backend",
		},
		{
			scenario: "Given a named backend has no config block, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCUDA, BackendCPU},
				Backend:  map[string]BackendConfig{BackendCPU: {}},
			},
			wantErr: "there is no acceleration.cuda config block",
		},
		{
			scenario: "Given a config block names a backend absent from backends, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCPU},
				Backend:  map[string]BackendConfig{BackendCPU: {}, BackendROCm: {}},
			},
			wantErr: "is configured but",
		},
		{
			scenario: "Given backends repeats a backend, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCUDA, BackendCUDA},
				Backend:  map[string]BackendConfig{BackendCUDA: {}},
			},
			wantErr: "more than once",
		},
		{
			scenario: "Given require is outside the closed set, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCPU},
				Require:  "mandatory",
				Backend:  map[string]BackendConfig{BackendCPU: {}},
			},
			wantErr: `require "mandatory" is invalid`,
		},
		{
			scenario: "Given require is required but only CPU is declared, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCPU},
				Require:  RequireRequired,
				Backend:  map[string]BackendConfig{BackendCPU: {}},
			},
			wantErr: "names no backend other than",
		},
		{
			scenario: "Given min_compute is declared on a non-CUDA backend, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendMetal},
				Backend:  map[string]BackendConfig{BackendMetal: {MinCompute: "8.9"}},
			},
			wantErr: "only meaningful for backend",
		},
		{
			scenario: "Given min_compute is not a positive decimal, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCUDA},
				Backend:  map[string]BackendConfig{BackendCUDA: {MinCompute: "ampere"}},
			},
			wantErr: "must be a positive decimal number",
		},
		{
			scenario: "Given the CPU backend declares a compose overlay, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCPU},
				Backend:  map[string]BackendConfig{BackendCPU: {ComposeOverlay: "docker/cpu.yml"}},
			},
			wantErr: "compose_overlay is not allowed",
		},
		{
			scenario: "Given verify.kind is outside the closed set, Then it is rejected",
			spec: AccelerationSpec{
				Backends: []string{BackendCUDA},
				Backend:  map[string]BackendConfig{BackendCUDA: {Verify: VerifySpec{Kind: "guess"}}},
			},
			wantErr: `verify.kind "guess" is invalid`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When the contract validates the block
			err := tc.spec.Validate()
			// Then it is rejected with the contract's reason
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

type accelerationCase struct {
	scenario string
	spec     AccelerationSpec
	wantErr  string
}

// Scenario: a VRAM claim without a degrade ladder cannot be declared.
//
// This is the kokoro and speaker-verification gap: both reserve VRAM the broker
// can never ask them to release.
func TestAccelerationValidateRejectsUnsteppableVRAMClaim(t *testing.T) {
	cases := []struct {
		scenario string
		mutate   func(*capacity.ResourceClaimSpec)
		wantErr  string
	}{
		{
			scenario: "Given a VRAM claim with no degrade profile, Then it is rejected",
			mutate:   func(c *capacity.ResourceClaimSpec) { c.Profile = nil },
			wantErr:  "profile with at least one step is required",
		},
		{
			scenario: "Given a VRAM claim that never yields when idle, Then it is rejected",
			mutate:   func(c *capacity.ResourceClaimSpec) { c.YieldWhenIdle = false },
			wantErr:  "yield_when_idle is required",
		},
		{
			scenario: "Given a degrade profile with no apply verb, Then it is rejected",
			mutate:   func(c *capacity.ResourceClaimSpec) { c.Profile.Apply.Verb = "" },
			wantErr:  "apply.verb is required",
		},
		{
			scenario: "Given a degrade ladder that does not end at the floor, Then it is rejected",
			mutate:   func(c *capacity.ResourceClaimSpec) { c.Profile.Steps[1].AmountBytes = 2 << 30 },
			wantErr:  "must end at the floor",
		},
		{
			scenario: "Given a claim with no resource_kind, Then it is rejected",
			mutate:   func(c *capacity.ResourceClaimSpec) { c.ResourceKind = "" },
			wantErr:  "resource_kind is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a resource that declares CUDA and a VRAM claim
			claim := vramClaim()
			tc.mutate(claim)
			spec := AccelerationSpec{
				Backends: []string{BackendCUDA, BackendCPU},
				Backend:  map[string]BackendConfig{BackendCUDA: {}, BackendCPU: {}},
				Claim:    claim,
			}

			// When the contract validates it
			err := spec.Validate()

			// Then it is rejected
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// Scenario: a VRAM claim requires a non-CPU backend.
func TestAccelerationValidateRejectsVRAMClaimWithoutAccelerator(t *testing.T) {
	// Given a CPU-only resource that reserves VRAM
	spec := AccelerationSpec{
		Backends: []string{BackendCPU},
		Backend:  map[string]BackendConfig{BackendCPU: {}},
		Claim:    vramClaim(),
	}

	// When the contract validates it
	err := spec.Validate()

	// Then it is rejected, because VRAM cannot be claimed without a device
	if err == nil || !strings.Contains(err.Error(), "names no backend other than") {
		t.Fatalf("Validate() = %v, want a VRAM-without-accelerator rejection", err)
	}
}

// Scenario: the sibling-keys JSON shape round-trips and rejects typos.
func TestAccelerationJSONRoundTripAndUnknownKeyRejection(t *testing.T) {
	// Given the documented JSON shape
	raw := []byte(`{
	  "backends": ["cuda", "cpu"],
	  "require": "required",
	  "cuda": {"min_compute": "8.9", "compose_overlay": "docker/docker-compose.gpu.yml", "env": {"DEVICE": "cuda"}, "verify": {"kind": "process-device", "device": "nvidia"}},
	  "cpu": {"env": {"DEVICE": "cpu"}}
	}`)

	// When it is decoded
	var spec AccelerationSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("Unmarshal() = %v, want nil", err)
	}

	// Then every field lands where the contract says it does
	if got := spec.Backends; len(got) != 2 || got[0] != BackendCUDA || got[1] != BackendCPU {
		t.Fatalf("Backends = %v, want [cuda cpu]", got)
	}
	cuda, ok := spec.Config(BackendCUDA)
	if !ok {
		t.Fatal("Config(cuda) = not found, want the cuda block")
	}
	if cuda.MinCompute != "8.9" || cuda.ComposeOverlay != "docker/docker-compose.gpu.yml" || cuda.Env["DEVICE"] != "cuda" {
		t.Fatalf("cuda config = %+v, want the declared values", cuda)
	}
	if cuda.Verify.Kind != VerifyProcessDevice || cuda.Verify.Device != "nvidia" {
		t.Fatalf("cuda verify = %+v, want process-device/nvidia", cuda.Verify)
	}

	// And re-encoding then decoding produces the same spec
	encoded, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Marshal() = %v, want nil", err)
	}
	var round AccelerationSpec
	if err := json.Unmarshal(encoded, &round); err != nil {
		t.Fatalf("Unmarshal(round trip) = %v, want nil", err)
	}
	if !accelerationEqual(spec, round) {
		t.Fatalf("round trip = %+v, want %+v", round, spec)
	}

	// And a CPU-only block does not emit an empty verify object
	if strings.Contains(string(encoded), `"verify":{}`) {
		t.Fatalf("Marshal() emitted an empty verify object: %s", encoded)
	}
}

// Scenario: a mistyped backend name is a load failure, not a silent no-op.
func TestAccelerationUnmarshalRejectsUnknownKey(t *testing.T) {
	// Given a block with a mistyped backend key
	raw := []byte(`{"backends":["cuda"],"cuda":{},"metall":{}}`)

	// When it is decoded
	var spec AccelerationSpec
	err := json.Unmarshal(raw, &spec)

	// Then the load fails and names the offending key
	if err == nil || !strings.Contains(err.Error(), "metall") {
		t.Fatalf("Unmarshal() = %v, want an unknown-key rejection naming metall", err)
	}
}

func accelerationEqual(a, b AccelerationSpec) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	var lv, rv any
	if json.Unmarshal(left, &lv) != nil || json.Unmarshal(right, &rv) != nil {
		return false
	}
	lb, _ := json.Marshal(lv)
	rb, _ := json.Marshal(rv)
	return string(lb) == string(rb)
}
