package validation_run

import "context"

// Repository is the persistence seam for validation_runs.
//
// seam: Repository
type Repository interface {
	Create(ctx context.Context, r Run) error
	Get(ctx context.Context, id string) (Run, error)
	UpdateStatus(ctx context.Context, r Run) error
	ListActive(ctx context.Context) ([]Run, error)

	// FindRecentMatching returns the most recent terminal run for the
	// (tuple_kind, subject_id, golden_slug) tuple. Used by Start when
	// force=false to short-circuit a duplicate run. Returns ErrRunNotFound
	// when no prior run exists.
	FindRecentMatching(ctx context.Context, kind int, subjectID, goldenSlug string) (Run, error)

	// ClaimNextQueued atomically transitions one queued run into
	// status=Running (returning it) so the worker can claim it. Returns
	// ErrRunNotFound if none are queued.
	ClaimNextQueued(ctx context.Context) (Run, error)
}
