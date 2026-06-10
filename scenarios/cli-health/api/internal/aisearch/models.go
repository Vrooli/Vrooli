// Package aisearch is cli-health's thin consumer of the shared retrieval engine
// in packages/aisearch-go. This package owns only the cli-health-specific
// concerns: command discovery (discovery.go / helpparser.go), the Source adapter
// + result projection (command_index.go), and the search/reindex orchestration
// surface the Connect handlers call (service.go). The index/store/reconcile core
// — embedding, the Qdrant vector store, drift reconciliation, the sync loop, and
// env config — lives in github.com/vrooli/aisearch-go.
package aisearch

// DefaultCollection is the Qdrant collection cli-health indexes its commands
// into. Single named-dense vector layout (the shared engine's CollectionSpec).
const DefaultCollection = "cli-health-commands"

// CommandRecord is the canonical view of a single CLI command, regardless of
// source (manifest or --help fallback). It is the unit that gets embedded,
// stored, and returned by Search.
type CommandRecord struct {
	Origin      string `json:"origin"`
	Group       string `json:"group,omitempty"`
	Name        string `json:"name"`
	FullPath    string `json:"fullPath"` // "<scenario> <group> <name>" canonical command
	Description string `json:"description,omitempty"`
	// GroupDescription is the nearest enclosing group's one-line summary. It
	// carries the real-world vocabulary a user queries with (a leaf's own
	// description is often terse machine syntax) and is folded into the
	// embedding text (composeCommandEmbeddingText) on both the manifest and
	// --help discovery paths. Empty when there is no group or it merely repeats
	// the leaf's own description.
	GroupDescription string   `json:"groupDescription,omitempty"`
	Flags            []string `json:"flags,omitempty"`
	Positionals      []string `json:"positionals,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Binding          string   `json:"binding,omitempty"` // "Service.Method" when source=manifest
	Source           string   `json:"source"`            // "manifest" | "help" | "help-failed"
	// Measure carries the parsed `measure` block + proto-derived param schema
	// when this command declares one (source=manifest only). Nil otherwise. Its
	// questions/intent are folded into the embedding text (appendMeasureText) so
	// the command is retrievable by the natural-language questions it answers.
	Measure *MeasureRecord `json:"measure,omitempty"`
}

// MeasureRecord is the discovery projection of a command's `measure` block: the
// curated prose + result shape from the manifest, joined with the proto-derived
// param schema and the graded adoption tier. It is the unit measures-health (a
// later phase) harvests and the central index embeds.
type MeasureRecord struct {
	Name       string               `json:"name"`   // "<domain>.<command>"
	Domain     string               `json:"domain"` // group name, or measure.domain override
	Intent     string               `json:"intent,omitempty"`
	Questions  []string             `json:"questions,omitempty"`
	Params     []MeasureParamRecord `json:"params,omitempty"`
	ResultKind string               `json:"resultKind,omitempty"`
	ValueField string               `json:"valueField,omitempty"`
	Unit       string               `json:"unit,omitempty"`
	Effect     string               `json:"effect,omitempty"`
	Tier       string               `json:"tier,omitempty"` // full | partial | fallback; empty if unassembled
}

// MeasureParamRecord is a single measure parameter's discovery projection: the
// proto-derived type/enum/required (authoritative) overlaid with the manifest's
// default/values_source. Param types are NEVER authored in the manifest.
type MeasureParamRecord struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Required     bool     `json:"required,omitempty"`
	EnumValues   []string `json:"enumValues,omitempty"`
	Default      string   `json:"default,omitempty"`
	ValuesSource string   `json:"valuesSource,omitempty"`
}

// Read-path adoption status (WS5): SearchHit / SearchResponse / StatusReport
// here — and SearchMode in service.go — are cli-health's command-domain
// *projection* of the generic read-path in github.com/vrooli/aisearch-go
// (pkg.SearchHit / pkg.SearchResponse / pkg.StatusReport / pkg.SearchMode).
// They are kept local on purpose, not by drift:
//   - SearchHit carries command fields the generic type lacks (Origin, Group,
//     Binding, ScorePercent) — a 1:1 merge would be wrong.
//   - SearchMode is the proto-facing enum (MODE_AI/MODE_TEXT/MODE_AUTO) the
//     Connect handler maps to; pkg.SearchMode (hybrid/dense) has no "ai" member.
//
// The generic pkg.Service/SearchQuery read-path is intentionally exercised
// first by the KO docs adopter, so that contract is deferred-not-dead. See
// docs/internal/DECISIONS.md (2026-06-03). If the federation SearchHit shape
// (search-hub Appendix A.5) changes, update that descriptor in lockstep.

// SearchHit is the per-result projection returned by Service.Search.
type SearchHit struct {
	ID           string   `json:"id"`
	Origin       string   `json:"origin"`
	Group        string   `json:"group,omitempty"`
	Name         string   `json:"name"`
	FullPath     string   `json:"fullPath"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Binding      string   `json:"binding,omitempty"`
	Source       string   `json:"source"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
	// Weak is the regime-aware weak-match flag, computed once in Service.Search
	// via the shared engine's regime-aware weak labeling (pkg.LabelWeakForMethod). Carried to the
	// proto so CLI and UI render an identical "(weak)" badge — neither client
	// re-derives a threshold (the duplicated 0.55 constants are deleted).
	Weak bool `json:"weak"`
}

// SearchResponse wraps results with the request echo + retrieval method.
type SearchResponse struct {
	Results  []SearchHit `json:"results"`
	Total    int         `json:"total"`
	Query    string      `json:"query"`
	Method   string      `json:"method"`             // "ai" | "text"
	Reranker string      `json:"reranker,omitempty"` // active reranker leg or "none"
}

// StatusReport describes search backend availability.
type StatusReport struct {
	Available            bool   `json:"available"`
	Ollama               bool   `json:"ollama"`
	Qdrant               bool   `json:"qdrant"`
	IndexedCount         int    `json:"indexedCount"`
	LastReconcileAt      string `json:"lastReconcileAt,omitempty"`
	LastReconcileOutcome string `json:"lastReconcileOutcome,omitempty"`
	// Reranker is the active reranker leg ("cross-encoder:…" / "llm:…"), "none"
	// when disabled, or "degraded" when enabled but no leg is reachable.
	Reranker string `json:"reranker,omitempty"`
}
