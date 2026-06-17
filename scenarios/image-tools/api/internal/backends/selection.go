package backends

import (
	"context"
	"fmt"
)

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
		matchLocal   Provider // standalone, name == ModelBackend
		otherLocal   Provider // standalone, different backend
		comfy        Provider // non-standalone local (ComfyUI)
		cloud        Provider // BYOK cloud
		anyAvailable bool
	)
	for _, p := range providers {
		if !p.Available(ctx) {
			continue
		}
		anyAvailable = true
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

	localTier := TierLocalCPU
	if req.GPUViable {
		localTier = TierLocalGPU
	}
	cpuTransition := func(s *Selection) {
		if !req.GPUViable {
			s.Warnings = append(s.Warnings, "no GPU-viable path; running locally on CPU (slower)")
		}
	}

	switch {
	case matchLocal != nil:
		sel := Selection{Provider: matchLocal, Tier: localTier, Reason: fmt.Sprintf("local backend %q (%s)", matchLocal.Name(), localTier)}
		cpuTransition(&sel)
		return sel, nil

	case otherLocal != nil:
		sel := Selection{Provider: otherLocal, Tier: localTier, Reason: fmt.Sprintf("local backend %q (%s)", otherLocal.Name(), localTier)}
		if req.ModelBackend != "" {
			sel.Warnings = append(sel.Warnings, fmt.Sprintf("model's native backend %q unavailable; substituting %q", req.ModelBackend, otherLocal.Name()))
		}
		cpuTransition(&sel)
		return sel, nil

	case comfy != nil:
		sel := Selection{Provider: comfy, Tier: localTier, Reason: fmt.Sprintf("ComfyUI provider %q (%s)", comfy.Name(), localTier)}
		sel.Warnings = append(sel.Warnings, "no standalone backend available; using the optional ComfyUI plug-in")
		cpuTransition(&sel)
		return sel, nil

	case cloud != nil && req.AllowBYOK:
		sel := Selection{Provider: cloud, Tier: TierBYOK, Reason: fmt.Sprintf("BYOK cloud provider %q", cloud.Name())}
		sel.Warnings = append(sel.Warnings, "running on a paid BYOK cloud provider; confirm the cost estimate before proceeding")
		return sel, nil

	case cloud != nil && !req.AllowBYOK:
		return Selection{}, fmt.Errorf("%w for %q: only a BYOK cloud provider is available and BYOK is not enabled for this request", ErrNoneAvailable, req.Operation)

	case !anyAvailable:
		return Selection{}, fmt.Errorf("%w for %q: providers are registered but none is ready (install a model/backend, or enable BYOK)", ErrNoneAvailable, req.Operation)

	default:
		return Selection{}, fmt.Errorf("%w for %q", ErrNoneAvailable, req.Operation)
	}
}
