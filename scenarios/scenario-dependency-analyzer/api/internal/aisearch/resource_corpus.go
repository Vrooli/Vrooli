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
	// resourcesCollection is the Qdrant collection for the resource-usage corpus.
	resourcesCollection = "scenario-dependency-analyzer-resources"
	// resourcesKind is the logical collection resource-usage records belong to.
	resourcesKind = "resource-usage"
	// resourcesIDPrefix namespaces the resources corpus's point IDs inside the
	// shared engine.
	resourcesIDPrefix = "sda-res:"
)

// ResourceUsage is one local resource's fleet footprint: which scenarios declare
// (use) it. It is the unit the .resources corpus indexes (one SourceDoc each).
type ResourceUsage struct {
	// Resource is the resource name (stable natural ID): postgres, qdrant, ...
	Resource string
	// Type is the declared resource type when present (often equal to Resource).
	Type string
	// UsedBy lists the scenarios that declare this resource, sorted, deduplicated.
	UsedBy []string
}

// ResourceUsageProvider supplies the resource-usage records the .resources corpus
// indexes. The resourceusage package implements it via a fleet service.json scan.
type ResourceUsageProvider interface {
	ResourceUsages(ctx context.Context) ([]ResourceUsage, error)
}

// resourceCorpus assembles the .resources corpus descriptor over the given usage
// provider.
func resourceCorpus(provider ResourceUsageProvider) corpusSpec {
	return corpusSpec{
		id:           CorpusResources,
		collection:   resourcesCollection,
		kind:         resourcesKind,
		idPrefix:     resourcesIDPrefix,
		tuning:       resourcesDefaultTuning(),
		source:       &resourceSource{provider: provider},
		rerankText:   func(r pkg.SearchResult) string { return resourceCandidateText(r.Payload) },
		textFallback: resourceTextFallback(provider),
	}
}

// resourcesDefaultTuning is the v1 engine recipe for the resource-usage corpus.
// The embed recipe (EmbedModel + EmbedTaskPrefix) MUST match every other SDA
// corpus (the corpusSpec INVARIANT). Phase 5 reads this from .vrooli/search.json.
func resourcesDefaultTuning() pkg.TuningConfig {
	return pkg.TuningConfig{
		Engine:          pkg.EngineDense,
		EmbedModel:      pkg.DefaultEmbedModel,
		EmbedTaskPrefix: true,
		RerankEnabled:   false,
	}.WithDefaults()
}

// composeResourceEmbeddingText renders the embeddable body in the natural
// language a user queries with ("which scenarios use postgres").
func composeResourceEmbeddingText(u ResourceUsage) string {
	var parts []string
	parts = append(parts, "Resource: "+u.Resource)
	if t := strings.TrimSpace(u.Type); t != "" && t != u.Resource {
		parts = append(parts, "Type: "+t)
	}
	if len(u.UsedBy) > 0 {
		parts = append(parts, "Used by scenarios: "+strings.Join(u.UsedBy, ", "))
	} else {
		parts = append(parts, "Used by scenarios: (none)")
	}
	return strings.Join(parts, ". ") + "."
}

func resourceMeta(u ResourceUsage) map[string]any {
	return map[string]any{
		"resource":  u.Resource,
		"type":      u.Type,
		"used_by":   u.UsedBy,
		"source_id": u.Resource,
	}
}

func resourceContentHash(u ResourceUsage) string {
	h := sha256.New()
	h.Write([]byte(composeResourceEmbeddingText(u)))
	sum := h.Sum(nil)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// resourceSource adapts the ResourceUsageProvider to the engine's Source.
type resourceSource struct {
	provider ResourceUsageProvider
}

func (s *resourceSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	usages, err := s.provider.ResourceUsages(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, len(usages))
	for _, u := range usages {
		if strings.TrimSpace(u.Resource) == "" {
			continue
		}
		out = append(out, pkg.SourceDoc{
			ID:          u.Resource,
			Kind:        resourcesKind,
			ContentHash: resourceContentHash(u),
			Body:        composeResourceEmbeddingText(u),
			Meta:        resourceMeta(u),
		})
	}
	return out, nil
}

func resourceCandidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	res, _ := payload["resource"].(string)
	return strings.TrimSpace(res)
}

// resourceTextFallback is the offline-safe keyword leg over freshly scanned
// resource usages, used when Ollama/Qdrant are unavailable.
func resourceTextFallback(provider ResourceUsageProvider) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		usages, err := provider.ResourceUsages(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			id    string
		}
		hits := make([]scored, 0, len(usages))
		for _, u := range usages {
			score := scoreResource(u, terms)
			if score <= 0 {
				continue
			}
			payload := resourceMeta(u)
			hits = append(hits, scored{
				res:   pkg.SearchResult{ID: pkg.PointIDFor(resourcesIDPrefix, u.Resource, 0, 1), Score: score, Payload: payload, SourceID: u.Resource},
				score: score,
				id:    u.Resource,
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

// scoreResource is the keyword-leg score for one resource-usage record. The
// resource identity dominates; the consuming-scenario list contributes so a query
// naming a scenario can surface the resources it uses.
func scoreResource(u ResourceUsage, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	ident := strings.ToLower(u.Resource + " " + u.Type)
	consumers := strings.ToLower(strings.Join(u.UsedBy, " "))
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(ident, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(consumers, t) {
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
