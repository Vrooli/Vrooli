package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

// idPrefix namespaces ui-health's point IDs inside the shared engine so its
// natural keys never collide with another consumer's. (scenario, file_path) is
// the surface identity; surfaces are single-chunk dense sources.
const idPrefix = "ui-health:"

// surfaceKind is the logical collection the surface records belong to.
const surfaceKind = "surface"

// composeSurfaceEmbeddingText builds the input passed to the embedder. Short
// and dense: identity first, then description, then provenance/widget hints.
func composeSurfaceEmbeddingText(r SurfaceRecord) string {
	var parts []string
	parts = append(parts, r.Scenario+" "+r.Slot+" "+r.DisplayName)
	if r.Description != "" {
		parts = append(parts, r.Description)
	}
	parts = append(parts, "Kind: "+r.Kind)
	if r.FilePath != "" {
		parts = append(parts, "File: "+r.FilePath)
	}
	if r.Provenance != nil && r.Provenance.Library != "" {
		parts = append(parts, "Library: "+r.Provenance.Library)
	}
	if r.Widget != nil && r.Widget.WidgetID != "" {
		parts = append(parts, "Widget: "+r.Widget.WidgetID)
	}
	return strings.Join(parts, "\n\n")
}

// surfaceMeta returns the per-surface payload fields propagated into the chunk
// payload by the shared engine (it appends body / source_id / payload_hash).
// Provenance/Widget ride along as nested maps; payloadToHit projects them back.
func surfaceMeta(r SurfaceRecord) map[string]any {
	p := map[string]any{
		"scenario":     r.Scenario,
		"slot":         r.Slot,
		"kind":         r.Kind,
		"display_name": r.DisplayName,
		"description":  r.Description,
		"file_path":    r.FilePath,
	}
	if r.Provenance != nil {
		p["provenance"] = map[string]any{
			"provenance":      r.Provenance.Provenance,
			"library":         r.Provenance.Library,
			"library_version": r.Provenance.LibraryVersion,
			"component_name":  r.Provenance.ComponentName,
			"adoption_id":     r.Provenance.AdoptionID,
			"applied_at":      r.Provenance.AppliedAt,
			"source_sha256":   r.Provenance.SourceSha256,
			"drift_hash":      r.Provenance.DriftHash,
			"file_path":       r.Provenance.FilePath,
		}
	}
	if r.Widget != nil {
		p["widget"] = map[string]any{
			"widget_id":         r.Widget.WidgetID,
			"component_name":    r.Widget.ComponentName,
			"props_schema_json": r.Widget.PropsSchemaJSON,
			"slot":              r.Widget.Slot,
			"scope":             r.Widget.Scope,
			"description":       r.Widget.Description,
			"file_path":         r.Widget.FilePath,
		}
	}
	return p
}

// surfaceContentHash is the source-level drift gate: a stable hash of the whole
// record so a warm reconcile tick skips an unchanged surface before chunking or
// embedding. Editing any field changes the hash.
func surfaceContentHash(r SurfaceRecord) string {
	b, _ := json.Marshal(r)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// surfaceToSourceDoc adapts one SurfaceRecord to the engine's SourceDoc. Body is
// the pre-composed embedding text (the identity composer embeds it verbatim);
// the surface fields ride along as Meta for result projection.
func surfaceToSourceDoc(r SurfaceRecord) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:          r.Scenario + ":" + r.FilePath,
		Kind:        surfaceKind,
		ContentHash: surfaceContentHash(r),
		Body:        composeSurfaceEmbeddingText(r),
		Meta:        surfaceMeta(r),
	}
}

// surfaceSource adapts ui-health's discovery source to the engine's Source
// interface: one SourceDoc per surface record across every scenario.
type surfaceSource struct {
	discovery DiscoverySource
}

func newSurfaceSource(d DiscoverySource) *surfaceSource {
	return &surfaceSource{discovery: d}
}

// LoadAll enumerates every surface record as a SourceDoc. A discovery failure
// for one scenario is logged and skipped — indexing never crashes.
func (s *surfaceSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	scenarios, err := s.discovery.ListScenarios(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, 128)
	for _, scenario := range scenarios {
		records, err := s.discovery.Discover(ctx, scenario)
		if err != nil {
			log.Printf("[ui-health/aisearch] discover %s: %v", scenario, err)
			continue
		}
		for i := range records {
			out = append(out, surfaceToSourceDoc(records[i]))
		}
	}
	return out, nil
}

// PointIDForSurface returns the deterministic Qdrant point ID for a surface.
// Surfaces are single-chunk sources, so this is the un-suffixed UUIDv5 keyed on
// (scenario, file_path).
func PointIDForSurface(scenario, filePath string) string {
	return pkg.PointIDFor(idPrefix, strings.TrimSpace(scenario)+":"+strings.TrimSpace(filePath), 0, 1)
}

// payloadToHit projects a vector-store payload back into a SearchHit. Returns an
// empty hit when the payload is missing required fields — never panics.
func payloadToHit(id string, score float64, payload map[string]any) SearchHit {
	hit := SearchHit{ID: id, Score: score, ScorePercent: int(score*100 + 0.5)}
	hit.Scenario, _ = payload["scenario"].(string)
	hit.Slot, _ = payload["slot"].(string)
	hit.Kind, _ = payload["kind"].(string)
	hit.DisplayName, _ = payload["display_name"].(string)
	hit.Description, _ = payload["description"].(string)
	hit.FilePath, _ = payload["file_path"].(string)
	if raw, ok := payload["provenance"].(map[string]any); ok {
		p := &ProvenancePayload{}
		p.Provenance, _ = raw["provenance"].(string)
		p.Library, _ = raw["library"].(string)
		p.LibraryVersion, _ = raw["library_version"].(string)
		p.ComponentName, _ = raw["component_name"].(string)
		p.AdoptionID, _ = raw["adoption_id"].(string)
		p.AppliedAt, _ = raw["applied_at"].(string)
		p.SourceSha256, _ = raw["source_sha256"].(string)
		p.DriftHash, _ = raw["drift_hash"].(string)
		p.FilePath, _ = raw["file_path"].(string)
		hit.Provenance = p
	}
	if raw, ok := payload["widget"].(map[string]any); ok {
		w := &WidgetPayload{}
		w.WidgetID, _ = raw["widget_id"].(string)
		w.ComponentName, _ = raw["component_name"].(string)
		w.PropsSchemaJSON, _ = raw["props_schema_json"].(string)
		w.Slot, _ = raw["slot"].(string)
		w.Scope, _ = raw["scope"].(string)
		w.Description, _ = raw["description"].(string)
		w.FilePath, _ = raw["file_path"].(string)
		hit.Widget = w
	}
	return hit
}
