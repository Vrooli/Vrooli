// Package adapters owns the declarative image-conditioning adapter catalog —
// LoRA, ControlNet, and IP-Adapter entries that *modify* an existing operation
// on a compatible base model rather than serving an operation themselves. It is
// the sibling of internal/models: a license-verified read-only seed
// (adapters.seed.json), a typed loader + validator, the runtime enabled/install
// overlay (SQLite), the catalog doctor, and the base-model compatibility rule.
//
// Why a separate package (decision C1): an adapter has no operations of its own
// and a different ontology from a base model — it carries a kind, a single
// compatible architecture, a consent weight, and (for ControlNet) a preprocessor.
// Modelling it as a `kind` field on models.Model would force every adapter to
// pretend it serves operations and would entangle two independent governance and
// licensing lifecycles. Adapters share the download/checksum/artifact spine
// (internal/fetch) and the architecture SSOT (internal/models) but nothing else.
//
// No vaporware (CC3): every adapter ships Ready=false until an attended GPU run
// proves that kind×architecture combination produces real output (plan Phases
// 4–6). An un-Ready adapter is inspectable and installable but is never offered
// for execution by the resolver/picker.
package adapters

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"image-tools/internal/fetch"
	"image-tools/internal/models"
	"image-tools/internal/safety"
)

//go:embed adapters.seed.json
var seedBytes []byte

// Kind classifies how an adapter conditions a generation. Each kind has a
// distinct execution contract (LoRA fuses weights; ControlNet adds a structural
// conditioning image; IP-Adapter adds a reference image) proven in a later phase.
type Kind string

const (
	// KindLoRA is a low-rank weight delta fused into the base UNet/text-encoder
	// (style / subject / step-count LoRAs). Self-contained: no extra image input.
	KindLoRA Kind = "lora"
	// KindControlNet is a structural conditioner driven by a preprocessed control
	// image (canny edges, depth, pose, segmentation).
	KindControlNet Kind = "controlnet"
	// KindIPAdapter is an image-prompt adapter driven by a reference image
	// (identity / style transfer).
	KindIPAdapter Kind = "ip-adapter"
)

func (k Kind) valid() bool {
	switch k {
	case KindLoRA, KindControlNet, KindIPAdapter:
		return true
	default:
		return false
	}
}

// Kinds returns the adapter kinds in canonical order.
func Kinds() []Kind { return []Kind{KindLoRA, KindControlNet, KindIPAdapter} }

// Preprocessor names the analysis step a ControlNet runs over a raw input image
// to derive its conditioning map. It is meaningful only for KindControlNet
// (PreprocessorNone elsewhere). canny is a deterministic op (internal/ops);
// depth/pose/segment reuse analysis models. A pre-made conditioning map skips it.
type Preprocessor string

const (
	// PreprocessorNone means no automatic preprocessing (LoRA/IP-Adapter, or a
	// ControlNet fed a pre-made conditioning map).
	PreprocessorNone Preprocessor = "none"
	// PreprocessorCanny derives a Canny edge map (deterministic; internal/ops).
	PreprocessorCanny Preprocessor = "canny"
	// PreprocessorDepth derives a depth map (reuses the depth_map analysis op).
	PreprocessorDepth Preprocessor = "depth"
	// PreprocessorPose derives an OpenPose skeleton (pose_estimate model).
	PreprocessorPose Preprocessor = "pose"
	// PreprocessorSegment derives a segmentation map (reuses the segment op).
	PreprocessorSegment Preprocessor = "segment"
)

func (p Preprocessor) valid() bool {
	switch p {
	case PreprocessorNone, PreprocessorCanny, PreprocessorDepth, PreprocessorPose, PreprocessorSegment:
		return true
	default:
		return false
	}
}

// ScaleRange is the allowed conditioning-strength range for an adapter, with the
// default applied when a request omits an explicit scale. Min<=Default<=Max.
type ScaleRange struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Default float64 `json:"default"`
}

// Clamp returns v constrained to [Min, Max]. A zero/absent range (all fields 0)
// is treated as "no clamp" so a caller that never set a range is unaffected.
func (r ScaleRange) Clamp(v float64) float64 {
	if r.Min == 0 && r.Max == 0 && r.Default == 0 {
		return v
	}
	if v < r.Min {
		return r.Min
	}
	if v > r.Max {
		return r.Max
	}
	return v
}

// Source records where an adapter's weights come from, reusing the shared fetch
// spine. Like a model, an adapter installs from direct Assets, a pinned Repo
// snapshot, or a local path (mutually exclusive); the checksum is pinned on first
// real download (never hand-written).
type Source struct {
	// Assets are direct, resolvable weight artifacts (a single .safetensors for a
	// LoRA / IP-Adapter; the ControlNet files). Artifact-validated on install.
	Assets []fetch.Asset `json:"assets,omitempty"`
	// Repo is a multi-file diffusers repo snapshot (some ControlNets ship this
	// way), fetched in whole at Repo.Revision. Mutually exclusive with Assets.
	Repo fetch.RepoSpec `json:"repo,omitempty"`
	// LocalPath points at already-present weights (custom/imported adapters).
	LocalPath string `json:"local_path,omitempty"`
	// SourceRepo / DocsURL are documentation-only provenance pointers.
	SourceRepo string `json:"source_repo,omitempty"`
	DocsURL    string `json:"docs_url,omitempty"`
	// Checksum is the integrity record, pinned on first download.
	Checksum models.Checksum `json:"checksum"`
}

// HasRepo reports whether the source installs from a multi-file repo snapshot.
func (s Source) HasRepo() bool { return strings.TrimSpace(s.Repo.RepoID) != "" }

// HasFetchStrategy reports whether the adapter declares a concrete, auto-
// resolvable install source (assets, a pinned repo, or a local path). An adapter
// with none cannot be installed and is caught by the catalog doctor.
func (s Source) HasFetchStrategy() bool {
	return strings.TrimSpace(s.LocalPath) != "" || s.HasRepo() || len(s.Assets) > 0
}

// Adapter is one catalog entry: a conditioning modifier compatible with exactly
// one base architecture.
type Adapter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind Kind   `json:"kind"`
	// Architecture is the SINGLE base architecture this adapter conditions. The
	// compatibility rule is data: adapter.Architecture == model.Architecture
	// (decision CC2). It reuses the models architecture SSOT so the two never drift.
	Architecture models.Architecture `json:"architecture"`
	// Weight is the consent weight this adapter contributes (decision C4): the
	// effective op weight is max(op, adapters...). IP-Adapter (identity/style) and
	// identity-class LoRAs carry WeightHigh; structural ControlNets carry none.
	Weight safety.Weight `json:"weight"`
	// Preprocessor is the control-image derivation for a ControlNet (PreprocessorNone
	// otherwise). A raw image is auto-preprocessed through it (decision C3); a
	// pre-made map bypasses it.
	Preprocessor Preprocessor `json:"preprocessor,omitempty"`
	// ScaleRange bounds the conditioning strength + default.
	ScaleRange       ScaleRange              `json:"scale_range"`
	SizeMBApprox     int                     `json:"size_mb_approx"`
	Source           Source                  `json:"source"`
	CapabilityLabels models.CapabilityLabels `json:"capability_labels"`
	// Ready reports the adapter's execution is proven on its architecture (no
	// vaporware). False until the attended GPU run flips it; an un-Ready adapter is
	// never offered for execution.
	Ready bool `json:"ready"`
	// Pending explains, for a not-Ready adapter, what blocks proving it.
	Pending string `json:"pending,omitempty"`
	// Enabled is the seed default enabled state (overlaid at runtime by the store).
	Enabled bool `json:"enabled"`
}

// RequiresReferenceImage reports whether running this adapter needs a caller-
// supplied reference/conditioning image. IP-Adapters always do (the reference is
// the prompt); ControlNets do unless a preprocessor can derive the map from the
// generation input — but the conditioning still rides an image input, so both
// ControlNet and IP-Adapter require one. LoRAs never do.
func (a Adapter) RequiresReferenceImage() bool {
	return a.Kind == KindControlNet || a.Kind == KindIPAdapter
}

// CompatibleWith reports whether this adapter can condition a model of the given
// architecture (the data compatibility rule, decision CC2).
func (a Adapter) CompatibleWith(arch models.Architecture) bool {
	return a.Architecture != "" && a.Architecture == arch
}

// seedFile mirrors the on-disk adapters.seed.json top-level shape.
type seedFile struct {
	SchemaVersion string    `json:"schema_version"`
	Adapters      []Adapter `json:"adapters"`
}

// Registry is the validated, indexed view of the adapter catalog.
type Registry struct {
	schemaVersion string
	adapters      []Adapter
	byID          map[string]Adapter
}

// Load parses and validates the embedded seed catalog, additionally asserting the
// seed-integrity invariants (license discipline, no-vaporware) so a bad edit
// fails loud at boot rather than shipping a non-compliant adapter.
func Load() (*Registry, error) {
	r, err := Parse(seedBytes)
	if err != nil {
		return nil, err
	}
	if err := r.validateSeedInvariants(); err != nil {
		return nil, fmt.Errorf("adapters seed invariant: %w", err)
	}
	return r, nil
}

// Parse builds a Registry from raw JSON, enforcing structural validity for any
// entry (seed or custom overlay). Malformed entries are rejected.
func Parse(data []byte) (*Registry, error) {
	var sf seedFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("decode adapter catalog: %w", err)
	}
	if sf.SchemaVersion == "" {
		return nil, fmt.Errorf("missing schema_version")
	}
	r := &Registry{
		schemaVersion: sf.SchemaVersion,
		byID:          make(map[string]Adapter, len(sf.Adapters)),
	}
	for i := range sf.Adapters {
		a := sf.Adapters[i]
		if err := validateAdapter(a); err != nil {
			return nil, fmt.Errorf("adapter %q: %w", a.ID, err)
		}
		if _, dup := r.byID[a.ID]; dup {
			return nil, fmt.Errorf("duplicate adapter id %q", a.ID)
		}
		r.adapters = append(r.adapters, a)
		r.byID[a.ID] = a
	}
	return r, nil
}

// validateAdapter enforces per-entry structural validity used for any entry.
func validateAdapter(a Adapter) error {
	if strings.TrimSpace(a.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("missing name")
	}
	if !a.Kind.valid() {
		return fmt.Errorf("invalid kind %q", a.Kind)
	}
	// An adapter conditions exactly one concrete architecture; ArchNone/empty is a
	// nonsense compatibility key (it would match nothing) and is rejected.
	if a.Architecture == "" || a.Architecture == models.ArchNone || !a.Architecture.Valid() {
		return fmt.Errorf("invalid architecture %q (must be a concrete base architecture)", a.Architecture)
	}
	if !a.Weight.Valid() {
		return fmt.Errorf("invalid weight %q", a.Weight)
	}
	if a.Preprocessor != "" && !a.Preprocessor.valid() {
		return fmt.Errorf("invalid preprocessor %q", a.Preprocessor)
	}
	// A preprocessor only makes sense for a ControlNet; any other kind must not
	// declare one (other than the explicit none).
	if a.Kind != KindControlNet && a.Preprocessor != "" && a.Preprocessor != PreprocessorNone {
		return fmt.Errorf("preprocessor %q is only valid for a controlnet adapter", a.Preprocessor)
	}
	if err := validateScaleRange(a.ScaleRange); err != nil {
		return err
	}
	if a.SizeMBApprox < 0 {
		return fmt.Errorf("negative size_mb_approx")
	}
	if !a.CapabilityLabels.CommercialUse.Valid() {
		return fmt.Errorf("invalid commercial_use %q", a.CapabilityLabels.CommercialUse)
	}
	if len(a.Source.Assets) > 0 && a.Source.HasRepo() {
		return fmt.Errorf("source declares both assets and repo (mutually exclusive fetch strategies)")
	}
	return nil
}

func validateScaleRange(r ScaleRange) error {
	if r.Min == 0 && r.Max == 0 && r.Default == 0 {
		return fmt.Errorf("scale_range is required (min/max/default)")
	}
	if r.Min < 0 || r.Max < 0 {
		return fmt.Errorf("scale_range has a negative bound")
	}
	if r.Max < r.Min {
		return fmt.Errorf("scale_range max %.3g is below min %.3g", r.Max, r.Min)
	}
	if r.Default < r.Min || r.Default > r.Max {
		return fmt.Errorf("scale_range default %.3g is outside [%.3g, %.3g]", r.Default, r.Min, r.Max)
	}
	return nil
}

// validateSeedInvariants asserts the policy guarantees the bundled seed upholds:
// license discipline (never seed an outright non-commercial adapter; conditional
// stays opt-in with notes) and no-vaporware (every seed adapter ships Ready=false
// until a later phase proves it, carrying a Pending reason).
func (r *Registry) validateSeedInvariants() error {
	for _, a := range r.adapters {
		switch a.CapabilityLabels.CommercialUse {
		case models.CommercialUseNo:
			return fmt.Errorf("adapter %q is commercial_use=no (commercial-clean gate)", a.ID)
		case models.CommercialUseConditional:
			if a.Enabled {
				return fmt.Errorf("adapter %q is commercial_use=conditional and must NOT be enabled by default", a.ID)
			}
			if a.CapabilityLabels.CommercialUseNotes == "" {
				return fmt.Errorf("adapter %q is commercial_use=conditional but carries no commercial_use_notes", a.ID)
			}
		}
		if a.Ready {
			return fmt.Errorf("adapter %q ships Ready=true in the seed; readiness is flipped only after an attended GPU run proves the kind×architecture (no vaporware)", a.ID)
		}
		if strings.TrimSpace(a.Pending) == "" {
			return fmt.Errorf("adapter %q is not Ready but declares no pending reason", a.ID)
		}
	}
	return nil
}

// SchemaVersion returns the catalog schema version.
func (r *Registry) SchemaVersion() string { return r.schemaVersion }

// Adapters returns all catalog entries in seed order.
func (r *Registry) Adapters() []Adapter { return append([]Adapter(nil), r.adapters...) }

// ByID returns the adapter with the given id.
func (r *Registry) ByID(id string) (Adapter, bool) {
	a, ok := r.byID[id]
	return a, ok
}

// ByKind returns every adapter of the given kind, in seed order.
func (r *Registry) ByKind(k Kind) []Adapter {
	var out []Adapter
	for _, a := range r.adapters {
		if a.Kind == k {
			out = append(out, a)
		}
	}
	return out
}

// Compatible returns every adapter compatible with arch, in seed order. It is the
// raw architecture filter; the resolver/picker further restrict to installed +
// Ready before offering an adapter for execution.
func (r *Registry) Compatible(arch models.Architecture) []Adapter {
	var out []Adapter
	for _, a := range r.adapters {
		if a.CompatibleWith(arch) {
			out = append(out, a)
		}
	}
	return out
}

// Architectures returns the distinct architectures the catalog has adapters for,
// sorted (discovery helper).
func (r *Registry) Architectures() []models.Architecture {
	seen := make(map[models.Architecture]struct{}, len(r.adapters))
	for _, a := range r.adapters {
		seen[a.Architecture] = struct{}{}
	}
	out := make([]models.Architecture, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
