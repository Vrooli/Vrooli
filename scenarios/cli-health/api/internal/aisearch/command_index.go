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

// idPrefix namespaces cli-health's point IDs inside the shared engine so its
// natural keys never collide with another consumer's in a shared collection.
// Kept byte-identical to the legacy "cli-health:" prefix so existing point IDs
// are recognized after the migration (single-chunk sources keep their
// un-suffixed UUIDv5 — see pkg.PointIDFor).
const idPrefix = "cli-health:"

// commandKind is the logical collection the command records belong to.
const commandKind = "command"

// composeCommandEmbeddingText builds the input passed to the embedder. Short
// and dense: identity first, then description, then flags/tags.
func composeCommandEmbeddingText(r CommandRecord) string {
	var parts []string
	parts = append(parts, r.FullPath)
	if r.Description != "" {
		parts = append(parts, r.Description)
	}
	// Parent-group prose carries the real-world vocabulary a user queries with
	// (a leaf's own description is often terse machine syntax). Folding it in
	// uniformly aligns command identity with natural query language — a generic
	// retrieval improvement, not a per-command tweak.
	if r.GroupDescription != "" {
		parts = append(parts, r.GroupDescription)
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
	parts = appendMeasureText(parts, r)
	parts = append(parts, "Source: "+r.Source)
	return strings.Join(parts, "\n\n")
}

// appendMeasureText folds a command's measure intent + the natural-language
// questions it answers into the embedding text, so a measure command is
// retrievable by the analytical question a user actually asks ("how many backlog
// items closed this week") rather than only by its machine identity. No-op for
// commands without a measure block.
func appendMeasureText(parts []string, r CommandRecord) []string {
	if r.Measure == nil {
		return parts
	}
	if r.Measure.Intent != "" {
		parts = append(parts, "Measures: "+r.Measure.Intent)
	}
	if len(r.Measure.Questions) > 0 {
		parts = append(parts, "Answers: "+strings.Join(r.Measure.Questions, "; "))
	}
	return parts
}

// commandMeta returns the per-command payload fields propagated into the chunk
// payload by the shared engine (it appends body / source_id / payload_hash).
// payloadToHit projects these keys back into a SearchHit.
func commandMeta(r CommandRecord) map[string]any {
	return map[string]any{
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
}

// commandContentHash is the source-level drift gate: a stable hash of the whole
// record so a warm reconcile tick skips an unchanged command before chunking or
// embedding (§4.1). Editing any field changes the hash.
func commandContentHash(r CommandRecord) string {
	b, _ := json.Marshal(r)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// commandToSourceDoc adapts one CommandRecord to the engine's SourceDoc. Body is
// the pre-composed embedding text (the identity composer embeds it verbatim);
// the command fields ride along as Meta for filtering + result projection. The
// compose function is injectable so the embedding-text strategy stays a
// measurable seam (production uses composeCommandEmbeddingText; an enriched
// variant was measured to HURT recall 0.70→0.40 and was removed — see
// packages/ai-go/search/docs/graduation-retrospective.md).
func commandToSourceDoc(r CommandRecord, compose func(CommandRecord) string) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:          r.FullPath,
		Kind:        commandKind,
		ContentHash: commandContentHash(r),
		Body:        compose(r),
		Meta:        commandMeta(r),
	}
}

// commandSource adapts the cli-health discovery source to the engine's Source
// interface: one SourceDoc per command record across every scenario CLI and
// each configured external CLI.
type commandSource struct {
	discovery DiscoverySource
	compose   func(CommandRecord) string
}

func newCommandSource(d DiscoverySource, compose func(CommandRecord) string) *commandSource {
	if compose == nil {
		compose = composeCommandEmbeddingText
	}
	return &commandSource{discovery: d, compose: compose}
}

// LoadAll enumerates every command record as a SourceDoc. A discovery failure
// for one scenario/CLI is logged and skipped — indexing never crashes.
func (s *commandSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	scenarios, err := s.discovery.ListScenarios(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, 256)
	for _, scenario := range scenarios {
		records, err := s.discovery.Discover(ctx, scenario)
		if err != nil {
			log.Printf("[cli-health/aisearch] discover %s: %v", scenario, err)
			continue
		}
		for i := range records {
			// WS3: help-failed stubs stay in discovery (so a CLI-less scenario is
			// still listable by name) but are kept out of the vector index — they
			// are not real commands and otherwise surface as semantic hits.
			if records[i].Source == SourceHelpFailed {
				continue
			}
			out = append(out, commandToSourceDoc(records[i], s.compose))
		}
	}
	for _, cli := range s.discovery.ListExternalCLIs() {
		records, err := s.discovery.DiscoverExternal(ctx, cli)
		if err != nil {
			log.Printf("[cli-health/aisearch] discover external %s: %v", cli.Name, err)
			continue
		}
		for i := range records {
			if records[i].Source == SourceHelpFailed { // WS3: see above.
				continue
			}
			out = append(out, commandToSourceDoc(records[i], s.compose))
		}
	}
	return out, nil
}

// pointIDForCommand returns the deterministic Qdrant point ID for a command.
// Commands are single-chunk sources, so this is the un-suffixed UUIDv5 the
// legacy collection already used.
func pointIDForCommand(fullPath string) string {
	return pkg.PointIDFor(idPrefix, strings.TrimSpace(fullPath), 0, 1)
}

// payloadToHit projects a vector-store payload back into a SearchHit. Returns
// an empty hit when the payload is missing required fields — never panics.
func payloadToHit(id string, score float64, payload map[string]any) SearchHit {
	hit := SearchHit{ID: id, Score: score, ScorePercent: int(score*100 + 0.5)}
	hit.Origin, _ = payload["origin"].(string)
	hit.Group, _ = payload["group"].(string)
	hit.Name, _ = payload["name"].(string)
	hit.FullPath, _ = payload["full_path"].(string)
	hit.Description, _ = payload["description"].(string)
	hit.Binding, _ = payload["binding"].(string)
	hit.Source, _ = payload["source"].(string)
	hit.Tags = stringSliceFromPayload(payload["tags"])
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
