package stagedmessages

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func queuedSnapshotDigest(snapshot *v1.StagedMessageSnapshot) (string, error) {
	type fileIdentity struct {
		Path, OldMode, NewMode, OldBlobID, NewBlobID, Status string
		Binary                                               bool
	}
	canonical := struct {
		RepositoryID, BaseHead string
		Files                  []fileIdentity
	}{RepositoryID: snapshot.GetRepositoryId(), BaseHead: snapshot.GetBaseHead()}
	for _, file := range snapshot.GetFiles() {
		canonical.Files = append(canonical.Files, fileIdentity{Path: file.GetPath(), OldMode: file.GetOldMode(), NewMode: file.GetNewMode(), OldBlobID: file.GetOldBlobId(), NewBlobID: file.GetNewBlobId(), Status: file.GetStatus(), Binary: file.GetBinary()})
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s *Service) QueueWorktreeSelection(ctx context.Context, repo, dir string, req *v1.QueueWorktreeSelectionRequest) (*v1.QueuedWorktreeSelection, error) {
	if req == nil || strings.TrimSpace(req.SelectionId) == "" || len(req.SelectionId) > 200 {
		return nil, fmt.Errorf("bounded selection id is required")
	}
	paths, err := normalizeWorktreePaths(req.Paths)
	if err != nil {
		return nil, err
	}
	var captured Captured
	switch req.Source {
	case v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_INDEX_TO_WORKTREE:
		captured, err = captureWorktreeSnapshot(ctx, repo, dir, paths, s.now)
	case v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX:
		captured, err = s.captureSelectedStaged(ctx, repo, dir, paths)
	default:
		return nil, fmt.Errorf("selection source is required")
	}
	if err != nil {
		return nil, err
	}
	if len(captured.Snapshot.Files) == 0 {
		return nil, fmt.Errorf("selected snapshot is empty")
	}
	captured.Snapshot.Digest, err = queuedSnapshotDigest(captured.Snapshot)
	if err != nil {
		return nil, err
	}
	var current Captured
	if req.Source == v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX {
		current, err = s.captureSelectedStaged(ctx, repo, dir, paths)
	} else {
		current, err = captureWorktreeSnapshot(ctx, repo, dir, paths, s.now)
	}
	if err != nil {
		return nil, err
	}
	currentDigest, err := queuedSnapshotDigest(current.Snapshot)
	if err != nil {
		return nil, err
	}
	if currentDigest != captured.Snapshot.Digest {
		return nil, fmt.Errorf("%w: selected content changed during queue capture", ErrConflict)
	}
	input := DraftInput{FormatVersion: 2, Snapshot: captured.Snapshot, WorktreeCapture: true, CapturedWorktree: captured.Bytes, QueuedSelection: true, QueuedSource: req.Source}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, fmt.Errorf("immutable selection exceeds %d bytes", MaxDraftInputBytes)
	}
	state := reviewjobs.WorktreeQueueQueued
	if req.Source == v1.WorktreeSelectionSource_WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX {
		state = reviewjobs.WorktreeQueueAwaitingUnstage
	}
	entry := reviewjobs.WorktreeQueueEntry{ID: req.SelectionId, RepositoryID: repo, SnapshotDigest: captured.Snapshot.Digest, Paths: paths, RequestKey: "selection:" + req.SelectionId, State: state, SelectionID: req.SelectionId, SelectionSource: int32(req.Source), SelectionPayload: string(raw)}
	queued, _, err := s.store.EnqueueCapturedSelection(ctx, entry)
	if err != nil {
		return nil, err
	}
	return queuedSelection(queued), nil
}

func (s *Service) captureSelectedStaged(ctx context.Context, repo, dir string, paths []string) (Captured, error) {
	head, err := gitRead(ctx, dir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		head = nil
	}
	baseHead := strings.TrimSpace(string(head))
	snap := &v1.StagedMessageSnapshot{RepositoryId: repo, BaseHead: baseHead}
	data := map[string]FileBytes{}
	total := 0
	args := []string{"diff", "--cached", "--raw", "--no-abbrev", "--no-renames", "--no-ext-diff", "--no-textconv", "-z"}
	if baseHead != "" {
		args = append(args, baseHead)
	}
	args = append(args, "--")
	args = append(args, paths...)
	raw, err := gitRead(ctx, dir, args...)
	if err != nil {
		return Captured{}, err
	}
	rows := bytes.Split(raw, []byte{0})
	for i := 0; i < len(rows) && len(rows[i]) > 0; i += 2 {
		if i+1 >= len(rows) {
			return Captured{}, fmt.Errorf("incomplete selected staged diff")
		}
		fields := strings.Fields(strings.TrimPrefix(string(rows[i]), ":"))
		path := string(rows[i+1])
		if len(fields) != 5 || !utf8.ValidString(path) {
			return Captured{}, fmt.Errorf("invalid selected staged diff entry")
		}
		f := &v1.StagedMessageFile{Path: path, OldPath: path, OldMode: fields[0], NewMode: fields[1], OldBlobId: fields[2], NewBlobId: fields[3], Status: fields[4]}
		if zeroOID(f.OldBlobId) {
			f.OldBlobId = ""
		}
		if zeroOID(f.NewBlobId) {
			f.NewBlobId = ""
		}
		idx, readErr := gitRead(ctx, dir, "ls-files", "--stage", "-z", "--", path)
		if readErr != nil {
			return Captured{}, readErr
		}
		for _, row := range bytes.Split(idx, []byte{0}) {
			if len(row) == 0 {
				continue
			}
			meta, got, ok := strings.Cut(string(row), "\t")
			idxFields := strings.Fields(meta)
			if !ok || got != path || len(idxFields) != 3 || idxFields[2] != "0" {
				return Captured{}, fmt.Errorf("selected path %q has an unmerged or invalid index entry", path)
			}
		}
		var before, after []byte
		if f.OldBlobId != "" {
			before, err = gitRead(ctx, dir, "cat-file", "blob", f.OldBlobId)
			if err != nil {
				return Captured{}, err
			}
		}
		if f.NewBlobId != "" {
			after, err = gitRead(ctx, dir, "cat-file", "blob", f.NewBlobId)
			if err != nil {
				return Captured{}, err
			}
		}
		if len(before) > maxFileBytes || len(after) > maxFileBytes || total+len(before)+len(after) > maxTotalBytes {
			return Captured{}, fmt.Errorf("selected staged content exceeds immutable capture bounds")
		}
		total += len(before) + len(after)
		f.Binary = bytes.IndexByte(after, 0) >= 0 || !utf8.Valid(after)
		snap.Files = append(snap.Files, f)
		data[path] = FileBytes{Before: before, After: after}
	}
	if len(snap.Files) != len(paths) {
		return Captured{}, fmt.Errorf("selected paths are not all staged")
	}
	snap.CapturedAt = snapshotTime(s.now)
	snap.Digest, err = queuedSnapshotDigest(snap)
	if err != nil {
		return Captured{}, err
	}
	return Captured{Snapshot: snap, Bytes: data}, nil
}

func queuedSelection(e reviewjobs.WorktreeQueueEntry) *v1.QueuedWorktreeSelection {
	return &v1.QueuedWorktreeSelection{SelectionId: e.SelectionID, RepositoryId: e.RepositoryID, Source: v1.WorktreeSelectionSource(e.SelectionSource), Paths: append([]string(nil), e.Paths...), SnapshotDigest: e.SnapshotDigest, State: queuedSelectionState(e.State), FailureReason: e.FailureReason}
}

func queuedSelectionState(state reviewjobs.WorktreeQueueState) v1.QueuedWorktreeSelectionState {
	if state == reviewjobs.WorktreeQueueAwaitingUnstage {
		return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_AWAITING_UNSTAGE
	}
	if state == reviewjobs.WorktreeQueueFailed {
		return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_FAILED
	}
	if state == reviewjobs.WorktreeQueueStale {
		return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_STALE
	}
	return v1.QueuedWorktreeSelectionState_QUEUED_WORKTREE_SELECTION_STATE_QUEUED
}

func (s *Service) FinalizeQueuedSelection(ctx context.Context, repo, dir string, req *v1.FinalizeQueuedSelectionRequest) (*v1.QueuedWorktreeSelection, error) {
	if req == nil || req.SelectionId == "" || req.ExpectedSnapshotDigest == "" {
		return nil, fmt.Errorf("selection id and expected snapshot digest are required")
	}
	q, err := s.store.ListWorktreeQueue(ctx, repo)
	if err != nil {
		return nil, err
	}
	var entry *reviewjobs.WorktreeQueueEntry
	for i := range q.Entries {
		if q.Entries[i].SelectionID == req.SelectionId {
			entry = &q.Entries[i]
			break
		}
	}
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
	var entry *reviewjobs.WorktreeQueueEntry
	for i := range q.Entries {
		if q.Entries[i].SelectionID == req.SelectionId {
			entry = &q.Entries[i]
			break
		}
	}
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
	manifest := make([]string, 0, len(input.Snapshot.Files))
	for _, f := range input.Snapshot.Files {
		manifest = append(manifest, f.Path)
	}
	sort.Strings(manifest)
	wanted := append([]string(nil), paths...)
	sort.Strings(wanted)
	if strings.Join(manifest, "\x00") != strings.Join(wanted, "\x00") {
		return reviewjobs.WorktreeQueueEntry{}, ErrConflict
	}
	entryID, err := s.generateID("worktree-queue-")
	if err != nil {
		return reviewjobs.WorktreeQueueEntry{}, err
	}
	return reviewjobs.WorktreeQueueEntry{ID: entryID, RepositoryID: repo, JobID: job.ID, SnapshotDigest: digest, Paths: manifest, Mode: int32(normalizeMode(mode)), RequestKey: requestKey}, nil
}

func (s *Service) UpdateWorktreeQueue(ctx context.Context, repo string, req *v1.UpdateWorktreeSuggestionQueueEntryRequest) (*v1.WorktreeSuggestionQueueMutationResponse, error) {
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
		q, e, err = s.store.UpdateWorktreeQueueEntry(ctx, repo, req.EntryId, req.ExpectedHeadEntryId, req.ExpectedRevision, reviewjobs.WorktreeQueueState(req.State), req.FailureReason)
	}
	if err != nil {
		return nil, err
	}
	return &v1.WorktreeSuggestionQueueMutationResponse{Queue: protoQueue(q), ChangedEntry: protoQueueEntry(*e)}, nil
}

func (s *Service) RetireWorktreeQueueHead(ctx context.Context, repo string, req *v1.RetireWorktreeSuggestionQueueHeadRequest) (*v1.WorktreeSuggestionQueueMutationResponse, error) {
	if req == nil || req.RepositoryId != repo {
		return nil, fmt.Errorf("repository id is required")
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
