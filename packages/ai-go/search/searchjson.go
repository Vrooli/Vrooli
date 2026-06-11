package aisearch

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

// ProviderConfig is one provider's full descriptor + tuning + test corpus.
type ProviderConfig struct {
	ProviderID    string `json:"provider_id"`
	ProviderGroup string `json:"provider_group"`
	Bucket        string `json:"bucket"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Scope         string `json:"scope"`

	// Descriptor sub-objects — opaque to aisearch-go (search-hub registry shapes).
	Endpoint       json.RawMessage `json:"endpoint,omitempty"`
	StatusEndpoint json.RawMessage `json:"status_endpoint,omitempty"`
	ResultMapping  json.RawMessage `json:"result_mapping,omitempty"`
	// Secured control-plane targets (search-hub.v1.control.SearchControlService):
	// reindex_endpoint drives an async corpus reconcile; config_endpoint writes a
	// new tuning block back into this file. Both are opaque registry Endpoint
	// shapes here (mapped to the proto by searchregister-go) — a provider that
	// exposes the token-gated control plane declares them so search-hub can route
	// reindex/config-write to it. Absent for routable-but-not-tunable providers.
	ReindexEndpoint json.RawMessage `json:"reindex_endpoint,omitempty"`
	ConfigEndpoint  json.RawMessage `json:"config_endpoint,omitempty"`

	// Tuning is the factor values the adopter reads at boot and the sweep writes.
	Tuning TuningConfig `json:"tuning"`
	// Scoring is the provider's corpus gate policy. Defaults apply when omitted.
	Scoring ScoringConfig `json:"scoring,omitempty"`
	// Tests is the scenario's evaluation corpus — the SSOT for the suite that
	// search-hub's eval store mirrors (see searchregister.RegisterCorpus).
	Tests TestSuite `json:"tests"`
}

// ScoringConfig is the JSON shape for a provider's corpus gate policy.
type ScoringConfig struct {
	RecallAt     int     `json:"recall_at,omitempty"`
	RecallTarget float64 `json:"recall_target,omitempty"`
	MRRAt        int     `json:"mrr_at,omitempty"`
	DeepK        int     `json:"deep_k,omitempty"`
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
		if err := p.Tuning.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
		}
		if err := p.Scoring.Validate(); err != nil {
			return fmt.Errorf("search.json: provider %q: %w", p.ProviderID, err)
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
