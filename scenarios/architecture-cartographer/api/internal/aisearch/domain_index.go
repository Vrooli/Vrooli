package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"

	"architecture-cartographer/internal/domains"

	pkg "github.com/vrooli/ai-go/search"
)

// DomainProvider is the seam over the in-process domain derivation. Production
// wires the cartographer's domains.Service; tests inject a fake. The Source
// derives the map in-process — it NEVER shells `architecture-cartographer
// domains show` (plan §12).
type DomainProvider interface {
	ExtractDomains(ctx context.Context, scenario string) (domains.DerivedDomainMap, error)
}

// ScenarioLister enumerates the scenarios whose domains are indexed. Production
// wires audit.DirScenarioLister (every directory under scenarios/); tests inject
// a fake. Project-scope: the corpus spans ALL scenarios' domains, not just the
// cartographer's own (plan C3).
type ScenarioLister interface {
	List(ctx context.Context) ([]string, error)
}

// toDomainRecord flattens one DerivedDomain (plus its map-level authority
// context) into the indexable DomainRecord.
func toDomainRecord(scenario string, m domains.DerivedDomainMap, d domains.DerivedDomain) DomainRecord {
	return DomainRecord{
		ID:             scenario + "/" + d.Name,
		Scenario:       scenario,
		Name:           d.Name,
		Responsibility: strings.TrimSpace(d.Responsibility),
		Purpose:        strings.TrimSpace(d.Purpose),
		OwnsData:       strings.TrimSpace(d.OwnsData),
		Glossary:       append([]string(nil), d.Glossary...),
		Archetype:      d.PrimaryArchetype(),
		Surfaces:       append([]string(nil), d.Surfaces...),
		Paths:          append([]string(nil), d.Paths...),
		Authority:      string(m.Authority),
		Confidence:     string(m.AuthorityConfidence),
	}
}

// composeDomainEmbeddingText builds the input passed to the embedder — the
// term-agnostic anchor. It folds the natural-language IDENTITY of the domain:
// the scenario + name (so "...in plan-manager" lands on plan-manager's domains)
// followed by the human-authored responsibility, purpose, owns-data, and
// glossary. The structured FACETS (archetype, surfaces, paths, authority) are
// deliberately excluded from the prose — they are metadata filters, not
// retrieval signal (plan §12 / design principle 2). This is the seam the eval
// sweep measures; an enriched variant should be A/B'd via search-hub evals
// before adoption.
func composeDomainEmbeddingText(r DomainRecord) string {
	parts := make([]string, 0, 6)
	identity := strings.TrimSpace(r.Scenario + " " + r.Name)
	if identity != "" {
		parts = append(parts, identity)
	}
	if r.Responsibility != "" {
		parts = append(parts, r.Responsibility)
	}
	if r.Purpose != "" {
		parts = append(parts, r.Purpose)
	}
	if r.OwnsData != "" {
		parts = append(parts, "Owns: "+r.OwnsData)
	}
	if len(r.Glossary) > 0 {
		parts = append(parts, "Vocabulary: "+strings.Join(r.Glossary, ", "))
	}
	return strings.Join(parts, "\n\n")
}

// domainMeta returns the per-domain payload fields propagated into the chunk
// payload by the shared engine (it appends body / source_id / payload_hash).
// payloadToHit projects these keys back into a DomainHit.
func domainMeta(r DomainRecord) map[string]any {
	return map[string]any{
		"id":             r.ID,
		"scenario":       r.Scenario,
		"name":           r.Name,
		"responsibility": r.Responsibility,
		"purpose":        r.Purpose,
		"owns_data":      r.OwnsData,
		"glossary":       r.Glossary,
		"archetype":      r.Archetype,
		"surfaces":       r.Surfaces,
		"paths":          r.Paths,
		"authority":      r.Authority,
		"confidence":     r.Confidence,
	}
}

// domainContentHash is the source-level drift gate: a stable hash of the whole
// record so a warm reconcile tick skips an unchanged domain before embedding.
// Editing any field changes the hash, so a re-derivation after a DOMAINS.md edit
// re-embeds only the affected domains.
func domainContentHash(r DomainRecord) string {
	b, _ := json.Marshal(r)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// domainToSourceDoc adapts one DomainRecord to the engine's SourceDoc. Body is
// the pre-composed embedding text (the identity composer embeds it verbatim);
// the domain facets ride along as Meta for filtering + result projection. The
// compose function is injectable so the embedding-text strategy stays a
// measurable seam.
func domainToSourceDoc(r DomainRecord, compose func(DomainRecord) string) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:          r.ID,
		Kind:        domainKind,
		ContentHash: domainContentHash(r),
		Body:        compose(r),
		Meta:        domainMeta(r),
	}
}

// domainSource adapts the cartographer's in-process domain derivation to the
// engine's Source interface: one SourceDoc per domain across every scenario.
type domainSource struct {
	provider DomainProvider
	lister   ScenarioLister
	compose  func(DomainRecord) string
}

func newDomainSource(provider DomainProvider, lister ScenarioLister, compose func(DomainRecord) string) *domainSource {
	if compose == nil {
		compose = composeDomainEmbeddingText
	}
	return &domainSource{provider: provider, lister: lister, compose: compose}
}

// LoadAll enumerates every scenario's domains as SourceDocs. A derivation
// failure for one scenario (no DOMAINS.md, unreadable dir) is logged and
// skipped — indexing never crashes because one scenario lacks a domain map.
func (s *domainSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	scenarios, err := s.lister.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, 256)
	for _, scenario := range scenarios {
		m, err := s.provider.ExtractDomains(ctx, scenario)
		if err != nil {
			log.Printf("[architecture-cartographer/aisearch] extract domains %s: %v", scenario, err)
			continue
		}
		for _, d := range m.Domains {
			out = append(out, domainToSourceDoc(toDomainRecord(scenario, m, d), s.compose))
		}
	}
	return out, nil
}

// pointIDForDomain returns the deterministic Qdrant point ID for a domain.
// Domains are single-chunk sources, so this is the un-suffixed UUIDv5.
func pointIDForDomain(id string) string {
	return pkg.PointIDFor(idPrefix, strings.TrimSpace(id), 0, 1)
}

// payloadToHit projects a vector-store payload back into a DomainHit. Returns a
// best-effort hit when fields are missing — never panics.
func payloadToHit(id string, score float64, payload map[string]any) DomainHit {
	hit := DomainHit{ID: id, Score: score, ScorePercent: int(score*100 + 0.5)}
	if v, _ := payload["id"].(string); v != "" {
		hit.ID = v
	}
	hit.Scenario, _ = payload["scenario"].(string)
	hit.Name, _ = payload["name"].(string)
	hit.Responsibility, _ = payload["responsibility"].(string)
	hit.Purpose, _ = payload["purpose"].(string)
	hit.Archetype, _ = payload["archetype"].(string)
	hit.Paths = stringSliceFromPayload(payload["paths"])
	return hit
}

// stringSliceFromPayload extracts a []string from a payload value that may be a
// []string (in-memory upsert) or a []interface{} (decoded from Qdrant JSON).
func stringSliceFromPayload(v any) []string {
	switch raw := v.(type) {
	case []string:
		return append([]string(nil), raw...)
	case []any:
		out := make([]string, 0, len(raw))
		for _, e := range raw {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
