// Package aisearch adapts the shared packages/ai-go/search engine to
// scenario-dependency-analyzer's search corpora. SDA hosts multiple federated
// leaves — the dependency-governance corpus (.dependencies), the
// scenario-connection corpus (.scenarios), and the resource-usage corpus
// (.resources) — each its own Qdrant collection and federated provider, but all
// driven by ONE Reconciler / sync loop / embedder (the engine's native
// []SourceBinding model). This file owns the .dependencies corpus: one SourceDoc
// per governance record, a dense single-chunk index, and the primitive RankIDs
// surface dependencygovernance consumes without importing this package (no import
// cycle).
package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

const (
	// dependenciesCollection is the Qdrant collection for the dependency
	// governance corpus.
	dependenciesCollection = "scenario-dependency-analyzer-dependencies"
	// dependenciesKind is the logical collection the governance records belong to.
	dependenciesKind = "dependency"
	// dependenciesIDPrefix namespaces the dependency corpus's point IDs inside the
	// shared engine.
	dependenciesIDPrefix = "sda-dep:"
)

// dependenciesDefaultTuning is the v1 engine recipe for the governance corpus:
// dense, nomic task prefixes on, rerank off. (Phase 5 reads this from
// .vrooli/search.json so it becomes a sweep-tunable lever; this is the hardcoded
// fallback.) The embed recipe (EmbedModel + EmbedTaskPrefix) must match every
// other SDA corpus — see the corpusSpec INVARIANT.
func dependenciesDefaultTuning() pkg.TuningConfig {
	return pkg.TuningConfig{
		Engine:          pkg.EngineDense,
		EmbedModel:      pkg.DefaultEmbedModel,
		EmbedTaskPrefix: true,
		RerankEnabled:   false,
	}.WithDefaults()
}

// dependencyCorpus assembles the .dependencies corpus descriptor over the given
// record provider.
func dependencyCorpus(provider RecordProvider) corpusSpec {
	return corpusSpec{
		id:           CorpusDependencies,
		collection:   dependenciesCollection,
		kind:         dependenciesKind,
		idPrefix:     dependenciesIDPrefix,
		tuning:       dependenciesDefaultTuning(),
		source:       &recordSource{provider: provider},
		rerankText:   func(r pkg.SearchResult) string { return candidateText(r.Payload) },
		textFallback: dependencyTextFallback(provider),
	}
}

// RecordProvider supplies the governance records the index is built from. The
// Registry implements it via its exported AllRecords method; defining the
// interface here (rather than importing dependencygovernance) keeps this package
// dependency-free of its consumer.
type RecordProvider interface {
	AllRecords(ctx context.Context) ([]*governancev1.ApprovedDependencyRecord, error)
}

// recordID is the stable natural identity of a record: ecosystem/package,
// lower-cased so lookups are case-insensitive. It matches the recordKey scheme
// the registry uses elsewhere.
func recordID(r *governancev1.ApprovedDependencyRecord) string {
	return strings.ToLower(strings.TrimSpace(r.GetEcosystem()) + "/" + strings.TrimSpace(r.GetPackageName()))
}

// composeEmbeddingText builds the input passed to the embedder: identity first,
// then the human-vocabulary fields a user actually queries with (use-cases,
// rationale, keywords), then security + provenance. Short and dense.
func composeEmbeddingText(r *governancev1.ApprovedDependencyRecord) string {
	var parts []string
	parts = append(parts, r.GetEcosystem()+"/"+r.GetPackageName())
	if v := strings.TrimSpace(r.GetVersionRange()); v != "" {
		parts = append(parts, "Version range: "+v)
	}
	if len(r.GetUseCases()) > 0 {
		parts = append(parts, "Use cases: "+strings.Join(r.GetUseCases(), "; "))
	}
	if rat := strings.TrimSpace(r.GetRationale()); rat != "" {
		parts = append(parts, rat)
	}
	if len(r.GetKeywords()) > 0 {
		parts = append(parts, "Keywords: "+strings.Join(r.GetKeywords(), ", "))
	}
	if len(r.GetAllowedSurfaces()) > 0 {
		parts = append(parts, "Surfaces: "+strings.Join(r.GetAllowedSurfaces(), ", "))
	}
	if sn := strings.TrimSpace(r.GetSecurityNotes()); sn != "" {
		parts = append(parts, "Security: "+sn)
	}
	if len(r.GetExampleScenarios()) > 0 {
		parts = append(parts, "Used by: "+strings.Join(r.GetExampleScenarios(), ", "))
	}
	parts = append(parts, "State: "+r.GetState())
	return strings.Join(parts, "\n\n")
}

// recordMeta is the payload propagated into the chunk payload; the registry maps
// the ranked IDs back to full records, so Meta only needs the facet fields used
// for filtering and quick projection.
func recordMeta(r *governancev1.ApprovedDependencyRecord) map[string]any {
	return map[string]any{
		"ecosystem":         r.GetEcosystem(),
		"package_name":      r.GetPackageName(),
		"version_range":     r.GetVersionRange(),
		"state":             r.GetState(),
		"allowed_surfaces":  r.GetAllowedSurfaces(),
		"keywords":          r.GetKeywords(),
		"use_cases":         r.GetUseCases(),
		"example_scenarios": r.GetExampleScenarios(),
	}
}

// contentHash is the source-level drift gate: a stable hash over the fields that
// affect retrieval, so a warm reconcile tick skips an unchanged record before
// embedding. Editing any retrieval-relevant field changes the hash.
func contentHash(r *governancev1.ApprovedDependencyRecord) string {
	h := sha256.New()
	h.Write([]byte(composeEmbeddingText(r)))
	sum := h.Sum(nil)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// recordToSourceDoc adapts one governance record to the engine's SourceDoc.
func recordToSourceDoc(r *governancev1.ApprovedDependencyRecord) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:          recordID(r),
		Kind:        dependenciesKind,
		ContentHash: contentHash(r),
		Body:        composeEmbeddingText(r),
		Meta:        recordMeta(r),
	}
}

// recordSource adapts the RecordProvider to the engine's Source interface: one
// SourceDoc per governance record.
type recordSource struct {
	provider RecordProvider
}

func (s *recordSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	records, err := s.provider.AllRecords(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, len(records))
	for _, r := range records {
		if strings.TrimSpace(r.GetPackageName()) == "" {
			continue
		}
		out = append(out, recordToSourceDoc(r))
	}
	return out, nil
}

// pointIDForRecord returns the deterministic Qdrant point ID for a record.
// Records are single-chunk sources, so this is the un-suffixed UUIDv5.
func pointIDForRecord(id string) string {
	return pkg.PointIDFor(dependenciesIDPrefix, strings.TrimSpace(id), 0, 1)
}

// sourceIDFromResult recovers the natural record ID (ecosystem/package) from a
// result payload — the engine stores it under source_id; fall back to composing
// it from the ecosystem/package_name meta fields.
func sourceIDFromResult(r pkg.SearchResult) string {
	if r.SourceID != "" {
		return strings.ToLower(strings.TrimSpace(r.SourceID))
	}
	if v, ok := r.Payload["source_id"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.ToLower(strings.TrimSpace(v))
	}
	eco, _ := r.Payload["ecosystem"].(string)
	pkgName, _ := r.Payload["package_name"].(string)
	if strings.TrimSpace(eco) == "" || strings.TrimSpace(pkgName) == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(eco) + "/" + strings.TrimSpace(pkgName))
}

// candidateText is the passage handed to the reranker for one dependency hit.
func candidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	eco, _ := payload["ecosystem"].(string)
	pkgName, _ := payload["package_name"].(string)
	return strings.TrimSpace(eco + "/" + pkgName)
}

// dependencyTextFallback is the offline-safe keyword leg over the live governance
// records, used when Ollama/Qdrant are unavailable. It returns pkg.SearchResult
// with the record's natural ID under source_id so RankIDs treats it identically
// to a vector hit.
func dependencyTextFallback(provider RecordProvider) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		records, err := provider.AllRecords(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			id    string
		}
		hits := make([]scored, 0, len(records))
		for _, r := range records {
			score := scoreRecord(r, terms)
			if score <= 0 {
				continue
			}
			id := recordID(r)
			payload := recordMeta(r)
			payload["source_id"] = id
			hits = append(hits, scored{
				res:   pkg.SearchResult{ID: pointIDForRecord(id), Score: score, Payload: payload, SourceID: id},
				score: score,
				id:    id,
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

// scoreRecord is a weighted substring match: identity > keywords/use-cases >
// rationale/security. Normalized to (0,1].
func scoreRecord(r *governancev1.ApprovedDependencyRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	identity := strings.ToLower(r.GetEcosystem() + " " + r.GetPackageName())
	vocab := strings.ToLower(strings.Join(r.GetKeywords(), " ") + " " + strings.Join(r.GetUseCases(), " "))
	prose := strings.ToLower(r.GetRationale() + " " + r.GetSecurityNotes())
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(identity, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(vocab, t) {
			score += 0.5
			hit = true
		}
		if strings.Contains(prose, t) {
			score += 0.3
			hit = true
		}
		if !hit {
			score -= 0.05
		}
	}
	if score <= 0 {
		return 0
	}
	max := float64(len(terms)) * 1.8
	if max <= 0 {
		return 0
	}
	n := score / max
	if n > 1 {
		n = 1
	}
	return n
}
