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
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"connectrpc.com/connect"
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
	ErrConflict = errors.New("staged message conflict")
	ErrNotFound = errors.New("staged message not found")
)

type Service struct {
	store     *reviewjobs.Store
	snapshots Snapshotter
	messages  MessageReader
	provider  Provider
}

func New(store *reviewjobs.Store, snapshots Snapshotter, messages MessageReader, provider Provider) *Service {
	return &Service{store, snapshots, messages, provider}
}

func id(prefix string) string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return prefix + hex.EncodeToString(b[:])
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
	c := &v1.StagedMessageCollection{Id: id("collection-"), Snapshot: captured.Snapshot, Status: "no_suggestions", CollectedAt: time.Now().UTC().Format(time.RFC3339Nano)}
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
		return s.continueDraft(ctx, repo, dir, req)
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
	input := DraftInput{FormatVersion: 2, Snapshot: proto.Clone(c.Snapshot).(*v1.StagedMessageSnapshot), OperatorNote: req.OperatorNote}
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
		return s.read(ctx, repo, dir, existing, "get", 0)
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
	job, _, e := s.store.Create(ctx, reviewjobs.Job{ID: id("message-"), IdempotencyKey: key(repo, req.RequestKey), InputDigest: inputDigest, InputJSON: string(raw), ResultRef: req.CollectionId, Operation: operation, RepositoryID: repo, PolicyVersion: "staged-message-v1", State: reviewjobs.Queued})
	if e != nil {
		return nil, e
	}
	if job.InputDigest != inputDigest && !sameAdmissionIntent(job, repo, req.CollectionId, input) {
		return nil, ErrConflict
	}
	return s.read(ctx, repo, dir, job, "get", 0)
}

// Idempotency binds operator intent to immutable collection/source revisions,
// not to a packaging version or the historical explanatory patch byte budget.
func sameAdmissionIntent(job reviewjobs.Job, repo, collection string, input DraftInput) bool {
	if job.Operation != operation || job.RepositoryID != repo || job.ResultRef != collection {
		return false
	}
	var old DraftInput
	if json.Unmarshal([]byte(job.InputJSON), &old) != nil || old.Snapshot == nil || input.Snapshot == nil || old.Snapshot.Digest != input.Snapshot.Digest || old.OperatorNote != input.OperatorNote || len(old.Contexts) != len(input.Contexts) {
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

func (s *Service) Job(ctx context.Context, repo, dir, jobID, requestKey, action string, timeout time.Duration) (*v1.StagedMessageJob, error) {
	var job reviewjobs.Job
	var e error
	if jobID != "" {
		job, e = s.store.Get(ctx, jobID)
	} else if requestKey != "" {
		job, e = s.store.GetByIdempotency(ctx, key(repo, requestKey))
	} else {
		return nil, fmt.Errorf("job id or request key is required")
	}
	if e != nil {
		return nil, e
	}
	if job.Operation != operation || job.RepositoryID != repo {
		return nil, ErrNotFound
	}
	return s.read(ctx, repo, dir, job, action, timeout)
}

func terminal(state reviewjobs.State) bool {
	return state == reviewjobs.Succeeded || state == reviewjobs.Failed || state == reviewjobs.Cancelled
}

func (s *Service) read(ctx context.Context, repo, dir string, job reviewjobs.Job, action string, timeout time.Duration) (*v1.StagedMessageJob, error) {
	var input DraftInput
	if e := json.Unmarshal([]byte(job.InputJSON), &input); e != nil {
		return nil, e
	}
	reconciled, e := s.reconcile(ctx, repo, job, input, action, timeout)
	if e != nil {
		return nil, e
	}
	job = reconciled
	result := &v1.StagedMessageJob{Id: job.ID, RepositoryId: repo, SnapshotDigest: input.Snapshot.GetDigest(), Status: string(job.State), CreatedAt: job.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: job.UpdatedAt.UTC().Format(time.RFC3339Nano), ExecutionId: job.ExecutionID, CancelRequested: job.CancelRequested, Error: job.Error, Snapshot: input.Snapshot, Phase: string(job.State)}
	result.ContinuationOfJobId = input.ContinuationOfJobID
	if input.FormatVersion >= 3 {
		result.EvidenceFilesRead = uint32(len(suppliedEvidence(job.Checks, input.Snapshot.Digest, job.ExecutionID)))
		result.EvidenceFilesTotal = uint32(len(input.Snapshot.Files) - len(priorCovered(input)))
		if input.FormatVersion == 4 {
			result.EvidenceFilesTotal = uint32(len(input.Snapshot.Files))
		}
	}
	if job.ResultJSON != "" {
		result.Result = &v1.StagedMessageSuggestion{}
		if e := protojson.Unmarshal([]byte(job.ResultJSON), result.Result); e != nil {
			return nil, e
		}
	}
	standing, e := s.Check(ctx, repo, dir, result.SnapshotDigest)
	if e != nil {
		result.ApplicabilityReason = "Current staged selection unavailable: " + e.Error()
	} else {
		result.Applicable = standing.Applicable
		result.ApplicabilityReason = standing.Reason
	}
	return result, nil
}

// reconcile owns exactly one persisted/provider lifecycle step. Projection and
// applicability checks run once in read, even after a CAS or cancelled admission.
func (s *Service) reconcile(ctx context.Context, repo string, job reviewjobs.Job, input DraftInput, action string, timeout time.Duration) (reviewjobs.Job, error) {
	if !terminal(job.State) {
		if action == "cancel" && !job.CancelRequested {
			if e := s.store.RequestAdvisoryCancel(ctx, job.ID, repo); e != nil {
				return reviewjobs.Job{}, e
			}
			var e error
			job, e = s.store.Get(ctx, job.ID)
			if e != nil {
				return reviewjobs.Job{}, e
			}
		}
		if terminal(job.State) {
			return job, nil
		}
		if s.provider == nil {
			next := job
			if job.State == reviewjobs.Queued && job.ExecutionID == "" {
				next.State = reviewjobs.Failed
			}
			next.Error = "Agent Manager message execution unavailable"
			if _, e := s.store.UpdateAdvisory(ctx, job, next); e != nil {
				return reviewjobs.Job{}, e
			}
			saved, e := s.store.Get(ctx, job.ID)
			if e != nil {
				return reviewjobs.Job{}, e
			}
			return saved, nil
		}
		attachedAt := time.Now()
		var ex Execution
		var err error
		firstDispatch := false
		if job.ExecutionID == "" {
			// Admission is durable before dispatch. A crash/lost acknowledgement is
			// reconciled with the same owner idempotency key, never a fresh execution.
			firstDispatch = job.State == reviewjobs.Queued
			claimed, changed, e := s.store.ClaimAdvisoryDispatch(ctx, job)
			if e != nil {
				return reviewjobs.Job{}, e
			}
			current, e := s.store.Get(ctx, job.ID)
			if e != nil {
				return reviewjobs.Job{}, e
			}
			if !changed || terminal(current.State) || current.ExecutionID != "" || !current.UpdatedAt.Equal(claimed.UpdatedAt) {
				return current, nil
			}
			// Retain this exact claim for the response CAS; a reload must never
			// lend a later retry's revision to this older owner response.
			job = claimed
			ex, err = s.provider.Start(ctx, input, job.IdempotencyKey)
		} else if job.CancelRequested {
			ex, err = s.provider.Cancel(ctx, job.ExecutionID, input)
		} else if action == "wait" {
			ex, err = s.provider.Wait(ctx, job.ExecutionID, input, timeout)
		} else {
			ex, err = s.provider.Get(ctx, job.ExecutionID, input)
		}
		if err == nil && job.ExecutionID != "" && ex.ID != "" && ex.ID != job.ExecutionID {
			err = fmt.Errorf("execution owner returned a different handle; original execution retained")
		}
		if err != nil {
			next := job
			next.Error = err.Error()
			switch connect.CodeOf(err) {
			case connect.CodeInvalidArgument, connect.CodePermissionDenied, connect.CodeUnauthenticated, connect.CodeFailedPrecondition, connect.CodeUnimplemented:
				if firstDispatch && job.ExecutionID == "" {
					next.State = reviewjobs.Failed
				}
			}
			// Only the first dispatch can establish a definitive admission refusal.
			// Once Running is durable, a prior lost acknowledgement may conceal an
			// accepted owner execution. Later auth/outage errors retain its key so
			// restored credentials can recover that execution rather than replace it.
			if action == "wait" && !terminal(next.State) {
				if e := waitAttachment(ctx, attachedAt, timeout); e != nil {
					return reviewjobs.Job{}, e
				}
			}
			if _, e := s.store.UpdateAdvisory(ctx, job, next); e != nil {
				return reviewjobs.Job{}, e
			}
		} else {
			next := job
			next.ExecutionID = ex.ID
			if next.ExecutionID == "" {
				next.ExecutionID = job.ExecutionID
			}
			next.Error = ex.Error
			if next.ExecutionID == "" {
				next.State = reviewjobs.Failed
				next.Error = "execution owner returned no handle"
			} else {
				switch ex.Status {
				case "queued", "running", "cancelling", "succeeded", "failed", "cancelled":
					next.State = reviewjobs.State(ex.Status)
				default:
					next.State = reviewjobs.Failed
					next.Error = "invalid execution-owner state"
				}
				if job.CancelRequested {
					if ex.Status == "succeeded" {
						next.State = reviewjobs.Cancelled
						next.Error = "Execution completed after cancellation request; result discarded"
						next.ResultJSON = ""
					} else if ex.Status != "cancelled" && ex.Status != "failed" {
						next.State = reviewjobs.State("cancelling")
					}
				}
				if ex.Status == "succeeded" && !job.CancelRequested {
					if input.FormatVersion >= 3 {
						current, e := s.store.Get(ctx, job.ID)
						if e != nil {
							return reviewjobs.Job{}, e
						}
						input.EvidenceEligibility = suppliedEvidence(current.Checks, input.Snapshot.Digest, ex.ID)
						input.InventoryEligibility = suppliedInventory(current.Checks, input.Snapshot.Digest, ex.ID)
					}
					if e := validateResult(input, ex.Result); e != nil {
						next.State = reviewjobs.Failed
						next.Error = e.Error()
					} else {
						data, _ := protojson.Marshal(ex.Result)
						next.ResultJSON = string(data)
					}
				}
			}
			if _, e := s.store.UpdateAdvisory(ctx, job, next); e != nil {
				return reviewjobs.Job{}, e
			}
		}
		var e error
		job, e = s.store.Get(ctx, job.ID)
		if e != nil {
			return reviewjobs.Job{}, e
		}
	}
	return job, nil
}

func waitAttachment(ctx context.Context, started time.Time, timeout time.Duration) error {
	remaining := timeout - time.Since(started)
	if remaining <= 0 {
		return nil
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func validateResult(input DraftInput, r *v1.StagedMessageSuggestion) error {
	if r == nil || strings.TrimSpace(r.Title) == "" || strings.ContainsAny(r.Title, "\r\n") || len(r.Title) > 240 || len(r.Body) > 32000 || len(r.Trailers) > 30 || r.Coverage == nil {
		return fmt.Errorf("invalid structured title/body/trailers/coverage result")
	}
	if input.FormatVersion == 4 {
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
		if !ok || seen[p] || (input.FormatVersion >= 3 && !deepEligible[p]) || (input.FormatVersion < 3 && f.OmissionReason != "") {
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
