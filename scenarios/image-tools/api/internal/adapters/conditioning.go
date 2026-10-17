package adapters

import (
	"fmt"
	"sort"

	"image-tools/internal/models"
	"image-tools/internal/safety"
)

// Conditioning resolution (plan capability C, Phase 3). This turns an ordered
// list of requested adapters (AdapterRequest) against a chosen base model into a
// validated, execution-ready []ResolvedAdapter — or fails closed with an
// actionable message. It is pure (no I/O, no catalog ownership): the caller
// supplies the merged catalog lookup + the enabled/installed predicates, so the
// resolver stays testable and the same logic serves the dry-run --explain surface
// and the submit edge.
//
// No vaporware: an adapter is offered for execution ONLY when it is compatible
// with the model's architecture, enabled, installed, AND Ready (its kind×arch
// proven by an attended GPU run). Until a later phase flips Ready, every request
// that names an adapter is honestly rejected here — never silently dropped or run.

// AdapterRequest is one requested conditioning modifier on a generation, exactly
// as it arrives from the wire (untyped scale/keys). The resolver validates it
// into a ResolvedAdapter.
type AdapterRequest struct {
	// ID is the catalog adapter id.
	ID string `json:"id"`
	// Scale is the requested conditioning strength; 0 means "use the adapter's
	// default", and any value is clamped to the adapter's ScaleRange.
	Scale float64 `json:"scale,omitempty"`
	// ConditioningImageKey is the blob key / path of the control or reference
	// image (ControlNet / IP-Adapter). Empty for a LoRA, or for a ControlNet whose
	// conditioning map is auto-derived from the generation input via a preprocessor.
	ConditioningImageKey string `json:"conditioning_image_key,omitempty"`
	// PreprocessorOverride forces a specific ControlNet preprocessor instead of the
	// adapter's declared one (ignored for non-ControlNet kinds).
	PreprocessorOverride Preprocessor `json:"preprocessor_override,omitempty"`
}

// ResolvedAdapter is a validated, execution-ready conditioning modifier: the
// catalog facts the technique/sidecar needs plus the resolved scale, effective
// preprocessor, conditioning image, and (filled by the engine) the on-disk dir.
type ResolvedAdapter struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name,omitempty"`
	Kind                 Kind                `json:"kind"`
	Architecture         models.Architecture `json:"architecture"`
	Scale                float64             `json:"scale"`
	Weight               safety.Weight       `json:"weight"`
	Preprocessor         Preprocessor        `json:"preprocessor,omitempty"`
	ConditioningImageKey string              `json:"conditioning_image_key,omitempty"`
	// Dir is the absolute directory the adapter's weights live in; empty until the
	// engine fills it from the adapters root at execution time.
	Dir string `json:"dir,omitempty"`
}

// kindOrder encodes the deterministic application order (decision C6):
// LoRA (fused into weights) → ControlNet (structural conditioning) → IP-Adapter
// (reference image), so a stack always composes in the same, well-defined way.
func kindOrder(k Kind) int {
	switch k {
	case KindLoRA:
		return 0
	case KindControlNet:
		return 1
	case KindIPAdapter:
		return 2
	default:
		return 3
	}
}

// ResolveConditioning validates an ordered adapter request list against a base
// model's architecture and returns the execution-ready adapters (sorted into the
// canonical application order) plus any non-fatal warnings. It fails closed on the
// first incompatible / disabled / not-installed / not-Ready adapter, or a missing
// required reference image. byID is the merged (seed+custom) catalog lookup;
// enabled/installed may be nil (nil enabled ⇒ seed default; nil installed ⇒ treated
// as not installed, so a request fails honestly until the adapter is installed).
func ResolveConditioning(
	modelArch models.Architecture,
	reqs []AdapterRequest,
	byID func(id string) (Adapter, bool),
	enabled func(id string) bool,
	installed func(id string) bool,
) ([]ResolvedAdapter, []string, error) {
	if len(reqs) == 0 {
		return nil, nil, nil
	}
	if byID == nil {
		return nil, nil, fmt.Errorf("conditioning: no adapter catalog available")
	}
	var (
		out      []ResolvedAdapter
		warnings []string
		seen     = make(map[string]struct{}, len(reqs))
	)
	for _, rq := range reqs {
		if rq.ID == "" {
			return nil, nil, fmt.Errorf("conditioning: an adapter request has an empty id")
		}
		if _, dup := seen[rq.ID]; dup {
			return nil, nil, fmt.Errorf("conditioning: adapter %q requested more than once", rq.ID)
		}
		seen[rq.ID] = struct{}{}

		a, ok := byID(rq.ID)
		if !ok {
			return nil, nil, fmt.Errorf("conditioning: adapter %q not found in the catalog", rq.ID)
		}
		if !a.CompatibleWith(modelArch) {
			return nil, nil, fmt.Errorf("conditioning: adapter %q targets architecture %q, incompatible with the chosen model's architecture %q", a.ID, a.Architecture, modelArch)
		}
		if enabled != nil && !enabled(a.ID) {
			return nil, nil, fmt.Errorf("conditioning: adapter %q is disabled — enable it with `image-tools adapters enable %s`", a.ID, a.ID)
		}
		if !a.Ready {
			reason := a.Pending
			if reason == "" {
				reason = "not yet proven runnable"
			}
			return nil, nil, fmt.Errorf("conditioning: adapter %q is not yet runnable (%s)", a.ID, reason)
		}
		if installed == nil || !installed(a.ID) {
			return nil, nil, fmt.Errorf("conditioning: adapter %q is not installed — run `image-tools adapters install %s`", a.ID, a.ID)
		}

		pre := a.Preprocessor
		if a.Kind == KindControlNet && rq.PreprocessorOverride != "" {
			pre = rq.PreprocessorOverride
		}
		if a.Kind != KindControlNet {
			pre = ""
		}

		// An IP-Adapter's reference image is its prompt and is mandatory. A
		// ControlNet may auto-derive its conditioning map from the generation input
		// via its preprocessor, so a missing image there is allowed (warned).
		if a.Kind == KindIPAdapter && rq.ConditioningImageKey == "" {
			return nil, nil, fmt.Errorf("conditioning: adapter %q (ip-adapter) requires a reference image", a.ID)
		}
		if a.Kind == KindControlNet && rq.ConditioningImageKey == "" {
			if pre == PreprocessorNone || pre == "" {
				return nil, nil, fmt.Errorf("conditioning: adapter %q (controlnet) has no conditioning image and no preprocessor to derive one", a.ID)
			}
			warnings = append(warnings, fmt.Sprintf("controlnet %q will auto-derive its conditioning map from the input image via %s", a.ID, pre))
		}

		scale := rq.Scale
		if scale == 0 {
			scale = a.ScaleRange.Default
		}
		scale = a.ScaleRange.Clamp(scale)

		out = append(out, ResolvedAdapter{
			ID:                   a.ID,
			Name:                 a.Name,
			Kind:                 a.Kind,
			Architecture:         a.Architecture,
			Scale:                scale,
			Weight:               a.Weight,
			Preprocessor:         pre,
			ConditioningImageKey: rq.ConditioningImageKey,
		})
	}

	// Canonical application order (stable: same kind keeps request order).
	sort.SliceStable(out, func(i, j int) bool { return kindOrder(out[i].Kind) < kindOrder(out[j].Kind) })
	return out, warnings, nil
}

// EffectiveWeight returns the consent weight elevation for a base op weight given
// a resolved adapter stack: max(opWeight, adapter weights...) — decision C4.
func EffectiveWeight(opWeight safety.Weight, resolved []ResolvedAdapter) safety.Weight {
	ws := make([]safety.Weight, 0, len(resolved)+1)
	ws = append(ws, opWeight)
	for _, a := range resolved {
		ws = append(ws, a.Weight)
	}
	return safety.MaxWeight(ws...)
}
