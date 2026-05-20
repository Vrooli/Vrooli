// Package aisearch is the AI-powered semantic search for ui-health. It
// indexes every scenario's UI surface (components, pages, features, hooks)
// into a local Qdrant collection (ui-health-surface) and serves AI-first
// semantic queries with deterministic text fallback.
//
// Structure mirrors cli-health/aisearch verbatim where reusable; ui-health
// substitutes its own DiscoverySource (which dispatches to InventoryService
// on per-framework component-library scenarios) for cli-health's CLI help
// parser.
//
// Duplicate-before-extract: this package shares shape with cli-health on
// purpose. Bug fixes flow there first, then are mirrored here.
package aisearch

import "time"

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

// EntityKind names the collection a reconciler item belongs to.
type EntityKind string

const KindSurface EntityKind = "surface"

// ItemRef is the reconciler's handle on a single planned item.
type ItemRef struct {
	Kind        EntityKind  `json:"kind"`
	PointID     string      `json:"pointId"`
	Name        string      `json:"name"`
	PayloadHash string      `json:"payloadHash"`
	Snapshot    interface{} `json:"-"`
}

// CollectionDriftReport captures Plan output for one collection.
type CollectionDriftReport struct {
	Kind           EntityKind `json:"kind"`
	ToUpsert       []ItemRef  `json:"toUpsert,omitempty"`
	ToDelete       []string   `json:"toDelete,omitempty"`
	UnchangedCount int        `json:"unchangedCount"`
	LegacyCount    int        `json:"legacyCount"`
}

// DriftReport is the full Plan output across all configured collections.
type DriftReport struct {
	PlannedAt   time.Time               `json:"plannedAt"`
	Collections []CollectionDriftReport `json:"collections"`
}

func (d *DriftReport) HasWork() bool {
	if d == nil {
		return false
	}
	for _, c := range d.Collections {
		if len(c.ToUpsert) > 0 || len(c.ToDelete) > 0 {
			return true
		}
	}
	return false
}

type CollectionApplyResult struct {
	Kind     EntityKind `json:"kind"`
	Upserted int        `json:"upserted"`
	Deleted  int        `json:"deleted"`
}

type ReconcileError struct {
	Kind    EntityKind `json:"kind"`
	PointID string     `json:"pointId,omitempty"`
	Name    string     `json:"name,omitempty"`
	Op      string     `json:"op"`
	Err     string     `json:"err"`
}

type ApplyResult struct {
	StartedAt   time.Time               `json:"startedAt"`
	FinishedAt  time.Time               `json:"finishedAt"`
	Collections []CollectionApplyResult `json:"collections"`
	Errors      []ReconcileError        `json:"errors,omitempty"`
}

type ReconcileStatus struct {
	Running    bool         `json:"running"`
	StartedAt  string       `json:"startedAt,omitempty"`
	FinishedAt string       `json:"finishedAt,omitempty"`
	LastPlan   *DriftReport `json:"lastPlan,omitempty"`
	LastResult *ApplyResult `json:"lastResult,omitempty"`
	LastError  string       `json:"lastError,omitempty"`
	Canceled   bool         `json:"canceled,omitempty"`
}
