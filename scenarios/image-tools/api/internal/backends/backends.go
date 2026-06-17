// Package backends is the provider-abstraction seam for image operations. Every
// operation is executed by a Provider; the registry enforces the scenario's
// headless tenet at boot — every registered operation must have at least one
// standalone (non-ComfyUI) provider, or the process refuses to start.
//
// Selection follows the ladder Local (standalone preferred, ComfyUI as an
// optional alternative) → BYOK cloud → refuse, surfacing a user-visible reason
// at each transition. Whether a local run uses the GPU or the CPU is decided by
// hardware-fit in internal/models (the model selector); this package consumes
// that verdict (GPUViable) to label the chosen tier and emit the GPU→CPU
// transition message.
package backends

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"image-tools/internal/models"
	"image-tools/internal/storage"
)

// Tier is the chosen position in the selection ladder.
type Tier int

const (
	// TierLocalGPU runs locally on a GPU.
	TierLocalGPU Tier = iota
	// TierLocalCPU runs locally on the CPU.
	TierLocalCPU
	// TierBYOK runs on a bring-your-own-key cloud provider (last resort).
	TierBYOK
)

func (t Tier) String() string {
	switch t {
	case TierLocalGPU:
		return "local-gpu"
	case TierLocalCPU:
		return "local-cpu"
	case TierBYOK:
		return "byok-cloud"
	default:
		return "unknown"
	}
}

// Request is the execution payload handed to a Provider.
type Request struct {
	// Operation is the op to run (a registry op vocabulary member).
	Operation string
	// Model is the selected registry model (zero value for provider-intrinsic
	// ops like deterministic edits that need no model).
	Model models.Model
	// GPU asks the provider to run on the GPU when true.
	GPU bool
	// InputKeys are blob-store keys (or local paths) of the operation inputs.
	InputKeys []string
	// Params carries op-specific parameters (e.g. prompt, scale, seed).
	Params map[string]string
	// Output selects where the result is written.
	Output storage.OutputTarget
}

// Result is the outcome of a Provider execution.
type Result struct {
	// OutputRef is the blob key or local path the result was written to.
	OutputRef string
	// Tier records where the op actually ran.
	Tier Tier
	// Meta carries provider-reported metadata (timings, seed used, …).
	Meta map[string]string
}

// Provider executes one or more operations through a concrete backend.
type Provider interface {
	// Name identifies the backend (matches models.Model.Backend for the local
	// backend that natively runs a model, e.g. "stable-diffusion.cpp").
	Name() string
	// Operations lists the ops this provider can execute.
	Operations() []string
	// Standalone reports whether the provider runs WITHOUT ComfyUI. At least
	// one standalone provider per op is required (the headless tenet).
	Standalone() bool
	// IsCloud reports whether this is a BYOK cloud provider (the last tier).
	IsCloud() bool
	// Available reports whether the provider can run right now (binary present,
	// key configured, resource /ready). Probed at selection time.
	Available(ctx context.Context) bool
	// Execute runs the request.
	Execute(ctx context.Context, req Request) (Result, error)
}

// Registry holds the registered providers, indexed by operation.
type Registry struct {
	byOp map[string][]Provider
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{byOp: make(map[string][]Provider)}
}

// Registration errors.
var (
	// ErrNoOperations is returned when a provider registers with no operations.
	ErrNoOperations = errors.New("backends: provider has no operations")
	// ErrMissingStandalone is returned by Validate when an op lacks a standalone provider.
	ErrMissingStandalone = errors.New("backends: operation has no standalone (non-ComfyUI) provider")
	// ErrNoProvider is returned by SelectProvider when no provider serves the op.
	ErrNoProvider = errors.New("backends: no provider for operation")
	// ErrNoneAvailable is returned when providers exist but none can run now.
	ErrNoneAvailable = errors.New("backends: no available provider")
)

// Register adds a provider for each of its operations. It does not enforce the
// standalone invariant (a ComfyUI-only op is legal until Validate runs at boot,
// so providers may register in any order).
func (r *Registry) Register(p Provider) error {
	ops := p.Operations()
	if len(ops) == 0 {
		return fmt.Errorf("%w: %s", ErrNoOperations, p.Name())
	}
	for _, op := range ops {
		r.byOp[op] = append(r.byOp[op], p)
	}
	return nil
}

// Validate enforces the headless tenet: every registered operation must have at
// least one available-by-construction standalone provider. Call once at boot;
// fail loud on error.
func (r *Registry) Validate() error {
	var bad []string
	for op, providers := range r.byOp {
		hasStandalone := false
		for _, p := range providers {
			if p.Standalone() && !p.IsCloud() {
				hasStandalone = true
				break
			}
		}
		if !hasStandalone {
			bad = append(bad, op)
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("%w: %s", ErrMissingStandalone, strings.Join(bad, ", "))
	}
	return nil
}

// Operations returns the registered operations, sorted.
func (r *Registry) Operations() []string {
	ops := make([]string, 0, len(r.byOp))
	for op := range r.byOp {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}

// Providers returns the providers registered for op, in registration order.
func (r *Registry) Providers(op string) []Provider {
	return append([]Provider(nil), r.byOp[op]...)
}
