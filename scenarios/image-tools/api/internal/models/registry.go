// Package models owns the declarative image-model registry (OT-P0-006): the
// license-verified seed catalog (registry.seed.json), its typed loader +
// validator, and the hardware-fit selector that picks the best enabled model
// for an operation given the host probed via internal/capabilities.
//
// The seed is the read-only baseline. Runtime state (enabled/installed/checksum)
// is overlaid from SQLite by the management layer; the loader here never mutates
// the seed. See docs/internal/DECISIONS.md for the policy behind the catalog and
// docs/reference/model-registry.md for the human-readable catalog + blocklist.
package models

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"image-tools/internal/capability"
	"image-tools/internal/fetch"
)

//go:embed registry.seed.json
var seedBytes []byte

// Tier ranks a model's quality/cost class. Lower tiers are the CPU-capable,
// commercial-clean defaults that ship enabled; "quality" tiers are heavier and
// seed disabled (opt-in).
type Tier string

const (
	TierDefault        Tier = "default"
	TierDefaultVariant Tier = "default-variant"
	TierQuality        Tier = "quality"
	TierNiceToHave     Tier = "nice-to-have"
)

func (t Tier) valid() bool {
	switch t {
	case TierDefault, TierDefaultVariant, TierQuality, TierNiceToHave:
		return true
	default:
		return false
	}
}

// rank orders tiers for selection. Higher is "better output" when the host can
// run it. Defaults sit above nice-to-have but below quality.
func (t Tier) rank() int {
	switch t {
	case TierQuality:
		return 3
	case TierDefaultVariant:
		return 2
	case TierDefault:
		return 1
	default: // nice-to-have / unknown
		return 0
	}
}

// CommercialUse classifies the model's commercial-use license posture.
type CommercialUse string

const (
	CommercialUseYes         CommercialUse = "yes"
	CommercialUseNo          CommercialUse = "no"
	CommercialUseConditional CommercialUse = "conditional"
)

func (c CommercialUse) valid() bool {
	switch c {
	case CommercialUseYes, CommercialUseNo, CommercialUseConditional:
		return true
	default:
		return false
	}
}

// Valid reports whether c is a known commercial-use enum member. Exported so the
// sibling adapter catalog (internal/adapters), which reuses CapabilityLabels, can
// validate its seed against this SSOT without duplicating the enum.
func (c CommercialUse) Valid() bool { return c.valid() }

// Hardware captures a model's host requirements. MinVRAMGB == 0 means "no
// dedicated VRAM required" (a CPU-capable tier), not "unknown".
type Hardware struct {
	CPUCapable  bool `json:"cpu_capable"`
	GPURequired bool `json:"gpu_required"`
	MinVRAMGB   int  `json:"min_vram_gb"`
	// ResidentVRAMGB is the VRAM a run needs to keep every stage GPU-resident
	// instead of streaming weights from host RAM. It is a preference above
	// MinVRAMGB (the streamed floor), never a requirement: admission asks for it
	// first, which lets the capacity broker reclaim idle resident models, and
	// falls back to the floor when it is not granted. Zero means the model has
	// no faster placement to ask for.
	ResidentVRAMGB int      `json:"resident_vram_gb,omitempty"`
	MinRAMGB       int      `json:"min_ram_gb"`
	OSArch         []string `json:"os_arch"`
	SpeedNote      string   `json:"speed_note"`
}

// IO describes native input/output dimensions and constraints.
type IO struct {
	NativeResolution string `json:"native_resolution"`
	NativeScale      string `json:"native_scale"`
	Notes            string `json:"notes"`
}

// CapabilityLabels carries the safety/license metadata surfaced to operators.
type CapabilityLabels struct {
	NSFWCapable        bool          `json:"nsfw_capable"`
	License            string        `json:"license"`
	CommercialUse      CommercialUse `json:"commercial_use"`
	CommercialUseNotes string        `json:"commercial_use_notes"`
	BaseModelLineage   string        `json:"base_model_lineage"`
	KnownRisks         string        `json:"known_risks"`
	// ExplicitUseOnly keeps an entry out of automatic selection: it runs only when a
	// caller names it (--model, a pinned per-operation default, a role). It is how an
	// uncensored or otherwise opt-in model stays installed and enabled without ever
	// being picked for an ordinary request that did not ask for it.
	ExplicitUseOnly bool `json:"explicit_use_only,omitempty"`
	// Provenance records how the entry entered the catalog. Empty for seed models;
	// ProvenanceUserImported for entries added via the guided import flow (plan
	// capability D). The safety tier gate consults it: a user-imported entry with
	// an unverified license is refused on the public/BYOK serving tier unless the
	// operator attested commercial rights (decision D4).
	Provenance string `json:"provenance,omitempty"`
}

// ProvenanceUserImported marks a catalog entry added via the guided model-import
// flow (not part of the license-verified seed). See CapabilityLabels.Provenance.
const ProvenanceUserImported = "user-imported"

type Checksum = fetch.Checksum

// The downloadable-weight vocabulary (ArtifactKind + Asset) lives in the shared
// internal/fetch spine so the model and adapter catalogs describe their artifacts
// with one type. The model catalog re-exports them under their historical names
// so the seed schema (registry.seed.json) and every consumer are unchanged.
type (
	// ArtifactKind classifies a downloadable weight artifact so the installer can
	// validate the bytes it fetched are actually that kind of file. See fetch.
	ArtifactKind = fetch.ArtifactKind
	// Asset is one downloadable artifact a model needs on disk. See fetch.Asset.
	Asset = fetch.Asset
)

const (
	// ArtifactGeneric applies the page/size guards but no format-specific magic.
	ArtifactGeneric = fetch.ArtifactGeneric
	// ArtifactONNX is an ONNX model (protobuf; starts with the ir_version tag).
	ArtifactONNX = fetch.ArtifactONNX
	// ArtifactGGUF is a llama.cpp/stable-diffusion.cpp GGUF weight.
	ArtifactGGUF = fetch.ArtifactGGUF
	// ArtifactSafetensors is a safetensors weight file.
	ArtifactSafetensors = fetch.ArtifactSafetensors
	// ArtifactNCNNParam / ArtifactNCNNBin are the ncnn model pair (realesrgan).
	ArtifactNCNNParam = fetch.ArtifactNCNNParam
	ArtifactNCNNBin   = fetch.ArtifactNCNNBin
	// ArtifactBinary is a standalone executable artifact.
	ArtifactBinary = fetch.ArtifactBinary
)

// Runtime declares HOW a weight-backed model executes, so a single generic
// sidecar runner can dispatch the correct pipeline class + call kwargs from data
// instead of hardcoded per-model Python (the diffusers backend previously
// supported exactly one model because the class + kwargs were baked into
// edit_instruct.py). It is required for diffusers-backed models — the catalog
// doctor asserts an enabled diffusers model names a registered family adapter —
// and is the zero value for backends that need no such dispatch.
type Runtime struct {
	// Family keys the sidecar adapter table AND the Go-side registered-family set
	// (families.go). One family per pipeline architecture; a genuinely new
	// architecture is one new adapter row, an existing one is reuse.
	Family string `json:"family"`
	// PipelineClass is the diffusers pipeline class the family loads/expects
	// (e.g. "QwenImageEditPlusPipeline"). diffusers auto-resolves the class from
	// model_index.json on load; this is asserted post-load and documents intent.
	PipelineClass string `json:"pipeline_class"`
	// Dtype is the torch dtype the family loads in ("float16", "bfloat16"). Empty
	// means the sidecar's device default (fp16 on cuda, fp32 on cpu).
	Dtype string `json:"dtype"`
	// Variant selects an upstream Diffusers weight variant when a pinned repo
	// intentionally omits full-precision weights (for example fp16 safetensors).
	Variant string `json:"variant"`
	// MinRuntime is the minimum runtime/library constraint surfaced to the
	// provision/availability check (e.g. "diffusers>=0.36.0"; a git ref for a
	// bleeding-edge family). Phase 4 enforces it; informational otherwise.
	MinRuntime string `json:"min_runtime"`
	// GatewayRole names the AI Gateway role a gateway-served vision model runs
	// through (see GatewayServed). The gateway's resource policy maps the role to
	// a concrete model (an Ollama vision model locally, a hosted one remotely), so
	// switching or adding a general vision model is this one field on a row.
	GatewayRole string `json:"gateway_role,omitempty"`
	// BoxOrder and BoxScale say how the model writes detection boxes, because
	// vision LLMs disagree: "xyxy" is [left, top, right, bottom] and "yxyx" is
	// [top, left, bottom, right]; BoxScale is the grid the numbers live on (1000
	// for the normalised 0..1000 grid most use, 0 for source pixels).
	BoxOrder string `json:"box_order,omitempty"`
	BoxScale int    `json:"box_scale,omitempty"`
}

// Box orders a gateway vision model can declare.
const (
	BoxOrderXYXY = "xyxy"
	BoxOrderYXYX = "yxyx"
)

// VisionOperations are the operations a general vision model can serve through
// the gateway: each is a question asked of the image in words and answered in
// text that the provider parses into the operation's output contract.
var VisionOperations = []string{"caption", "object_detection", "tagging"}

// IsVisionOperation reports whether op is one a gateway vision model serves.
func IsVisionOperation(op string) bool {
	for _, v := range VisionOperations {
		if v == op {
			return true
		}
	}
	return false
}

// GatewayServed reports whether this model's vision operations run through an AI
// Gateway role rather than weights image-tools manages: a local vision model the
// gateway reaches through Ollama (backend gateway-vision) or a hosted one (the
// openrouter backend with a gateway role).
func (m Model) GatewayServed() bool {
	return m.Backend == BackendGatewayVision || (m.Backend == BackendOpenRouter && strings.TrimSpace(m.Runtime.GatewayRole) != "")
}

// RepoSource is a HuggingFace-style multi-file model repository — the install
// shape diffusers models use. It is the shared fetch.RepoSpec, re-exported under
// the catalog's historical name so the seed schema is unchanged.
type RepoSource = fetch.RepoSpec

type Source = fetch.Source

// HasFetchStrategy reports whether a weight-backed model declares a concrete,
// auto-resolvable install source (direct assets, a pinned repo snapshot, or a
// local path). A model with none is installable only if it is honestly marked
// Manual; otherwise its download_url is a documentation page the installer must
// never fetch. Weightless models (builtin/computed/library) need no strategy.
func (m Model) HasFetchStrategy() bool {
	if !m.RequiresWeights() {
		return true
	}
	if strings.TrimSpace(m.Source.LocalPath) != "" || m.Source.HasRepo() || len(m.Source.Assets) > 0 {
		return true
	}
	return false
}

// Model is one registry entry: a concrete model/library backing one or more
// operations.
type Model struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Operations       []string         `json:"operations"`
	DefaultFor       []string         `json:"default_for"`
	Architecture     Architecture     `json:"architecture,omitempty"`
	Tier             Tier             `json:"tier"`
	Backend          string           `json:"backend"`
	AltBackends      []string         `json:"alt_backends"`
	RequiresComfyUI  bool             `json:"requires_comfyui"`
	SizeMBApprox     int              `json:"size_mb_approx"`
	QuantVariants    []string         `json:"quant_variants"`
	Hardware         Hardware         `json:"hardware"`
	IO               IO               `json:"io"`
	CapabilityLabels CapabilityLabels `json:"capability_labels"`
	Source           Source           `json:"source"`
	// Rights carries the source-backed shared evaluator record. RawMessage keeps
	// unknown rights facts intact until the scenario can depend on the shared
	// evaluator package; selection must fail closed when this record is absent.
	Rights json.RawMessage `json:"rights"`
	// BuildProfile is a closed, built-in derivation recipe for an artifact that
	// the installer can deterministically build from pinned Source.Assets. It is
	// not a general-purpose command runner.
	BuildProfile *BuildProfile `json:"build_profile,omitempty"`
	Runtime      Runtime       `json:"runtime"`
	Enabled      bool          `json:"enabled"`
	// Disabled is the visible reason a seeded row is off. The seed requires it on
	// every enabled=false row and forbids it on enabled rows.
	Disabled *DisabledReason `json:"disabled,omitempty"`
}

// BuildProfile describes one reviewed model-export shape. Its values are
// validated against the single supported LaMa profile in validateModel.
type BuildProfile struct {
	Kind             string `json:"kind"`
	SourceAsset      string `json:"source_asset"`
	SourceSHA256     string `json:"source_sha256"`
	CheckpointAsset  string `json:"checkpoint_asset"`
	CheckpointSHA256 string `json:"checkpoint_sha256"`
	Resolution       int    `json:"resolution"`
	Opset            int    `json:"opset"`
	OutputFilename   string `json:"output_filename"`
	OutputSHA256     string `json:"output_sha256,omitempty"`
}

// BackendBuiltin is the backend name for deterministic, in-process models that
// ship in the binary and need no downloaded weights (e.g. the naturalize detail
// compositor). The ai package registers an always-available builtin provider
// under this name, and such models are always "installed" (RequiresWeights is
// false) so the install/provisioning flow skips them.
const BackendBuiltin = "builtin"

const (
	BackendComputed  = "computed"
	BackendLibraryGo = "library-go"
	// BackendDiffusers is the python diffusers sidecar backend. edit_instruct
	// models on this backend run through the generic _diffusers runner and so
	// must declare a registered, runnable runtime family (DoctorCatalog enforces).
	BackendDiffusers = "diffusers"
	// BackendOpenRouter is the BYOK cloud backend: image generation/editing
	// served by OpenRouter image-capable models. It holds no local weights (the
	// model lives on the provider), so it is always "installed"; runnability is
	// gated instead by the cloud provider's Available() (a usable API key).
	BackendOpenRouter = "openrouter"
	// BackendGatewayVision is a general vision-language model served locally by
	// AI Gateway (Ollama behind a gateway role). The weights belong to the
	// gateway's Ollama resource, not to image-tools managed acquisition, so the
	// row is always "installed"; runnability is gated by the gateway being
	// reachable and the role resolving to a model.
	BackendGatewayVision = "gateway-vision"
	// BackendMaskFill is the standalone builtin object-removal provider: a
	// deterministic membrane fill that is always available, so object-removal
	// works on a host with no iopaint. Ranked below iopaint for quality.
	BackendMaskFill = "mask-fill"
	// BackendTesseract is the Tesseract OCR command line.
	BackendTesseract = "tesseract"
)

// RequiresWeights reports whether a model needs downloaded artifacts on disk to
// run. In-process deterministic/Go-library models and the BYOK cloud backend
// have no local weights and are always considered installed; everything else
// must be fetched + artifact-validated by the Installer before an AI op will
// launch.
func (m Model) RequiresWeights() bool {
	switch m.Backend {
	case BackendBuiltin, BackendComputed, BackendLibraryGo, BackendOpenRouter, BackendGatewayVision, BackendMaskFill:
		return false
	default:
		return true
	}
}

// WeightlessInstallRecord is the install state of a model that downloads no
// weights (remote, builtin, computed and pure-Go providers): there is nothing to
// install, so it is installed from the start. The installer's install/lifecycle
// path and the wire list/get views all read this one rule, so `models list`
// cannot report not-installed for a model `models install` calls already
// installed.
func WeightlessInstallRecord(m Model) (InstallRecord, bool) {
	if m.RequiresWeights() {
		return InstallRecord{}, false
	}
	return InstallRecord{ID: m.ID, Installed: true}, true
}

// ServesOperation reports whether this model can run the given operation today:
// natively (declared) OR through a PROVEN derived technique of its architecture.
// A derived op whose technique is not yet Ready is intentionally NOT served (it
// is surfaced honestly elsewhere as derived_pipeline_unproven, never silently
// run). This is the literal-array gate's replacement — capability is now a
// derivation over the architecture table, not a hand-list.
func (m Model) ServesOperation(op string) bool {
	for _, eo := range m.EffectiveOps() {
		if eo.Op == op && eo.Offerable() {
			return true
		}
	}
	return false
}

// IsDefaultFor reports whether this model is the seeded default for op.
func (m Model) IsDefaultFor(op string) bool {
	for _, o := range m.DefaultFor {
		if o == op {
			return true
		}
	}
	return false
}

// GateKind says what a blocklist entry restricts.
type GateKind string

const (
	// GateLicense restricts only the uses a license leaves out (monetized,
	// hosted-for-others, redistributed). Normal local use is never gated by a
	// license (operator stance 2026-11-21), so a catalog row may carry a
	// license-gated model; its rights facts must then agree with the gate so the
	// shared rights evaluator refuses the gated uses.
	GateLicense GateKind = "license"
	// GateExclusion keeps a model out of the catalog entirely, for a safety,
	// technical or hardware-fit reason that holds regardless of license.
	GateExclusion GateKind = "exclusion"
)

// ExclusionKind is the non-license reason behind a GateExclusion entry.
type ExclusionKind string

const (
	ExclusionSafety      ExclusionKind = "safety"
	ExclusionTechnical   ExclusionKind = "technical"
	ExclusionHardwareFit ExclusionKind = "hardware_fit"
)

var exclusionKinds = map[ExclusionKind]bool{ExclusionSafety: true, ExclusionTechnical: true, ExclusionHardwareFit: true}

// licenseGatedUses are the rights uses a license gate may restrict: every use
// the shared evaluator keeps license-bound. Normal local inference is absent on
// purpose; a gate listing it is rejected at parse time.
var licenseGatedUses = map[string]bool{
	RightsUseMonetizedUse:        true,
	RightsUseHostedInference:     true,
	RightsUseRedistributeWeights: true,
	"redistribute_derivative":    true,
	"distribute_runtime":         true,
}

// BlocklistEntry records a model that is restricted, with the reason. A
// GateLicense entry is a monetization/redistribution gate, not a normal-use
// exclusion; a GateExclusion entry is a real block.
type BlocklistEntry struct {
	ID         string   `json:"id"`
	Operations []string `json:"operations"`
	License    string   `json:"license"`
	Reason     string   `json:"reason"`
	Gate       GateKind `json:"gate"`
	// GatedUses lists the rights uses the license refuses (license gates only).
	GatedUses []string `json:"gated_uses,omitempty"`
	// ExclusionKind names why an exclusion holds (exclusion gates only).
	ExclusionKind                   ExclusionKind `json:"exclusion_kind,omitempty"`
	ExportingONNXRemovesRestriction bool          `json:"exporting_onnx_removes_restriction"`
}

// IsExclusion reports whether the entry keeps its model out of the catalog.
func (b BlocklistEntry) IsExclusion() bool { return b.Gate == GateExclusion }

func (b BlocklistEntry) validate() error {
	if strings.TrimSpace(b.Reason) == "" {
		return fmt.Errorf("blocklist entry %q has no reason", b.ID)
	}
	switch b.Gate {
	case GateLicense:
		if len(b.GatedUses) == 0 {
			return fmt.Errorf("license gate %q lists no gated_uses", b.ID)
		}
		for _, use := range b.GatedUses {
			if !licenseGatedUses[use] {
				return fmt.Errorf("license gate %q lists %q: a license gate restricts only monetized, hosted or redistributed use, never normal local inference", b.ID, use)
			}
		}
		if b.ExclusionKind != "" {
			return fmt.Errorf("license gate %q must not set exclusion_kind", b.ID)
		}
	case GateExclusion:
		if !exclusionKinds[b.ExclusionKind] {
			return fmt.Errorf("exclusion %q needs exclusion_kind safety, technical or hardware_fit (got %q); a license-only reason is a license gate", b.ID, b.ExclusionKind)
		}
		if len(b.GatedUses) != 0 {
			return fmt.Errorf("exclusion %q must not list gated_uses", b.ID)
		}
	default:
		return fmt.Errorf("blocklist entry %q has gate %q, want license or exclusion", b.ID, b.Gate)
	}
	return nil
}

// licenseGateAgrees reports whether a catalog row carrying a license-gated
// model has rights facts that keep the gated uses refused: it must not claim
// commercial use, and its shared rights record must not be unrestricted.
func licenseGateAgrees(m Model) error {
	if m.CapabilityLabels.CommercialUse == CommercialUseYes {
		return fmt.Errorf("model %q is license-gated but labelled commercial_use=yes", m.ID)
	}
	var facts struct {
		Classification string `json:"classification"`
	}
	if len(m.Rights) == 0 {
		return fmt.Errorf("model %q is license-gated but carries no rights record for the shared evaluator", m.ID)
	}
	if err := json.Unmarshal(m.Rights, &facts); err != nil {
		return fmt.Errorf("model %q rights record: %w", m.ID, err)
	}
	if facts.Classification == "" || facts.Classification == "unrestricted" {
		return fmt.Errorf("model %q is license-gated but its rights classification is %q, which would grant the gated uses", m.ID, facts.Classification)
	}
	return nil
}

// seedFile mirrors the on-disk registry.seed.json top-level shape. The operation
// vocabulary is NOT declared here — it lives in the SSOT (internal/operations);
// the registry reads it from there and validates every model op against it.
type seedFile struct {
	SchemaVersion string           `json:"schema_version"`
	Models        []Model          `json:"models"`
	Blocklist     []BlocklistEntry `json:"blocklist"`
}

// Registry is the validated, indexed view of the model catalog.
type Registry struct {
	schemaVersion string
	models        []Model
	byID          map[string]Model
	byOperation   map[string][]Model // op -> models serving it (seed order)
	defaultFor    map[string]string  // op -> default model id
	vocab         map[string]struct{}
	vocabOrder    []string
	blocklist     []BlocklistEntry
	blockByID     map[string]BlocklistEntry
}

// Load parses and validates the embedded seed catalog. It additionally asserts
// the seed-integrity invariants (restrictive licenses carry notes, ComfyUI
// optional, every op has a default, exclusions disjoint from the catalog,
// license-gated rows' rights agree with their gate) so a bad edit to the seed
// fails loud at boot rather than silently shipping a non-compliant model.
func Load() (*Registry, error) {
	r, err := Parse(seedBytes)
	if err != nil {
		return nil, err
	}
	if err := r.validateSeedInvariants(); err != nil {
		return nil, fmt.Errorf("registry seed invariant: %w", err)
	}
	return r, nil
}

// Parse builds a Registry from raw JSON, enforcing structural validity (the
// rules a usable, well-formed catalog must satisfy regardless of whether it is
// the bundled seed or a user/custom overlay). Malformed entries are rejected.
func Parse(data []byte) (*Registry, error) {
	var sf seedFile
	// Unknown top-level keys in the seed (field_reference, checksum_policy,
	// notes, …) are documentation-only and intentionally ignored.
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if sf.SchemaVersion == "" {
		return nil, fmt.Errorf("missing schema_version")
	}
	if len(sf.Models) == 0 {
		return nil, fmt.Errorf("registry has no models")
	}

	// The operation vocabulary is the capability manifest set; the registry reads
	// it rather than carrying its own copy. Per-model operations[] are validated
	// against it below.
	vocabOrder := capability.VocabularyNames()
	r := &Registry{
		schemaVersion: sf.SchemaVersion,
		byID:          make(map[string]Model, len(sf.Models)),
		byOperation:   make(map[string][]Model),
		defaultFor:    make(map[string]string),
		vocab:         make(map[string]struct{}, len(vocabOrder)),
		vocabOrder:    vocabOrder,
		blockByID:     make(map[string]BlocklistEntry, len(sf.Blocklist)),
	}
	for _, op := range vocabOrder {
		r.vocab[op] = struct{}{}
	}

	for i := range sf.Models {
		m := sf.Models[i]
		if err := r.validateModel(m); err != nil {
			return nil, fmt.Errorf("model %q: %w", m.ID, err)
		}
		if _, dup := r.byID[m.ID]; dup {
			return nil, fmt.Errorf("duplicate model id %q", m.ID)
		}
		r.models = append(r.models, m)
		r.byID[m.ID] = m
		// Index by the OFFERABLE effective op set: declared natives plus any
		// architecture-derived op whose technique is proven (Ready). Today every
		// derivation is Ready=false, so this is exactly m.Operations; once the
		// attended GPU run proves a derivation, that model auto-joins the op's
		// picker with no per-model code.
		for _, eo := range m.EffectiveOps() {
			if eo.Offerable() {
				r.byOperation[eo.Op] = append(r.byOperation[eo.Op], m)
			}
		}
		for _, op := range m.DefaultFor {
			if existing, ok := r.defaultFor[op]; ok {
				return nil, fmt.Errorf("operation %q has two defaults: %q and %q", op, existing, m.ID)
			}
			r.defaultFor[op] = m.ID
		}
	}

	for _, b := range sf.Blocklist {
		if b.ID == "" {
			return nil, fmt.Errorf("blocklist entry with empty id")
		}
		if _, dup := r.blockByID[b.ID]; dup {
			return nil, fmt.Errorf("duplicate blocklist id %q", b.ID)
		}
		if err := b.validate(); err != nil {
			return nil, err
		}
		r.blocklist = append(r.blocklist, b)
		r.blockByID[b.ID] = b
	}

	return r, nil
}

// WithCustom returns a shallow copy of the registry with the given custom
// (user-imported, add-only) models merged in: each becomes resolvable by id and
// joins its offerable ops' candidate lists, but NEVER becomes an op's default
// (the add-only, no-seed-shadow discipline — defaults stay seed-owned). A custom
// id colliding with an existing id, or a model failing validation, is an error.
// The receiver is left unchanged.
//
// The AI engine builds this per request from the custom-model store so a model
// imported by `models import` is resolvable for generation immediately after
// install — the seed registry alone (Load) does not know about runtime imports.
func (r *Registry) WithCustom(customs []Model) (*Registry, error) {
	if len(customs) == 0 {
		return r, nil
	}
	clone := &Registry{
		schemaVersion: r.schemaVersion,
		models:        append([]Model(nil), r.models...),
		byID:          make(map[string]Model, len(r.byID)+len(customs)),
		byOperation:   make(map[string][]Model, len(r.byOperation)),
		defaultFor:    r.defaultFor,
		vocab:         r.vocab,
		vocabOrder:    r.vocabOrder,
		blocklist:     r.blocklist,
		blockByID:     r.blockByID,
	}
	for id, m := range r.byID {
		clone.byID[id] = m
	}
	for op, ms := range r.byOperation {
		clone.byOperation[op] = append([]Model(nil), ms...)
	}
	for i := range customs {
		m := customs[i]
		if err := clone.validateModel(m); err != nil {
			return nil, fmt.Errorf("custom model %q: %w", m.ID, err)
		}
		if _, dup := clone.byID[m.ID]; dup {
			return nil, fmt.Errorf("custom model %q collides with an existing model id", m.ID)
		}
		clone.models = append(clone.models, m)
		clone.byID[m.ID] = m
		for _, eo := range m.EffectiveOps() {
			if eo.Offerable() {
				clone.byOperation[eo.Op] = append(clone.byOperation[eo.Op], m)
			}
		}
	}
	return clone, nil
}

// validateModel enforces per-entry structural validity used for any entry
// (seed or custom). Seed-only policy lives in validateSeedInvariants.
func (r *Registry) validateModel(m Model) error {
	if m.ID == "" {
		return fmt.Errorf("missing id")
	}
	if m.Name == "" {
		return fmt.Errorf("missing name")
	}
	if len(m.Operations) == 0 {
		return fmt.Errorf("no operations")
	}
	if !m.Tier.valid() {
		return fmt.Errorf("invalid tier %q", m.Tier)
	}
	if m.Backend == "" {
		return fmt.Errorf("missing backend")
	}
	if !m.CapabilityLabels.CommercialUse.valid() {
		return fmt.Errorf("invalid commercial_use %q", m.CapabilityLabels.CommercialUse)
	}
	seen := make(map[string]struct{}, len(m.Operations))
	for _, op := range m.Operations {
		if _, ok := r.vocab[op]; !ok {
			return fmt.Errorf("operation %q not in vocabulary", op)
		}
		if _, dup := seen[op]; dup {
			return fmt.Errorf("operation %q listed twice", op)
		}
		seen[op] = struct{}{}
	}
	for _, op := range m.DefaultFor {
		if _, ok := seen[op]; !ok {
			return fmt.Errorf("default_for %q is not in this model's operations", op)
		}
	}
	if m.Hardware.MinVRAMGB < 0 || m.Hardware.MinRAMGB < 0 || m.SizeMBApprox < 0 {
		return fmt.Errorf("negative hardware/size figure")
	}
	if !m.Hardware.CPUCapable && !m.Hardware.GPURequired {
		// A model that is neither CPU-capable nor GPU-required is unrunnable.
		return fmt.Errorf("model is neither cpu_capable nor gpu_required")
	}
	// A declared architecture must be a known enum member (typo guard). An empty
	// architecture is allowed (treated as ArchNone — derives nothing); the seed
	// invariant below additionally REQUIRES a concrete architecture on diffusion
	// generation backends so a base checkpoint can never vanish from the derived
	// pickers for lack of a lineage tag.
	if m.Architecture != "" && !m.Architecture.valid() {
		return fmt.Errorf("invalid architecture %q", m.Architecture)
	}
	// A declared runtime family must name a registered adapter (typo guard), and
	// any declared pipeline_class must match that adapter's — so the registry
	// can't drift from the executor. The doctor (DoctorCatalog) additionally
	// REQUIRES the family on enabled diffusers models; here we only validate what
	// is declared so non-diffusers entries stay unaffected.
	// Combined repo+asset sources have no pre-known tree checksum; each asset is
	// individually pinned and the installer captures the complete tree receipt.
	if m.Source.HasRepo() && len(m.Source.Assets) > 0 && m.Source.Checksum.Value != "" {
		return fmt.Errorf("combined repo+asset source uses per-asset hashes; source checksum must be empty")
	}
	if err := fetch.ValidateHubCompanions(m.Source); err != nil {
		return err
	}
	if profile := m.BuildProfile; profile != nil {
		if m.ID != "big-lama-1024-export" || profile.Kind != "lama-onnx-fixed" || profile.SourceAsset != "lama-source.zip" || profile.CheckpointAsset != "big-lama.safetensors" || profile.Resolution != 1024 || profile.Opset != 17 || profile.OutputFilename != "lama_1024.onnx" {
			return fmt.Errorf("unsupported model build profile %q", profile.Kind)
		}
		if !fetch.ValidSHA256(profile.SourceSHA256) || profile.CheckpointSHA256 != "12cb5eb5d515400f262b48d4e7c1994183e170aedb4b86699d9a55b770038e16" {
			return fmt.Errorf("LaMa build profile must pin source and known generator-only safetensors SHA-256 values")
		}
		if profile.OutputSHA256 != "" && !fetch.ValidSHA256(profile.OutputSHA256) {
			return fmt.Errorf("LaMa build profile output_sha256 must be a SHA-256 hex digest")
		}
		if m.Enabled && profile.OutputSHA256 == "" {
			return fmt.Errorf("LaMa model cannot be enabled until its built ONNX output is pinned")
		}
		assets := map[string]Asset{}
		for _, asset := range m.Source.Assets {
			assets[asset.Filename] = asset
		}
		if assets[profile.SourceAsset].SHA256 != profile.SourceSHA256 || assets[profile.CheckpointAsset].SHA256 != profile.CheckpointSHA256 {
			return fmt.Errorf("LaMa build profile asset hashes must match the pinned source assets")
		}
	}
	if err := validateGatewayServed(m); err != nil {
		return err
	}
	if fam := strings.TrimSpace(m.Runtime.Family); fam != "" {
		reg, ok := DiffusersFamilyByName(fam)
		if !ok {
			return fmt.Errorf("runtime.family %q is not a registered diffusers adapter", fam)
		}
		if pc := strings.TrimSpace(m.Runtime.PipelineClass); pc != "" && pc != reg.PipelineClass {
			return fmt.Errorf("runtime.pipeline_class %q does not match family %q (%s)", pc, fam, reg.PipelineClass)
		}
	}
	return nil
}

// validateGatewayServed holds a gateway-served vision row to the one contract
// that makes adding such a model a single row: it names its gateway role, serves
// only vision operations, and (when it detects objects) says how it writes boxes.
func validateGatewayServed(m Model) error {
	vision := false
	for _, op := range m.Operations {
		if IsVisionOperation(op) {
			vision = true
		}
	}
	if m.Backend == BackendGatewayVision || (m.Backend == BackendOpenRouter && vision) {
		if strings.TrimSpace(m.Runtime.GatewayRole) == "" {
			return fmt.Errorf("a gateway-served vision model must name its runtime.gateway_role")
		}
	}
	if !m.GatewayServed() {
		return nil
	}
	for _, op := range m.Operations {
		if m.Backend == BackendGatewayVision && !IsVisionOperation(op) {
			return fmt.Errorf("backend %s serves only %v, not %q", BackendGatewayVision, VisionOperations, op)
		}
		if op == "object_detection" {
			if order := m.Runtime.BoxOrder; order != BoxOrderXYXY && order != BoxOrderYXYX {
				return fmt.Errorf("an object_detection gateway model must declare runtime.box_order %q or %q", BoxOrderXYXY, BoxOrderYXYX)
			}
			if m.Runtime.BoxScale < 0 {
				return fmt.Errorf("runtime.box_scale must not be negative")
			}
		}
	}
	return nil
}

// validateSeedInvariants asserts the policy guarantees the bundled seed must
// uphold (see DECISIONS.md). These are stricter than structural validity and
// apply only to the shipped catalog.
func (r *Registry) validateSeedInvariants() error {
	for _, m := range r.models {
		if m.RequiresComfyUI {
			return fmt.Errorf("model %q sets requires_comfyui=true (ComfyUI is an optional plug-in only)", m.ID)
		}
		switch {
		case !m.Enabled && m.Disabled == nil:
			return fmt.Errorf("model %q is seeded disabled with no \"disabled\" reason: say why (kind + one line) or enable it", m.ID)
		case m.Enabled && m.Disabled != nil:
			return fmt.Errorf("model %q is enabled but still carries a \"disabled\" reason; remove it", m.ID)
		case m.Disabled != nil:
			if err := m.Disabled.Validate(); err != nil {
				return fmt.Errorf("model %q: %w", m.ID, err)
			}
		}
		// A diffusion-generation checkpoint MUST declare a concrete (non-none)
		// architecture: the architecture is what derives its image-conditioned ops,
		// so a missing lineage tag is exactly the "usable base checkpoint vanishes
		// from the img2img/inpaint pickers" bug this refactor removes.
		if m.Backend == BackendDiffusers || m.Backend == "stable-diffusion.cpp" {
			if m.Architecture == "" || m.Architecture == ArchNone {
				return fmt.Errorf("model %q on backend %q must declare a concrete architecture (none/empty derives nothing)", m.ID, m.Backend)
			}
		}
		// License labels do not keep a model out of the seed (operator stance
		// 2026-11-21: best models for normal use; licensing matters for
		// monetization). A restrictive label must say what to verify before any
		// monetized, hosted or redistributed use, and must carry a rights record
		// whose classification agrees, so the shared rights owner still refuses
		// those uses.
		switch m.CapabilityLabels.CommercialUse {
		case CommercialUseNo, CommercialUseConditional:
			if m.CapabilityLabels.CommercialUseNotes == "" {
				return fmt.Errorf("model %q is commercial_use=%s but carries no commercial_use_notes", m.ID, m.CapabilityLabels.CommercialUse)
			}
		}
		if b, gated := r.blockByID[m.ID]; gated {
			if b.IsExclusion() {
				return fmt.Errorf("model %q is excluded (%s: %s)", m.ID, b.ExclusionKind, b.Reason)
			}
			if err := licenseGateAgrees(m); err != nil {
				return err
			}
		}
	}
	// Every operation with an enabled model must have a default. Disabled-only
	// operations remain structurally registered but unavailable until an
	// experimental backend has passed its native qualification; DoctorCatalog
	// reports that state as a warning.
	for op := range r.byOperation {
		if _, ok := r.defaultFor[op]; ok {
			continue
		}
		available := false
		for _, model := range r.byOperation[op] {
			if model.Enabled {
				available = true
				break
			}
		}
		if available {
			return fmt.Errorf("operation %q has models but no seeded default", op)
		}
	}
	return nil
}

// SchemaVersion returns the catalog schema version.
func (r *Registry) SchemaVersion() string { return r.schemaVersion }

// Models returns all catalog entries in seed order.
func (r *Registry) Models() []Model { return append([]Model(nil), r.models...) }

// ByID returns the model with the given id.
func (r *Registry) ByID(id string) (Model, bool) {
	m, ok := r.byID[id]
	return m, ok
}

// ForOperation returns every model serving op, in seed order.
func (r *Registry) ForOperation(op string) []Model {
	return append([]Model(nil), r.byOperation[op]...)
}

// OpCandidate pairs a model with how it would serve a requested operation: its
// EffectiveOp carries the native|derived support, the technique, the caveat, and
// whether a derived technique is proven (Ready). Unlike ForOperation (which lists
// only OFFERABLE models — native or proven-derived), OpCandidates includes models
// that could serve the op via an UNPROVEN derivation, so the picker can show
// "inpaint ⚠ via workflow (unproven)" instead of the model disappearing.
type OpCandidate struct {
	Model     Model
	Effective EffectiveOp
}

// OpCandidates returns every model whose effective op set contains op — native
// OR architecture-derived, proven or not — in seed order, each tagged with its
// EffectiveOp. This is the picker's full menu; the handler styles each by support
// + Ready. ForOperation stays the offerable-only set the engine/selector use.
func (r *Registry) OpCandidates(op string) []OpCandidate {
	var out []OpCandidate
	for _, m := range r.models {
		for _, eo := range m.EffectiveOps() {
			if eo.Op == op {
				out = append(out, OpCandidate{Model: m, Effective: eo})
				break
			}
		}
	}
	return out
}

// DefaultFor returns the seeded default model for op.
func (r *Registry) DefaultFor(op string) (Model, bool) {
	id, ok := r.defaultFor[op]
	if !ok {
		return Model{}, false
	}
	return r.byID[id], true
}

// IsOperation reports whether op is a known operation.
func (r *Registry) IsOperation(op string) bool {
	_, ok := r.vocab[op]
	return ok
}

// Operations returns the operation vocabulary in declaration order.
func (r *Registry) Operations() []string { return append([]string(nil), r.vocabOrder...) }

// Blocklist returns every gate: license gates (monetized, hosted and
// redistributed use only) and real exclusions.
func (r *Registry) Blocklist() []BlocklistEntry { return append([]BlocklistEntry(nil), r.blocklist...) }

// Gate returns the blocklist entry for id, whichever kind it is.
func (r *Registry) Gate(id string) (BlocklistEntry, bool) {
	b, ok := r.blockByID[id]
	return b, ok
}

// SortedOperations returns the operations that have at least one model, sorted.
func (r *Registry) SortedOperations() []string {
	ops := make([]string, 0, len(r.byOperation))
	for op := range r.byOperation {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}
