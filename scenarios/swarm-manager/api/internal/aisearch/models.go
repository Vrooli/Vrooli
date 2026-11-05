// Package aisearch provides embedding-based semantic search over swarm-manager
// backlog items and goals, with graceful fallback to text search when
// Ollama or Qdrant is unavailable.
package aisearch

import (
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

// EntityType identifies which collection a search result belongs to.
type EntityType string

const (
	EntityBacklog EntityType = "backlog"
	EntityGoal    EntityType = "goal"
	EntityRecord  EntityType = "record"
	EntityBoth    EntityType = "both"
)

// Valid reports whether e is one of the known entity values accepted by the
// API surface.
func (e EntityType) Valid() bool {
	switch e {
	case EntityBacklog, EntityGoal, EntityRecord, EntityBoth:
		return true
	default:
		return false
	}
}

// SearchFilters constrains results to a subset of the indexed entities. All
// fields are optional; zero-valued filters match everything.
type SearchFilters struct {
	Status          []string `json:"status,omitempty"`
	Kind            []string `json:"kind,omitempty"`
	Goal            string   `json:"goal,omitempty"`
	TargetScenario  string   `json:"target_scenario,omitempty"`
	IncludeArchived bool     `json:"include_archived,omitempty"`
}

// AISearchRequest is the input shape for a semantic search.
type AISearchRequest struct {
	Query     string        `json:"query"`
	Entity    EntityType    `json:"entity,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Threshold float64       `json:"threshold,omitempty"`
	Filters   SearchFilters `json:"filters,omitempty"`
}

// AISearchResult is a single ranked result. Payload preserves the raw Qdrant
// payload for the client; typed helpers can unwrap it when the entity is known.
// can unwrap it when the entity is known.
type AISearchResult struct {
	Entity       EntityType             `json:"entity"`
	ID           string                 `json:"id"`
	Score        float64                `json:"score"`
	ScorePercent int                    `json:"scorePercent"`
	Payload      map[string]interface{} `json:"payload"`
}

// FallbackMethod describes whether and how a request fell back when embeddings
// were unavailable.
type FallbackMethod string

const (
	FallbackNone        FallbackMethod = "none"
	FallbackTextSearch  FallbackMethod = "text-search"
	FallbackUnavailable FallbackMethod = "unavailable"
)

// AISearchResponse wraps ranked results with metadata describing how the
// search was fulfilled.
type AISearchResponse struct {
	Results   []AISearchResult `json:"results"`
	Total     int              `json:"total"`
	Query     string           `json:"query"`
	Entity    EntityType       `json:"entity"`
	Fallback  FallbackMethod   `json:"fallback"`
	LatencyMs int64            `json:"latencyMs"`
}

// SimilarTarget identifies an indexed entity whose composed text is used as a
// semantic query. It intentionally does not expose Qdrant point retrieval:
// re-embedding keeps the VectorStore seam unchanged.
type SimilarTarget struct {
	Entity      EntityType
	BacklogKind backlog.BacklogKind
	Name        string
}

// SimilarResponse is deliberately separate from Search so callers can tell a
// semantic outage from an ordinary zero-result search. Similar never uses the
// text-search fallback because that would make its scores misleading.
type SimilarResponse struct {
	Results  []AISearchResult
	Degraded bool
}

// AvailabilityStatus reports whether AI search is currently usable. Exposed by
// GET /api/v1/search/ai/status.
type AvailabilityStatus struct {
	Available      bool   `json:"available"`
	Ollama         bool   `json:"ollama"`
	Qdrant         bool   `json:"qdrant"`
	IndexedBacklog int    `json:"indexedBacklog"`
	IndexedGoals   int    `json:"indexedGoals"`
	OnDiskBacklog  int    `json:"onDiskBacklog"`
	OnDiskGoals    int    `json:"onDiskGoals"`
	Message        string `json:"message,omitempty"`
}

// EntityKind discriminates which collection an ItemRef or ReconcileError
// belongs to. Distinct from EntityType (which has a Both value used by search
// requests) — the reconciler never operates on "both" as a single bucket.
type EntityKind string

const (
	KindBacklogItem EntityKind = "backlog"
	KindGoal        EntityKind = "goal"
)

// ItemRef is the minimal handle the Reconciler needs to embed-and-upsert one
// item without re-walking disk during Apply. Holding the full struct inline
// means Apply doesn't need a second LoadAll round-trip — at current scale
// (~326 items, ~1KB each) the memory cost is negligible.
//
// Exactly one of Backlog/Goal is set per Kind. PayloadHash is the freshly
// computed hash, ready to be stored under "payload_hash" in the Qdrant payload.
type ItemRef struct {
	Kind        EntityKind
	PointID     string
	Name        string
	PayloadHash string
	Backlog     *backlog.BacklogItem // set when Kind == KindBacklogItem
	Goal        *goals.Goal          // set when Kind == KindGoal
}

// CaptureDocument is the searchable projection of an ephemeral capture. It
// intentionally contains no lifecycle or graph semantics; captures are
// indexed for deduplication and grounding, not promoted to work entities.
type CaptureDocument struct {
	ID          string
	Text        string
	Note        string
	Attachments []string
}

// DriftReport is the structured "what work needs doing?" decision returned by
// Reconciler.Plan. Apply consumes this verbatim — no second comparison pass.
//
// Per-collection breakdowns let the dry-run output show "12 backlog upserts,
// 3 goal orphans" rather than a single opaque total. LegacyBacklog and
// LegacyGoal track the post-deploy "absent payload_hash → re-embed once"
// drain so operators can see when convergence is complete.
type DriftReport struct {
	PlannedAt        time.Time `json:"plannedAt"`
	ToUpsertBacklog  []ItemRef `json:"-"` // items, kept out of JSON; counts below
	ToUpsertGoal     []ItemRef `json:"-"`
	ToDeleteBacklog  []string  `json:"toDeleteBacklog,omitempty"`
	ToDeleteGoal     []string  `json:"toDeleteGoal,omitempty"`
	UnchangedBacklog int       `json:"unchangedBacklog"`
	UnchangedGoal    int       `json:"unchangedGoal"`
	LegacyBacklog    int       `json:"legacyBacklog"`
	LegacyGoal       int       `json:"legacyGoal"`
}

// HasWork reports whether Apply would do anything for this report. The
// periodic SyncLoop uses it to skip the apply phase entirely when the index
// has already converged — the optimization that turns 5-minute ticks into
// "one disk walk + one qdrant scroll + zero embeds" when nothing changed.
func (d DriftReport) HasWork() bool {
	return len(d.ToUpsertBacklog)+len(d.ToUpsertGoal)+
		len(d.ToDeleteBacklog)+len(d.ToDeleteGoal) > 0
}

// UpsertCount returns total items needing upsert across both collections.
// Convenience for status/log lines.
func (d DriftReport) UpsertCount() int {
	return len(d.ToUpsertBacklog) + len(d.ToUpsertGoal)
}

// DeleteCount returns total point IDs needing delete across both collections.
func (d DriftReport) DeleteCount() int {
	return len(d.ToDeleteBacklog) + len(d.ToDeleteGoal)
}

// ReconcileError captures one per-item failure during Apply so operators can
// triage exactly which item failed and where in the pipeline. The reconciler
// is best-effort: a single item's failure does not abort the rest of the pass.
type ReconcileError struct {
	Kind    EntityKind `json:"kind"`
	PointID string     `json:"pointId,omitempty"`
	Name    string     `json:"name,omitempty"`
	Op      string     `json:"op"` // "embed" | "upsert" | "delete"
	Err     string     `json:"err"`
}

// ApplyResult is the structured outcome of Reconciler.Apply. Counts reflect
// items that completed successfully; failed items appear in Errors with their
// failing op.
type ApplyResult struct {
	StartedAt       time.Time        `json:"startedAt"`
	FinishedAt      time.Time        `json:"finishedAt"`
	UpsertedBacklog int              `json:"upsertedBacklog"`
	UpsertedGoal    int              `json:"upsertedGoal"`
	DeletedBacklog  int              `json:"deletedBacklog"`
	DeletedGoal     int              `json:"deletedGoal"`
	Errors          []ReconcileError `json:"errors,omitempty"`
}

// ReconcileStatus is the wire-shape returned by GET /api/v1/search/ai/reconcile/status.
// It is the user-visible projection of Reconciler's internal state.
type ReconcileStatus struct {
	Running    bool         `json:"running"`
	StartedAt  string       `json:"startedAt,omitempty"`  // RFC3339, "" when never run
	FinishedAt string       `json:"finishedAt,omitempty"` // RFC3339, "" while running
	LastPlan   *DriftReport `json:"lastPlan,omitempty"`
	LastResult *ApplyResult `json:"lastResult,omitempty"`
	LastError  string       `json:"lastError,omitempty"`
	Canceled   bool         `json:"canceled,omitempty"`
}

// BacklogPayload is the typed shape of what aisearch stores in a backlog
// vector's payload. Keys here must match the map keys used in
// index.go:buildBacklogPayload.
//
// PayloadHash is the reconciler's skip-if-unchanged signal — see
// index.go:composePayloadHash. Absent (empty) hash means "legacy point, force
// re-embed once" so post-deploy drains converge in one tick.
type BacklogPayload struct {
	Kind            string   `json:"kind"`
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Status          string   `json:"status"`
	Priority        int      `json:"priority"`
	Tags            []string `json:"tags"`
	Milestone       string   `json:"milestone"`
	Effort          string   `json:"effort"`
	Archived        bool     `json:"archived"`
	TargetScenarios []string `json:"target_scenarios"`
	PayloadHash     string   `json:"payload_hash,omitempty"`
}

// RecordPayload is the typed shape of what aisearch stores in a record
// vector's payload. Keys here must match the map keys used in
// records.go:buildRecordPayload.
//
// entity_type is fixed to "record" so future code that pools records and
// backlog in shared filters (e.g. ai-search --kind fix) can discriminate.
type RecordPayload struct {
	EntityType   string   `json:"entity_type"`
	RecordID     string   `json:"record_id"`
	Kind         string   `json:"kind"`
	Scenario     string   `json:"scenario"`
	BacklogRef   string   `json:"backlog_ref,omitempty"`
	MilestoneID  string   `json:"milestone_id,omitempty"`
	Supersedes   string   `json:"supersedes,omitempty"`
	SupersededBy string   `json:"superseded_by,omitempty"`
	Outcome      string   `json:"outcome,omitempty"`
	Commit       string   `json:"commit,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`
	Stub         bool     `json:"stub"`
	Title        string   `json:"title,omitempty"`
	PayloadHash  string   `json:"payload_hash,omitempty"`
}

// GoalPayload is the typed shape of what aisearch stores in a goal vector's
// payload. PayloadHash is the same skip-if-unchanged
// signal documented on BacklogPayload.
type GoalPayload struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	Archived    bool   `json:"archived"`
	PayloadHash string `json:"payload_hash,omitempty"`
}
