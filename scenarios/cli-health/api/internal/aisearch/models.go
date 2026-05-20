// Package aisearch is the AI-powered semantic search for cli-health.
// Structure mirrors scenarios/prompt-manager/api/aisearch verbatim where
// reusable; cli-health-specific types (CommandRecord, SearchHit) live here.
//
// Duplicate-before-extract: this package is a copy of the prompt-manager
// implementation. Bug fixes flow upstream first, then are mirrored here.
// Extraction into a shared package is deferred to a future scenario.
package aisearch

import "time"

// CommandRecord is the canonical view of a single CLI command, regardless of
// source (manifest or --help fallback). It is the unit that gets embedded,
// stored, and returned by Search.
type CommandRecord struct {
	Scenario    string   `json:"scenario"`
	Group       string   `json:"group,omitempty"`
	Name        string   `json:"name"`
	FullPath    string   `json:"fullPath"` // "<scenario> <group> <name>" canonical command
	Description string   `json:"description,omitempty"`
	Flags       []string `json:"flags,omitempty"`
	Positionals []string `json:"positionals,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Binding     string   `json:"binding,omitempty"` // "Service.Method" when source=manifest
	Source      string   `json:"source"`            // "manifest" | "help" | "help-failed"
}

// SearchHit is the per-result projection returned by Service.Search.
type SearchHit struct {
	ID           string   `json:"id"`
	Scenario     string   `json:"scenario"`
	Group        string   `json:"group,omitempty"`
	Name         string   `json:"name"`
	FullPath     string   `json:"fullPath"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Binding      string   `json:"binding,omitempty"`
	Source       string   `json:"source"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
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

// EntityKind names the collection a reconciler item belongs to. cli-health
// has one kind today; the type stays so the reconciler stays generic.
type EntityKind string

const KindCommand EntityKind = "command"

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

// HasWork returns true when any collection has upserts or deletes pending.
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

// CollectionApplyResult captures Apply output for one collection.
type CollectionApplyResult struct {
	Kind     EntityKind `json:"kind"`
	Upserted int        `json:"upserted"`
	Deleted  int        `json:"deleted"`
}

// ReconcileError is one entry in ApplyResult.Errors.
type ReconcileError struct {
	Kind    EntityKind `json:"kind"`
	PointID string     `json:"pointId,omitempty"`
	Name    string     `json:"name,omitempty"`
	Op      string     `json:"op"`
	Err     string     `json:"err"`
}

// ApplyResult captures the actions Apply took.
type ApplyResult struct {
	StartedAt   time.Time               `json:"startedAt"`
	FinishedAt  time.Time               `json:"finishedAt"`
	Collections []CollectionApplyResult `json:"collections"`
	Errors      []ReconcileError        `json:"errors,omitempty"`
}

// ReconcileStatus is the Reconciler state surfaced to handlers/CLI.
type ReconcileStatus struct {
	Running    bool         `json:"running"`
	StartedAt  string       `json:"startedAt,omitempty"`
	FinishedAt string       `json:"finishedAt,omitempty"`
	LastPlan   *DriftReport `json:"lastPlan,omitempty"`
	LastResult *ApplyResult `json:"lastResult,omitempty"`
	LastError  string       `json:"lastError,omitempty"`
	Canceled   bool         `json:"canceled,omitempty"`
}
