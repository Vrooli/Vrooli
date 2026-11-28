// Package modelcatalog is the audio-tools model and voice registry. One
// checked-in, schema-versioned seed (models.seed.json) describes every
// STT/TTS/summarize/speaker model the scenario can select, with the facts the
// selector needs: stable provider ids, STT engine ids, runtime, languages,
// licence and commercial eligibility, readiness, hardware, and benchmark
// evidence.
//
// The package owns static facts and deterministic selection only. It does not
// start resources, probe health, or install weights. Live availability belongs
// to internal/capabilities and internal/sttengine; importing either package
// here would create a cycle (they are the consumers of this data).
//
// Model choice is data plus evidence, not an engine-id branch in code. A
// record that is not validated as runnable is marked candidate, experimental,
// or watch; it can never become a default by accident. Promotion is a seed
// edit plus qualification evidence.
package modelcatalog

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// SupportedSchemaVersion is the only schema version this loader accepts.
// Bump it when a field changes meaning, not when a record is added.
const SupportedSchemaVersion = "1.0.0"

//go:embed models.seed.json
var seedJSON []byte

// Regenerate docs/reference/models.md after a seed edit. The drift test fails
// when the checked-in document and this renderer disagree.
//
//go:generate go run ./cmd/render-models -out ../../../docs/reference/models.md

// Capability is the product capability a model serves.
type Capability string

const (
	CapabilitySTT       Capability = "stt"
	CapabilityTTS       Capability = "tts"
	CapabilitySummarize Capability = "summarize"
	CapabilitySpeaker   Capability = "speaker"
	CapabilityDuplex    Capability = "duplex"
)

// Capabilities lists the supported capabilities in canonical display order.
var Capabilities = []Capability{
	CapabilitySTT,
	CapabilityTTS,
	CapabilitySummarize,
	CapabilitySpeaker,
	CapabilityDuplex,
}

// DefaultCapabilities are independently selectable product routes. Duplex
// models are catalogued for provenance and pricing, but are not yet runnable.
var DefaultCapabilities = []Capability{CapabilitySTT, CapabilityTTS, CapabilitySummarize, CapabilitySpeaker}

// Readiness is the honest state of a record's runnability in this scenario.
//
//   - available:    an adapter exists and the model is expected to run.
//   - candidate:    adapter work or qualification is pending; not a default
//     unless allow_default is explicitly set.
//   - experimental: runnable in a lab or draft setting only.
//   - watch:        no confirmed release or licence; never selected or shipped.
type Readiness string

const (
	ReadinessAvailable    Readiness = "available"
	ReadinessCandidate    Readiness = "candidate"
	ReadinessExperimental Readiness = "experimental"
	ReadinessWatch        Readiness = "watch"
)

// Runtime declares a selectable model execution backend and where its model
// weights are owned.
type Runtime struct {
	ID             string
	ArtifactPolicy RuntimeArtifactPolicy
	Capabilities   []Capability
}

// RuntimeArtifactPolicy states whether a model record must identify a
// resource data artifact or whether another execution service owns the weights.
type RuntimeArtifactPolicy string

const (
	RuntimeArtifactRequired RuntimeArtifactPolicy = "required"
	RuntimeArtifactExternal RuntimeArtifactPolicy = "external"
)

// Runtimes is the single declaration of supported runtime IDs and their
// artifact ownership policy.
var Runtimes = []Runtime{
	{ID: "whisper.cpp", ArtifactPolicy: RuntimeArtifactRequired, Capabilities: []Capability{CapabilitySTT}},
	{ID: "kyutai", ArtifactPolicy: RuntimeArtifactRequired, Capabilities: []Capability{CapabilitySTT}},
	{ID: "sherpa-onnx", ArtifactPolicy: RuntimeArtifactRequired, Capabilities: []Capability{CapabilitySTT, CapabilityTTS, CapabilitySpeaker}},
	{ID: "kokoro", ArtifactPolicy: RuntimeArtifactRequired, Capabilities: []Capability{CapabilityTTS}},
	{ID: "onnxruntime", ArtifactPolicy: RuntimeArtifactRequired, Capabilities: []Capability{CapabilityTTS}},
	{ID: "pytorch", ArtifactPolicy: RuntimeArtifactExternal, Capabilities: []Capability{CapabilityTTS}},
	{ID: "vllm", ArtifactPolicy: RuntimeArtifactExternal, Capabilities: []Capability{CapabilitySTT}},
	{ID: "ollama", ArtifactPolicy: RuntimeArtifactExternal, Capabilities: []Capability{CapabilitySummarize}},
	{ID: "byok", ArtifactPolicy: RuntimeArtifactExternal, Capabilities: []Capability{CapabilitySTT, CapabilityTTS, CapabilityDuplex}},
	{ID: "gateway", ArtifactPolicy: RuntimeArtifactExternal, Capabilities: []Capability{CapabilitySummarize}},
	{ID: "browser", ArtifactPolicy: RuntimeArtifactExternal, Capabilities: []Capability{CapabilityTTS}},
}

func runtimeLookup() map[string]Runtime {
	byID := make(map[string]Runtime, len(Runtimes))
	for _, runtime := range Runtimes {
		byID[runtime.ID] = runtime
	}
	return byID
}

// BatchAdapterIDs identify batch protocol adapters that bootstrap can wire to
// an explicitly configured endpoint. They describe a protocol, not a model;
// additional models using the same protocol need only declare the adapter id.
var BatchAdapterIDs = []string{"vllm-openai-chat-audio"}

// CommercialUse says whether the weights/terms permit commercial use.
type CommercialUse string

const (
	CommercialYes     CommercialUse = "yes"
	CommercialNo      CommercialUse = "no"
	CommercialUnknown CommercialUse = "unknown"
)

// Roles are selection intents. A record may carry several. "default" is the
// promoted default; "fast" trades quality for latency; "quality" is the
// high-fidelity tier; "edge" targets constrained or browser/device hosts.
var Roles = []string{"default", "fast", "quality", "edge"}

// Artifact names the resource data artifact that provides a local model's
// weights. The resource manifest owns the URL, checksum, size, and install
// layout; the registry only names the artifact. A candidate can name a pinned
// artifact before runtime qualification, while unshipped candidates and remote
// providers leave this nil.
type Artifact struct {
	Resource string `json:"resource"`
	Name     string `json:"name"`
}

// Model is one model or voice family record.
type Model struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	ProviderIDs    []string      `json:"provider_ids"`
	EngineIDs      []string      `json:"engine_ids"`
	EngineDefaults []string      `json:"engine_defaults,omitempty"`
	Capability     Capability    `json:"capability"`
	Runtime        string        `json:"runtime"`
	RuntimeModelID string        `json:"runtime_model_id,omitempty"`
	BatchAdapterID string        `json:"batch_adapter_id,omitempty"`
	Artifact       *Artifact     `json:"artifact,omitempty"`
	Languages      []string      `json:"languages"`
	SizeMBApprox   float64       `json:"size_mb_approx"`
	Params         string        `json:"params"`
	License        string        `json:"license"`
	Commercial     CommercialUse `json:"commercial_use"`
	LicenseNotes   string        `json:"license_notes"`
	Source         Source        `json:"source"`
	Readiness      Readiness     `json:"readiness"`
	// AllowDefault is the explicit, auditable opt-in that lets a record carry a
	// generic default role or an engine-scoped default outside normal readiness
	// eligibility. In both cases license/readiness notes must state uncertainty,
	// commercial selection still fails closed, and promotion evidence remains
	// separately enforced.
	AllowDefault bool              `json:"allow_default,omitempty"`
	Promotion    PromotionEvidence `json:"promotion_evidence,omitempty"`
	Roles        []string          `json:"roles"`
	Hardware     Hardware          `json:"hardware"`
	Benchmarks   Benchmarks        `json:"benchmarks"`
	Pricing      *Pricing          `json:"pricing,omitempty"`
	Notes        string            `json:"notes"`
}

// PromotionEvidence references the existing owner-produced receipts for each
// required promotion dimension. It stores no duplicate measurements and does
// not define quality thresholds; only owner-issued qualified verdicts count.
type PromotionEvidence struct {
	CorpusQuality EvidenceFacet `json:"corpus_quality,omitempty"`
	Latency       EvidenceFacet `json:"latency,omitempty"`
	Memory        EvidenceFacet `json:"memory,omitempty"`
	Platform      EvidenceFacet `json:"platform,omitempty"`
	License       EvidenceFacet `json:"license,omitempty"`
	verified      bool          // Never loaded from seed JSON; requires an integrated owner-receipt verifier.
	// LegacyException preserves an existing operational default without
	// claiming that it has passed the promotion gate.
	LegacyException string `json:"legacy_exception,omitempty"`
}

type EvidenceFacet struct {
	Status string `json:"status,omitempty"` // qualified, unqualified, not_measured, unknown
	Ref    string `json:"ref,omitempty"`
}

// legacyDefaultExceptions is a code-owned sunset list for specifically
// authorized operational routes that lack owner-verified promotion receipts.
// Seed text alone cannot grant this exception to a newly declared model.
var legacyDefaultExceptions = map[string]string{
	"whisper-small":                  "Existing operational default retained pending a joined corpus-quality, latency, memory, platform, and license receipt.",
	"sherpa-zipformer-en-2023-06-26": "Existing operator-selected Sherpa streaming route retained pending joined corpus-quality, latency, memory, platform, and license receipts.",
	"kokoro-82m-v1.0":                "Existing operational default retained pending a joined corpus-quality, latency, memory, platform, and license receipt.",
	"qwen3:1.7b":                     "Audio Tools compact summarize target retained pending owner-qualified receipts; evidence is one short report at 8/8 facts and one 300-word report at 14/22 facts, so this is not a general fact-faithfulness qualification.",
	"sherpa-onnx-speaker-embedding":  "Existing operational default retained pending a joined corpus-quality, latency, memory, platform, and license receipt; current resource score has no EER or threshold qualification.",
}

func legacyExceptionAllowed(modelID, reason string) bool {
	expected, ok := legacyDefaultExceptions[modelID]
	return ok && strings.TrimSpace(reason) == expected
}

func (e PromotionEvidence) Qualified() bool {
	if !e.verified {
		return false
	}
	for _, facet := range []EvidenceFacet{e.CorpusQuality, e.Latency, e.Memory, e.Platform, e.License} {
		if facet.Status != "qualified" || strings.TrimSpace(facet.Ref) == "" {
			return false
		}
	}
	return true
}

// Pricing records published provider rates without any authorization,
// credential, estimate, or settlement policy. Rates are integer micro-USD per declared unit.
type Pricing struct {
	Currency                string `json:"currency"`
	Unit                    string `json:"unit"`
	ActiveMicrosPerMinute   *int64 `json:"active_micros_per_minute,omitempty"`
	StandbyMicrosPerMinute  *int64 `json:"standby_micros_per_minute,omitempty"`
	BillingIncrementSeconds int    `json:"billing_increment_seconds,omitempty"`
	SourceURL               string `json:"source_url"`
	Notes                   string `json:"notes,omitempty"`
}

// Source is provenance for the weights and the runtime adapter.
type Source struct {
	URL  string `json:"url"`
	Repo string `json:"repo"`
}

// Hardware describes the host envelope. Empty OSArch means "no constraint
// declared". MinVRAMGB/MinRAMGB of 0 means "not established".
type Hardware struct {
	CPUCapable  bool     `json:"cpu_capable"`
	GPURequired bool     `json:"gpu_required"`
	MinVRAMGB   float64  `json:"min_vram_gb"`
	MinRAMGB    int      `json:"min_ram_gb"`
	OSArch      []string `json:"os_arch"`
}

// Benchmarks records measured evidence. Nil values mean "not established in
// this seed", which is different from zero. WER is a percentage, SIM is 0..1,
// and RTF is an x-realtime factor (higher is faster).
type Benchmarks struct {
	WER   *float64 `json:"wer,omitempty"`
	SIM   *float64 `json:"sim,omitempty"`
	RTF   *float64 `json:"rtf,omitempty"`
	Notes string   `json:"notes,omitempty"`
}

// SelectionPolicy names the deterministic rank orders the selector uses.
// Keeping the order in data prevents a selection policy change from becoming
// a code change.
type SelectionPolicy struct {
	ReadinessRank    []string `json:"readiness_rank"`
	RoleOrder        []string `json:"role_order"`
	LanguageFallback string   `json:"language_fallback"`
}

// Seed is the decoded models.seed.json.
type Seed struct {
	SchemaVersion    string            `json:"schema_version"`
	Generated        string            `json:"generated"`
	Source           string            `json:"source"`
	Notes            []string          `json:"notes"`
	FieldReference   map[string]string `json:"field_reference"`
	KnownProviderIDs []string          `json:"known_provider_ids"`
	SelectionPolicy  SelectionPolicy   `json:"selection_policy"`
	Models           []Model           `json:"models"`
}

// Catalog is a loaded, indexed registry.
type Catalog struct {
	seed         Seed
	byID         map[string]Model
	byCapability map[Capability][]Model
	byProvider   map[string][]Model
	byEngine     map[string][]Model
}

// Default returns the catalog loaded from the embedded seed. It panics on an
// invalid seed because a broken static registry is a build-time defect.
func Default() *Catalog {
	c, err := Load(seedJSON)
	if err != nil {
		panic(fmt.Sprintf("modelcatalog: embedded seed invalid: %v", err))
	}
	return c
}

// Load parses and validates a seed from raw JSON.
func Load(raw []byte) (*Catalog, error) {
	var seed Seed
	if err := json.Unmarshal(raw, &seed); err != nil {
		return nil, fmt.Errorf("modelcatalog: parse seed: %w", err)
	}
	c := New(seed)
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// New builds a catalog without validation. Tests use it for fixtures; callers
// loading operator data should use Load.
func New(seed Seed) *Catalog {
	c := &Catalog{
		seed:         seed,
		byID:         make(map[string]Model, len(seed.Models)),
		byCapability: make(map[Capability][]Model, len(Capabilities)),
		byProvider:   make(map[string][]Model),
		byEngine:     make(map[string][]Model),
	}
	for _, m := range seed.Models {
		c.byID[m.ID] = m
		c.byCapability[m.Capability] = append(c.byCapability[m.Capability], m)
		for _, p := range m.ProviderIDs {
			c.byProvider[p] = append(c.byProvider[p], m)
		}
		for _, e := range m.EngineIDs {
			c.byEngine[e] = append(c.byEngine[e], m)
		}
	}
	return c
}

// Seed returns a copy of the decoded seed metadata.
func (c *Catalog) Seed() Seed {
	seed := c.seed
	seed.Notes = append([]string(nil), c.seed.Notes...)
	seed.KnownProviderIDs = append([]string(nil), c.seed.KnownProviderIDs...)
	seed.Models = nil
	return seed
}

// All returns every record in seed order.
func (c *Catalog) All() []Model {
	out := make([]Model, len(c.seed.Models))
	copy(out, c.seed.Models)
	return out
}

// ByID looks up one record.
func (c *Catalog) ByID(id string) (Model, bool) {
	m, ok := c.byID[id]
	return m, ok
}

// ByCapability returns records for a capability in seed order.
func (c *Catalog) ByCapability(capability Capability) []Model {
	out := make([]Model, len(c.byCapability[capability]))
	copy(out, c.byCapability[capability])
	return out
}

// CapabilityOrder returns the canonical display position of a capability and
// whether it is known. Consumers use it to keep listings stable without
// hardcoding the vocabulary.
func CapabilityOrder(capability Capability) (int, bool) {
	for i, candidate := range Capabilities {
		if candidate == capability {
			return i, true
		}
	}
	return 0, false
}

// SortedForDisplay returns every record grouped by canonical capability order
// and then by id. Unlike ByCapability it does not imply runnability: a
// registry listing must be able to show planned and watch-only records
// honestly, so the caller filters on readiness separately.
func (c *Catalog) SortedForDisplay() []Model {
	out := make([]Model, len(c.seed.Models))
	copy(out, c.seed.Models)
	sort.SliceStable(out, func(i, j int) bool {
		oi, _ := CapabilityOrder(out[i].Capability)
		oj, _ := CapabilityOrder(out[j].Capability)
		if oi != oj {
			return oi < oj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// DefaultsByCapability resolves the unconstrained default for each capability.
// A capability with no promoted default is omitted; callers that need an error
// per capability use DefaultFor instead.
func (c *Catalog) DefaultsByCapability(platform Platform) map[Capability]Model {
	out := make(map[Capability]Model, len(Capabilities))
	for _, capability := range DefaultCapabilities {
		if model, err := c.DefaultFor(capability, "", platform); err == nil {
			out[capability] = model
		}
	}
	return out
}

// ByProvider returns records that name a stable provider id.
func (c *Catalog) ByProvider(providerID string) []Model {
	out := make([]Model, len(c.byProvider[providerID]))
	copy(out, c.byProvider[providerID])
	return out
}

// ByEngine returns records that name an STT stream-selection engine id.
// Non-STT records have no engine ids and therefore never appear here.
func (c *Catalog) ByEngine(engineID string) []Model {
	out := make([]Model, len(c.byEngine[engineID]))
	copy(out, c.byEngine[engineID])
	return out
}

// ByEngineModelID resolves either a canonical registry ID or the declared
// runtime model ID for one engine. Runtime IDs are protocol-facing names; the
// returned Model retains the canonical ID used by evidence and promotion.
// Ambiguous runtime aliases fail closed.
func (c *Catalog) ByEngineModelID(engineID, modelID string) (Model, bool) {
	engineID = strings.TrimSpace(engineID)
	modelID = strings.TrimSpace(modelID)
	if engineID == "" || modelID == "" {
		return Model{}, false
	}
	var match Model
	found := false
	for _, model := range c.byEngine[engineID] {
		if model.ID != modelID && model.RuntimeModelID != modelID {
			continue
		}
		if found && match.ID != model.ID {
			return Model{}, false
		}
		match = model
		found = true
	}
	return match, found
}

// EngineIDs returns the distinct engine ids in seed order.
func (c *Catalog) EngineIDs() []string {
	seen := make(map[string]struct{}, len(c.byEngine))
	var out []string
	for _, m := range c.seed.Models {
		for _, id := range m.EngineIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// Platform is the bounded host description used for selection. Zero values
// mean "unconstrained or unknown": an empty OS/Arch skips the platform filter,
// Accelerated=false excludes GPU-required records, and a zero RAM/VRAM value
// skips that capacity check. Callers supply probed facts; this package never
// probes.
type Platform struct {
	OS          string
	Arch        string
	Accelerated bool
	FreeVRAMGB  float64
	RAMGB       int
	// CPULimit is the caller's CPU quota in cores. Zero means unknown or
	// unconstrained. Models currently have no evidence-backed minimum CPU
	// declaration, so this is carried as context and is not a selection gate.
	CPULimit int
}

// Selection errors.
var (
	// ErrNoEligibleModel means no record matched the capability, language,
	// platform, and default-eligibility filters.
	ErrNoEligibleModel = errors.New("modelcatalog: no eligible model")
	// ErrNoCommercialModel means no default-eligible commercial record
	// matched. A non-commercial (or unknown-licence) record is never returned.
	ErrNoCommercialModel = errors.New("modelcatalog: no commercial-eligible model")
)

// DefaultEligible reports whether a record may be selected as a default. A
// default role needs either all five owner-referenced qualified evidence facets
// or an exact code-authorized exception for an operational route.
// Readiness eligibility is evaluated too.
func (m Model) DefaultEligible() bool {
	if m.Readiness != ReadinessAvailable && !m.AllowDefault {
		return false
	}
	return !m.HasRole("default") || m.Promotion.Qualified() || legacyExceptionAllowed(m.ID, m.Promotion.LegacyException)
}

// EngineDefaultEligible applies the evidence or code-authorized legacy
// exception gate to a model selected as an engine-scoped default.
func (m Model) EngineDefaultEligible() bool {
	if m.Readiness != ReadinessAvailable && !m.AllowDefault {
		return false
	}
	return m.Promotion.Qualified() || legacyExceptionAllowed(m.ID, m.Promotion.LegacyException)
}

// HasEngineDefault reports whether this model is the declared default for the
// given STT stream-selection engine.
func (m Model) HasEngineDefault(engineID string) bool {
	for _, id := range m.EngineDefaults {
		if id == engineID {
			return true
		}
	}
	return false
}

// HasRole reports whether the record carries a selection intent.
func (m Model) HasRole(role string) bool {
	for _, r := range m.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Speaks reports whether the record declares a language. A request-level
// language matches by primary subtag ("en-US" matches "en"); the ISO 639-3
// codes "mul" (multiple) and "und" (undetermined) match every language. An
// empty request language imposes no filter.
func (m Model) Speaks(language string) bool {
	if language == "" {
		return true
	}
	want := primarySubtag(language)
	for _, tag := range m.Languages {
		if tag == "mul" || tag == "und" {
			return true
		}
		if primarySubtag(tag) == want {
			return true
		}
	}
	return false
}

// RunnableOn reports whether the record's declared hardware envelope admits
// the platform. Unknown platform facts are not treated as a shortfall; a known
// shortfall is.
func (m Model) RunnableOn(p Platform) bool {
	return len(m.platformExclusionReasons(p)) == 0
}

// platformExclusionReasons explains known host constraints that prevent a
// model from running. Unknown host facts remain unknown and are not treated as
// failures, matching RunnableOn's selection semantics.
func (m Model) platformExclusionReasons(p Platform) []string {
	var reasons []string
	if !m.supportsPlatform(p) {
		reasons = append(reasons, fmt.Sprintf("does not declare support for %s/%s", p.OS, p.Arch))
	}
	if m.Hardware.GPURequired && !p.Accelerated {
		reasons = append(reasons, "requires an accelerated GPU")
	}
	if !m.Hardware.CPUCapable && !p.Accelerated {
		reasons = append(reasons, "does not support CPU execution")
	}
	if p.FreeVRAMGB > 0 && m.Hardware.MinVRAMGB > 0 && p.FreeVRAMGB < m.Hardware.MinVRAMGB {
		reasons = append(reasons, fmt.Sprintf("requires %.3g GB free VRAM; host reports %.3g GB", m.Hardware.MinVRAMGB, p.FreeVRAMGB))
	}
	if p.RAMGB > 0 && m.Hardware.MinRAMGB > 0 && p.RAMGB < m.Hardware.MinRAMGB {
		reasons = append(reasons, fmt.Sprintf("requires at least %d GB RAM; host reports %d GB", m.Hardware.MinRAMGB, p.RAMGB))
	}
	return reasons
}

func (m Model) supportsPlatform(p Platform) bool {
	if len(m.Hardware.OSArch) == 0 || (p.OS == "" && p.Arch == "") {
		return true
	}
	for _, entry := range m.Hardware.OSArch {
		osName, arch, ok := strings.Cut(entry, "/")
		if !ok {
			continue
		}
		osMatch := osName == "any" || p.OS == "" || osName == p.OS
		archMatch := arch == "any" || p.Arch == "" || arch == p.Arch
		if osMatch && archMatch {
			return true
		}
	}
	return false
}

// CandidatesFor returns every runnable, language-compatible record for a
// capability in deterministic policy order, including records that are not
// default-eligible. Callers use it for pickers and fallback ladders.
func (c *Catalog) CandidatesFor(capability Capability, language string, platform Platform) []Model {
	return c.CandidatesForEngine(capability, "", language, platform)
}

// CandidatesForEngine is CandidatesFor with an optional engine-id filter. An
// empty engineID keeps every record; a non-empty id keeps only records that
// declare it (STT stream selection is engine-scoped).
func (c *Catalog) CandidatesForEngine(capability Capability, engineID, language string, platform Platform) []Model {
	out := make([]Model, 0, len(c.byCapability[capability]))
	for _, m := range c.byCapability[capability] {
		if engineID != "" && !m.hasEngine(engineID) {
			continue
		}
		if !m.Speaks(language) || !m.RunnableOn(platform) {
			continue
		}
		out = append(out, m)
	}
	sortModels(out, c.seed.SelectionPolicy)
	return out
}

func (m Model) hasEngine(engineID string) bool {
	for _, id := range m.EngineIDs {
		if id == engineID {
			return true
		}
	}
	return false
}

// DefaultFor returns the highest-ranked promoted default. A default candidate
// must explicitly carry the "default" role in addition to being
// default-eligible; a model that only happens to be available is an
// alternative, not a default. Ranking is readiness (available first), then the
// role signature (default, fast, quality, edge), then commercial clarity, then
// id.
func (c *Catalog) DefaultFor(capability Capability, language string, platform Platform) (Model, error) {
	return c.defaultFor(capability, "", language, platform, nil)
}

// DefaultForVerified is the generic bridge for runtime owner verification.
// The callback must accept only owner-verified results bound to this exact
// model; seed declarations and caller supplied facet strings are not proof.
// Legacy exceptions continue to be evaluated by DefaultEligible unchanged.
func (c *Catalog) DefaultForVerified(capability Capability, language string, platform Platform, verified func(Model) bool) (Model, error) {
	return c.defaultFor(capability, "", language, platform, verified)
}

// DefaultForEngine is DefaultFor constrained to one STT engine id. A model
// must carry the generic default role or declare this id in engine_defaults.
// The result never substitutes a model from another engine.
func (c *Catalog) DefaultForEngine(capability Capability, engineID, language string, platform Platform) (Model, error) {
	return c.defaultForEngine(capability, engineID, language, platform, nil, false)
}

// DefaultForEngineVerified is DefaultForVerified constrained to one engine.
func (c *Catalog) DefaultForEngineVerified(capability Capability, engineID, language string, platform Platform, verified func(Model) bool) (Model, error) {
	return c.defaultForEngine(capability, engineID, language, platform, verified, false)
}

// CommercialDefaultForEngineVerified resolves a declared engine-scoped default
// while requiring explicit commercial permission. It never substitutes a
// model from another engine.
func (c *Catalog) CommercialDefaultForEngineVerified(capability Capability, engineID, language string, platform Platform, verified func(Model) bool) (Model, error) {
	return c.defaultForEngine(capability, engineID, language, platform, verified, true)
}

func (c *Catalog) defaultForEngine(capability Capability, engineID, language string, platform Platform, verified func(Model) bool, commercial bool) (Model, error) {
	engineID = strings.TrimSpace(engineID)
	if engineID == "" {
		return Model{}, fmt.Errorf("%w: engine id is required", ErrNoEligibleModel)
	}
	candidates := c.CandidatesForEngine(capability, engineID, language, platform)
	for _, m := range candidates {
		if !m.HasEngineDefault(engineID) {
			continue
		}
		if commercial && m.Commercial != CommercialYes {
			return Model{}, fmt.Errorf("%w: capability=%s engine=%q language=%q", ErrNoCommercialModel, capability, engineID, language)
		}
		eligible := m.EngineDefaultEligible() || (verified != nil && verified(m))
		if eligible {
			return m, nil
		}
		if commercial {
			return Model{}, fmt.Errorf("%w: capability=%s engine=%q language=%q", ErrNoCommercialModel, capability, engineID, language)
		}
		return Model{}, fmt.Errorf("%w: capability=%s engine=%q language=%q", ErrNoEligibleModel, capability, engineID, language)
	}
	for _, m := range candidates {
		if !m.HasRole("default") {
			continue
		}
		if commercial && m.Commercial != CommercialYes {
			continue
		}
		if m.DefaultEligible() || (verified != nil && verified(m)) {
			return m, nil
		}
	}
	return Model{}, c.noDefaultError(capability, engineID, language, platform, commercial, verified)
}

func (c *Catalog) defaultFor(capability Capability, engineID, language string, platform Platform, verified func(Model) bool) (Model, error) {
	for _, m := range c.CandidatesForEngine(capability, engineID, language, platform) {
		if m.HasRole("default") && (m.DefaultEligible() || (verified != nil && verified(m))) {
			return m, nil
		}
	}
	return Model{}, c.noDefaultError(capability, engineID, language, platform, false, verified)
}

// CommercialDefaultFor is DefaultFor with a fail-closed commercial filter. It
// only considers records that carry the default role, are default-eligible,
// and whose commercial_use is explicitly "yes". Unknown or non-commercial
// records are never returned, even when they rank highest.
func (c *Catalog) CommercialDefaultFor(capability Capability, language string, platform Platform) (Model, error) {
	return c.CommercialDefaultForVerified(capability, language, platform, nil)
}

// CommercialDefaultForVerified is CommercialDefaultFor with owner verification
// available for newly declared defaults.
func (c *Catalog) CommercialDefaultForVerified(capability Capability, language string, platform Platform, verified func(Model) bool) (Model, error) {
	for _, m := range c.CandidatesFor(capability, language, platform) {
		if m.HasRole("default") && m.Commercial == CommercialYes && (m.DefaultEligible() || (verified != nil && verified(m))) {
			return m, nil
		}
	}
	return Model{}, c.noDefaultError(capability, "", language, platform, true, verified)
}

func (c *Catalog) noDefaultError(capability Capability, engineID, language string, platform Platform, commercial bool, verified func(Model) bool) error {
	var excluded []string
	for _, model := range c.byCapability[capability] {
		if engineID != "" && !model.hasEngine(engineID) {
			continue
		}
		if !model.Speaks(language) || (commercial && model.Commercial != CommercialYes) {
			continue
		}
		isDefault := model.HasRole("default") && (model.DefaultEligible() || (verified != nil && verified(model)))
		if engineID != "" && model.HasEngineDefault(engineID) {
			isDefault = model.EngineDefaultEligible() || (verified != nil && verified(model))
		}
		if !isDefault {
			continue
		}
		reasons := model.platformExclusionReasons(platform)
		if len(reasons) > 0 {
			excluded = append(excluded, model.ID+": "+strings.Join(reasons, ", "))
		}
	}
	if len(excluded) > 0 {
		return fmt.Errorf("%w: capability=%s engine=%q language=%q; host constraints exclude default model(s): %s",
			ErrNoEligibleModel, capability, engineID, language, strings.Join(excluded, "; "))
	}
	if commercial {
		return fmt.Errorf("%w: capability=%s engine=%q language=%q", ErrNoCommercialModel, capability, engineID, language)
	}
	return fmt.Errorf("%w: capability=%s engine=%q language=%q (no model declares an eligible default)",
		ErrNoEligibleModel, capability, engineID, language)
}

func sortModels(models []Model, policy SelectionPolicy) {
	readinessRank := rankMap(policy.ReadinessRank)
	roleRank := rankMap(policy.RoleOrder)
	sort.SliceStable(models, func(i, j int) bool {
		a, b := models[i], models[j]
		if ra, rb := readinessRank[string(a.Readiness)], readinessRank[string(b.Readiness)]; ra != rb {
			return ra < rb
		}
		if lessRoleSignature(roleSignature(a, roleRank), roleSignature(b, roleRank)) {
			return true
		}
		if lessRoleSignature(roleSignature(b, roleRank), roleSignature(a, roleRank)) {
			return false
		}
		if ra, rb := commercialRank(a.Commercial), commercialRank(b.Commercial); ra != rb {
			return ra < rb
		}
		return a.ID < b.ID
	})
}

func rankMap(order []string) map[string]int {
	out := make(map[string]int, len(order))
	for i, value := range order {
		out[value] = i
	}
	return out
}

// roleSignature turns a model's roles into a comparable vector in policy
// order. Unfilled slots pad with len(roleRank), so [default,fast] ranks above
// [default,quality] and both rank above a roleless model. This is what makes
// two promoted defaults resolve deterministically by their secondary intent.
func roleSignature(m Model, roleRank map[string]int) []int {
	ranks := make([]int, 0, len(m.Roles))
	for _, role := range m.Roles {
		if rank, ok := roleRank[role]; ok {
			ranks = append(ranks, rank)
		}
	}
	sort.Ints(ranks)
	sig := make([]int, len(roleRank))
	for i := range sig {
		sig[i] = len(roleRank)
		if i < len(ranks) {
			sig[i] = ranks[i]
		}
	}
	return sig
}

func lessRoleSignature(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func commercialRank(use CommercialUse) int {
	switch use {
	case CommercialYes:
		return 0
	case CommercialUnknown:
		return 1
	default:
		return 2
	}
}

func primarySubtag(tag string) string {
	primary, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	return primary
}
