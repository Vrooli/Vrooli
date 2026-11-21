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
)

// Capabilities lists the supported capabilities in canonical display order.
var Capabilities = []Capability{
	CapabilitySTT,
	CapabilityTTS,
	CapabilitySummarize,
	CapabilitySpeaker,
}

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

// Runtimes are the execution backends a record may name. "byok" and "gateway"
// run remotely; "browser" runs in the consumer's browser.
var Runtimes = []string{
	"whisper.cpp",
	"kyutai",
	"sherpa-onnx",
	"kokoro",
	"onnxruntime",
	"pytorch",
	"vllm",
	"ollama",
	"byok",
	"gateway",
	"browser",
}

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
// layout; the registry only names the artifact so a conformance test can prove
// that a record marked "available" is actually shipped. Remote providers and
// unshipped candidates leave this nil.
type Artifact struct {
	Resource string `json:"resource"`
	Name     string `json:"name"`
}

// Model is one model or voice family record.
type Model struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	ProviderIDs  []string      `json:"provider_ids"`
	EngineIDs    []string      `json:"engine_ids"`
	Capability   Capability    `json:"capability"`
	Runtime      string        `json:"runtime"`
	Artifact     *Artifact     `json:"artifact,omitempty"`
	Languages    []string      `json:"languages"`
	SizeMBApprox float64       `json:"size_mb_approx"`
	Params       string        `json:"params"`
	License      string        `json:"license"`
	Commercial   CommercialUse `json:"commercial_use"`
	LicenseNotes string        `json:"license_notes"`
	Source       Source        `json:"source"`
	Readiness    Readiness     `json:"readiness"`
	// AllowDefault is the explicit, auditable opt-in that lets a record carry
	// the "default" role outside the normal eligibility rule. There are two
	// legitimate uses: a non-available (candidate) record whose runtime is the
	// intended primary, and an available runtime primary whose upstream licence
	// is not yet verified in-repo. In both cases license/readiness notes must
	// state the uncertainty, and CommercialDefaultFor still refuses to select a
	// record whose commercial use is not explicitly allowed.
	AllowDefault bool       `json:"allow_default,omitempty"`
	Roles        []string   `json:"roles"`
	Hardware     Hardware   `json:"hardware"`
	Benchmarks   Benchmarks `json:"benchmarks"`
	Notes        string     `json:"notes"`
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
	for _, capability := range Capabilities {
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

// DefaultEligible reports whether a record may be selected as a default. Only
// available records are eligible by default; candidate, experimental, and
// watch records require the explicit allow_default flag in the seed.
func (m Model) DefaultEligible() bool {
	return m.Readiness == ReadinessAvailable || m.AllowDefault
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
	if !m.supportsPlatform(p) {
		return false
	}
	if m.Hardware.GPURequired && !p.Accelerated {
		return false
	}
	if !m.Hardware.CPUCapable && !p.Accelerated {
		return false
	}
	if p.FreeVRAMGB > 0 && m.Hardware.MinVRAMGB > 0 && p.FreeVRAMGB < m.Hardware.MinVRAMGB {
		return false
	}
	if p.RAMGB > 0 && m.Hardware.MinRAMGB > 0 && p.RAMGB < m.Hardware.MinRAMGB {
		return false
	}
	return true
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
	return c.defaultFor(capability, "", language, platform)
}

// DefaultForEngine is DefaultFor constrained to one STT engine id. It is the
// safe bridge for callers that pin StreamConfig.EngineID: the result is the
// model record for that engine, not a silent engine substitution.
func (c *Catalog) DefaultForEngine(capability Capability, engineID, language string, platform Platform) (Model, error) {
	return c.defaultFor(capability, engineID, language, platform)
}

func (c *Catalog) defaultFor(capability Capability, engineID, language string, platform Platform) (Model, error) {
	for _, m := range c.CandidatesForEngine(capability, engineID, language, platform) {
		if m.HasRole("default") && m.DefaultEligible() {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("%w: capability=%s engine=%q language=%q (no model declares the default role)",
		ErrNoEligibleModel, capability, engineID, language)
}

// CommercialDefaultFor is DefaultFor with a fail-closed commercial filter. It
// only considers records that carry the default role, are default-eligible,
// and whose commercial_use is explicitly "yes". Unknown or non-commercial
// records are never returned, even when they rank highest.
func (c *Catalog) CommercialDefaultFor(capability Capability, language string, platform Platform) (Model, error) {
	for _, m := range c.CandidatesFor(capability, language, platform) {
		if m.HasRole("default") && m.Commercial == CommercialYes && m.DefaultEligible() {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("%w: capability=%s language=%q", ErrNoCommercialModel, capability, language)
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
