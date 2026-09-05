package manifest

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/capacity"
)

// Backend names form a closed set. A closed set is what makes placement
// verification decidable and lets the JSON schema reject a typo; adding a
// sixth backend is a deliberate contract change, not a manifest edit.
const (
	BackendCUDA   = "cuda"
	BackendMetal  = "metal"
	BackendROCm   = "rocm"
	BackendVulkan = "vulkan"
	BackendCPU    = "cpu"
)

// AllowedBackends is the closed backend set, in the order the contract
// documents them.
var AllowedBackends = []string{BackendCUDA, BackendMetal, BackendROCm, BackendVulkan, BackendCPU}

// Require states declare how strictly a resource needs a non-CPU backend.
const (
	// RequireRequired fails install and start when no declared non-CPU backend
	// is ready on the host.
	RequireRequired = "required"
	// RequirePreferred falls through to the next declared backend and records
	// mode drift. This is the default.
	RequirePreferred = "preferred"
	// RequireNone declares the accelerator purely opportunistic: falling back
	// is not drift.
	RequireNone = "none"
)

// AllowedAccelerationRequire is the closed set of `require` values.
var AllowedAccelerationRequire = []string{RequireRequired, RequirePreferred, RequireNone}

// Placement verification kinds. Empty selects the default for the backend and
// the placement target, which is what every manifest in the fleet uses.
const (
	// VerifyProcessDevice asserts the resource's host process holds a device of
	// the backend's family.
	VerifyProcessDevice = "process-device"
	// VerifyContainerDevice asserts the resource's container can open the
	// backend's device node.
	VerifyContainerDevice = "container-device"
	// VerifyNone declares placement unverifiable for this backend, so observed
	// mode is reported as unknown rather than guessed.
	VerifyNone = "none"
)

// DegradeVerb is the single subcommand the capacity broker calls on every
// resource CLI. A resource declares what each rung means; it does not get to
// declare how it is asked.
const DegradeVerb = "capacity"

// DegradeArgv is the argument vector that goes with DegradeVerb. The {label}
// placeholder is filled with the target rung.
var DegradeArgv = []string{"degrade", "--to", "{label}"}

// AllowedVerifyKinds is the closed set of verification kinds.
var AllowedVerifyKinds = []string{VerifyProcessDevice, VerifyContainerDevice, VerifyNone}

// accelerationReservedKeys are the non-backend keys inside an `acceleration`
// block. Everything else must name a backend from AllowedBackends.
var accelerationReservedKeys = []string{"backends", "require", "claim"}

// AccelerationSpec is the single accelerator declaration in resource.json. It
// replaces the `gpu` block, `requirements.gpu`, and the top-level `capacity`
// block, which produced two contradictory definitions of "this resource uses
// the GPU" between fleet_contract.go and capacity/declared.go.
//
// The JSON shape puts backend configs beside the scalar fields:
//
//	"acceleration": {
//	  "backends": ["cuda", "cpu"],
//	  "require": "preferred",
//	  "cuda": { "min_compute": "8.9", "env": {"DEVICE": "cuda"} },
//	  "cpu":  { "env": {"DEVICE": "cpu"} },
//	  "claim": { "resource_kind": "vram", ... }
//	}
type AccelerationSpec struct {
	// Backends is an ordered preference list drawn from AllowedBackends. The
	// last entry is the floor the resource can always fall back to.
	Backends []string
	// Require is one of AllowedAccelerationRequire. Absent means
	// RequirePreferred.
	Require string
	// Backend holds one config per name in Backends, keyed by backend name.
	Backend map[string]BackendConfig
	// Claim is the capacity broker's claim spec, moved inside the accelerator
	// declaration so a resource cannot claim VRAM without declaring a backend.
	Claim *capacity.ResourceClaimSpec
}

// BackendConfig is what a resource needs in order to run on one backend.
type BackendConfig struct {
	// MinCompute is the minimum vendor compute capability, as a decimal string.
	// Only meaningful for BackendCUDA today.
	MinCompute string `json:"min_compute,omitempty"`
	// ComposeOverlay is a compose file the container drivers layer on when this
	// backend is selected, relative to the resource root or absolute.
	ComposeOverlay string `json:"compose_overlay,omitempty"`
	// Env is applied to the resource process when this backend is selected.
	Env map[string]string `json:"env,omitempty"`
	// LibraryPaths are directories the dynamic loader must search for this
	// backend's shared libraries, relative to the staged artifact or absolute.
	// Acquisition checks the artifact's runtime closure against them, so a
	// target that ships its libraries outside the artifact's own directory is
	// accepted rather than rejected for libraries the host does not have
	// system-wide.
	LibraryPaths []string `json:"library_paths,omitempty"`
	// Verify overrides the default placement evidence for this backend.
	Verify VerifySpec `json:"verify,omitzero"`
}

// VerifySpec overrides how the control plane reads observed placement.
type VerifySpec struct {
	// Kind is one of AllowedVerifyKinds. Empty selects the default for the
	// backend and the placement target kind.
	Kind string `json:"kind,omitempty"`
	// Device narrows a device check to one device-node family.
	Device string `json:"device,omitempty"`
}

// IsZero reports whether the spec carries no override, so `omitzero` can drop
// it from the marshalled form.
func (v VerifySpec) IsZero() bool { return v.Kind == "" && v.Device == "" }

// accelerationScalars is the non-backend half of an acceleration block. It
// exists so UnmarshalJSON can decode the scalar fields with the standard
// decoder and treat every remaining key as a backend name.
type accelerationScalars struct {
	Backends []string                    `json:"backends"`
	Require  string                      `json:"require,omitempty"`
	Claim    *capacity.ResourceClaimSpec `json:"claim,omitempty"`
}

// UnmarshalJSON decodes the sibling-keys shape and rejects any key that is
// neither a reserved scalar nor a member of the closed backend set. Rejecting
// unknown keys here is what stops a typo becoming a silently ignored backend.
func (a *AccelerationSpec) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("acceleration: %w", err)
	}

	var scalars accelerationScalars
	if err := json.Unmarshal(data, &scalars); err != nil {
		return fmt.Errorf("acceleration: %w", err)
	}
	a.Backends = scalars.Backends
	a.Require = scalars.Require
	a.Claim = scalars.Claim
	a.Backend = nil

	var unknown []string
	for key, value := range raw {
		if slices.Contains(accelerationReservedKeys, key) {
			continue
		}
		if !slices.Contains(AllowedBackends, key) {
			unknown = append(unknown, key)
			continue
		}
		var config BackendConfig
		if err := json.Unmarshal(value, &config); err != nil {
			return fmt.Errorf("acceleration.%s: %w", key, err)
		}
		if a.Backend == nil {
			a.Backend = make(map[string]BackendConfig, len(raw))
		}
		a.Backend[key] = config
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		return fmt.Errorf("acceleration has unknown key(s) %v (allowed backends: %v)", unknown, AllowedBackends)
	}
	return nil
}

// MarshalJSON writes the sibling-keys shape back, so a normalised manifest can
// be compared against a hand-written one byte for byte.
func (a AccelerationSpec) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(a.Backend)+len(accelerationReservedKeys))
	if len(a.Backends) > 0 {
		out["backends"] = a.Backends
	}
	if a.Require != "" {
		out["require"] = a.Require
	}
	if a.Claim != nil {
		out["claim"] = a.Claim
	}
	for name, config := range a.Backend {
		out[name] = config
	}
	return json.Marshal(out)
}

// EffectiveRequire resolves the declared require state, defaulting absent to
// RequirePreferred.
func (a AccelerationSpec) EffectiveRequire() string {
	if strings.TrimSpace(a.Require) == "" {
		return RequirePreferred
	}
	return a.Require
}

// Config returns the config for one backend.
func (a AccelerationSpec) Config(backend string) (BackendConfig, bool) {
	config, ok := a.Backend[backend]
	return config, ok
}

// AcceleratedBackends returns the declared backends other than CPU, in
// preference order. An empty result means the resource declares no accelerator.
func (a AccelerationSpec) AcceleratedBackends() []string {
	out := make([]string, 0, len(a.Backends))
	for _, backend := range a.Backends {
		if backend != BackendCPU {
			out = append(out, backend)
		}
	}
	return out
}

// DeclaresAcceleration reports whether the resource asks for any non-CPU
// backend. This is the single answer to the question `capacity/declared.go` and
// `fleet_contract.go` used to answer differently.
func (a AccelerationSpec) DeclaresAcceleration() bool { return len(a.AcceleratedBackends()) > 0 }

// Validate enforces every rule the contract makes about an acceleration block.
func (a AccelerationSpec) Validate() error {
	if len(a.Backends) == 0 {
		return fmt.Errorf("acceleration.backends must list at least one backend (allowed: %v)", AllowedBackends)
	}
	seen := make(map[string]struct{}, len(a.Backends))
	for _, backend := range a.Backends {
		if !slices.Contains(AllowedBackends, backend) {
			return fmt.Errorf("acceleration.backends entry %q is invalid (allowed: %v)", backend, AllowedBackends)
		}
		if _, duplicate := seen[backend]; duplicate {
			return fmt.Errorf("acceleration.backends lists %q more than once", backend)
		}
		seen[backend] = struct{}{}
		if _, ok := a.Backend[backend]; !ok {
			return fmt.Errorf("acceleration.backends names %q but there is no acceleration.%s config block", backend, backend)
		}
	}
	for backend := range a.Backend {
		if _, ok := seen[backend]; !ok {
			return fmt.Errorf("acceleration.%s is configured but %q is absent from acceleration.backends", backend, backend)
		}
	}
	if require := strings.TrimSpace(a.Require); require != "" && !slices.Contains(AllowedAccelerationRequire, require) {
		return fmt.Errorf("acceleration.require %q is invalid (allowed: %v)", require, AllowedAccelerationRequire)
	}
	if a.EffectiveRequire() == RequireRequired && !a.DeclaresAcceleration() {
		return fmt.Errorf("acceleration.require is %q but acceleration.backends names no backend other than %q", RequireRequired, BackendCPU)
	}
	for _, backend := range a.Backends {
		if err := a.Backend[backend].validate(backend); err != nil {
			return err
		}
	}
	return a.validateClaim()
}

// validateClaim verifies the broker's claim ladder. A CPU-only declaration may
// declare a VRAM claim only when a device backend is declared; a CPU-only
// resource cannot reserve video memory.
func (a AccelerationSpec) validateClaim() error {
	if a.Claim == nil {
		return nil
	}
	kind := strings.TrimSpace(a.Claim.ResourceKind)
	if kind == "" {
		return fmt.Errorf("acceleration.claim.resource_kind is required")
	}
	if kind != capacity.ResourceKindVRAM {
		return nil
	}
	if !a.DeclaresAcceleration() {
		return fmt.Errorf("acceleration.claim.resource_kind %q requires a declared non-CPU accelerator backend", kind)
	}
	if a.Claim.Profile == nil || len(a.Claim.Profile.Steps) == 0 {
		return fmt.Errorf("acceleration.claim.profile with at least one step is required for a %q claim, otherwise the broker can never step the resource down", capacity.ResourceKindVRAM)
	}
	if !a.Claim.YieldWhenIdle {
		return fmt.Errorf("acceleration.claim.yield_when_idle is required for a %q claim, otherwise an idle claim never releases capacity to active work", capacity.ResourceKindVRAM)
	}
	if verb := strings.TrimSpace(a.Claim.Profile.Apply.Verb); verb != DegradeVerb {
		if verb == "" {
			return fmt.Errorf("acceleration.claim.profile.apply.verb is required so the broker knows how to ask the resource to step down; it must be %q", DegradeVerb)
		}
		// One broker contract had four call signatures across the fleet. The
		// broker had to know which resource it was talking to in order to talk
		// to it, which is the opposite of a contract.
		return fmt.Errorf("acceleration.claim.profile.apply.verb is %q; every resource exposes the same contract, so it must be %q with argv %v", verb, DegradeVerb, DegradeArgv)
	}
	last := a.Claim.Profile.Steps[len(a.Claim.Profile.Steps)-1]
	if last.AmountBytes != a.Claim.FloorBytes {
		return fmt.Errorf("acceleration.claim.profile last step %q is %d bytes but claim.floor_bytes is %d; the ladder must end at the floor", last.Label, last.AmountBytes, a.Claim.FloorBytes)
	}
	return nil
}

func (b BackendConfig) validate(backend string) error {
	if compute := strings.TrimSpace(b.MinCompute); compute != "" {
		if backend != BackendCUDA {
			return fmt.Errorf("acceleration.%s.min_compute is only meaningful for backend %q", backend, BackendCUDA)
		}
		if value, err := strconv.ParseFloat(compute, 64); err != nil || value <= 0 {
			return fmt.Errorf("acceleration.%s.min_compute %q must be a positive decimal number", backend, b.MinCompute)
		}
	}
	if backend == BackendCPU && strings.TrimSpace(b.ComposeOverlay) != "" {
		return fmt.Errorf("acceleration.%s.compose_overlay is not allowed; the CPU backend is the base compose file", BackendCPU)
	}
	if kind := strings.TrimSpace(b.Verify.Kind); kind != "" && !slices.Contains(AllowedVerifyKinds, kind) {
		return fmt.Errorf("acceleration.%s.verify.kind %q is invalid (allowed: %v)", backend, kind, AllowedVerifyKinds)
	}
	for key := range b.Env {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("acceleration.%s.env has an empty variable name", backend)
		}
	}
	for _, path := range b.LibraryPaths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("acceleration.%s.library_paths has an empty entry", backend)
		}
	}
	return nil
}
