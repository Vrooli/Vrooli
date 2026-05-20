package aisearch

import (
	"context"
	"log"
	"strings"
)

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

// buildSurfacePayload returns the Qdrant payload for one surface, including
// the payload_hash that drives the reconciler's drift detection.
func buildSurfacePayload(r SurfaceRecord, embeddingText string) map[string]interface{} {
	p := map[string]interface{}{
		"scenario":     r.Scenario,
		"slot":         r.Slot,
		"kind":         r.Kind,
		"display_name": r.DisplayName,
		"description":  r.Description,
		"file_path":    r.FilePath,
	}
	if r.Provenance != nil {
		p["provenance"] = map[string]interface{}{
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
		p["widget"] = map[string]interface{}{
			"widget_id":         r.Widget.WidgetID,
			"component_name":    r.Widget.ComponentName,
			"props_schema_json": r.Widget.PropsSchemaJSON,
			"slot":              r.Widget.Slot,
			"scope":             r.Widget.Scope,
			"description":       r.Widget.Description,
			"file_path":         r.Widget.FilePath,
		}
	}
	p[payloadHashKey] = composePayloadHash(embeddingText, p)
	return p
}

// NewSurfaceDescriptor wires a CollectionDescriptor for the surface
// collection. LoadAll iterates every scenario via the discovery source.
func NewSurfaceDescriptor(store VectorStore, src DiscoverySource) CollectionDescriptor {
	return CollectionDescriptor{
		Kind:  KindSurface,
		Store: store,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			scenarios, err := src.ListScenarios(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]ItemSnapshot, 0, 128)
			for _, scenario := range scenarios {
				records, err := src.Discover(ctx, scenario)
				if err != nil {
					log.Printf("[ui-health/aisearch] discover %s: %v", scenario, err)
					continue
				}
				for i := range records {
					r := records[i]
					out = append(out, &r)
				}
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string {
			return composeSurfaceEmbeddingText(*snap.(*SurfaceRecord))
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			return buildSurfacePayload(*snap.(*SurfaceRecord), text)
		},
		PointID: func(snap ItemSnapshot) string {
			r := snap.(*SurfaceRecord)
			return PointIDForSurface(r.Scenario, r.FilePath)
		},
		DisplayName: func(snap ItemSnapshot) string {
			r := snap.(*SurfaceRecord)
			return r.Scenario + ":" + r.FilePath
		},
	}
}

// payloadToHit projects a vector-store payload back into a SearchHit.
func payloadToHit(id string, score float64, payload map[string]interface{}) SearchHit {
	hit := SearchHit{ID: id, Score: score, ScorePercent: int(score*100 + 0.5)}
	hit.Scenario, _ = payload["scenario"].(string)
	hit.Slot, _ = payload["slot"].(string)
	hit.Kind, _ = payload["kind"].(string)
	hit.DisplayName, _ = payload["display_name"].(string)
	hit.Description, _ = payload["description"].(string)
	hit.FilePath, _ = payload["file_path"].(string)
	if raw, ok := payload["provenance"].(map[string]interface{}); ok {
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
	if raw, ok := payload["widget"].(map[string]interface{}); ok {
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

// PointIDForSurface returns the deterministic UUIDv5 used as the Qdrant
// point ID for a surface record. (scenario, file_path) is the identity key.
func PointIDForSurface(scenario, filePath string) string {
	id := strings.TrimSpace(scenario) + ":" + strings.TrimSpace(filePath)
	if id == ":" {
		id = "unknown"
	}
	return uuidV5(qdrantNamespace, "ui-health:"+id)
}
