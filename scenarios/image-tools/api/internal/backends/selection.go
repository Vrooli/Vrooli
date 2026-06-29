package backends

import (
	"context"
	"fmt"
)

// gpuCapableProvider is an optional Provider capability: a backend that can run
// on the GPU implements it returning true. Backends that do not implement it are
// assumed GPU-capable (the historical default); a CPU-only backend implements it
// returning false so the selector never labels its run "local-gpu".
type gpuCapableProvider interface {
	GPUCapable() bool
}

// providerGPUCapable reports whether p can use a GPU. Defaults to true when the
// provider does not declare its capability.
func providerGPUCapable(p Provider) bool {
	if g, ok := p.(gpuCapableProvider); ok {
		return g.GPUCapable()
	}
	return true
}

// adapterCapableProvider is an optional Provider capability: a backend that can
// apply the conditioning adapter stack (LoRA/ControlNet/IP-Adapter) for an op
// implements it. A backend that does not implement it cannot honor a conditioned
// request, so selection skips it when adapters are required.
type adapterCapableProvider interface {
	SupportsAdapters(op string) bool
}

// providerSupportsAdapters reports whether p can apply conditioning adapters for
// op. Defaults to false: a backend must opt in (only the diffusers sidecar does).
func providerSupportsAdapters(p Provider, op string) bool {
	if a, ok := p.(adapterCapableProvider); ok {
		return a.SupportsAdapters(op)
	}
	return false
}

// SelectRequest drives provider selection along the ladder.
type SelectRequest struct {
	// Operation is the op to run.
	Operation string
	// ModelBackend is the chosen model's backend name (models.Model.Backend).
	// When set, the matching local provider is preferred. Empty for intrinsic
	// ops (any provider for the op is acceptable).
	ModelBackend string
	// GPUViable reports whether the model selector judged the chosen model
	// runnable on this host's GPU. Used to label the local tier and emit the
	// GPU→CPU transition message.
	GPUViable bool
	// AllowBYOK permits falling back to a cloud provider when no local provider
	// is available. Off by default — BYOK is opt-in and metered.
	AllowBYOK bool
	// RequireAdapters constrains selection to a backend that can apply the
	// conditioning adapter stack (LoRA/ControlNet/IP-Adapter) for Operation. Set
	// when the request carries adapters: a model's default backend that cannot
	// honor them (stable-diffusion.cpp) is skipped in favor of one that can (the
	// diffusers sidecar), so a conditioned request never silently drops or rejects
	// its modifiers.
	RequireAdapters bool
}

// Selection is the chosen provider plus the ladder decision context.
type Selection struct {
	// Provider is the chosen backend.
	Provider Provider
	// Tier is where the op will run.
	Tier Tier
	// Reason is a user-facing explanation of the choice.
	Reason string
	// Warnings carries non-fatal transition notices (GPU→CPU, backend
	// substitution, ComfyUI fallback, BYOK cost caution).
	Warnings []string
}

// SelectProvider walks the ladder for req.Operation:
//
//  1. Local standalone provider matching ModelBackend (the model's native backend)
//  2. Any other available local standalone provider (with a substitution warning)
//  3. An available ComfyUI (non-standalone local) provider (with a fallback warning)
//  4. An available BYOK cloud provider, only if AllowBYOK (with a cost warning)
//  5. Otherwise refuse with an actionable message
//
// The local tier is labeled GPU or CPU from GPUViable; a CPU label on a host
// where GPU was expected carries the GPU→CPU transition warning.
func (r *Registry) SelectProvider(ctx context.Context, req SelectRequest) (Selection, error) {
	providers := r.byOp[req.Operation]
	if len(providers) == 0 {
		return Selection{}, fmt.Errorf("%w: %q", ErrNoProvider, req.Operation)
	}

	var (
		matchLocal        Provider // standalone, name == ModelBackend
		otherLocal        Provider // standalone, different backend
		comfy             Provider // non-standalone local (ComfyUI)
		cloud             Provider // BYOK cloud
		anyAvailable      bool
		unavailable       []Provider
		unavailableCloud  []Provider
		unavailableLocals []Provider
	)
	for _, p := range providers {
		if !providerAvailability(ctx, p).Available {
			unavailable = append(unavailable, p)
			if p.IsCloud() {
				unavailableCloud = append(unavailableCloud, p)
			} else {
				unavailableLocals = append(unavailableLocals, p)
			}
			continue
		}
		anyAvailable = true
		// A conditioned request needs a backend that can apply the adapter stack;
		// one that cannot (e.g. stable-diffusion.cpp) is not a candidate even when
		// it is the model's native backend.
		if req.RequireAdapters && p.Standalone() && !providerSupportsAdapters(p, req.Operation) {
			continue
		}
		switch {
		case p.IsCloud():
			if cloud == nil {
				cloud = p
			}
		case p.Standalone():
			if req.ModelBackend != "" && p.Name() == req.ModelBackend && matchLocal == nil {
				matchLocal = p
			} else if otherLocal == nil {
				otherLocal = p
			}
		default: // local but not standalone == ComfyUI plug-in
			if comfy == nil {
				comfy = p
			}
		}
	}

	// applyLocalTier sets the honest execution tier for a chosen local provider.
	// A GPU is only claimed when the host has a GPU-viable path AND the backend
	// can actually use it: a CPU-only backend (e.g. the onnxruntime sidecar bound
	// to CPUExecutionProvider) reports local-cpu even on a GPU host, so the tier
	// label never overstates where the op really runs.
	applyLocalTier := func(s *Selection, p Provider) {
		switch {
		case req.GPUViable && providerGPUCapable(p):
			s.Tier = TierLocalGPU
		case req.GPUViable && !providerGPUCapable(p):
			s.Tier = TierLocalCPU
			s.Warnings = append(s.Warnings, fmt.Sprintf("backend %q is CPU-only; running on CPU despite an available GPU", p.Name()))
		default:
			s.Tier = TierLocalCPU
			s.Warnings = append(s.Warnings, "no GPU-viable path; running locally on CPU (slower)")
		}
		s.Reason = fmt.Sprintf("local backend %q (%s)", p.Name(), s.Tier)
	}

	switch {
	case matchLocal != nil:
		sel := Selection{Provider: matchLocal}
		applyLocalTier(&sel, matchLocal)
		return sel, nil

	case otherLocal != nil:
		sel := Selection{Provider: otherLocal}
		applyLocalTier(&sel, otherLocal)
		switch {
		case req.RequireAdapters && req.ModelBackend != "" && req.ModelBackend != otherLocal.Name():
			sel.Warnings = append(sel.Warnings, fmt.Sprintf("conditioning adapters require the %q backend; the model's native backend %q cannot apply them", otherLocal.Name(), req.ModelBackend))
		case req.ModelBackend != "":
			sel.Warnings = append(sel.Warnings, fmt.Sprintf("model's native backend %q unavailable; substituting %q", req.ModelBackend, otherLocal.Name()))
		}
		return sel, nil

	case comfy != nil:
		sel := Selection{Provider: comfy}
		applyLocalTier(&sel, comfy)
		sel.Reason = fmt.Sprintf("ComfyUI provider %q (%s)", comfy.Name(), sel.Tier)
		sel.Warnings = append(sel.Warnings, "no standalone backend available; using the optional ComfyUI plug-in")
		return sel, nil

	case cloud != nil && req.AllowBYOK:
		sel := Selection{Provider: cloud, Tier: TierBYOK, Reason: fmt.Sprintf("BYOK cloud provider %q", cloud.Name())}
		sel.Warnings = append(sel.Warnings, "running on a paid BYOK cloud provider; confirm the cost estimate before proceeding")
		return sel, nil

	case cloud != nil && !req.AllowBYOK:
		return Selection{}, fmt.Errorf("%w for %q: only a BYOK cloud provider is available and BYOK is not enabled for this request", ErrNoneAvailable, req.Operation)

	case !anyAvailable:
		details := unavailableProviderDetails(ctx, unavailable)
		if details == "" {
			details = "install a model/backend, or enable BYOK"
		}
		return Selection{}, fmt.Errorf("%w for %q: providers are registered but none is ready (%s)", ErrNoneAvailable, req.Operation, details)

	default:
		if len(unavailableLocals) > 0 && len(unavailableCloud) > 0 {
			return Selection{}, fmt.Errorf("%w for %q: local providers unavailable (%s); BYOK providers unavailable (%s)", ErrNoneAvailable, req.Operation, unavailableProviderDetails(ctx, unavailableLocals), unavailableProviderDetails(ctx, unavailableCloud))
		}
		return Selection{}, fmt.Errorf("%w for %q", ErrNoneAvailable, req.Operation)
	}
}
