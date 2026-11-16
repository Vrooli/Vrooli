package stagedmessages

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	operation            = "staged-message"
	MaxOperatorNoteBytes = 4000
	MaxSelectedContexts  = 20
	MaxDraftInputBytes   = 16 << 20
	MaxCollectionBytes   = 16 << 20
)

var (
	ErrConflict         = errors.New("staged message conflict")
	ErrNotFound         = errors.New("staged message not found")
	ErrProviderNotReady = errors.New("runner provider readiness is not confirmed")
)

type Service struct {
	store          *reviewjobs.Store
	snapshots      Snapshotter
	messages       MessageReader
	provider       Provider
	continuations  ContinuationFinder
	now            func() time.Time
	generateID     func(string) (string, error)
	settleMu       sync.Mutex
	settleCursor   string
	beforeDispatch func(context.Context, reviewjobs.Job)
}

func New(store *reviewjobs.Store, snapshots Snapshotter, messages MessageReader, provider Provider) *Service {
	return &Service{
		store: store, snapshots: snapshots, messages: messages, provider: provider,
		now: time.Now, generateID: func(prefix string) (string, error) { return newID(prefix, rand.Reader) },
	}
}

// SetContinuations enables continuation suggestions for collections and
// succeeded drafts. Suggestions are advisory; a lookup failure never blocks.
func (s *Service) SetContinuations(finder ContinuationFinder) { s.continuations = finder }

func newID(prefix string, source io.Reader) (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(source, b[:]); err != nil {
		return "", fmt.Errorf("generate %s identifier: %w", prefix, err)
	}
	return prefix + hex.EncodeToString(b[:]), nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func key(repo, request string) string { return operation + ":" + repo + ":" + request }

func (s *Service) Check(ctx context.Context, repo, dir, want string) (*v1.CheckStagedMessageResponse, error) {
	got, e := s.snapshots.Identity(ctx, repo, dir)
	if e != nil {
		return nil, e
	}
	r := &v1.CheckStagedMessageResponse{Applicable: got == want, CurrentSnapshotDigest: got}
	if !r.Applicable {
		r.Reason = "Staged selection changed; existing text and suggestions are retained"
	}
	return r, nil
}

func (s *Service) Collect(ctx context.Context, repo, dir string) (*v1.StagedMessageCollection, error) {
	captured, e := s.snapshots.Capture(ctx, repo, dir)
	if e != nil {
		return nil, e
	}
	collectionID, e := s.generateID("collection-")
	if e != nil {
		return nil, e
	}
	c := &v1.StagedMessageCollection{Id: collectionID, Snapshot: captured.Snapshot, Status: "no_suggestions", CollectedAt: s.now().UTC().Format(time.RFC3339Nano)}
	for _, f := range c.Snapshot.Files {
		if f.OmissionReason != "" {
			c.Omissions = append(c.Omissions, &v1.Omission{Path: f.Path, Reason: f.OmissionReason})
		}
	}
	if len(c.Snapshot.Files) == 0 {
		c.Status = "empty"
	} else {
		var messages []RetainedMessage
		if s.messages != nil {
			var omissions []*v1.Omission
			messages, omissions, e = s.messages.Messages(ctx, repo, c.Snapshot)
			c.Omissions = append(c.Omissions, omissions...)
			if e != nil {
				c.Omissions = append(c.Omissions, &v1.Omission{Reason: "retained messages unavailable: " + e.Error()})
			}
		}
		var reusable []RetainedMessage
		union := map[string]map[Atom]bool{}
		for _, m := range messages {
			candidate, covered, ready := matchRetained(captured, m)
			if candidate.MatchedEdits > 0 {
				c.Contexts = append(c.Contexts, candidate)
			}
			if !ready {
				continue
			}
			reusable = append(reusable, m)
			c.Suggestions = append(c.Suggestions, suggestion(captured, m, covered))
			for p, atoms := range covered {
				if union[p] == nil {
					union[p] = map[Atom]bool{}
				}
				for a := range atoms {
					union[p][a] = true
				}
			}
		}
		if len(reusable) > 1 {
			c.Suggestions = append([]*v1.StagedMessageSuggestion{combineSuggestions(captured, reusable, union)}, c.Suggestions...)
		}
		if s.provider != nil {
			contexts, omissions, err := s.provider.Search(ctx, c.Snapshot)
			c.Omissions = append(c.Omissions, omissions...)
			if err != nil {
				c.Omissions = append(c.Omissions, &v1.Omission{Reason: "Agent Manager context unavailable: " + err.Error()})
			} else {
				for _, candidate := range contexts {
					if candidate != nil {
						c.Contexts = append(c.Contexts, candidate)
					}
				}
			}
		} else {
			c.Omissions = append(c.Omissions, &v1.Omission{Reason: "Agent Manager context unavailable"})
		}
		if s.continuations != nil {
			// Advisory only: a history read failure (e.g. no HEAD yet) is not a
			// coverage omission and must not mark the collection partial.
			c.Continuations, _ = s.continuations.Continuations(ctx, dir, c.Snapshot)
		}
		if len(c.Contexts) > 100 {
			c.Contexts = c.Contexts[:100]
			c.Omissions = append(c.Omissions, &v1.Omission{Reason: "additional contexts omitted by 100-source bound"})
		}
		if len(c.Suggestions) > 10 {
			c.Suggestions = c.Suggestions[:10]
			c.Omissions = append(c.Omissions, &v1.Omission{Reason: "additional reusable suggestions omitted by 10-suggestion bound"})
		}
		for _, source := range c.Contexts {
			if len(source.MatchedPaths) > 64 {
				count := len(source.MatchedPaths)
				source.MatchedPaths = source.MatchedPaths[:64]
				source.Unknowns = append(source.Unknowns, fmt.Sprintf("matched-path examples limited to 64 of %d; full staged identity and suggestion coverage are retained", count))
			}
			if len(source.Body) > 8192 {
				source.Body = boundedText(source.Body, 8192)
				source.Unknowns = append(source.Unknowns, "source excerpt truncated to 8192 bytes; revision reference retained")
			}
			if len(source.Title) > 240 {
				source.Title = boundedText(source.Title, 240)
			}
		}
		if len(c.Suggestions) > 0 {
			c.Status = "ready"
			if len(c.Omissions) > 0 {
				c.Status = "partial"
			}
		}
	}
	encoded, e := encodeCollection(c)
	if e != nil {
		return nil, e
	}
	_, _, e = s.store.Create(ctx, reviewjobs.Job{ID: c.Id, IdempotencyKey: c.Id, InputDigest: c.Snapshot.Digest, Operation: "staged-message-collection", RepositoryID: repo, State: reviewjobs.Succeeded, PolicyVersion: "staged-message-v1", ResultJSON: string(encoded)})
	if e == nil {
		e = s.store.PruneAdvisoryCollections(ctx, repo, 20)
	}
	return c, e
}

// StartWorktree captures the selected index-to-worktree changes and immediately
// admits an independent durable job whose input contains that immutable manifest.
func (s *Service) StartWorktree(ctx context.Context, repo, dir string, req *v1.StartWorktreeStagedMessageRequest) (*v1.StagedMessageJob, error) {
	if req == nil || strings.TrimSpace(req.RequestKey) == "" || len(req.RequestKey) > 200 || len(req.OperatorNote) > MaxOperatorNoteBytes {
		return nil, fmt.Errorf("bounded request key and operator note are required")
	}
	if (req.ReplaceQueueEntryId == "") != (req.ExpectedQueueRevision == 0) {
		return nil, fmt.Errorf("queue replacement requires both an entry id and expected queue revision")
	}
	mode := normalizeMode(req.Mode)
	if mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK && mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		return nil, fmt.Errorf("unsupported staged message mode")
	}
	selected, err := normalizeWorktreePaths(req.Paths)
	if err != nil {
		return nil, err
	}
	idempotency := "worktree-staged-message:" + repo + ":" + req.RequestKey
	if existing, e := s.store.GetByIdempotency(ctx, idempotency); e == nil {
		old, decodeErr := decodeDraftInput([]byte(existing.InputJSON))
		if decodeErr != nil || existing.Operation != operation || existing.RepositoryID != repo || !old.WorktreeCapture || old.Snapshot == nil || old.OperatorNote != req.OperatorNote || normalizeInputMode(old) != mode || !sameWorktreePaths(old.Snapshot, selected) || old.ReplaceQueueEntryID != req.ReplaceQueueEntryId || old.ExpectedQueueRevision != req.ExpectedQueueRevision {
			return nil, fmt.Errorf("%w: request key reused for different worktree input", ErrConflict)
		}
		if req.ReplaceQueueEntryId != "" {
			replacement, replaceErr := s.queueEntryForJob(ctx, repo, existing.ID, old.Snapshot.Digest, selected, mode, req.RequestKey)
			if replaceErr != nil {
				return nil, replaceErr
			}
			if _, _, replaceErr = s.store.ReplaceWorktreeQueueEntry(ctx, repo, req.ReplaceQueueEntryId, req.ExpectedQueueRevision, replacement); replaceErr != nil {
				return nil, replaceErr
			}
		}
		return s.advanceRead(ctx, repo, dir, existing, "get", 0)
	} else if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	captured, err := captureWorktreeSnapshot(ctx, repo, dir, selected, s.now)
	if err != nil {
		return nil, err
	}
	currentDigest, digestErr := queuedSnapshotDigest(captured.Snapshot)
	if digestErr != nil {
		return nil, digestErr
	}
	if req.ExpectedSnapshotDigest != "" && currentDigest != req.ExpectedSnapshotDigest {
		return nil, fmt.Errorf("%w: selected worktree content differs from queued snapshot", ErrConflict)
	}
	if len(captured.Snapshot.Files) == 0 {
		return nil, fmt.Errorf("selected worktree snapshot is empty")
	}
	input := DraftInput{FormatVersion: 2, Mode: mode, Snapshot: captured.Snapshot, OperatorNote: req.OperatorNote, WorktreeCapture: true, CapturedWorktree: captured.Bytes, ReplaceQueueEntryID: req.ReplaceQueueEntryId, ExpectedQueueRevision: req.ExpectedQueueRevision}
	jobID, err := s.generateID("message-")
	if err != nil {
		return nil, err
	}
	if mode == v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		input.FormatVersion = 4
		input.JobID = jobID
	}
	input.Contexts, input.Omissions = s.searchDraftContexts(ctx, captured.Snapshot)
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, fmt.Errorf("immutable worktree manifest exceeds %d bytes", MaxDraftInputBytes)
	}
	// Recompute the selected path identities after context lookup and encoding.
	standing, err := WorktreeSelectionApplicable(ctx, dir, captured.Snapshot)
	if err != nil {
		return nil, err
	}
	if !standing {
		return nil, fmt.Errorf("%w: selected worktree paths changed during capture", ErrConflict)
	}
	queueID, err := s.generateID("worktree-queue-")
	if err != nil {
		return nil, err
	}
	jobSpec := reviewjobs.Job{ID: jobID, IdempotencyKey: idempotency, InputDigest: digest(raw), InputJSON: string(raw), Operation: operation, RepositoryID: repo, PolicyVersion: "staged-message-worktree-v1", State: reviewjobs.Queued}
	queueSpec := reviewjobs.WorktreeQueueEntry{ID: queueID, RepositoryID: repo, JobID: jobID, SnapshotDigest: input.Snapshot.Digest, Paths: selected, Mode: int32(mode), RequestKey: req.RequestKey}
	var job reviewjobs.Job
	if req.ReplaceQueueEntryId != "" {
		job, _, _, err = s.store.CreateWorktreeJobAndReplaceQueueEntry(ctx, jobSpec, req.ReplaceQueueEntryId, req.ExpectedQueueRevision, queueSpec)
	} else {
		job, _, _, err = s.store.CreateWorktreeJobAndEnqueue(ctx, jobSpec, queueSpec)
	}
	if err != nil {
		return nil, err
	}
	if job.InputDigest != digest(raw) {
		return nil, fmt.Errorf("%w: request key reused for different worktree input", ErrConflict)
	}
	return s.advanceRead(ctx, repo, dir, job, "get", 0)
}

// Preserve complete UTF-8 characters when limiting already-valid source text.
func boundedText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit]
}

// encodeCollection budgets actual wire JSON, whose escaping differs from
// protobuf size. Only redundant explanations are reduced; the staged manifest
// and any retained suggestion's complete coverage are never silently truncated.
func encodeCollection(c *v1.StagedMessageCollection) ([]byte, error) {
	manifest, err := protojson.Marshal(c.Snapshot)
	if err != nil {
		return nil, err
	}
	if len(manifest) > MaxCollectionBytes {
		return nil, fmt.Errorf("canonical staged manifest exceeds %d-byte identity bound", MaxCollectionBytes)
	}
	encode := func() ([]byte, error) { return protojson.Marshal(c) }
	data, err := encode()
	if err != nil || len(data) <= MaxCollectionBytes {
		return data, err
	}
	// Per-file omissions already live on every canonical manifest entry. The
	// collection list can summarize that duplicate explanation without losing it.
	manifestPaths := map[string]bool{}
	for _, file := range c.Snapshot.Files {
		manifestPaths[file.Path] = true
	}
	compactOmissions := func(input []*v1.Omission) []*v1.Omission {
		counts := map[string]int{}
		var omissions []*v1.Omission
		for _, omission := range input {
			if !manifestPaths[omission.Path] {
				omissions = append(omissions, omission)
			} else {
				counts[omission.Reason]++
			}
		}
		var reasons []string
		for reason := range counts {
			reasons = append(reasons, reason)
		}
		sort.Strings(reasons)
		for _, reason := range reasons {
			omissions = append(omissions, &v1.Omission{Reason: fmt.Sprintf("%d files: %s; exact paths remain in staged manifest", counts[reason], reason)})
		}
		return omissions
	}
	c.Omissions = compactOmissions(c.Omissions)
	for _, suggestion := range c.Suggestions {
		if suggestion.Coverage != nil {
			suggestion.Coverage.Omissions = compactOmissions(suggestion.Coverage.Omissions)
		}
	}
	if data, err = encode(); err != nil || len(data) <= MaxCollectionBytes {
		return data, err
	}
	if len(c.Suggestions) > 1 {
		c.Suggestions = c.Suggestions[:1]
		c.Omissions = append(c.Omissions, &v1.Omission{Reason: "additional alternative suggestions omitted by collection byte bound; full staged manifest retained"})
		c.Status = "partial"
	}
	if data, err = encode(); err != nil || len(data) <= MaxCollectionBytes {
		return data, err
	}
	for _, source := range c.Contexts {
		source.MatchedPaths = nil
		source.Body = ""
		source.Unknowns = append(source.Unknowns, "source excerpt and matched-path examples omitted by collection byte bound; source revision retained")
	}
	if data, err = encode(); err != nil || len(data) <= MaxCollectionBytes {
		return data, err
	}
	if len(c.Suggestions) > 0 {
		c.Suggestions = nil
		c.Status = "no_suggestions"
		c.Omissions = append(c.Omissions, &v1.Omission{Reason: "retained suggestion omitted because complete coverage exceeds collection byte bound; full staged manifest retained"})
	}
	data, err = encode()
	if err != nil {
		return nil, err
	}
	if len(data) > MaxCollectionBytes {
		return nil, fmt.Errorf("canonical staged manifest plus required collection metadata exceeds %d-byte bound", MaxCollectionBytes)
	}
	return data, nil
}

func (s *Service) Start(ctx context.Context, repo, dir string, req *v1.StartStagedMessageRequest) (*v1.StagedMessageJob, error) {
	if req == nil || strings.TrimSpace(req.RequestKey) == "" || len(req.RequestKey) > 200 || len(req.OperatorNote) > MaxOperatorNoteBytes || len(req.SelectedContextIds) > MaxSelectedContexts {
		return nil, fmt.Errorf("bounded request key and operator note are required")
	}
	if req.ContinuationOfJobId != "" {
		if req.Mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_UNSPECIFIED && req.Mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
			return nil, ErrConflict
		}
		return s.continueDraft(ctx, repo, dir, req)
	}
	mode := normalizeMode(req.Mode)
	if mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK && mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		return nil, fmt.Errorf("unsupported staged message mode")
	}
	collection, e := s.store.Get(ctx, req.CollectionId)
	if e != nil {
		return nil, e
	}
	if collection.Operation != "staged-message-collection" || collection.RepositoryID != repo {
		return nil, ErrNotFound
	}
	var c v1.StagedMessageCollection
	if e = protojson.Unmarshal([]byte(collection.ResultJSON), &c); e != nil {
		return nil, e
	}
	if c.Snapshot == nil || req.SnapshotDigest != c.Snapshot.Digest || len(c.Snapshot.Files) == 0 {
		return nil, fmt.Errorf("%w: collection subject is absent or different", ErrConflict)
	}
	available := map[string]*v1.StagedMessageContext{}
	for _, source := range c.Contexts {
		available[source.Id] = source
	}
	selected := append([]string(nil), req.SelectedContextIds...)
	sort.Strings(selected)
	input := DraftInput{FormatVersion: 2, Mode: mode, Snapshot: proto.Clone(c.Snapshot).(*v1.StagedMessageSnapshot), OperatorNote: req.OperatorNote}
	jobID, e := s.generateID("message-")
	if e != nil {
		return nil, e
	}
	if mode == v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		input.FormatVersion = 4
		input.JobID = jobID
	}
	seen := map[string]bool{}
	for _, ref := range selected {
		source, ok := available[ref]
		if !ok || seen[ref] {
			return nil, fmt.Errorf("selected context is absent or duplicated")
		}
		seen[ref] = true
		input.Contexts = append(input.Contexts, source)
	}
	if existing, e := s.store.GetByIdempotency(ctx, key(repo, req.RequestKey)); e == nil {
		if !sameAdmissionIntent(existing, repo, req.CollectionId, input) {
			return nil, fmt.Errorf("%w: request key reused for different input", ErrConflict)
		}
		return s.advanceRead(ctx, repo, dir, existing, "get", 0)
	} else if !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	raw, e := json.Marshal(input)
	if e != nil {
		return nil, e
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, fmt.Errorf("immutable manifest and selected context exceed %d bytes", MaxDraftInputBytes)
	}
	inputDigest := digest(raw)
	standing, e := s.Check(ctx, repo, dir, req.SnapshotDigest)
	if e != nil {
		return nil, e
	}
	if !standing.Applicable {
		return nil, fmt.Errorf("%w: %s", ErrConflict, standing.Reason)
	}
	job, _, e := s.store.Create(ctx, reviewjobs.Job{ID: jobID, IdempotencyKey: key(repo, req.RequestKey), InputDigest: inputDigest, InputJSON: string(raw), ResultRef: req.CollectionId, Operation: operation, RepositoryID: repo, PolicyVersion: "staged-message-v1", State: reviewjobs.Queued})
	if e != nil {
		return nil, e
	}
	if job.InputDigest != inputDigest && !sameAdmissionIntent(job, repo, req.CollectionId, input) {
		return nil, ErrConflict
	}
	return s.advanceRead(ctx, repo, dir, job, "get", 0)
}

// Retry starts a new owner job from a terminal job's saved immutable input.
// It allows recovery when the browser no longer has the original collection
// and admission envelope, while preserving the exact staged subject and context.
func (s *Service) Retry(ctx context.Context, repo, dir string, req *v1.RetryStagedMessageRequest) (*v1.StagedMessageJob, error) {
	if req == nil || strings.TrimSpace(req.JobId) == "" || strings.TrimSpace(req.SnapshotDigest) == "" || strings.TrimSpace(req.RequestKey) == "" || len(req.RequestKey) > 200 {
		return nil, fmt.Errorf("job id, snapshot digest, and bounded request key are required")
	}
	mode := normalizeMode(req.Mode)
	if mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK && mode != v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		return nil, fmt.Errorf("unsupported staged message mode")
	}
	previous, err := s.store.Get(ctx, req.JobId)
	if err != nil {
		return nil, err
	}
	if previous.Operation != operation || previous.RepositoryID != repo {
		return nil, ErrNotFound
	}
	if previous.State != reviewjobs.Failed && previous.State != reviewjobs.Cancelled && previous.State != reviewjobs.Interrupted && previous.State != reviewjobs.AwaitingProvider {
		return nil, fmt.Errorf("only a failed or provider-paused draft can be retried: %w", ErrConflict)
	}
	input, err := decodeDraftInput([]byte(previous.InputJSON))
	if err != nil {
		return nil, fmt.Errorf("decode saved draft for retry: %w", err)
	}
	if input.Snapshot == nil || input.Snapshot.Digest != req.SnapshotDigest || input.WorktreeCapture || input.QueuedSelection {
		return nil, fmt.Errorf("saved draft does not match the requested staged subject: %w", ErrConflict)
	}
	if normalizeInputMode(input) != mode {
		return nil, fmt.Errorf("retry mode differs from the saved draft: %w", ErrConflict)
	}
	idempotencyKey := key(repo, req.RequestKey)
	if existing, lookupErr := s.store.GetByIdempotency(ctx, idempotencyKey); lookupErr == nil {
		existingInput, decodeErr := decodeDraftInput([]byte(existing.InputJSON))
		if decodeErr != nil || existing.Operation != operation || existing.RepositoryID != repo || !sameRetryInput(existingInput, input) {
			return nil, fmt.Errorf("%w: request key reused for a different saved draft", ErrConflict)
		}
		if existing.ID != previous.ID || !canResumeUnconfirmedDispatch(previous) {
			return s.advanceRead(ctx, repo, dir, existing, "get", 0)
		}
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		return nil, lookupErr
	}
	if previous.ErrorCode == "provider_admission_maintenance" {
		readiness := s.ProviderReadiness(ctx)
		if !readiness.Known || readiness.State != "open" {
			return nil, fmt.Errorf("%w: %s", ErrProviderNotReady, readiness.Message)
		}
	}
	standing, err := s.Check(ctx, repo, dir, req.SnapshotDigest)
	if err != nil {
		return nil, err
	}
	if !standing.Applicable {
		return nil, fmt.Errorf("%w: %s", ErrConflict, standing.Reason)
	}
	if canResumeUnconfirmedDispatch(previous) {
		resumed, changed, resumeErr := s.store.RequeueUnconfirmedDispatch(ctx, previous)
		if resumeErr != nil {
			return nil, resumeErr
		}
		if changed {
			return s.advanceRead(ctx, repo, dir, resumed, "get", 0)
		}
	}
	jobID, err := s.generateID("message-")
	if err != nil {
		return nil, err
	}
	input.JobID = ""
	if mode == v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP {
		input.FormatVersion = 4
		input.JobID = jobID
	}
	input.Mode = mode
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxDraftInputBytes {
		return nil, fmt.Errorf("saved retry input exceeds %d bytes", MaxDraftInputBytes)
	}
	retried, created, err := s.store.Create(ctx, reviewjobs.Job{
		ID: jobID, IdempotencyKey: idempotencyKey, InputDigest: digest(raw), InputJSON: string(raw),
		ResultRef: previous.ResultRef, Operation: operation, RepositoryID: repo,
		PolicyVersion: previous.PolicyVersion, State: reviewjobs.Queued,
	})
	if err != nil {
		return nil, err
	}
	if !created {
		existingInput, decodeErr := decodeDraftInput([]byte(retried.InputJSON))
		if decodeErr != nil || !sameRetryInput(existingInput, input) {
			return nil, fmt.Errorf("%w: request key raced with a different saved draft", ErrConflict)
		}
	}
	return s.advanceRead(ctx, repo, dir, retried, "get", 0)
}

func (s *Service) ProviderReadiness(ctx context.Context) ProviderReadiness {
	provider, ok := s.provider.(ReadinessProvider)
	if !ok {
		return ProviderReadiness{Provider: "workspace-sandbox", State: "unknown", Known: false, Message: "Workspace Sandbox readiness could not be confirmed."}
	}
	readiness := provider.Readiness(ctx)
	if readiness.Provider != "workspace-sandbox" || !readiness.Known || (readiness.State != "open" && readiness.State != "closed") || readiness.ObservedAt == nil || readiness.ObservedAt.IsZero() {
		return ProviderReadiness{Provider: "workspace-sandbox", State: "unknown", Known: false, Message: "Workspace Sandbox readiness could not be confirmed."}
	}
	if age := time.Since(*readiness.ObservedAt); age < -time.Minute || age > 30*time.Second {
		return ProviderReadiness{Provider: "workspace-sandbox", State: "unknown", Known: false, Message: "Workspace Sandbox readiness could not be confirmed."}
	}
	return readiness
}

func sameRetryInput(left, right DraftInput) bool {
	left.JobID, right.JobID = "", ""
	return reflect.DeepEqual(left, right)
}

func canResumeUnconfirmedDispatch(job reviewjobs.Job) bool {
	return job.State == reviewjobs.Failed && job.DispatchAttempted && job.ExecutionID == "" &&
		(job.ErrorCode == "provider_precondition_failed" || job.ErrorCode == "dispatch_unconfirmed")
}

// Idempotency binds operator intent to immutable collection/source revisions,
// not to a packaging version or the historical explanatory patch byte budget.
func sameAdmissionIntent(job reviewjobs.Job, repo, collection string, input DraftInput) bool {
	if job.Operation != operation || job.RepositoryID != repo || job.ResultRef != collection {
		return false
	}
	old, decodeErr := decodeDraftInput([]byte(job.InputJSON))
	if decodeErr != nil || old.Snapshot == nil || input.Snapshot == nil || old.Snapshot.Digest != input.Snapshot.Digest || old.OperatorNote != input.OperatorNote || old.ContinuationOfJobID != input.ContinuationOfJobID || normalizeInputMode(old) != normalizeInputMode(input) || len(old.Contexts) != len(input.Contexts) {
		return false
	}
	refs := map[string]string{}
	for _, source := range old.Contexts {
		if source == nil {
			return false
		}
		if _, duplicate := refs[source.Id]; duplicate {
			return false
		}
		refs[source.Id] = source.Revision
	}
	for _, source := range input.Contexts {
		if source == nil {
			return false
		}
		revision, exists := refs[source.Id]
		if !exists || revision != source.Revision {
			return false
		}
	}
	return true
}

func normalizeMode(mode v1.StagedMessageMode) v1.StagedMessageMode {
	if mode == v1.StagedMessageMode_STAGED_MESSAGE_MODE_UNSPECIFIED {
		return v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK
	}
	return mode
}

func normalizeInputMode(input DraftInput) v1.StagedMessageMode {
	if input.Mode == v1.StagedMessageMode_STAGED_MESSAGE_MODE_UNSPECIFIED && input.supportsSynthesis() {
		// Format 4 was continuation-only before direct deep admission. Retain
		// that meaning for durable jobs written before the mode field existed.
		return v1.StagedMessageMode_STAGED_MESSAGE_MODE_DEEP
	}
	return normalizeMode(input.Mode)
}

func validateResult(input DraftInput, r *v1.StagedMessageSuggestion) error {
	if r == nil || strings.TrimSpace(r.Title) == "" || strings.ContainsAny(r.Title, "\r\n") || len(r.Title) > 240 || len(r.Body) > 32000 || len(r.Trailers) > 30 || r.Coverage == nil {
		return fmt.Errorf("invalid structured title/body/trailers/coverage result")
	}
	semantics := input.semantics()
	if semantics.synthesis {
		if r.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION && r.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_LIMITED {
			return fmt.Errorf("synthesis must declare whole-selection or limited message scope")
		}
		if strings.TrimSpace(r.MessageScopeExplanation) == "" || len(r.MessageScopeExplanation) > 1000 {
			return fmt.Errorf("synthesis requires a bounded scope and investigation explanation")
		}
		if r.MessageScope == v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION {
			for i := range input.Snapshot.Files {
				if !input.InventoryEligibility[i+1] {
					return fmt.Errorf("whole-selection synthesis requires the complete immutable inventory")
				}
			}
		}
	} else if r.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_UNSPECIFIED || r.MessageScopeExplanation != "" {
		return fmt.Errorf("legacy drafting cannot reinterpret message scope")
	}
	allowed := map[string]bool{}
	for _, c := range input.Contexts {
		allowed[c.Id] = true
	}
	for _, id := range r.SourceIds {
		if !allowed[id] {
			return fmt.Errorf("result references unselected context")
		}
	}
	files := map[string]*v1.StagedMessageFile{}
	deepEligible := priorCovered(input)
	total := int32(0)
	for i, f := range input.Snapshot.Files {
		files[f.Path] = f
		if input.EvidenceEligibility[i+1] {
			deepEligible[f.Path] = true
		}
		n := f.EditCount
		if n < 1 {
			n = 1
		}
		total += n
	}
	seen := map[string]bool{}
	coveredEdits := int32(0)
	for _, p := range r.Coverage.CoveredPaths {
		f, ok := files[p]
		if !ok || seen[p] || (semantics.deepEvidence && !deepEligible[p]) || (semantics.legacyDraft && f.OmissionReason != "") {
			return fmt.Errorf("invalid or duplicated covered file")
		}
		seen[p] = true
		n := f.EditCount
		if n < 1 {
			n = 1
		}
		coveredEdits += n
	}
	for _, p := range r.Coverage.UncoveredPaths {
		if _, ok := files[p]; !ok || seen[p] {
			return fmt.Errorf("invalid or duplicated uncovered file")
		}
		seen[p] = true
	}
	if len(seen) != len(files) || r.Coverage.TotalEdits != total || r.Coverage.MatchedEdits < coveredEdits || r.Coverage.MatchedEdits > total || r.Coverage.MatchedEdits < 0 || r.Coverage.Partial != (len(r.Coverage.UncoveredPaths) > 0) {
		return fmt.Errorf("incomplete or inconsistent structured coverage")
	}
	for _, t := range r.Trailers {
		if t == nil || len(t.Key) < 1 || len(t.Key) > 64 || len(t.Value) > 2048 || strings.TrimSpace(t.Value) == "" || strings.ContainsAny(t.Key+t.Value, "\r\n") || strings.EqualFold(t.Key, "Co-Authored-By") || strings.HasPrefix(strings.ToLower(t.Key), "vrooli-") {
			return fmt.Errorf("invalid or inferred attribution trailer")
		}
		for _, r := range t.Key {
			if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') {
				return fmt.Errorf("invalid trailer key")
			}
		}
	}
	return nil
}

// ConstrainSynthesisScope downgrades an unsupported whole-selection claim to
// an explicit limited draft. Missing inventory receipts reduce confidence in
// the scope; they do not make an otherwise valid title/body unusable.
func ConstrainSynthesisScope(input DraftInput, result *v1.StagedMessageSuggestion) {
	if !input.supportsSynthesis() || input.Snapshot == nil || result == nil || result.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION {
		return
	}
	for i := range input.Snapshot.Files {
		if !input.InventoryEligibility[i+1] {
			result.MessageScope = v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_LIMITED
			result.MessageScopeExplanation = "The workflow did not provide verifiable evidence of the complete immutable inventory. Treat this as a limited draft and investigate unresolved themes before relying on it."
			return
		}
	}
}
