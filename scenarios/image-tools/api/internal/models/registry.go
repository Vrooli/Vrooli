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
}

// Checksum integrity record. Value is captured + pinned on first real download
// (never hand-written); Status flips to "pinned" then.
type Checksum struct {
	Algo   string `json:"algo"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

// Source records where the model artifact comes from.
type Source struct {
	DownloadURL  string   `json:"download_url"`
	SourceRepo   string   `json:"source_repo"`
	DocsURL      string   `json:"docs_url"`
	UpdateSource string   `json:"update_source"`
	LocalPath    string   `json:"local_path,omitempty"` // set for custom/local entries
	Checksum     Checksum `json:"checksum"`
}

// Model is one registry entry: a concrete model/library backing one or more
// operations.
type Model struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Operations       []string         `json:"operations"`
	DefaultFor       []string         `json:"default_for"`
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
	Enabled          bool             `json:"enabled"`
}

// ServesOperation reports whether this model can run the given operation.
func (m Model) ServesOperation(op string) bool {
	for _, o := range m.Operations {
		if o == op {
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

// seedFile mirrors the on-disk registry.seed.json top-level shape.
type seedFile struct {
	SchemaVersion        string           `json:"schema_version"`
	OperationsVocabulary []string         `json:"operations_vocabulary"`
	Models               []Model          `json:"models"`
	Blocklist            []BlocklistEntry `json:"blocklist"`
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
	if len(sf.OperationsVocabulary) == 0 {
		return nil, fmt.Errorf("empty operations_vocabulary")
	}
	if len(sf.Models) == 0 {
		return nil, fmt.Errorf("registry has no models")
	}

	r := &Registry{
		schemaVersion: sf.SchemaVersion,
		byID:          make(map[string]Model, len(sf.Models)),
		byOperation:   make(map[string][]Model),
		defaultFor:    make(map[string]string),
		vocab:         make(map[string]struct{}, len(sf.OperationsVocabulary)),
		vocabOrder:    append([]string(nil), sf.OperationsVocabulary...),
		blockByID:     make(map[string]BlocklistEntry, len(sf.Blocklist)),
	}
	for _, op := range sf.OperationsVocabulary {
		if op == "" {
			return nil, fmt.Errorf("operations_vocabulary contains an empty entry")
		}
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
		for _, op := range m.Operations {
			r.byOperation[op] = append(r.byOperation[op], m)
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
