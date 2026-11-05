// Package retention owns BAS evidence-retention planning across executions,
// recordings, captures, and pressure cleanup. Transport adapters convert
// contracts; deletion remains bounded by configured roots and filesystem seams.
package retention

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/database"
)

const (
	ReasonNonTerminal     = "non-terminal execution"
	ReasonKeepLatest      = "protected by keep_latest"
	ReasonTooNew          = "newer than max_age_days"
	ReasonUnsafePath      = "artifact directory outside recordings root"
	ReasonMissingDir      = "artifact directory already absent"
	ReasonDeleteDirFailed = "failed to delete artifact directory"
	ReasonDeleteRowFailed = "failed to delete execution row"
	ReasonMaxBytes        = "bounded by max_bytes per sweep"
)

// ErrRecordingsRootNotConfigured is returned when no recordings root is set, in
// which case retention cannot resolve or safely delete artifact directories.
var ErrRecordingsRootNotConfigured = errors.New("recordings root not configured")

type FileSystem interface {
	// DirSize returns the total bytes under dir and whether dir exists. A missing
	// directory returns (0, false, nil) rather than an error.
	DirSize(dir string) (sizeBytes int64, exists bool, err error)
	// RemoveAll removes dir and everything beneath it. Removing a missing dir is
	// a no-op (no error), mirroring os.RemoveAll.
	RemoveAll(dir string) error
}

type containedDeleter interface {
	DeleteContained(context.Context, string, string) error
}

type pathExistence interface {
	Exists(path string) (bool, error)
}

type ExecutionStore interface {
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	ListExecutions(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error)
	DeleteExecution(ctx context.Context, id uuid.UUID) error
}

type latestTerminalExecutionsPerWorkflow interface {
	ListLatestTerminalExecutionsPerWorkflow(
		ctx context.Context,
		workflowIDs []uuid.UUID,
		projectID *uuid.UUID,
		statuses []string,
		limit int,
	) ([]*database.ExecutionIndex, error)
}

type Service struct {
	store            ExecutionStore
	fs               FileSystem
	recordingsRoot   string
	capturesRoot     string
	recoveryLockPath string
	now              func() time.Time
	log              *logrus.Logger
	cleanupMu        sync.Mutex
	cleanupDone      map[string]CleanupApplyResponse
	recoveryCacheMu  sync.Mutex
	protectedCacheAt time.Time
	protectedCache   map[string]struct{}
	orphanCacheAt    time.Time
	orphanCacheKey   string
	orphanCache      []CleanupItem
}

// NewService constructs a retention Service. When now is nil, time.Now is used.
func NewService(store ExecutionStore, fs FileSystem, recordingsRoot string, log *logrus.Logger) *Service {
	return &Service{
		store:          store,
		fs:             fs,
		recordingsRoot: strings.TrimSpace(recordingsRoot),
		cleanupDone:    make(map[string]CleanupApplyResponse),
		now:            time.Now,
		log:            log,
	}
}

func (s *Service) WithCaptureRoot(root string) *Service {
	if s != nil {
		s.capturesRoot = strings.TrimSpace(root)
	}
	return s
}

func (s *Service) WithRecoveryLockPath(path string) *Service {
	if s != nil {
		s.recoveryLockPath = strings.TrimSpace(path)
	}
	return s
}

func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

type Options struct {
	// MaxAgeDays removes executions older than this many days (by completed_at
	// when present, otherwise started_at). 0 disables the age filter.
	MaxAgeDays int
	// MaxAgeSeconds is the transport-precise equivalent used by the owner
	// cleanup contract. When set it takes precedence over MaxAgeDays.
	MaxAgeSeconds int64
	// KeepLatest spares this many most-recent terminal executions per workflow.
	KeepLatest int
	// WorkflowID, ProjectID optionally narrow the candidate set.
	WorkflowID *uuid.UUID
	ProjectID  *uuid.UUID
	// Status optionally restricts to a single terminal status; empty means both
	// completed and failed are eligible.
	Status string
	// ExecutionIDs restricts apply to the items returned by a preview.
	ExecutionIDs []uuid.UUID
	// MaxBytes caps the bytes selected by one sweep. It is a reclaim-batch cap,
	// not a declaration of total storage capacity; the owner declaration remains
	// the source of the age/size policy shown to operators.
	MaxBytes int64
	// MaxItems bounds one scheduled sweep by execution directories. Zero keeps
	// the existing unbounded behavior for explicit operator previews.
	MaxItems int
	// EstimatedBytes optionally supplies sizes from an already validated
	// preview. Apply callers may use this to avoid re-walking large artifact
	// trees; deletion still revalidates containment through the deleter.
	EstimatedBytes map[uuid.UUID]int64
	// Apply performs deletion. When false, the sweep is a pure dry-run.
	Apply bool
}

type Item struct {
	ExecutionID    uuid.UUID
	Status         string
	WorkflowID     uuid.UUID
	StartedAt      time.Time
	CompletedAt    *time.Time
	ResultPath     string
	ArtifactDir    string
	EstimatedBytes int64
	Reason         string
}

type Report struct {
	DryRun          bool
	Removed         []Item
	Skipped         []Item
	EstimatedBytes  int64
	RemovedCount    int
	SkippedCount    int
	ErrorCount      int
	RemovedByStatus map[string]int
}

// CleanupItem is one candidate and the bounded selection passed to apply.
type CleanupItem struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Bytes      int64     `json:"bytes"`
	AgeSeconds int64     `json:"age_seconds"`
	Protected  bool      `json:"protected"`
	ModifiedAt time.Time `json:"-"`
}

type CleanupApplyResponse struct {
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
}

type CleanupEstimate struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	BlockedReason  string    `json:"blocked_reason,omitempty"`
	MinAgeSeconds  int64     `json:"min_age_seconds,omitempty"`
	KeepCount      int       `json:"keep_count,omitempty"`
	MaxBytes       int64     `json:"max_bytes,omitempty"`
	ObservedAt     time.Time `json:"observed_at"`
}

type CleanupPreview struct {
	ProviderID    string        `json:"provider_id"`
	Items         []CleanupItem `json:"items"`
	BlockedReason string        `json:"blocked_reason,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
	MinAgeSeconds int64         `json:"min_age_seconds,omitempty"`
	KeepCount     int           `json:"keep_count,omitempty"`
	MaxBytes      int64         `json:"max_bytes,omitempty"`
}
