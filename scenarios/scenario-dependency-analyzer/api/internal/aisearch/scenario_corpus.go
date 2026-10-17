package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

const (
	// scenariosCollection is the Qdrant collection for the scenario-connection
	// corpus.
	scenariosCollection = "scenario-dependency-analyzer-scenarios"
	// scenariosKind is the logical collection scenario-connection records belong to.
	scenariosKind = "scenario-connection"
	// scenariosIDPrefix namespaces the scenarios corpus's point IDs inside the
	// shared engine.
	scenariosIDPrefix = "sda-scn:"
)

// ScenarioConnection is one scenario's place in the cross-scenario interface
// graph: what it depends on (forward edges) and what depends on it (reverse
// edges). It is the unit the .scenarios corpus indexes (one SourceDoc each).
type ScenarioConnection struct {
	// Scenario is the scenario name (stable natural ID).
	Scenario string
	// DependsOn lists the scenarios this scenario connects out to (its
	// dependencies), sorted, deduplicated.
	DependsOn []string
	// UsedBy lists the scenarios that connect in to this scenario (its reverse
	// dependents), sorted, deduplicated.
	UsedBy []string
}

// ScenarioGraphProvider supplies the scenario-connection records the .scenarios
// corpus indexes. The graph package implements it from the live interface graph;
// defining the interface here keeps aisearch dependency-free of its provider.
type ScenarioGraphProvider interface {
	ScenarioConnections(ctx context.Context) ([]ScenarioConnection, error)
}

// scenarioCorpus assembles the .scenarios corpus descriptor over the given
// connection provider. Connection retrieval is a precision/junk-rejection task
// where a connection doc and a domain-map purpose doc are close in embedding
// space (the prototype watch-item, ~0.035 margin), so rerank is the lever Phase
// 5 sweeps — the hardcoded fallback leaves it on dense+task-prefix, rerank-off
// until the sweep confirms it.
func scenarioCorpus(provider ScenarioGraphProvider) corpusSpec {
	return corpusSpec{
		id:           CorpusScenarios,
		collection:   scenariosCollection,
		kind:         scenariosKind,
		idPrefix:     scenariosIDPrefix,
		tuning:       scenariosDefaultTuning(),
		source:       &scenarioSource{provider: provider},
		rerankText:   func(r pkg.SearchResult) string { return scenarioCandidateText(r.Payload) },
		textFallback: scenarioTextFallback(provider),
	}
}

// scenariosDefaultTuning is the v1 engine recipe for the connection corpus. The
// embed recipe (EmbedModel + EmbedTaskPrefix) MUST match every other SDA corpus
// (the corpusSpec INVARIANT). Phase 5 reads this from .vrooli/search.json.
func scenariosDefaultTuning() pkg.TuningConfig {
	return pkg.TuningConfig{
		Engine:          pkg.EngineDense,
		EmbedModel:      pkg.DefaultEmbedModel,
		EmbedTaskPrefix: true,
		RerankEnabled:   false,
	}.WithDefaults()
}

// composeScenarioEmbeddingText renders the embeddable body. Identity first, then
// the two connection directions in natural language a user queries with ("what
// does X depend on", "what depends on X"). Prototype-validated rendering.
func composeScenarioEmbeddingText(c ScenarioConnection) string {
	var parts []string
	parts = append(parts, "Scenario: "+c.Scenario)
	if len(c.DependsOn) > 0 {
		parts = append(parts, "Depends on: "+strings.Join(c.DependsOn, ", "))
	} else {
		parts = append(parts, "Depends on: (no scenario dependencies)")
	}
	if len(c.UsedBy) > 0 {
		parts = append(parts, "Used by: "+strings.Join(c.UsedBy, ", "))
	} else {
		parts = append(parts, "Used by: (no scenario dependents)")
	}
	return strings.Join(parts, ". ") + "."
}

// scenarioMeta is the payload propagated into the chunk payload for projection.
func scenarioMeta(c ScenarioConnection) map[string]any {
	return map[string]any{
		"scenario":   c.Scenario,
		"depends_on": c.DependsOn,
		"used_by":    c.UsedBy,
		"source_id":  c.Scenario,
	}
}

// scenarioContentHash is the source-level drift gate over the retrieval-relevant
// fields, so an unchanged scenario skips re-embedding on a warm reconcile.
func scenarioContentHash(c ScenarioConnection) string {
	h := sha256.New()
	h.Write([]byte(composeScenarioEmbeddingText(c)))
	sum := h.Sum(nil)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// scenarioSource adapts the ScenarioGraphProvider to the engine's Source.
type scenarioSource struct {
	provider ScenarioGraphProvider
}

func (s *scenarioSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	conns, err := s.provider.ScenarioConnections(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, len(conns))
	for _, c := range conns {
		if strings.TrimSpace(c.Scenario) == "" {
			continue
		}
		out = append(out, pkg.SourceDoc{
			ID:          c.Scenario,
			Kind:        scenariosKind,
			ContentHash: scenarioContentHash(c),
			Body:        composeScenarioEmbeddingText(c),
			Meta:        scenarioMeta(c),
		})
	}
	return out, nil
}

// scenarioCandidateText is the passage handed to the reranker for one hit.
func scenarioCandidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	scenario, _ := payload["scenario"].(string)
	return strings.TrimSpace(scenario)
}

// scenarioTextFallback is the offline-safe keyword leg over freshly derived
// connections, used when Ollama/Qdrant are unavailable.
func scenarioTextFallback(provider ScenarioGraphProvider) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		conns, err := provider.ScenarioConnections(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			id    string
		}
		hits := make([]scored, 0, len(conns))
		for _, c := range conns {
			score := scoreScenario(c, terms)
			if score <= 0 {
				continue
			}
			payload := scenarioMeta(c)
			hits = append(hits, scored{
				res:   pkg.SearchResult{ID: pkg.PointIDFor(scenariosIDPrefix, c.Scenario, 0, 1), Score: score, Payload: payload, SourceID: c.Scenario},
				score: score,
				id:    c.Scenario,
			})
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].score != hits[j].score {
				return hits[i].score > hits[j].score
			}
			return hits[i].id < hits[j].id
		})
		out := make([]pkg.SearchResult, 0, len(hits))
		for _, h := range hits {
			out = append(out, h.res)
		}
		return out, nil
	}
}

// scoreScenario is the keyword-leg score for one connection record. Identity
// dominates; connection lists contribute so "what depends on X" matches the
// scenarios that list X under used_by/depends_on.
func scoreScenario(c ScenarioConnection, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	ident := strings.ToLower(c.Scenario)
	conns := strings.ToLower(strings.Join(c.DependsOn, " ") + " " + strings.Join(c.UsedBy, " "))
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(ident, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(conns, t) {
			score += 0.5
			hit = true
		}
		if !hit {
			score -= 0.05
		}
	}
	if score <= 0 {
		return 0
	}
	max := float64(len(terms)) * 1.5
	if max <= 0 {
		return 0
	}
	n := score / max
	if n > 1 {
		n = 1
	}
	if n < 0 {
		n = 0
	}
	return n
}
