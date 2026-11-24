package stagedmessages

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func protoQueueEntry(e reviewjobs.WorktreeQueueEntry) *v1.WorktreeSuggestionQueueEntry {
	state := v1.WorktreeSuggestionQueueState(e.State)
	if e.State == reviewjobs.WorktreeQueueAwaitingUnstage {
		state = v1.WorktreeSuggestionQueueState_WORKTREE_SUGGESTION_QUEUE_STATE_AWAITING_UNSTAGE
	}
	return &v1.WorktreeSuggestionQueueEntry{Id: e.ID, RepositoryId: e.RepositoryID, JobId: e.JobID, SnapshotDigest: e.SnapshotDigest, Paths: append([]string(nil), e.Paths...), Mode: v1.StagedMessageMode(e.Mode), RequestKey: e.RequestKey, Sequence: e.Sequence, State: state, FailureReason: e.FailureReason, CreatedAt: e.CreatedAt.UTC().Format(timeFormat), UpdatedAt: e.UpdatedAt.UTC().Format(timeFormat), SelectionSource: v1.WorktreeSelectionSource(e.SelectionSource), SelectionId: e.SelectionID}
}

func protoQueue(q reviewjobs.WorktreeQueue) *v1.WorktreeSuggestionQueue {
	out := &v1.WorktreeSuggestionQueue{RepositoryId: q.RepositoryID, Revision: q.Revision}
	for _, e := range q.Entries {
		out.Entries = append(out.Entries, protoQueueEntry(e))
	}
	return out
}

func (s *Service) ListWorktreeQueue(ctx context.Context, repo string) (*v1.WorktreeSuggestionQueue, error) {
	q, err := s.store.ListWorktreeQueue(ctx, repo)
	if err != nil {
		return nil, err
	}
	return protoQueue(q), nil
}

// ValidateQueueCommitAdmission is the server-side commit fence. A queued
// selection may enter a commit only after it is the reserved head and its
// immutable versions still match Git's stage-0 index.
func (s *Service) ValidateQueueCommitAdmission(ctx context.Context, repo, dir string, stagedPaths []string) error {
	queue, err := s.store.ListWorktreeQueue(ctx, repo)
	if err != nil {
		return err
	}
	if len(queue.Entries) == 0 {
		return nil
	}
	staged := make(map[string]struct{}, len(stagedPaths))
	for _, path := range stagedPaths {
		staged[path] = struct{}{}
	}
	if queue.Entries[0].State == reviewjobs.WorktreeQueueCommitPending {
		entry := queue.Entries[0]
		snapshotEntry, snapshot, snapshotErr := s.worktreeQueueSnapshot(ctx, repo, entry.ID)
		if snapshotErr != nil {
			return snapshotErr
		}
		if len(entry.Paths) == 0 || !allPathsIncluded(entry.Paths, staged) {
			return fmt.Errorf("reserved queue head is no longer fully staged: %w", ErrConflict)
		}
		matches, verifyErr := WorktreeSelectionStagedApplicable(ctx, dir, snapshot)
		if verifyErr != nil {
			return verifyErr
		}
		if !matches || snapshotEntry.ID != entry.ID {
			return fmt.Errorf("reserved queue head no longer matches the Git index: %w", ErrConflict)
		}
		return nil
	}
	for index, entry := range queue.Entries {
		stagedCount := 0
		for _, path := range entry.Paths {
			if _, ok := staged[path]; ok {
				stagedCount++
			}
		}
		if stagedCount == 0 {
			continue
		}
		if index != 0 || stagedCount != len(entry.Paths) {
			return fmt.Errorf("staged files overlap an unresolved queued selection: %w", ErrConflict)
		}
		return fmt.Errorf("queue head must be reserved before its staged files can be committed: %w", ErrConflict)
	}
	return nil
}

func allPathsIncluded(paths []string, included map[string]struct{}) bool {
	for _, path := range paths {
		if _, ok := included[path]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) EnqueueWorktreeQueue(ctx context.Context, repo string, req *v1.EnqueueWorktreeSuggestionRequest) (*v1.WorktreeSuggestionQueueEntry, error) {
	if req == nil || req.RepositoryId != repo || strings.TrimSpace(req.RequestKey) == "" {
		return nil, fmt.Errorf("repository and request key are required")
	}
	entry, err := s.queueEntryForJob(ctx, repo, req.JobId, req.SnapshotDigest, req.Paths, req.Mode, req.RequestKey)
	if err != nil {
		return nil, err
	}
	entry, _, err = s.store.EnqueueWorktreeQueue(ctx, entry)
	if err != nil {
		return nil, err
	}
	return protoQueueEntry(entry), nil
}

func (s *Service) queueEntryForJob(ctx context.Context, repo, jobID, digest string, paths []string, mode v1.StagedMessageMode, requestKey string) (reviewjobs.WorktreeQueueEntry, error) {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return reviewjobs.WorktreeQueueEntry{}, err
	}
	if job.Operation != operation || job.RepositoryID != repo {
		return reviewjobs.WorktreeQueueEntry{}, ErrNotFound
	}
	input, err := decodeDraftInput([]byte(job.InputJSON))
	if err != nil {
		return reviewjobs.WorktreeQueueEntry{}, err
	}
	if !input.WorktreeCapture || input.Snapshot == nil || input.Snapshot.Digest != digest || normalizeInputMode(input) != normalizeMode(mode) || job.IdempotencyKey != "worktree-staged-message:"+repo+":"+requestKey {
		return reviewjobs.WorktreeQueueEntry{}, ErrConflict
	}
	manifest := worktreeSnapshotPaths(input.Snapshot)
	sort.Strings(manifest)
	if !sameWorktreePathSet(manifest, paths) {
		return reviewjobs.WorktreeQueueEntry{}, ErrConflict
	}
	entryID, err := s.generateID("worktree-queue-")
	if err != nil {
		return reviewjobs.WorktreeQueueEntry{}, err
	}
	return reviewjobs.WorktreeQueueEntry{ID: entryID, RepositoryID: repo, JobID: job.ID, SnapshotDigest: digest, Paths: manifest, Mode: int32(normalizeMode(mode)), RequestKey: requestKey}, nil
}

func (s *Service) UpdateWorktreeQueue(ctx context.Context, repo, dir string, req *v1.UpdateWorktreeSuggestionQueueEntryRequest) (*v1.WorktreeSuggestionQueueMutationResponse, error) {
	if req == nil || req.RepositoryId != repo {
		return nil, fmt.Errorf("repository id is required")
	}
	var q reviewjobs.WorktreeQueue
	var e *reviewjobs.WorktreeQueueEntry
	var err error
	if req.State == v1.WorktreeSuggestionQueueState_WORKTREE_SUGGESTION_QUEUE_STATE_UNSPECIFIED {
		replacement, replaceErr := s.queueEntryForJob(ctx, repo, req.ReplacementJobId, req.ReplacementSnapshotDigest, req.ReplacementPaths, req.ReplacementMode, req.ReplacementRequestKey)
		if replaceErr != nil {
			return nil, replaceErr
		}
		q, e, err = s.store.ReplaceWorktreeQueueEntry(ctx, repo, req.EntryId, req.ExpectedRevision, replacement)
	} else {
		target := reviewjobs.WorktreeQueueState(req.State)
		failureReason := req.FailureReason
		if target == reviewjobs.WorktreeQueueStaged || target == reviewjobs.WorktreeQueueCommitPending || target == reviewjobs.WorktreeQueueQueued {
			entry, snapshot, snapshotErr := s.worktreeQueueSnapshot(ctx, repo, req.EntryId)
			if snapshotErr != nil {
				return nil, snapshotErr
			}
			var matches bool
			switch target {
			case reviewjobs.WorktreeQueueStaged, reviewjobs.WorktreeQueueCommitPending:
				matches, err = WorktreeSelectionStagedApplicable(ctx, dir, snapshot)
			case reviewjobs.WorktreeQueueQueued:
				matches, err = WorktreeSelectionUnstagedApplicable(ctx, dir, snapshot)
			}
			if err != nil {
				return nil, err
			}
			if !matches {
				if target == reviewjobs.WorktreeQueueCommitPending {
					return nil, fmt.Errorf("queue state does not match the current Git index for %q: %w", entry.ID, ErrConflict)
				}
				target = reviewjobs.WorktreeQueueStale
				failureReason = "The saved file versions no longer match the Git index. Review the files and stage the current versions or remove this selection."
			}
		}
		q, e, err = s.store.UpdateWorktreeQueueEntry(ctx, repo, req.EntryId, req.ExpectedHeadEntryId, req.ExpectedRevision, target, failureReason)
	}
	if err != nil {
		return nil, err
	}
	return &v1.WorktreeSuggestionQueueMutationResponse{Queue: protoQueue(q), ChangedEntry: protoQueueEntry(*e)}, nil
}

// worktreeQueueSnapshot resolves the immutable snapshot owned by a queue row.
// Rows can be backed by either a saved captured selection or a suggestion job;
// state transitions must validate both forms against the same manifest.
func (s *Service) worktreeQueueSnapshot(ctx context.Context, repo, id string) (reviewjobs.WorktreeQueueEntry, *v1.StagedMessageSnapshot, error) {
	queue, err := s.store.ListWorktreeQueue(ctx, repo)
	if err != nil {
		return reviewjobs.WorktreeQueueEntry{}, nil, err
	}
	entry := findWorktreeQueueEntry(queue.Entries, func(entry reviewjobs.WorktreeQueueEntry) bool {
		return entry.ID == id
	})
	if entry != nil {
		var input DraftInput
		if entry.SelectionPayload != "" {
			input, err = decodeDraftInput([]byte(entry.SelectionPayload))
		} else if entry.JobID != "" {
			var job reviewjobs.Job
			job, err = s.store.Get(ctx, entry.JobID)
			if err == nil && (job.Operation != operation || job.RepositoryID != repo) {
				err = ErrNotFound
			}
			if err == nil {
				input, err = decodeDraftInput([]byte(job.InputJSON))
			}
		} else {
			err = ErrNotFound
		}
		if err != nil {
			return reviewjobs.WorktreeQueueEntry{}, nil, err
		}
		if !input.WorktreeCapture || input.Snapshot == nil || input.Snapshot.Digest != entry.SnapshotDigest {
			return reviewjobs.WorktreeQueueEntry{}, nil, fmt.Errorf("queue entry has no matching immutable snapshot: %w", ErrConflict)
		}
		if !sameWorktreePaths(input.Snapshot, entry.Paths) {
			return reviewjobs.WorktreeQueueEntry{}, nil, fmt.Errorf("queue entry paths do not match its immutable snapshot: %w", ErrConflict)
		}
		return *entry, input.Snapshot, nil
	}
	return reviewjobs.WorktreeQueueEntry{}, nil, ErrNotFound
}

func (s *Service) RetireWorktreeQueueHead(ctx context.Context, repo, dir string, req *v1.RetireWorktreeSuggestionQueueHeadRequest) (*v1.WorktreeSuggestionQueueMutationResponse, error) {
	if req == nil || req.RepositoryId != repo {
		return nil, fmt.Errorf("repository id is required")
	}
	if reviewjobs.WorktreeQueueDisposition(req.Disposition) == reviewjobs.WorktreeQueueDispositionCommitted {
		entry, snapshot, err := s.worktreeQueueSnapshot(ctx, repo, req.ExpectedHeadEntryId)
		if err != nil {
			return nil, err
		}
		if entry.State != reviewjobs.WorktreeQueueCommitPending {
			return nil, fmt.Errorf("only a reserved queue head can be committed: %w", ErrConflict)
		}
		committed, err := WorktreeSelectionCommittedAtHead(ctx, dir, snapshot)
		if err != nil {
			return nil, err
		}
		if !committed {
			q, e, err := s.store.UpdateWorktreeQueueEntry(ctx, repo, req.ExpectedHeadEntryId, req.ExpectedHeadEntryId, req.ExpectedRevision,
				reviewjobs.WorktreeQueueStale,
				"The commit completed without the exact saved file versions. Review this selection before staging or removing it.",
			)
			if err != nil {
				return nil, err
			}
			return &v1.WorktreeSuggestionQueueMutationResponse{Queue: protoQueue(q), ChangedEntry: protoQueueEntry(*e)}, nil
		}
	}
	q, e, err := s.store.RetireWorktreeQueueHead(ctx, repo, req.ExpectedHeadEntryId, req.ExpectedRevision, reviewjobs.WorktreeQueueDisposition(req.Disposition))
	if err != nil {
		return nil, err
	}
	return &v1.WorktreeSuggestionQueueMutationResponse{Queue: protoQueue(q), ChangedEntry: protoQueueEntry(*e)}, nil
}

func (s *Service) RemoveWorktreeQueueEntry(ctx context.Context, repo string, req *v1.RemoveWorktreeSuggestionQueueEntryRequest) (*v1.WorktreeSuggestionQueueMutationResponse, error) {
	if req == nil || req.RepositoryId != repo {
		return nil, fmt.Errorf("repository id is required")
	}
	q, e, err := s.store.RemoveWorktreeQueueEntry(ctx, repo, req.EntryId, req.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	return &v1.WorktreeSuggestionQueueMutationResponse{Queue: protoQueue(q), ChangedEntry: protoQueueEntry(*e)}, nil
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
