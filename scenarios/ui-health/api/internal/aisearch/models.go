// Package aisearch is ui-health's thin consumer of the shared retrieval engine
// in packages/ai-go/search. It owns only the ui-health-specific concerns: surface
// discovery (discovery.go — dispatching to InventoryService on per-framework
// component-library scenarios), the source adapter + result projection
// (surface_index.go), and the search/reindex orchestration surface the Connect
// handlers call (service.go). The index/store/reconcile core — embedding, the
// Qdrant vector store, drift reconciliation, the sync loop, and env config —
// lives in github.com/vrooli/ai-go/search, driven by the scenario-owned
// .vrooli/search.json (the SSOT). Mirrors the cli-health adopter shape.
package aisearch

// SurfaceRecord is the canonical view of one UI surface (component, page,
// feature, hook) returned by InventoryService.ScanScenario. It is the unit
// that gets embedded, stored, and returned by Search.
type SurfaceRecord struct {
	Scenario    string             `json:"scenario"`
	Slot        string             `json:"slot,omitempty"`
	Kind        string             `json:"kind"`
	DisplayName string             `json:"displayName"`
	Description string             `json:"description,omitempty"`
	FilePath    string             `json:"filePath"`
	Provenance  *ProvenancePayload `json:"provenance,omitempty"`
	Widget      *WidgetPayload     `json:"widget,omitempty"`
}

// ProvenancePayload is the index-time projection of ComponentProvenance.
type ProvenancePayload struct {
	Provenance     string `json:"provenance"`
	Library        string `json:"library,omitempty"`
	LibraryVersion string `json:"libraryVersion,omitempty"`
	ComponentName  string `json:"componentName,omitempty"`
	AdoptionID     string `json:"adoptionId,omitempty"`
	AppliedAt      string `json:"appliedAt,omitempty"`
	SourceSha256   string `json:"sourceSha256,omitempty"`
	DriftHash      string `json:"driftHash,omitempty"`
	FilePath       string `json:"filePath,omitempty"`
}

// WidgetPayload is the index-time projection of WidgetDeclaration.
type WidgetPayload struct {
	WidgetID        string `json:"widgetId"`
	ComponentName   string `json:"componentName"`
	PropsSchemaJSON string `json:"propsSchemaJson,omitempty"`
	Slot            string `json:"slot,omitempty"`
	Scope           string `json:"scope,omitempty"`
	Description     string `json:"description,omitempty"`
	FilePath        string `json:"filePath,omitempty"`
}

// SearchHit is the per-result projection returned by Service.Search.
type SearchHit struct {
	ID           string             `json:"id"`
	Scenario     string             `json:"scenario"`
	Slot         string             `json:"slot,omitempty"`
	Kind         string             `json:"kind"`
	DisplayName  string             `json:"displayName"`
	Description  string             `json:"description,omitempty"`
	FilePath     string             `json:"filePath"`
	Provenance   *ProvenancePayload `json:"provenance,omitempty"`
	Widget       *WidgetPayload     `json:"widget,omitempty"`
	Score        float64            `json:"score"`
	ScorePercent int                `json:"scorePercent"`
}

// SearchResponse wraps results with the request echo + retrieval method.
type SearchResponse struct {
	Results []SearchHit `json:"results"`
	Total   int         `json:"total"`
	Query   string      `json:"query"`
	Method  string      `json:"method"` // "ai" | "text"
}

// StatusReport describes search backend availability.
type StatusReport struct {
	Available            bool   `json:"available"`
	Ollama               bool   `json:"ollama"`
	Qdrant               bool   `json:"qdrant"`
	IndexedCount         int    `json:"indexedCount"`
	LastReconcileAt      string `json:"lastReconcileAt,omitempty"`
	LastReconcileOutcome string `json:"lastReconcileOutcome,omitempty"`
}
