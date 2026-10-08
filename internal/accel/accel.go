// Package accel owns accelerator truth for the control plane: which backends a
// host can reach, which backend a resource should run on, and which backend a
// running resource is actually on.
//
// Two rules keep this package honest and are enforced by its own tests:
//
//   - It never runs a command. Every host observation arrives through a
//     FactSource reading internal/hostinventory, and every container probe
//     arrives through an injected ContainerProbe. hostinventory keeps sole
//     ownership of vendor-tool calls.
//   - It never infers placement from configuration, from an environment
//     variable, or from a log line. Placement is read from the host or it is
//     unknown.
package accel

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Backend names the software path a resource uses to reach an accelerator. The
// set is closed so placement verification is decidable and a typo is a load
// failure rather than a silently ignored declaration.
type Backend string

const (
	BackendCUDA   Backend = "cuda"
	BackendMetal  Backend = "metal"
	BackendROCm   Backend = "rocm"
	BackendVulkan Backend = "vulkan"
	BackendCPU    Backend = "cpu"
)

// AllBackends is the closed backend set in preference order, CPU last.
var AllBackends = []Backend{BackendCUDA, BackendMetal, BackendROCm, BackendVulkan, BackendCPU}

// Require states declare how strictly a resource needs a non-CPU backend.
const (
	RequireRequired  = "required"
	RequirePreferred = "preferred"
	RequireNone      = "none"
)

// Spec is the decision input: what a resource declared, reduced to the fields
// readiness and placement need. It is built from a manifest's acceleration
// block by the caller, so this package never parses a manifest.
type Spec struct {
	// Resource is the resource name, used only in messages.
	Resource string
	// Backends is the declared preference order. The last entry is the floor.
	Backends []Backend
	// Require is one of RequireRequired, RequirePreferred or RequireNone.
	// Empty means RequirePreferred.
	Require string
}

// EffectiveRequire resolves an absent Require to RequirePreferred.
func (s Spec) EffectiveRequire() string {
	if strings.TrimSpace(s.Require) == "" {
		return RequirePreferred
	}
	return s.Require
}

// Accelerated reports whether the spec asks for any backend other than CPU.
func (s Spec) Accelerated() bool {
	for _, backend := range s.Backends {
		if backend != BackendCPU {
			return true
		}
	}
	return false
}

// FactSource is the injectable seam over live host sensing. Production reads
// the real snapshot; tests provide a fake, so no test needs an accelerator.
// This mirrors internal/capacity's CapacitySource on purpose.
type FactSource interface {
	Snapshot(ctx context.Context) (hostinventory.Snapshot, error)
}

// HostFactSource is the production FactSource. It uses the GPU-only collection
// path so a slow desktop probe cannot delay a resource start.
type HostFactSource struct{}

// Snapshot collects the accelerator-relevant host inventory.
func (HostFactSource) Snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	return hostinventory.CollectGPUFacts(ctx)
}

// StaticFactSource returns a fixed snapshot. Tests and callers that already
// hold a snapshot use it.
type StaticFactSource struct {
	Inventory hostinventory.Snapshot
	Err       error
}

// Snapshot returns the fixed snapshot.
func (s StaticFactSource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	return s.Inventory, s.Err
}

// ParseBackend converts declared text into a Backend.
func ParseBackend(value string) (Backend, error) {
	candidate := Backend(strings.TrimSpace(strings.ToLower(value)))
	if slices.Contains(AllBackends, candidate) {
		return candidate, nil
	}
	return "", fmt.Errorf("backend %q is not one of %v", value, AllBackends)
}

// ParseBackends converts a declared list into backends, preserving order.
func ParseBackends(values []string) ([]Backend, error) {
	out := make([]Backend, 0, len(values))
	for _, value := range values {
		backend, err := ParseBackend(value)
		if err != nil {
			return nil, err
		}
		out = append(out, backend)
	}
	return out, nil
}

// ReachableBackends reads the accel.backends fact the host published. It is the
// single answer to "what can this host reach"; nothing in this package probes
// a device itself.
func ReachableBackends(snapshot hostinventory.Snapshot) []Backend {
	value := snapshot.AcceleratorFacts()[hostinventory.FactAccelBackends]
	if strings.TrimSpace(value) == "" {
		return []Backend{BackendCPU}
	}
	parts := strings.Split(value, ",")
	out := make([]Backend, 0, len(parts))
	for _, part := range parts {
		if backend, err := ParseBackend(part); err == nil {
			out = append(out, backend)
		}
	}
	if len(out) == 0 {
		return []Backend{BackendCPU}
	}
	return out
}
