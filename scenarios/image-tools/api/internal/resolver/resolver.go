// Package resolver is the single home for image-tools' operation→model→technique
// resolution: the "requires-a-condition-to-be-true" decision that turns a
// requested operation (plus an optional model override and the probed host) into
// an explicit, inspectable Resolution value object.
//
// Before this package that decision was implicit and scattered: model selection
// lived in models.Registry.Select, backend-tier selection in backends.Registry,
// the native-vs-derived fact in models.Model.EffectiveOps, and the consent weight
// in safety.OpWeight — with nothing tying them into one answer a caller could
// read back before executing. The Resolver composes them into one value, so the
// --explain / dry-run surface (ExplainResolution RPC + `models explain` CLI) can
// return exactly what WOULD run without running it, and the AI submit edge can
// pin the same resolution into the job. See docs/internal/TECHNIQUE-SUBSTRATE.md
// (Phase 4) and DECISIONS.md (2026-06-26).
package resolver

import (
	"context"
	"fmt"

	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/models"
	"image-tools/internal/safety"
)

// Support classifies how a model serves an operation.
const (
	// SupportNative — the model declares the operation in its registry row; the
	// backend's own builder runs it.
	SupportNative = "native"
	// SupportDerived — the model does not declare the operation, but its
	// architecture derives it through a named technique (with a quality caveat).
	SupportDerived = "derived"
)

// Resolution is the explicit, inspectable answer to "what would run for this
// operation on this host". It is read-only data — producing it executes nothing.
type Resolution struct {
	// Operation is the resolved operation.
	Operation string
	// Model is the chosen registry entry.
	Model models.Model
	// Support is native | derived (see the Support constants).
	Support string
	// Technique names the derived technique that yields the op (empty for native
	// ops, which the backend's own builder resolves).
	Technique string
	// PipelineClass is the model's declared diffusers pipeline class when it has
	// one (informational; empty for backends with no such dispatch).
	PipelineClass string
	// Caveat is the derived-op quality note (empty for native ops).
	Caveat string
	// Weight is the safety consent weight of the OPERATION (none|low|high). It is
	// operation-keyed and invariant to native-vs-derived: a derived inpaint/edit
	// stays HIGH-weight exactly like a native one (decision 113). When the request
	// carries conditioning adapters it is ELEVATED to max(op, adapters...) — an
	// IP-Adapter / identity-class LoRA can push a none-weight op to high (C4).
	Weight string
	// Adapters is the validated, execution-ready conditioning stack (empty when the
	// request carried none). Sorted into canonical application order (LoRA →
	// ControlNet → IP-Adapter).
	Adapters []adapters.ResolvedAdapter
	// Tier is the backend tier the op would run on (local-gpu|local-cpu|byok).
	Tier string
	// GPUViable is true when the chosen model would run on a detected GPU with
	// known, sufficient VRAM.
	GPUViable bool
	// Warnings carries the selection + backend cautions (CPU-slow, VRAM
	// shortfall, BYOK cost, derived-op caveat, adapter auto-preprocess, …).
	Warnings []string
}

// Request drives a resolution. Host is the already-probed hardware snapshot (the
// resolver never probes — the caller owns probing so the resolver stays pure and
// testable). IsEnabled overlays runtime enable/disable state; nil uses the seed
// defaults.
type Request struct {
	Operation     string
	ModelOverride string
	Host          capabilities.Host
	AllowBYOK     bool
	IsEnabled     models.EnabledFunc
	// Adapters is the requested conditioning stack (LoRA / ControlNet / IP-Adapter).
	// Empty for an unconditioned op.
	Adapters []adapters.AdapterRequest
	// AdapterByID is the merged (seed + custom) adapter catalog lookup the
	// conditioning resolver validates each request against. nil when the caller
	// supplies no adapters; a non-empty Adapters with a nil lookup is rejected.
	AdapterByID func(id string) (adapters.Adapter, bool)
	// AdapterEnabled / AdapterInstalled overlay an adapter's runtime state (nil
	// AdapterEnabled ⇒ seed default; nil AdapterInstalled ⇒ treated as not
	// installed, so a conditioned request fails honestly until installed).
	AdapterEnabled   func(id string) bool
	AdapterInstalled func(id string) bool
}

// Resolver composes model selection, technique derivation, backend-tier
// selection, and the consent-weight table into one Resolution.
type Resolver struct {
	registry *models.Registry
	backends *backends.Registry
}

// New builds a Resolver over the model registry and backend provider registry.
// backends may be nil (the resolution then omits tier/backend warnings) — useful
// for a pure model/technique explanation that needs no host backends.
func New(registry *models.Registry, be *backends.Registry) *Resolver {
	return &Resolver{registry: registry, backends: be}
}

// Resolve produces the Resolution for an operation. It fails (executing nothing)
// when the op is unknown, no enabled model serves it on this host, or an explicit
// override cannot be honored — each error is the selector's already-actionable
// message. A derived operation whose technique is not yet proven (Ready=false) is
// not served and surfaces as a selection error, honestly, never a silent run.
func (r *Resolver) Resolve(ctx context.Context, req Request) (Resolution, error) {
	sel, err := r.registry.Select(models.SelectRequest{
		Operation:  req.Operation,
		Host:       req.Host,
		OverrideID: req.ModelOverride,
	}, req.IsEnabled)
	if err != nil {
		return Resolution{}, err
	}

	res := Resolution{
		Operation:     req.Operation,
		Model:         sel.Model,
		Support:       SupportNative,
		PipelineClass: sel.Model.Runtime.PipelineClass,
		Weight:        string(safety.OpWeight(req.Operation)),
		GPUViable:     sel.GPUViable,
		Warnings:      append([]string{}, sel.Warnings...),
	}
	if dt, derived := sel.Model.DerivesOperation(req.Operation); derived {
		res.Support = SupportDerived
		res.Technique = dt.Technique
		res.Caveat = dt.Caveat
		if dt.Caveat != "" {
			res.Warnings = append(res.Warnings, dt.Caveat)
		}
	}

	// Conditioning: validate the requested adapter stack against the chosen model's
	// architecture (fail closed on incompatible/disabled/not-installed/not-Ready),
	// attach it to the resolution, and elevate the consent weight to
	// max(op, adapters...) — decision C4. Producing this executes nothing.
	if len(req.Adapters) > 0 {
		resolved, warns, cerr := adapters.ResolveConditioning(
			sel.Model.Architecture,
			req.Adapters,
			req.AdapterByID,
			req.AdapterEnabled,
			req.AdapterInstalled,
		)
		if cerr != nil {
			return Resolution{}, cerr
		}
		res.Adapters = resolved
		res.Warnings = append(res.Warnings, warns...)
		res.Weight = string(adapters.EffectiveWeight(safety.OpWeight(req.Operation), resolved))
	}

	if r.backends != nil {
		bsel, berr := r.backends.SelectProvider(ctx, backends.SelectRequest{
			Operation:    req.Operation,
			ModelBackend: sel.Model.Backend,
			GPUViable:    sel.GPUViable,
			AllowBYOK:    req.AllowBYOK,
		})
		if berr != nil {
			return Resolution{}, berr // "no available provider — install a backend/enable BYOK"
		}
		res.Tier = bsel.Tier.String()
		res.Warnings = append(res.Warnings, bsel.Warnings...)
	}
	return res, nil
}

// Describe returns a compact one-line human summary of a resolution.
func (res Resolution) Describe() string {
	s := fmt.Sprintf("%s → model %q (%s)", res.Operation, res.Model.ID, res.Support)
	if res.Support == SupportDerived && res.Technique != "" {
		s += fmt.Sprintf(" via %s", res.Technique)
	}
	if res.Tier != "" {
		s += fmt.Sprintf(" on %s", res.Tier)
	}
	return s
}
