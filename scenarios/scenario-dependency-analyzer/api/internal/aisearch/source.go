// Package aisearch adapts the shared packages/ai-go/search engine to
// scenario-dependency-analyzer's governance records, giving SDA AI semantic
// search over the approved/denied/needs-review dependency corpus. It mirrors the
// cli-health adopter pattern (one SourceDoc per record, dense single-chunk
// index, text fallback when Qdrant/Ollama are down) and exposes a primitive
// RankIDs surface so dependencygovernance can rank records without importing this
// package (no import cycle).
package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

const (
	// DefaultCollection is the per-adopter Qdrant collection for the dependency
	// governance corpus.
	DefaultCollection = "scenario-dependency-analyzer-dependencies"
	// envPrefix scopes the engine's env config (e.g. SDA_QDRANT_URL,
	// SDA_EMBED_MODEL) so SDA's knobs never collide with another adopter's.
	envPrefix = "SDA"
	// recordKind is the logical collection the governance records belong to.
	recordKind = "dependency"
	// idPrefix namespaces SDA's point IDs inside the shared engine.
	idPrefix = "sda-dep:"
)

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
		Kind:        recordKind,
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
	return pkg.PointIDFor(idPrefix, strings.TrimSpace(id), 0, 1)
}
