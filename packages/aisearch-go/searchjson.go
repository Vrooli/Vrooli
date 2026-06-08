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
	// Tests is the unified evaluation corpus (cases + negatives).
	Tests TestSuite `json:"tests"`
}

// TestSuite is the unified evaluation corpus. It collapses the two divergent
// cli-health corpora — the recall@5 gate's full-path labels and the search-hub
// eval suite's leaf ids + difficulty tags + a gibberish negative — into one
// shape, so there is exactly one labelled corpus per provider.
type TestSuite struct {
	// RecallAt / RecallTarget drive the scenario's per-build recall gate
	// (REQ-P0-004 for cli-health). 0 means "use the harness default" (5 / 0.8).
	RecallAt     int     `json:"recall_at,omitempty"`
	RecallTarget float64 `json:"recall_target,omitempty"`
	// Cases are the positive labelled queries; Negatives are queries that should
	// return no strong hit (junk-rejection guards). The sweep optimizes against
	// Cases and validates that Negatives stay rejected.
	Cases     []TestCase `json:"cases"`
	Negatives []TestCase `json:"negatives,omitempty"`
}

// TestCase is one labelled query. ExpectedPaths (full command/doc paths) and
// ExpectIDs (leaf ids) are alternative label shapes — a case may carry either or
// both; a recall gate that compares full paths reads ExpectedPaths, a leaf-id
// eval reads ExpectIDs. The Expect* score/rank fields are soft eval signals; a
// negative case uses ExpectNoStrongHit / ExpectMaxScore.
type TestCase struct {
	ID                string   `json:"id"`
	Query             string   `json:"query"`
	ExpectedPaths     []string `json:"expected_paths,omitempty"`
	ExpectIDs         []string `json:"expect_ids,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	ExpectWithinTopK  int      `json:"expect_within_top_k,omitempty"`
	ExpectMinScore    float64  `json:"expect_min_score,omitempty"`
	ExpectMaxScore    float64  `json:"expect_max_score,omitempty"`
	ExpectNoStrongHit bool     `json:"expect_no_strong_hit,omitempty"`
	// Source marks provenance: "" / "curated" is the human-authored golden core;
	// "generated" is a corpus-gen proposal the sweep holds out of tuning (overfit
	// guard). The constants below name the two values.
	Source string `json:"source,omitempty"`
	Note   string `json:"note,omitempty"`
}

// Test-case provenance values (TestCase.Source).
const (
	SourceCurated   = "curated"
	SourceGenerated = "generated"
)

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
