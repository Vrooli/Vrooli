// Package aisearch is business-health's thin consumer of the shared
// retrieval engine in packages/ai-go/search. It owns only the
// intent-domain concerns: the fleet-wide intent corpus source (one doc per
// PRD overview, per operational target, per requirement — the type-faceted
// single corpus of plan decision D5), the embed-text composition, the
// typed-hit projection, and the search/reindex orchestration surface the
// Connect handlers call. The index/store/reconcile core lives in
// github.com/vrooli/ai-go/search.
package aisearch

// DefaultCollection is the Qdrant collection business-health indexes the
// intent corpus into (single named-dense vector layout).
const DefaultCollection = "business-health-intent"

// Facet values for IntentRecord.Type (the D5 type facet).
const (
	TypePRDOverview       = "prd_overview"
	TypeOperationalTarget = "operational_target"
	TypeRequirement       = "requirement"
)

// IntentRecord is the canonical view of one intent unit — the unit that
// gets embedded, stored, and returned by Search. Pointer-only by design
// (cell #22): Anchor points into the owning artifact; no synthesized "why".
type IntentRecord struct {
	// ID is the stable corpus identity: "<scenario>/prd",
	// "<scenario>/<OT-ID>", or "<scenario>/<REQ-ID>".
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	// Type: prd_overview | operational_target | requirement.
	Type    string `json:"type"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	// Anchor into the artifact, e.g. "scenarios/x/PRD.md#OT-P0-004".
	Anchor string `json:"anchor"`
	// PRDRef is the linked OT id for requirement records ("" otherwise).
	PRDRef string `json:"prd_ref,omitempty"`
	// ScenarioPurpose is the owning scenario's purpose line — folded into
	// the embed text so capability queries land on the right scenario.
	ScenarioPurpose string `json:"scenario_purpose,omitempty"`
}

// SearchHit is the per-result projection returned by Service.Search.
type SearchHit struct {
	ID           string  `json:"id"`
	Scenario     string  `json:"scenario"`
	Type         string  `json:"type"`
	Title        string  `json:"title"`
	Snippet      string  `json:"snippet"`
	Anchor       string  `json:"anchor"`
	PRDRef       string  `json:"prdRef,omitempty"`
	Score        float64 `json:"score"`
	ScorePercent int     `json:"scorePercent"`
	Weak         bool    `json:"weak"`
}

// SearchResponse wraps results with the request echo + retrieval method.
type SearchResponse struct {
	Results  []SearchHit `json:"results"`
	Total    int         `json:"total"`
	Query    string      `json:"query"`
	Method   string      `json:"method"` // "ai" | "text"
	Reranker string      `json:"reranker,omitempty"`
}

// StatusReport describes search backend availability.
type StatusReport struct {
	Available            bool   `json:"available"`
	Ollama               bool   `json:"ollama"`
	Qdrant               bool   `json:"qdrant"`
	IndexedCount         int    `json:"indexedCount"`
	LastReconcileAt      string `json:"lastReconcileAt,omitempty"`
	LastReconcileOutcome string `json:"lastReconcileOutcome,omitempty"`
	Reranker             string `json:"reranker,omitempty"`
}
