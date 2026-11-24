package stagedmessages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func findWorktreeQueueEntry(entries []reviewjobs.WorktreeQueueEntry, matches func(reviewjobs.WorktreeQueueEntry) bool) *reviewjobs.WorktreeQueueEntry {
	for i := range entries {
		if matches(entries[i]) {
			return &entries[i]
		}
	}
	return nil
}

func (s *Service) FinalizeQueuedSelection(ctx context.Context, repo, dir string, req *v1.FinalizeQueuedSelectionRequest) (*v1.QueuedWorktreeSelection, error) {
	if req == nil || req.SelectionId == "" || req.ExpectedSnapshotDigest == "" {
		return nil, fmt.Errorf("selection id and expected snapshot digest are required")
	}
	q, err := s.store.ListWorktreeQueue(ctx, repo)
	if err != nil {
		return nil, err
	}
	entry := findWorktreeQueueEntry(q.Entries, func(entry reviewjobs.WorktreeQueueEntry) bool {
		return entry.SelectionID == req.SelectionId
	})
	if entry == nil {
		return nil, ErrNotFound
	}
	if entry.SnapshotDigest != req.ExpectedSnapshotDigest {
		return nil, ErrConflict
	}
	input, err := decodeDraftInput([]byte(entry.SelectionPayload))
	if err != nil {
		return nil, err
	}
	var applicable bool
	if entry.State == reviewjobs.WorktreeQueueAwaitingUnstage && entry.SelectionSource == int32(v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX) {
		applicable, err = WorktreeSelectionUnstagedApplicable(ctx, dir, input.Snapshot)
	} else {
		applicable, err = WorktreeSelectionApplicable(ctx, dir, input.Snapshot)
	}
	if err != nil {
		return nil, err
	}
	if !applicable {
		return nil, fmt.Errorf("%w: selected content does not match saved snapshot", ErrConflict)
	}
	q, err = s.store.FinalizeCapturedSelection(ctx, repo, entry.ID, entry.SnapshotDigest, entry.SelectionPayload)
	if err != nil {
		return nil, err
	}
	for _, e := range q.Entries {
		if e.ID == entry.ID {
			return queuedSelection(e), nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) StartQueuedSelection(ctx context.Context, repo, dir string, req *v1.StartQueuedSelectionRequest) (*v1.StagedMessageJob, error) {
	if req == nil || req.SelectionId == "" || req.SnapshotDigest == "" {
		return nil, fmt.Errorf("selection id and snapshot digest are required")
	}
	q, err := s.store.ListWorktreeQueue(ctx, repo)
	if err != nil {
		return nil, err
	}
	entry := findWorktreeQueueEntry(q.Entries, func(entry reviewjobs.WorktreeQueueEntry) bool {
		return entry.SelectionID == req.SelectionId
	})
	if entry == nil {
		return nil, ErrNotFound
	}
	if (entry.State != reviewjobs.WorktreeQueueQueued && entry.State != reviewjobs.WorktreeQueueStaged && entry.State != reviewjobs.WorktreeQueueFailed) || entry.SnapshotDigest != req.SnapshotDigest {
		return nil, ErrConflict
	}
	input, err := decodeDraftInput([]byte(entry.SelectionPayload))
	if err != nil {
		return nil, err
	}
	applicable, err := WorktreeSelectionApplicable(ctx, dir, input.Snapshot)
	if err != nil {
		return nil, err
	}
	if !applicable {
		return nil, ErrConflict
	}
	if strings.TrimSpace(req.RequestKey) == "" || len(req.RequestKey) > 200 || len(req.OperatorNote) > MaxOperatorNoteBytes {
		return nil, fmt.Errorf("bounded request key and operator note are required")
	}
	mode := normalizeMode(req.Mode)
	if mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK && mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		return nil, fmt.Errorf("unsupported staged message mode")
	}
	idempotency := "worktree-staged-message:" + repo + ":" + req.RequestKey
	if existing, e := s.store.GetByIdempotency(ctx, idempotency); e == nil {
		prior, decodeErr := decodeDraftInput([]byte(existing.InputJSON))
		if decodeErr != nil || prior.QueuedSelectionID != req.SelectionId || prior.Snapshot == nil || prior.Snapshot.Digest != req.SnapshotDigest || normalizeInputMode(prior) != mode || prior.OperatorNote != req.OperatorNote {
			return nil, fmt.Errorf("%w: request key reused for different queued selection", ErrConflict)
		}
		return s.advanceRead(ctx, repo, dir, existing, "get", 0)
	} else if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	if entry.JobID != "" {
		previous, getErr := s.store.Get(ctx, entry.JobID)
		// The captured selection is the durable retry source. A job can be
		// missing after retention/repair even though its queue entry survives;
		// in that case allow a new job to replace the orphaned reference.
		// Other storage errors still stop admission because ownership is unclear.
		if err := validateQueuedJobReplacement(previous, getErr); err != nil {
			return nil, err
		}
		if previous.ErrorCode == "provider_admission_maintenance" {
			readiness := s.ProviderReadiness(ctx)
			if !readiness.Known || readiness.State != "open" {
				return nil, fmt.Errorf("%w: %s", ErrProviderNotReady, readiness.Message)
			}
		}
	}
	input.Mode = mode
	input.OperatorNote = req.OperatorNote
	input.QueuedSelectionID = req.SelectionId
	input.ReplaceQueueEntryID = entry.ID
	input.ExpectedQueueRevision = q.Revision
	input.WorktreeCapture = true
	jobID, err := s.generateID("message-")
	if err != nil {
		return nil, err
	}
	if mode == v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		input.FormatVersion = 4
		input.JobID = jobID
	} else {
		input.FormatVersion = 2
	}
	input.Contexts, input.Omissions = s.searchDraftContexts(ctx, input.Snapshot)
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, fmt.Errorf("immutable worktree manifest exceeds %d bytes", MaxDraftInputBytes)
	}
	jobSpec := reviewjobs.Job{ID: jobID, IdempotencyKey: idempotency, InputDigest: digest(raw), InputJSON: string(raw), Operation: operation, RepositoryID: repo, PolicyVersion: "staged-message-worktree-v1", State: reviewjobs.Queued}
	replacement := reviewjobs.WorktreeQueueEntry{ID: entry.ID, RepositoryID: repo, JobID: jobID, SnapshotDigest: req.SnapshotDigest, Paths: entry.Paths, Mode: int32(mode), RequestKey: req.RequestKey}
	job, _, _, err := s.store.CreateWorktreeJobAndReplaceQueueEntry(ctx, jobSpec, entry.ID, q.Revision, replacement)
	if err != nil {
		return nil, err
	}
	return s.advanceRead(ctx, repo, dir, job, "get", 0)
}

func validateQueuedJobReplacement(previous reviewjobs.Job, lookupErr error) error {
	if errors.Is(lookupErr, sql.ErrNoRows) {
		return nil // The immutable queued selection remains recoverable.
	}
	if lookupErr != nil {
		return lookupErr
	}
	if previous.State != reviewjobs.Failed && previous.State != reviewjobs.Cancelled && previous.State != reviewjobs.Interrupted && previous.State != reviewjobs.AwaitingProvider {
		return fmt.Errorf("queued selection already has a non-retryable suggestion job: %w", ErrConflict)
	}
	return nil
}
