package aisearch

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// searchjson.go parses the scenario-owned `.vrooli/search.json` — the single
// source of truth for everything search-related a scenario exposes: the provider
// DESCRIPTOR (how search-hub routes to it), the TUNING (the factor values the
// sweep reads and writes), and the TESTS (the evaluation corpus). One file, one
// home; nothing search-tunable is a Go literal or a build-time seed.
//
// aisearch-go owns this file format because the file's centre of gravity is the
// tuning factors it defines. It does NOT interpret the descriptor sub-objects
// (endpoint / result_mapping / status_endpoint) — those are search-hub's registry
// contract and are kept here as raw JSON to be mapped to the proto in Phase 2 —
// so this package stays free of the registry/transport vocabulary.

// SearchFile is a parsed `.vrooli/search.json`.
type SearchFile struct {
	// Schema is the optional "$schema" editor/validation pointer; ignored at runtime.
	Schema    string           `json:"$schema,omitempty"`
	Version   string           `json:"version"`
	Providers []ProviderConfig `json:"providers"`
}

// Provider operability classes. class is a scenario-owned descriptor policy
// field that tells Search Hub what operational posture to expect of a provider
// with no scenario-name special-casing:
//   - ClassLocalIndex — a locally-owned, tunable, indexed corpus. Search Hub
//     expects a reindex control endpoint (the corpus must be reconcilable);
//     config write-back is optional (hybrid-by-construction providers pin it).
//   - ClassLocalLive  — a locally-owned provider that computes results live with
//     no tunable index; no reindex/config control plane is expected.
//   - ClassExternal   — a third-party/external provider (e.g. web search); no
//     local index, control plane, or status endpoint is expected, and its
//     latency/reliability budget is looser (see the performance policy).
//   - ClassAsync      — a locally-owned corpus rebuilt asynchronously; like
//     ClassLocalIndex it must expose a reindex endpoint.
const (
	ClassLocalIndex = "local_index"
	ClassLocalLive  = "local_live"
	ClassExternal   = "external"
	ClassAsync      = "async"
)

// KnownProviderClasses is the closed set of provider operability classes.
var KnownProviderClasses = []string{ClassLocalIndex, ClassLocalLive, ClassExternal, ClassAsync}

// ValidProviderClass reports whether class is one of the known operability
// classes. The empty string is not valid; whether an empty class is tolerated is
// the consumer's policy (Search Hub requires it for active providers).
func ValidProviderClass(class string) bool {
	switch strings.TrimSpace(class) {
	case ClassLocalIndex, ClassLocalLive, ClassExternal, ClassAsync:
		return true
	default:
		return false
	}
}

// IndexedClass reports whether the class owns a rebuildable corpus that must
// expose a reindex control endpoint.
func IndexedClass(class string) bool {
	switch strings.TrimSpace(class) {
	case ClassLocalIndex, ClassAsync:
		return true
	default:
		return false
	}
}

// ProviderConfig is one provider's full descriptor + tuning + test corpus.
type ProviderConfig struct {
	ProviderID    string `json:"provider_id"`
	ProviderGroup string `json:"provider_group"`
	Bucket        string `json:"bucket"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Scope         string `json:"scope"`
	// RoutingProfile is optional declarative evidence for semantic provider
	// selection. Its positive fields are embedded; exclusions remain available
	// to lexical/policy diagnostics and are never embedded as positive text.
	RoutingProfile RoutingProfileConfig `json:"routing_profile,omitempty"`
	// Class is the operability class (one of the Class* constants). Optional in
	// this shared parser so capability-gap stubs and legacy descriptors still
	// load; Search Hub requires it for active providers.
	Class string `json:"class,omitempty"`
	// Lifecycle controls whether a provider participates in automatic routing.
	// Empty is production; fixture and experimental providers remain available
	// to explicit selectors while staying out of the classifier path.
	Lifecycle string `json:"lifecycle,omitempty"`

	// Descriptor sub-objects — opaque to aisearch-go (search-hub registry shapes).
	Endpoint       json.RawMessage `json:"endpoint,omitempty"`
	StatusEndpoint json.RawMessage `json:"status_endpoint,omitempty"`
	// IndexTimestampField names the status response field containing the last
	// successful index/reconcile timestamp. Search Hub defaults it to
	// last_indexed_at when omitted; dotted paths are supported.
	IndexTimestampField string `json:"index_timestamp_field,omitempty"`
	// FreshnessBudget is the maximum age Search Hub may accept for this
	// provider's active generation before withholding it from automatic routing.
	// It uses protobuf/Go duration syntax (for example "24h" or "900s").
	FreshnessBudget string          `json:"freshness_budget,omitempty"`
	ResultMapping   json.RawMessage `json:"result_mapping,omitempty"`
	// Secured control-plane targets (search-hub.v1.control.SearchControlService):
	// reindex_endpoint drives an async corpus reconcile; config_endpoint writes a
	// new tuning block back into this file. Both are opaque registry Endpoint
	// shapes here (mapped to the proto by searchregister-go) — a provider that
	// exposes the token-gated control plane declares them so search-hub can route
	// reindex/config-write to it. Absent for routable-but-not-tunable providers.
	ReindexEndpoint json.RawMessage `json:"reindex_endpoint,omitempty"`
	ConfigEndpoint  json.RawMessage `json:"config_endpoint,omitempty"`
	// ConfigWritable declares whether Search Hub may write tuning back through a
	// config endpoint. Indexed providers with no config_endpoint set this false
	// with ConfigPinnedReason when tuning is intentionally fixed by construction.
	ConfigWritable     *bool  `json:"config_writable,omitempty"`
	ConfigPinnedReason string `json:"config_pinned_reason,omitempty"`

	// Tuning is the factor values the adopter reads at boot and the sweep writes.
	Tuning TuningConfig `json:"tuning"`
	// Scoring is the provider's corpus gate policy. Defaults apply when omitted.
	Scoring ScoringConfig `json:"scoring,omitempty"`
	// Performance is the provider's latency/reliability budget. Optional: an
	// omitted block runs on the class default budget (DefaultP95BudgetMs).
	Performance PerformanceConfig `json:"performance,omitempty"`
	// Tests is the scenario's evaluation corpus — the SSOT for the suite that
	// search-hub's eval store mirrors (see searchregister.RegisterCorpus).
	Tests TestSuite `json:"tests"`
}

// RoutingProfileConfig describes the answer space and intent a provider leaf
// serves without coupling the router to a scenario or provider identity.
type RoutingProfileConfig struct {
	AnswerSpaces     []string `json:"answer_spaces,omitempty"`
	Intents          []string `json:"intents,omitempty"`
	PositiveExamples []string `json:"positive_examples,omitempty"`
	Exclusions       []string `json:"exclusions,omitempty"`
}

// Validate checks route facets as open vocabulary while rejecting values that
// cannot contribute useful evidence or would make the profile ambiguous.
func (p RoutingProfileConfig) Validate() error {
	if len(p.AnswerSpaces) == 0 && len(p.Intents) == 0 && len(p.PositiveExamples) == 0 && len(p.Exclusions) > 0 {
		return fmt.Errorf("routing_profile must declare at least one positive answer_space, intent, or positive_example")
	}
	for name, values := range map[string][]string{
		"answer_spaces":     p.AnswerSpaces,
		"intents":           p.Intents,
		"positive_examples": p.PositiveExamples,
		"exclusions":        p.Exclusions,
	} {
		seen := make(map[string]struct{}, len(values))
		for index, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				return fmt.Errorf("routing_profile.%s[%d] must not be blank", name, index)
			}
			key := strings.ToLower(value)
			if _, ok := seen[key]; ok {
				return fmt.Errorf("routing_profile.%s[%d] duplicates %q", name, index, value)
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

// PerformanceConfig is a provider's declared latency/reliability budget. The
// provider's operability class (see class) is the latency class; this block
// tightens the class default or opts into telemetry.
type PerformanceConfig struct {
	// P95Ms is the p95 query-latency budget in milliseconds. 0 ⇒ use the class
	// default (DefaultP95BudgetMs).
	P95Ms int `json:"p95_ms,omitempty"`
	// DegradedRateMax is the maximum tolerated fraction (0..1) of queries that
	// return no result. 0 ⇒ not checked (no declared reliability budget).
	DegradedRateMax float64 `json:"degraded_rate_max,omitempty"`
	// TelemetryRequired opts the provider into a hard requirement that latency
	// evidence exist: certification fails if no run latency can be measured.
	TelemetryRequired bool `json:"telemetry_required,omitempty"`
	// MinimumSamples is the minimum number of evaluated cases required before a
	// run can substantiate this provider's latency and degradation policy. Zero
	// means the class default (DefaultMinimumSamples).
	MinimumSamples int `json:"minimum_samples,omitempty"`
}

// Validate bounds-checks the performance policy.
func (p PerformanceConfig) Validate() error {
	if p.P95Ms < 0 {
		return fmt.Errorf("performance.p95_ms must be >= 0")
	}
	if p.DegradedRateMax < 0 || p.DegradedRateMax > 1 {
		return fmt.Errorf("performance.degraded_rate_max must be between 0 and 1")
	}
	if p.MinimumSamples < 0 {
		return fmt.Errorf("performance.minimum_samples must be >= 0")
	}
	return nil
}

// DefaultP95BudgetMs is the conservative p95 latency budget for a provider class
// when the descriptor declares no explicit performance.p95_ms. External and async
// providers get looser budgets by class — no scenario-name special-casing.
func DefaultP95BudgetMs(class string) int {
	switch strings.TrimSpace(class) {
	case ClassLocalIndex, ClassLocalLive:
		return 1500
	case ClassExternal:
		return 4000
	case ClassAsync:
		return 15000
	default:
		return 2000
	}
}

// DefaultMinimumSamples returns the smallest evaluated corpus that can support
// a production performance claim for a provider class. External providers are
// reachability/smoke-only by design; locally owned providers need a broader
// sample before their p95 and degradation figures are meaningful.
func DefaultMinimumSamples(class string) int {
	switch strings.TrimSpace(class) {
	case ClassExternal:
		return 1
	case ClassLocalIndex, ClassLocalLive, ClassAsync:
		return 8
	default:
		return 8
	}
}

// ScoringConfig is the JSON shape for a provider's corpus gate policy.
type ScoringConfig struct {
	RecallAt             int     `json:"recall_at,omitempty"`
	RecallTarget         float64 `json:"recall_target,omitempty"`
	MRRAt                int     `json:"mrr_at,omitempty"`
	DeepK                int     `json:"deep_k,omitempty"`
	JunkLeakOptOutReason string  `json:"junk_leak_opt_out_reason,omitempty"`
}

// TestSuite is a provider's evaluation corpus: the single source of truth that
// search-hub's eval store mirrors. After unification it is a 1:1 shape of the
// search-hub eval proto's EvalSuite (modulo the store-assigned provider_id /
// created_at / updated_at / state), so searchregister-go converts it losslessly
// and a scenario self-registers it at boot. There is exactly one labelled corpus
// per provider and exactly one case list — negatives are cases, not a separate
// array.
type TestSuite struct {
	// SuiteID overrides the default suite id "<provider_id>.primary"; only the
	// rare provider that owns more than one suite needs to set it.
	SuiteID string `json:"suite_id,omitempty"`
	// Name / Description are human-facing suite metadata mirrored into the store.
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// Cases is the whole corpus — positive labelled queries AND negatives. A
	// negative is just a case with ExpectNoStrongHit set; the sweep optimizes
	// against the positives and validates that negatives stay rejected.
	Cases []TestCase `json:"cases"`
	// Coverage declares the provider-owned question families/tags the reviewed
	// positive corpus must cover before production certification.
	Coverage CoverageConfig `json:"coverage,omitempty"`
	// Minimum is the registrant-declared bar for an ACTIVE production provider.
	// A zero value means no additional registration bar was declared.
	Minimum *EvalMinimum `json:"minimum,omitempty"`
}

// EvalMinimum is the generic registration bar for a provider-owned suite.
// Counts are deliberately about reviewed evidence, not scenario identity.
type EvalMinimum struct {
	ReviewedPositive int      `json:"reviewed_positive,omitempty"`
	Negative         int      `json:"negative,omitempty"`
	RequiredTags     []string `json:"required_tags,omitempty"`
}

// CoverageConfig declares corpus breadth requirements. Search Hub enforces
// these against reviewed positive cases only; candidates, generated cases, and
// negatives do not count as production acceptance evidence.
type CoverageConfig struct {
	// RequiredTags requires at least one reviewed positive for each tag.
	RequiredTags []string `json:"required_tags,omitempty"`
	// RequiredTagGroups lets a provider name a family and satisfy it with any one
	// of several tags. MinReviewedPositive defaults to 1.
	RequiredTagGroups []CoverageTagGroup `json:"required_tag_groups,omitempty"`
}

// CoverageTagGroup is one provider-owned question family.
type CoverageTagGroup struct {
	ID                  string   `json:"id"`
	Description         string   `json:"description,omitempty"`
	Tags                []string `json:"tags"`
	MinReviewedPositive int      `json:"min_reviewed_positive,omitempty"`
}

// TestCase is one labelled query in the canonical RANK-CENTRIC shape. A positive
// asserts ExpectIDs (leaf ids per the provider's id_field) landing within
// ExpectWithinTopK; a negative sets ExpectNoStrongHit (+ ExpectMaxScore). By
// design a positive carries NO absolute ExpectMinScore: the sweep compares arms
// across score regimes (dense cosine, cross-encoder, RRF-blend) where a shared
// absolute band would mislabel a rank-correct hit as a miss. ExpectMinScore /
// ExpectMaxScore stay available for a single-regime corpus but are not the
// default. Provenance rides Tags ("generated" is the load-bearing marker the
// sweep holds out of the tuning fold).
type TestCase struct {
	ID                string   `json:"id"`
	Query             string   `json:"query"`
	Scope             string   `json:"scope,omitempty"`
	Status            string   `json:"status,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	ExpectIDs         []string `json:"expect_ids,omitempty"`
	ExpectWithinTopK  int      `json:"expect_within_top_k,omitempty"`
	ExpectMinScore    float64  `json:"expect_min_score,omitempty"`
	ExpectMaxScore    float64  `json:"expect_max_score,omitempty"`
	ExpectNoStrongHit bool     `json:"expect_no_strong_hit,omitempty"`
	Note              string   `json:"note,omitempty"`
}

// TagGenerated marks a machine-generated (corpus-gen) case. The sweep ALWAYS
// holds these out of the tuning fold (overfit guard #2) — a tuning can never be
// selected on cases a machine wrote for it.
const TagGenerated = "generated"

const (
	CaseStatusReviewed  = "reviewed"
	CaseStatusCandidate = "candidate"
)

// ScoringPolicy is the resolved, defaults-filled corpus grading policy.
type ScoringPolicy struct {
	GateK        int
	RecallTarget float64
	MRRAt        int
	DeepK        int
}

// DefaultScoringPolicy is the shared gate policy used when search.json omits a
// provider scoring block.
var DefaultScoringPolicy = ScoringPolicy{
	GateK:        5,
	RecallTarget: 0.8,
	MRRAt:        3,
	DeepK:        50,
}

// WithDefaults fills omitted scoring fields.
func (c ScoringConfig) WithDefaults() ScoringPolicy {
	p := DefaultScoringPolicy
	if c.RecallAt > 0 {
		p.GateK = c.RecallAt
	}
	if c.RecallTarget > 0 {
		p.RecallTarget = c.RecallTarget
	}
	if c.MRRAt > 0 {
		p.MRRAt = c.MRRAt
	}
	if c.DeepK > 0 {
		p.DeepK = c.DeepK
	}
	if p.DeepK < p.GateK {
		p.DeepK = p.GateK
	}
	return p
}

// Validate checks scoring values for sane bounds.
func (c ScoringConfig) Validate() error {
	if c.RecallAt < 0 {
		return fmt.Errorf("scoring.recall_at must be >= 0")
	}
	if c.DeepK < 0 {
		return fmt.Errorf("scoring.deep_k must be >= 0")
	}
	if c.MRRAt < 0 {
		return fmt.Errorf("scoring.mrr_at must be >= 0")
	}
	if c.RecallTarget < 0 || c.RecallTarget > 1 {
		return fmt.Errorf("scoring.recall_target must be between 0 and 1")
	}
	if strings.TrimSpace(c.JunkLeakOptOutReason) == "" && c.JunkLeakOptOutReason != "" {
		return fmt.Errorf("scoring.junk_leak_opt_out_reason must not be blank")
	}
	return nil
}

// ResolvedStatus returns the effective review status. Empty means reviewed so
// existing hand-authored corpora do not need to spell the default.
func (c TestCase) ResolvedStatus() string {
	if strings.TrimSpace(c.Status) == "" {
		return CaseStatusReviewed
	}
	return c.Status
}

// IsCandidate reports whether this case is review-pending and excluded from
// acceptance recall.
func (c TestCase) IsCandidate() bool {
	return c.ResolvedStatus() == CaseStatusCandidate
}

// HasTag reports whether a case has tag.
func (c TestCase) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ResolvedScope parses the optional case-level scope string used by doc-like
// corpora. Empty/global search the whole provider; scenario:<id> and path:<p>
// map onto the shared SearchQuery scope contract.
func (c TestCase) ResolvedScope() Scope {
	scope := strings.TrimSpace(c.Scope)
	switch {
	case scope == "", scope == string(ScopeGlobal):
		return Scope{Kind: ScopeGlobal}
	case strings.HasPrefix(scope, string(ScopeScenario)+":"):
		return Scope{Kind: ScopeScenario, Value: strings.TrimSpace(strings.TrimPrefix(scope, string(ScopeScenario)+":"))}
	case strings.HasPrefix(scope, string(ScopePath)+":"):
		return Scope{Kind: ScopePath, Value: strings.TrimSpace(strings.TrimPrefix(scope, string(ScopePath)+":"))}
	default:
		return Scope{Kind: ScopeGlobal}
	}
}

// SuiteID returns the suite's id, defaulting to "<providerID>.primary" when the
// file does not set an explicit override.
func (s TestSuite) ResolvedSuiteID(providerID string) string {
	if strings.TrimSpace(s.SuiteID) != "" {
		return s.SuiteID
	}
	return providerID + ".primary"
}

// Validate checks the corpus is well-formed enough to persist: every case has a
// non-empty id + query and case ids are unique. It is intentionally light — the
// corpus is rank-centric data, not typed config; deeper adequacy (count floor,
// negatives, coverage) is search-hub's warn-level job, never a hard gate here.
func (s TestSuite) Validate() error {
	if err := s.Coverage.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool, len(s.Cases))
	for i, c := range s.Cases {
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("tests.cases[%d]: id is required", i)
		}
		if strings.TrimSpace(c.Query) == "" {
			return fmt.Errorf("tests.cases[%d] (%q): query is required", i, c.ID)
		}
		if err := validateCaseScope(c.Scope); err != nil {
			return fmt.Errorf("tests.cases[%d] (%q): %w", i, c.ID, err)
		}
		switch c.ResolvedStatus() {
		case CaseStatusReviewed, CaseStatusCandidate:
		default:
			return fmt.Errorf("tests.cases[%d] (%q): status must be %q or %q", i, c.ID, CaseStatusReviewed, CaseStatusCandidate)
		}
		if len(c.ExpectIDs) > 0 {
			ids := make(map[string]bool, len(c.ExpectIDs))
			for j, id := range c.ExpectIDs {
				id = strings.TrimSpace(id)
				if id == "" {
					return fmt.Errorf("tests.cases[%d] (%q): expect_ids[%d] is empty", i, c.ID, j)
				}
				if ids[id] {
					return fmt.Errorf("tests.cases[%d] (%q): duplicate expect_id %q", i, c.ID, id)
				}
				ids[id] = true
			}
		}
		if !c.ExpectNoStrongHit && hasPositiveExpectation(c) && len(c.ExpectIDs) == 0 {
			return fmt.Errorf("tests.cases[%d] (%q): positive cases require expect_ids", i, c.ID)
		}
		if seen[c.ID] {
			return fmt.Errorf("tests.cases: duplicate id %q", c.ID)
		}
		seen[c.ID] = true
	}
	return nil
}

// Validate checks that coverage declarations are coherent while leaving adequacy
// decisions to Search Hub's maturity validator.
func (c CoverageConfig) Validate() error {
	seenTags := make(map[string]bool, len(c.RequiredTags))
	for i, tag := range c.RequiredTags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return fmt.Errorf("tests.coverage.required_tags[%d] is empty", i)
		}
		if seenTags[tag] {
			return fmt.Errorf("tests.coverage.required_tags: duplicate tag %q", tag)
		}
		seenTags[tag] = true
	}
	seenGroups := make(map[string]bool, len(c.RequiredTagGroups))
	for i, group := range c.RequiredTagGroups {
		if strings.TrimSpace(group.ID) == "" {
			return fmt.Errorf("tests.coverage.required_tag_groups[%d].id is required", i)
		}
		if seenGroups[group.ID] {
			return fmt.Errorf("tests.coverage.required_tag_groups: duplicate id %q", group.ID)
		}
		seenGroups[group.ID] = true
		if len(group.Tags) == 0 {
			return fmt.Errorf("tests.coverage.required_tag_groups[%d].tags must not be empty", i)
		}
		seenGroupTags := make(map[string]bool, len(group.Tags))
		for j, tag := range group.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				return fmt.Errorf("tests.coverage.required_tag_groups[%d].tags[%d] is empty", i, j)
			}
			if seenGroupTags[tag] {
				return fmt.Errorf("tests.coverage.required_tag_groups[%d].tags: duplicate tag %q", i, tag)
			}
			seenGroupTags[tag] = true
		}
		if group.MinReviewedPositive < 0 {
			return fmt.Errorf("tests.coverage.required_tag_groups[%d].min_reviewed_positive must be >= 0", i)
		}
	}
	return nil
}

func hasPositiveExpectation(c TestCase) bool {
	return c.ExpectWithinTopK > 0 || c.ExpectMinScore > 0
}

func validateCaseScope(scope string) error {
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == string(ScopeGlobal) {
		return nil
	}
	for _, prefix := range []string{string(ScopeScenario) + ":", string(ScopePath) + ":"} {
		if strings.HasPrefix(scope, prefix) {
			if strings.TrimSpace(strings.TrimPrefix(scope, prefix)) == "" {
				return fmt.Errorf("scope %q requires a value", scope)
			}
			return nil
		}
	}
	return fmt.Errorf("scope must be global, scenario:<id>, or path:<prefix>")
}

// ParseSearchFile parses search.json bytes and validates every provider's
// tuning + structural invariants (version present, provider_ids present and
// unique). It does not touch the filesystem.
func ParseSearchFile(raw []byte) (SearchFile, error) {
	var f SearchFile
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return SearchFile{}, fmt.Errorf("parse search.json: %w", err)
	}
	if err := f.Validate(); err != nil {
		return SearchFile{}, err
	}
	return f, nil
}

// LoadSearchFile reads and validates the search.json at path.
func LoadSearchFile(path string) (SearchFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SearchFile{}, fmt.Errorf("read search.json %s: %w", path, err)
	}
	f, err := ParseSearchFile(raw)
	if err != nil {
		return SearchFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// Validate checks structural invariants and each provider's tuning.
func (f SearchFile) Validate() error {
	if strings.TrimSpace(f.Version) == "" {
		return fmt.Errorf("search.json: version is required")
	}
	if len(f.Providers) == 0 {
		return fmt.Errorf("search.json: at least one provider is required")
	}
	seen := make(map[string]bool, len(f.Providers))
	for i, p := range f.Providers {
		if strings.TrimSpace(p.ProviderID) == "" {
			return fmt.Errorf("search.json: providers[%d].provider_id is required", i)
		}
		if seen[p.ProviderID] {
			return fmt.Errorf("search.json: duplicate provider_id %q", p.ProviderID)
		}
		seen[p.ProviderID] = true
		if strings.TrimSpace(p.Class) != "" && !ValidProviderClass(p.Class) {
			return fmt.Errorf("search.json: provider %q: class %q must be one of %s", p.ProviderID, p.Class, strings.Join(KnownProviderClasses, ", "))
		}
		if err := p.Tuning.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
		}
		if err := p.Scoring.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
		}
		if err := p.Performance.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
		}
		if raw := strings.TrimSpace(p.FreshnessBudget); raw != "" {
			budget, err := time.ParseDuration(raw)
			if err != nil || budget < 0 {
				return fmt.Errorf("search.json: provider %q: freshness_budget %q is not a non-negative duration", p.ProviderID, raw)
			}
		}
		if err := p.RoutingProfile.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
		}
		if p.ConfigWritable != nil && !*p.ConfigWritable && strings.TrimSpace(p.ConfigPinnedReason) == "" {
			return fmt.Errorf("search.json: provider %q: config_pinned_reason is required when config_writable is false", p.ProviderID)
		}
		if err := p.Tests.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
		}
	}
	return nil
}

// Provider returns the provider with the given id (ok=false if absent).
func (f SearchFile) Provider(id string) (ProviderConfig, bool) {
	for _, p := range f.Providers {
		if p.ProviderID == id {
			return p, true
		}
	}
	return ProviderConfig{}, false
}

// ResolvedTuning returns the provider's tuning with taxonomy defaults filled.
func (p ProviderConfig) ResolvedTuning() TuningConfig { return p.Tuning.WithDefaults() }

// ResolvedScoring returns the provider's scoring policy with defaults filled.
func (p ProviderConfig) ResolvedScoring() ScoringPolicy { return p.Scoring.WithDefaults() }
