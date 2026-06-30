package aisearch

import (
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

// envPrefix scopes the engine's env config (e.g. SDA_QDRANT_URL, SDA_EMBED_MODEL)
// so SDA's knobs never collide with another adopter's. It is shared by every
// corpus: all SDA leaves embed into the same Qdrant/Ollama deployment under one
// reconcile/sync loop.
const envPrefix = "SDA"

// CorpusID is the stable identity of one SDA search corpus. Each corpus is its
// own Qdrant collection, point-id namespace, and federated provider leaf, but
// all share a single Reconciler / sync loop / embedder (the engine's native
// []SourceBinding model). New corpora (.scenarios, .resources) register here.
type CorpusID string

const (
	// CorpusDependencies is the original dependency-governance corpus
	// (scenario-dependency-analyzer.dependencies). It backs the SemanticRanker
	// surface dependencygovernance consumes.
	CorpusDependencies CorpusID = "dependencies"
	// CorpusScenarios is the scenario-connection corpus
	// (scenario-dependency-analyzer.scenarios): per scenario, what it depends on
	// and what depends on it.
	CorpusScenarios CorpusID = "scenarios"
	// CorpusResources is the resource-usage corpus
	// (scenario-dependency-analyzer.resources): per resource, which scenarios use
	// it.
	CorpusResources CorpusID = "resources"
)

// corpusSpec is the per-corpus identity + engine seams. One corpusSpec maps to
// exactly one Qdrant collection (collection), one logical kind, one point-id
// namespace (idPrefix), one index-time read tuning, and the three corpus-domain
// seams the shared read path overlays (source, rerankText, textFallback). The
// engine assembles N of these into one Reconciler via NewService below.
//
// INVARIANT: every corpus must declare the SAME embed recipe (tuning.EmbedModel
// + tuning.EmbedTaskPrefix) because all corpora share ONE embedder/Reconciler —
// a divergent embed recipe would index documents in a different vector space
// than the shared query embedder, silently breaking retrieval. Per-corpus tuning
// may differ ONLY on read-path factors (rerank_enabled/rerank_blend/floor/
// shortlist), which the per-corpus read Service honors independently.
type corpusSpec struct {
	id           CorpusID
	collection   string
	kind         string
	idPrefix     string
	tuning       pkg.TuningConfig
	source       pkg.Source
	rerankText   pkg.RerankTextFunc
	textFallback pkg.TextFallbackFunc
}

// Sources carries the per-corpus data providers Start binds. A nil field omits
// that corpus from the assembled Service, so a partially-wired deployment (e.g.
// tests, or a phase that has not landed its provider yet) degrades cleanly to
// the corpora it can build. Phase 2 adds Scenarios; Phase 3 adds Resources.
type Sources struct {
	// Dependencies supplies the governance records the .dependencies corpus
	// indexes (the dependencygovernance Registry implements RecordProvider).
	Dependencies RecordProvider
	// Scenarios supplies per-scenario connection records (.scenarios corpus); the
	// graph package implements it from the live interface graph.
	Scenarios ScenarioGraphProvider
	// Resources supplies per-resource usage records (.resources corpus); the
	// resourceusage package implements it via a fleet service.json scan.
	Resources ResourceUsageProvider
	// SearchJSONPath is the absolute path to the scenario's .vrooli/search.json.
	// When set, each corpus's tuning is read from its provider entry (the SSOT)
	// instead of the hardcoded default. A missing file/provider degrades to the
	// default (logged), never failing boot.
	SearchJSONPath string
}

// buildCorpusSpecs assembles the corpus descriptors for whichever providers are
// wired in sources, preserving a stable order (dependencies, scenarios,
// resources) so the reconciler's bindings are deterministic.
func buildCorpusSpecs(sources Sources) []corpusSpec {
	specs := make([]corpusSpec, 0, 3)
	if sources.Dependencies != nil {
		specs = append(specs, dependencyCorpus(sources.Dependencies))
	}
	if sources.Scenarios != nil {
		specs = append(specs, scenarioCorpus(sources.Scenarios))
	}
	if sources.Resources != nil {
		specs = append(specs, resourceCorpus(sources.Resources))
	}
	return specs
}

// ProviderID is the federated provider_id for a corpus
// (scenario-dependency-analyzer.<corpus>).
func ProviderID(corpus CorpusID) string {
	return "scenario-dependency-analyzer." + string(corpus)
}

// CorpusForProvider maps a federated provider_id back to its corpus. The control
// plane uses it to route a WriteConfig/ApplyTuning to the right corpus.
func CorpusForProvider(providerID string) (CorpusID, bool) {
	const prefix = "scenario-dependency-analyzer."
	if !strings.HasPrefix(providerID, prefix) {
		return "", false
	}
	id := CorpusID(strings.TrimPrefix(providerID, prefix))
	switch id {
	case CorpusDependencies, CorpusScenarios, CorpusResources:
		return id, true
	default:
		return "", false
	}
}

// tokenize splits a query into deduplicated lower-cased terms ≥2 chars, the
// shared keyword-leg tokenizer every corpus's text fallback uses.
func tokenize(q string) []string {
	q = strings.ToLower(q)
	fields := strings.FieldsFunc(q, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';', ':', '/', '\\', '-', '_', '.', '"', '\'', '(', ')', '[', ']':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len(f) < 2 {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}
