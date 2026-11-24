package conversationsearch

import (
	"context"
	"time"
)

// Document is a deterministic, bounded projection of one canonical message
// chunk. DocumentID is stable across rebuilds and never exposes SQLite rowid.
type Document struct {
	DocumentID           string
	SourceRunID          string
	SourceEventID        string
	SourceMessageID      string
	ChunkIndex           int
	ChunkTotal           int
	StartByte            int
	EndByte              int
	EventSequence        int64
	NativeRecordSequence int64
	Role                 string
	OccurredAt           time.Time
	Content              string
	ContentClass         ContentClass
	SourceHash           string
	ContentHash          string
	RecipeVersion        string
	Harness              string
	SourceSessionID      string
	ProviderOrigin       string
	Importer             string
	ProjectScope         string
	CWDScope             string
	Runner               string
	Model                string
	Profile              string
	RunStatus            string
	RunLabel             string
	Tags                 []string
	Workloads            []string
	EvidenceRef          string
	Visible              bool
	IndexedAt            time.Time
}

type SourceCursor struct {
	OccurredAt time.Time
	// OccurredAtText is the canonical timestamp text this cursor was formed
	// from. run_events writers keep variable RFC3339 precision, so comparing a
	// re-formatted time against that text is not safe: the cursor must carry
	// the exact string it was ordered by.
	OccurredAtText        string
	SourceRunID           string
	EventSequence         int64
	SnapshotMaxEventRowID int64
}

type SourcePage struct {
	Documents  []Document
	NextCursor *SourceCursor
}

type SourceRepository interface {
	LoadSourcePage(context.Context, *SourceCursor, int) (SourcePage, error)
}

// RunDocumentSource is an optional targeted seam for incremental projection.
// Implementations avoid rescanning the complete canonical corpus when the
// durable change queue identifies one run that needs replacement.
type RunDocumentSource interface {
	LoadRunDocuments(context.Context, string) ([]Document, error)
}

// EventDocumentSource narrows append-driven reconciliation to the canonical
// event already identified by the durable change queue.
type EventDocumentSource interface {
	LoadEventDocuments(context.Context, string, string) ([]Document, error)
}

// SnapshotSource can pin a paged traversal to an append-only source high-water
// mark. Sources without this optional seam retain the conservative two-scan
// mutation check used by deterministic fakes.
type SnapshotSource interface {
	SnapshotCursor(context.Context) (*SourceCursor, error)
}

type ProjectionRepository interface {
	UpsertDocument(context.Context, Document) error
	DeleteDocument(context.Context, string) error
	DeleteSourceEvent(context.Context, string, string) (int64, error)
	DeleteRun(context.Context, string) (int64, error)
	GetDocument(context.Context, string) (Document, error)
	ContextDocuments(context.Context, Document, int, int) ([]Document, error)
}

type CandidateQuery struct {
	Query            string
	PrefilterLiteral string
	MatchAllTerms    bool
	Limit            int
	ByteLimit        int
	Sort             SearchSort
	After            *CandidateCursor
	OccurredAfter    *time.Time
	OccurredBefore   *time.Time
	Roles            []string
	Harnesses        []string
	ProviderOrigins  []string
	ProjectScopes    []string
	CWDScopes        []string
	Runners          []string
	Models           []string
	Profiles         []string
	RunStatuses      []string
	Tags             []string
	Workloads        []string
	ContentClasses   []ContentClass
	IncludeHidden    bool
}

type SearchSort int

const (
	SearchSortRelevance SearchSort = iota + 1
	SearchSortNewest
	SearchSortOldest
	SearchSortNativeSequence
)

type CandidateCursor struct {
	Score                float64
	OccurredAt           time.Time
	DocumentID           string
	NativeRecordSequence int64
	EventSequence        int64
}

type Candidate struct {
	Document Document
	Score    float64
	Rank     int
}

type RegexCandidatePage struct {
	Documents    []Document
	ScannedBytes int
	HasMore      bool
	LimitReason  RegexLimitReason
}

type RegexLimitReason string

const (
	RegexLimitNone       RegexLimitReason = ""
	RegexLimitCandidates RegexLimitReason = "candidate_limit"
	RegexLimitBytes      RegexLimitReason = "byte_limit"
	RegexLimitDeadline   RegexLimitReason = "deadline"
)

type CandidateRepository interface {
	LexicalCandidates(context.Context, CandidateQuery) ([]Candidate, error)
}

type RegexCandidateRepository interface {
	RegexCandidates(context.Context, CandidateQuery) (RegexCandidatePage, error)
}

type VisibilityRepository interface {
	VisibleDocument(context.Context, string) (bool, error)
}

type Checkpoint struct {
	SourceName        string
	SourceCursor      string
	SourceFingerprint string
	UpdatedAt         time.Time
	LastErrorCode     string
}

type Generation struct {
	GenerationID       string
	State              string
	RecipeVersion      string
	SourceCheckpoint   string
	PlannedDocuments   uint64
	ProcessedDocuments uint64
	FailedDocuments    uint64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type StatusRepository interface {
	LoadCheckpoint(context.Context, string) (Checkpoint, error)
	SaveCheckpoint(context.Context, Checkpoint) error
	LoadGeneration(context.Context, string) (Generation, error)
	SaveGeneration(context.Context, Generation) error
	CountCoverage(context.Context) (visibleMessages, catalogDocuments, lexicalDocuments uint64, err error)
}

type ProjectionStatus struct {
	CanonicalMessages uint64
	CatalogDocuments  uint64
	LexicalDocuments  uint64
	SemanticDocuments uint64
	// SemanticEligibleDocuments is the denominator the semantic leg actually
	// covers: visible prose/quoted chunks, not tool payloads that are never
	// embedded. It keeps a 98.9% eligible index from reading as 4.7%.
	SemanticEligibleDocuments uint64
	// SemanticCoverageUnknown is true when no vector count has been observed
	// (the first probe is in flight or failed). SemanticDocuments is then 0 by
	// absence, not by measurement, and consumers must not read it as empty
	// coverage.
	SemanticCoverageUnknown bool
	PendingChanges          uint64
	DeletedDocuments        uint64
	OrphanDocuments         uint64
	ActiveGeneration        string
	CandidateGeneration     string
	LastSuccessAt           time.Time
	LastIndexedAt           time.Time
	LastErrorCode           string
	CollectionName          string
	CollectionLayout        string
	EmbeddingModel          string
	DegradedDependencies    []string
}

type ChangeOperation string

const (
	ChangeUpsertRun   ChangeOperation = "upsert_run"
	ChangeDeleteEvent ChangeOperation = "delete_event"
	ChangeDeleteRun   ChangeOperation = "delete_run"
	ChangeRepair      ChangeOperation = "repair"
)

type ProjectionChange struct {
	Sequence      int64
	Operation     ChangeOperation
	SourceRunID   string
	SourceEventID string
	CreatedAt     time.Time
}
