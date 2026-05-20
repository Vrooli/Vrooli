package aisearch

import (
	"context"
	"log"
	"strings"
)

// composeCommandEmbeddingText builds the input passed to the embedder. Short
// and dense: identity first, then description, then flags/tags.
func composeCommandEmbeddingText(r CommandRecord) string {
	var parts []string
	parts = append(parts, r.FullPath)
	if r.Description != "" {
		parts = append(parts, r.Description)
	}
	if len(r.Flags) > 0 {
		parts = append(parts, "Flags: "+strings.Join(r.Flags, ", "))
	}
	if len(r.Positionals) > 0 {
		parts = append(parts, "Args: "+strings.Join(r.Positionals, ", "))
	}
	if len(r.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(r.Tags, ", "))
	}
	if r.Binding != "" {
		parts = append(parts, "Binding: "+r.Binding)
	}
	parts = append(parts, "Source: "+r.Source)
	return strings.Join(parts, "\n\n")
}

// buildCommandPayload returns the Qdrant payload for one command, including
// the payload_hash that drives the reconciler's drift detection.
func buildCommandPayload(r CommandRecord, embeddingText string) map[string]interface{} {
	p := map[string]interface{}{
		"origin":      r.Origin,
		"group":       r.Group,
		"name":        r.Name,
		"full_path":   r.FullPath,
		"description": r.Description,
		"flags":       r.Flags,
		"positionals": r.Positionals,
		"tags":        r.Tags,
		"binding":     r.Binding,
		"source":      r.Source,
	}
	p[payloadHashKey] = composePayloadHash(embeddingText, p)
	return p
}

// NewCommandDescriptor wires a CollectionDescriptor for the commands
// collection. LoadAll iterates every scenario via the discovery source.
func NewCommandDescriptor(store VectorStore, src DiscoverySource) CollectionDescriptor {
	return CollectionDescriptor{
		Kind:  KindCommand,
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
					log.Printf("[cli-health/aisearch] discover %s: %v", scenario, err)
					continue
				}
				for i := range records {
					r := records[i]
					out = append(out, &r)
				}
			}
			for _, cli := range src.ListExternalCLIs() {
				records, err := src.DiscoverExternal(ctx, cli)
				if err != nil {
					log.Printf("[cli-health/aisearch] discover external %s: %v", cli.Name, err)
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
			return composeCommandEmbeddingText(*snap.(*CommandRecord))
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			return buildCommandPayload(*snap.(*CommandRecord), text)
		},
		PointID: func(snap ItemSnapshot) string {
			return PointIDForCommand(snap.(*CommandRecord).FullPath)
		},
		DisplayName: func(snap ItemSnapshot) string {
			return snap.(*CommandRecord).FullPath
		},
	}
}

// payloadToHit projects a vector-store payload back into a SearchHit. Returns
// an empty hit when the payload is missing required fields — never panics.
func payloadToHit(id string, score float64, payload map[string]interface{}) SearchHit {
	hit := SearchHit{ID: id, Score: score, ScorePercent: int(score*100 + 0.5)}
	hit.Origin, _ = payload["origin"].(string)
	hit.Group, _ = payload["group"].(string)
	hit.Name, _ = payload["name"].(string)
	hit.FullPath, _ = payload["full_path"].(string)
	hit.Description, _ = payload["description"].(string)
	hit.Binding, _ = payload["binding"].(string)
	hit.Source, _ = payload["source"].(string)
	if raw, ok := payload["tags"].([]interface{}); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				hit.Tags = append(hit.Tags, s)
			}
		}
	}
	return hit
}
