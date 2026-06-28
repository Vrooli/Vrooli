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

	"image-tools/internal/fetch"
	"image-tools/internal/operations"
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
	CPUCapable  bool     `json:"cpu_capable"`
	GPURequired bool     `json:"gpu_required"`
	MinVRAMGB   int      `json:"min_vram_gb"`
	MinRAMGB    int      `json:"min_ram_gb"`
	OSArch      []string `json:"os_arch"`
	SpeedNote   string   `json:"speed_note"`
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

// Checksum integrity record. Value is captured + pinned on first real download
// (never hand-written); Status flips to "pinned" then.
type Checksum struct {
	Algo   string `json:"algo"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

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
	// MinRuntime is the minimum runtime/library constraint surfaced to the
	// provision/availability check (e.g. "diffusers>=0.36.0"; a git ref for a
	// bleeding-edge family). Phase 4 enforces it; informational otherwise.
	MinRuntime string `json:"min_runtime"`
}

// RepoSource is a HuggingFace-style multi-file model repository — the install
// shape diffusers models use. It is the shared fetch.RepoSpec, re-exported under
// the catalog's historical name so the seed schema is unchanged.
type RepoSource = fetch.RepoSpec

// Source records where the model artifact comes from.
type Source struct {
	// DownloadURL is the single remote URL for an operator-registered CUSTOM
	// model (the AddCustomModel API/UI field). Seed models use Assets instead;
	// either way the downloaded bytes are artifact-validated before install.
	DownloadURL  string `json:"download_url"`
	SourceRepo   string `json:"source_repo"`
	DocsURL      string `json:"docs_url"`
	UpdateSource string `json:"update_source"`
	LocalPath    string `json:"local_path,omitempty"` // set for custom/local entries
	// Assets are the direct, resolvable weight artifacts for a seed model. When
	// present they are the install source (DownloadURL is ignored). The installer
	// downloads + validates each one into the model dir under its Filename.
	Assets []Asset `json:"assets,omitempty"`
	// Repo is a multi-file model repository (diffusers snapshot). When set it is
	// the install source, fetched in whole at Repo.Revision; integrity is the
	// tree-manifest checksum. Repo and Assets are mutually exclusive fetch
	// strategies behind one installer seam.
	Repo RepoSource `json:"repo,omitempty"`
	// Manual declares — honestly — that this weight-backed model has NO single
	// auto-fetchable artifact: its weights must be obtained manually per docs_url
	// (e.g. an upstream that ships only a landing page or a multi-file repo with no
	// pinned revision). The installer refuses to auto-install it (no HTML GET of a
	// documentation page), and registry-lint accepts it instead of demanding an
	// assets[]/repo it cannot honestly declare. download_url stays documentation-only.
	Manual   bool     `json:"manual,omitempty"`
	Checksum Checksum `json:"checksum"`
}

// HasRepo reports whether the source installs from a multi-file repo snapshot.
func (s Source) HasRepo() bool { return strings.TrimSpace(s.Repo.RepoID) != "" }

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
	Runtime          Runtime          `json:"runtime"`
	Enabled          bool             `json:"enabled"`
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
)

// RequiresWeights reports whether a model needs downloaded artifacts on disk to
// run. In-process deterministic/Go-library models have no weights and are
// always considered installed; everything else must be fetched +
// artifact-validated by the Installer before an AI op will launch.
func (m Model) RequiresWeights() bool {
	switch m.Backend {
	case BackendBuiltin, BackendComputed, BackendLibraryGo:
		return false
	default:
		return true
	}
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

// BlocklistEntry records a license-encumbered model that must never be seeded
// or accidentally adopted, with the reason it is excluded.
type BlocklistEntry struct {
	ID                              string   `json:"id"`
	Operations                      []string `json:"operations"`
	License                         string   `json:"license"`
	Reason                          string   `json:"reason"`
	ExportingONNXRemovesRestriction bool     `json:"exporting_onnx_removes_restriction"`
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
// the seed-integrity invariants (commercial-clean, ComfyUI-optional, every op
// has a default, blocklist disjoint from catalog) so a bad edit to the seed
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

	// The operation vocabulary is the SSOT (internal/operations); the registry
	// reads it rather than carrying its own copy. Per-model operations[] are
	// validated against it below.
	vocabOrder := operations.Names()
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
		r.blocklist = append(r.blocklist, b)
		r.blockByID[b.ID] = b
	}

	return r, nil
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
	// Assets and Repo are mutually exclusive fetch strategies.
	if len(m.Source.Assets) > 0 && m.Source.HasRepo() {
		return fmt.Errorf("source declares both assets and repo (mutually exclusive fetch strategies)")
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

// validateSeedInvariants asserts the policy guarantees the bundled seed must
// uphold (see DECISIONS.md). These are stricter than structural validity and
// apply only to the shipped catalog.
func (r *Registry) validateSeedInvariants() error {
	for _, m := range r.models {
		if m.RequiresComfyUI {
			return fmt.Errorf("model %q sets requires_comfyui=true (ComfyUI is an optional plug-in only)", m.ID)
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
		// Commercial-clean gate: never seed an outright non-commercial model.
		// "conditional" is tolerated ONLY as an opt-in (disabled-by-default)
		// entry carrying notes that say what to verify at bundle time — so a
		// careless enable can't ship a license-encumbered model.
		switch m.CapabilityLabels.CommercialUse {
		case CommercialUseNo:
			return fmt.Errorf("model %q is commercial_use=no (commercial-clean gate)", m.ID)
		case CommercialUseConditional:
			if m.Enabled {
				return fmt.Errorf("model %q is commercial_use=conditional and must NOT be enabled by default", m.ID)
			}
			if m.CapabilityLabels.CommercialUseNotes == "" {
				return fmt.Errorf("model %q is commercial_use=conditional but carries no commercial_use_notes", m.ID)
			}
		}
		if b, blocked := r.blockByID[m.ID]; blocked {
			return fmt.Errorf("model %q is also on the blocklist (%s)", m.ID, b.Reason)
		}
	}
	// Every operation in the vocabulary that any model serves must have a default.
	for op := range r.byOperation {
		if _, ok := r.defaultFor[op]; !ok {
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

// Blocklist returns the license-encumbered exclusions.
func (r *Registry) Blocklist() []BlocklistEntry { return append([]BlocklistEntry(nil), r.blocklist...) }

// IsBlocked reports whether id is on the blocklist.
func (r *Registry) IsBlocked(id string) (BlocklistEntry, bool) {
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
