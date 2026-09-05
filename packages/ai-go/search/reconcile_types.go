package aisearch

import "time"

// ItemRef is the reconciler's handle on one chunk that must be embedded and
// upserted. It snapshots the Chunk so Apply can re-compose the embedding text
// and rebuild the payload deterministically.
type ItemRef struct {
	Kind        string
	PointID     string
	SourceID    string
	Index       int
	Total       int
	Name        string
	PayloadHash string
	SourceHash  string
	Chunk       Chunk
}

// RefreshRef is a chunk whose body is unchanged but whose parent source changed
// elsewhere: only its source_hash needs refreshing (no re-embed).
type RefreshRef struct {
	Kind       string
	PointID    string
	SourceHash string
	Payload    map[string]any
}

// CollectionDriftReport is the Plan output for one binding.
type CollectionDriftReport struct {
	Kind             string       `json:"kind"`
	ToUpsert         []ItemRef    `json:"toUpsert,omitempty"`
	ToRefresh        []RefreshRef `json:"toRefresh,omitempty"`
	ToDelete         []string     `json:"toDelete,omitempty"`
	UnchangedSources int          `json:"unchangedSources"`
	UnchangedChunks  int          `json:"unchangedChunks"`
	LegacyCount      int          `json:"legacyCount"`
}

// DriftReport is the full Plan output across all bindings.
type DriftReport struct {
	PlannedAt   time.Time               `json:"plannedAt"`
	Collections []CollectionDriftReport `json:"collections"`
}

// HasWork reports whether any binding has upserts, refreshes, or deletes.
func (d *DriftReport) HasWork() bool {
	if d == nil {
		return false
	}
	for _, c := range d.Collections {
		if len(c.ToUpsert) > 0 || len(c.ToRefresh) > 0 || len(c.ToDelete) > 0 {
			return true
		}
	}
	return false
}

// CollectionApplyResult is the Apply output for one binding.
type CollectionApplyResult struct {
	Kind      string `json:"kind"`
	Upserted  int    `json:"upserted"`
	Refreshed int    `json:"refreshed"`
	Deleted   int    `json:"deleted"`
}

// ReconcileError is one failure recorded during Apply.
type ReconcileError struct {
	Kind    string `json:"kind"`
	PointID string `json:"pointId,omitempty"`
	Name    string `json:"name,omitempty"`
	Op      string `json:"op"`
	Err     string `json:"err"`
}

// ApplyResult captures what Apply did. Deferred counts upserts skipped because
// the per-tick embed budget was exhausted — they remain as drift for the next
// tick (resumable, §4.2).
type ApplyResult struct {
	StartedAt   time.Time               `json:"startedAt"`
	FinishedAt  time.Time               `json:"finishedAt"`
	Collections []CollectionApplyResult `json:"collections"`
	Errors      []ReconcileError        `json:"errors,omitempty"`
	Deferred    int                     `json:"deferred,omitempty"`
}

// ReconcileStatus is the reconciler state surfaced to handlers/CLI.
type ReconcileStatus struct {
	Running    bool         `json:"running"`
	StartedAt  string       `json:"startedAt,omitempty"`
	FinishedAt string       `json:"finishedAt,omitempty"`
	LastPlan   *DriftReport `json:"lastPlan,omitempty"`
	LastResult *ApplyResult `json:"lastResult,omitempty"`
	LastError  string       `json:"lastError,omitempty"`
	Canceled   bool         `json:"canceled,omitempty"`
}
