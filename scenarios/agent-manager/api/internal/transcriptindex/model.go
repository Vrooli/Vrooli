// Package transcriptindex is the Transcript Indexer (USAGE-INTELLIGENCE T-1):
// the one capture path for agent transcripts on this host. It discovers every
// transcript root, identifies each file by device and inode, tails it from a
// stored offset with the runner's own transcript parser, and writes durable
// usage summaries as it reads. Full events are an optional recall layer with
// its own byte budget; the Host owns runs, events and search.
package transcriptindex

import (
	"context"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// Root kinds name where a file was discovered.
const (
	RootWebConsoleCodex = "web-console-codex-home"
	RootCodexHome       = "codex-home"
	RootClaudeHome      = "claude-home"
	RootRunHome         = "agent-manager-run-home"
	RootConfigured      = "configured"
	RootManual          = "manual"
)

// ReplacementEffectsPending marks committed replacement evidence whose
// derived-owner notifications have not yet been acknowledged.
const ReplacementEffectsPending = "replacement committed; search and projection reconciliation pending"

// ReplacementReconciliationRequired marks old evidence retained while a
// source replacement needs an explicit owner retry.
const ReplacementReconciliationRequired = "transcript replacement requires reconciliation; prior evidence retained: "

// ReplacementPrecommitError means no promotion intent or owner transaction
// was attempted, so the prior checkpoint can be guarded with a CAS.
type ReplacementPrecommitError struct{ Err error }

func (e *ReplacementPrecommitError) Error() string { return e.Err.Error() }
func (e *ReplacementPrecommitError) Unwrap() error { return e.Err }

// Harness labels stored on file rows and as run import provenance.
const (
	HarnessCodex      = "codex"
	HarnessClaude     = "claude-code"
	HarnessConfigured = "configured"
	// HarnessMixed labels a coverage total over roots of several harnesses.
	HarnessMixed = "mixed"
)

// File statuses.
const (
	StatusPending    = "pending"    // seen, not yet (fully) read
	StatusIndexed    = "indexed"    // read up to its size
	StatusOwned      = "owned"      // an Agent Manager run records its events; the indexer records only its usage
	StatusExcluded   = "excluded"   // a visibility policy keeps it out
	StatusUnreadable = "unreadable" // open/parse failed; Error says why
)

// Owners of a file's usage.
const (
	OwnerIndexer = "indexer"
	OwnerRun     = "run"
)

// Root is one discovered directory of transcripts.
type Root struct {
	Path    string
	Kind    string
	Harness string
	// WebConsoleSessionID attributes a per-session web-console home.
	WebConsoleSessionID string
	// RunID is set for an Agent Manager run home: every file under it is
	// owned by that run.
	RunID string
}

// FileRow is one transcript file (transcript_files).
type FileRow struct {
	Key                 string
	Path                string
	RootKind            string
	RootPath            string
	Harness             string
	SessionID           string // the file's own session (fork child / sidechain agent id)
	HeaderSessionID     string // session id in the file's header (Codex thread)
	ParentSessionID     string
	ParentFileKey       string
	Sidechain           bool
	WebConsoleSessionID string
	RunID               string
	Owner               string
	Status              string
	Error               string
	Offset              int64
	Size                int64
	MtimeUnixNano       int64
	HeadHash            string
	Turns               int64
	UsageReports        int64
	Active              bool
	RecallBytes         int64
	RecallEvicted       bool
	// PendingFacts is the non-text work read after the newest usage report,
	// not yet attached to a report (encoded accumulator).
	PendingFacts string
	// ParserVersion is the version of the events the parser derived
	// (runner.TranscriptVersioner). A recall file read by an older version is
	// re-imported from zero into the same run.
	ParserVersion int
	// LastObservedEventAt is the latest timestamp actually parsed from this
	// transcript, not a claim that the native session has ended.
	LastObservedEventAt *time.Time
	FirstSeenAt         time.Time
	IndexedAt           *time.Time
}

// Mtime is the file's modification time.
func (f FileRow) Mtime() time.Time { return time.Unix(0, f.MtimeUnixNano) }

// Child reports whether the file is a sub-agent sidechain or a fork of a
// parent session file.
func (f FileRow) Child() bool { return f.Sidechain || f.ParentSessionID != "" }

// UsageRow is one durable usage report (transcript_usage) plus the non-text
// work the request consumed since the previous report (SPEC AM-3).
type UsageRow struct {
	FileKey    string
	LineOffset int64
	Ordinal    int
	RunID      string
	ObservedAt time.Time
	Usage      domain.UsageEventData
	Facts      TurnFacts
}

// TurnFacts counts non-text work. Image input tokens are inside the input
// counters; these counts explain them, they are never an extra charge.
type TurnFacts struct {
	ImagesIn           int            `json:"imagesIn,omitempty"`
	ImagesInUser       int            `json:"imagesInUser,omitempty"`
	ImagesInHighDetail int            `json:"imagesInHighDetail,omitempty"`
	ImageBytesIn       int64          `json:"imageBytesIn,omitempty"`
	ImagesOut          int            `json:"imagesOut,omitempty"`
	WebSearches        int            `json:"webSearches,omitempty"`
	WebFetches         int            `json:"webFetches,omitempty"`
	AudioIn            int            `json:"audioIn,omitempty"`
	ToolCalls          int            `json:"toolCalls,omitempty"`
	ToolKinds          map[string]int `json:"toolKinds,omitempty"`
	CreditsBalance     string         `json:"creditsBalance,omitempty"`
}

// Store persists file rows and usage summaries.
type Store interface {
	ListTranscriptFiles(ctx context.Context) ([]FileRow, error)
	// SaveTranscriptFile upserts the row and inserts its new usage rows in one
	// transaction. Usage rows are keyed by (file, line offset, ordinal), so a
	// re-read of the same bytes inserts nothing.
	SaveTranscriptFile(ctx context.Context, file FileRow, usage []UsageRow) error
	// ResetTranscriptFile deletes a file's usage rows before it is re-read
	// from offset 0 (truncation, rewrite or identity change).
	ResetTranscriptFile(ctx context.Context, key string) error
	DeleteTranscriptFile(ctx context.Context, key string) error
	TranscriptUsageForRun(ctx context.Context, runID string) ([]UsageRow, error)
	// RecallBytes measures a run's recall layer on its own tables: retained
	// run events plus derived conversation-search documents.
	RecallBytes(ctx context.Context, runID string) (int64, error)
}

// Adoption is the run that represents an external file.
type Adoption struct {
	RunID        string
	Excluded     bool
	LegacyRelink bool
	LegacyPath   string
}

// RunUpdate is the activity a tick observed for a file's run.
type RunUpdate struct {
	SessionID string
	Label     string
	StartedAt time.Time
	EndedAt   time.Time
	Active    bool
	// Recorded counts the events this tick appended to the run's recall. An
	// idle run that gained events was re-imported and must be re-derived.
	Recorded int
}

// Host is the run, event and search side of capture, implemented by the
// orchestrator.
type Host interface {
	// OwnedTranscripts maps each transcript path an Agent Manager run reads
	// itself (interactive tails) to that run.
	OwnedTranscripts(ctx context.Context) (map[string]string, error)
	// Adopt admits a file under the import visibility policy and returns its
	// run: file.RunID when it has one, else a new run or the relinked legacy
	// imported run that already names it. parentRunID links a sidechain or
	// fork to its parent run.
	Adopt(ctx context.Context, file FileRow, parentRunID string) (Adoption, error)
	// Parser returns a fresh transcript parser for the file's format.
	Parser(ctx context.Context, file FileRow) (runner.TranscriptParser, error)
	// ParserVersion is the current version of the events the harness's
	// parser derives; 0 for a parser that declares none.
	ParserVersion(harness string) int
	// Record applies one parsed batch: provider quota observations always;
	// the full events only when recall is true.
	Record(ctx context.Context, runID string, events []*domain.RunEvent, recall bool) error
	UpdateRun(ctx context.Context, runID string, update RunUpdate) error
	// DropRecall deletes a run's retained events and search documents. The
	// file's durable usage rows are untouched.
	DropRecall(ctx context.Context, runID string) error
}
