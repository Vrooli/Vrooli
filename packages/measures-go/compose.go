package measures

import (
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
)

// Meta keys the central measures index sets on each SourceDoc/Chunk so the
// MeasureComposer can build the embedding text from the curated questions +
// intent while the retrievable payload (Chunk.Body) stays the measure
// declaration. Exported so the Phase 4 index Source and the composer agree on
// one contract.
const (
	// MetaQuestions carries the measure's natural-language phrasings — the
	// embedding key. Value is []string (preferred) or a newline-joined string.
	MetaQuestions = "measure_questions"
	// MetaIntent carries the one-line measure intent used as grounding.
	MetaIntent = "measure_intent"
	// MetaMeasureName carries the measure declaration name (provenance/debug).
	MetaMeasureName = "measure_name"
)

// MeasureComposer is the custom EmbeddingTextComposer for measures. It embeds
// the JOINED example questions (the way users actually ask the question),
// grounded by the measure intent, while leaving the returned payload (Chunk.Body
// — set by the index to the serialized declaration) untouched. This is the
// clean separation aisearch-go's composer seam exists for: the vector lands in
// "how a user phrases this analytical question" space, but the hit carries the
// full typed measure so the provider can extract params and execute.
//
// Compose is deterministic and side-effect free, so the same measure embeds
// identically across reindexes (no spurious drift).
type MeasureComposer struct{}

var _ aisearch.EmbeddingTextComposer = MeasureComposer{}

// NewMeasureComposer returns the measures embedding composer.
func NewMeasureComposer() MeasureComposer { return MeasureComposer{} }

// Compose builds the embedded text as the questions (one per line) followed by
// the intent as a grounding line. When no questions are present it falls back to
// the intent, then to the chunk body — so a malformed index entry still embeds
// *something* rather than empty text.
func (MeasureComposer) Compose(chunk aisearch.Chunk) string {
	questions := metaStrings(chunk.Meta, MetaQuestions)
	intent := metaString(chunk.Meta, MetaIntent)

	var lines []string
	lines = append(lines, questions...)
	if intent != "" {
		lines = append(lines, intent)
	}
	if len(lines) == 0 {
		return chunk.Body
	}
	return strings.Join(lines, "\n")
}

// metaString reads a trimmed string value from a chunk's Meta map.
func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	s, _ := meta[key].(string)
	return strings.TrimSpace(s)
}

// metaStrings reads a string slice from Meta, accepting either a []string or a
// newline-joined string (the two shapes an index Source might serialize).
// Blank entries are dropped.
func metaStrings(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	var raw []string
	switch v := meta[key].(type) {
	case []string:
		raw = v
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				raw = append(raw, s)
			}
		}
	case string:
		raw = strings.Split(v, "\n")
	}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
