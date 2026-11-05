// [REQ:GCT-OT-P1-002]
package stagedmessages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

const legacyProviderInputBytes = 196608

type fakeSnapshot struct {
	mu       sync.Mutex
	identity string
	patch    string
}

func (f *fakeSnapshot) Identity(context.Context, string, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.identity, nil
}

func (f *fakeSnapshot) Capture(ctx context.Context, repo, dir string) (Captured, error) {
	identity, _ := f.Identity(ctx, repo, dir)
	return Captured{Snapshot: &v1.StagedMessageSnapshot{Digest: identity, RepositoryId: repo, Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1, Patch: f.patch}}}}, nil
}
func (f *fakeSnapshot) drift() { f.mu.Lock(); f.identity = "sha256:new"; f.mu.Unlock() }

type fakeProvider struct {
	contexts                  []*v1.StagedMessageContext
	lastInput                 DraftInput
	mu                        sync.Mutex
	starts, searches, cancels int
	status                    string
	result                    *v1.StagedMessageSuggestion
	entered, release          chan struct{}
}

func (f *fakeProvider) Search(context.Context, *v1.StagedMessageSnapshot) ([]*v1.StagedMessageContext, []*v1.Omission, error) {
	f.mu.Lock()
	f.searches++
	f.mu.Unlock()
	return f.contexts, nil, nil
}

func (f *fakeProvider) Start(_ context.Context, input DraftInput, _ string) (Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	f.lastInput = input
	return Execution{ID: "owner-execution", Status: f.status, Result: f.result}, nil
}

func (f *fakeProvider) Get(context.Context, string, DraftInput) (Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Execution{ID: "owner-execution", Status: f.status, Result: f.result}, nil
}

func (f *fakeProvider) Wait(ctx context.Context, id string, input DraftInput, _ time.Duration) (Execution, error) {
	if f.entered != nil {
		close(f.entered)
		select {
		case <-f.release:
		case <-ctx.Done():
			return Execution{}, ctx.Err()
		}
	}
	return f.Get(ctx, id, input)
}

func (f *fakeProvider) Cancel(context.Context, string, DraftInput) (Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancels++
	return Execution{ID: "owner-execution", Status: f.status, Result: f.result}, nil
}

func (f *fakeProvider) complete(status string, result *v1.StagedMessageSuggestion) {
	f.mu.Lock()
	f.status = status
	f.result = result
	f.mu.Unlock()
}

func serviceFixture(t *testing.T) (*Service, *reviewjobs.Store, *fakeSnapshot, *fakeProvider) {
	t.Helper()
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, e := reviewjobs.New(db)
	if e != nil {
		t.Fatal(e)
	}
	snap := &fakeSnapshot{identity: "sha256:original"}
	provider := &fakeProvider{status: "running"}
	return New(store, snap, nil, provider), store, snap, provider
}

func admission(t *testing.T, s *Service, key string) (*v1.StagedMessageCollection, *v1.StagedMessageJob) {
	t.Helper()
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: key})
	if e != nil {
		t.Fatal(e)
	}
	return c, j
}

func validResult() *v1.StagedMessageSuggestion {
	return &v1.StagedMessageSuggestion{Title: "Update selected behavior", Coverage: &v1.StagedMessageCoverage{CoveredPaths: []string{"file"}, MatchedEdits: 1, TotalEdits: 1}}
}

func TestCollectionNeverStartsAgentWhenNoReusableMessage(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	if c.Status != "no_suggestions" || len(c.Suggestions) != 0 || provider.starts != 0 || provider.searches != 1 {
		t.Fatalf("collection %#v starts=%d", c, provider.starts)
	}
}

func TestCollectionBoundsUnicodeContextWithoutSplittingCharacters(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	title := "A" + strings.Repeat("界", 100)
	body := strings.Repeat("界", 3000)
	provider.contexts = []*v1.StagedMessageContext{{Id: "unicode-source", Revision: "1", Title: title, Body: body}}
	c, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatalf("valid Unicode source must remain collectable: %v", err)
	}
	if len(c.Contexts) != 1 {
		t.Fatal("bounded Unicode context was dropped")
	}
	source := c.Contexts[0]
	if !utf8.ValidString(source.Title) || !utf8.ValidString(source.Body) || source.Title != title[:238] || source.Body != body[:8190] || len(source.Title) > 240 || len(source.Body) > 8192 {
		t.Fatalf("source clipping split characters or lost complete prefix: title bytes=%d body bytes=%d", len(source.Title), len(source.Body))
	}
	if _, err := protojson.Marshal(c); err != nil {
		t.Fatalf("bounded collection must encode as valid protobuf JSON: %v", err)
	}
	if provider.starts != 0 {
		t.Fatal("Unicode context collection started inference")
	}
}

func TestDurableAdmissionIsIdempotentAndReattachesByRequestKey(t *testing.T) {
	s, store, snap, p := serviceFixture(t)
	c, j := admission(t, s, "request")
	same, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "request"})
	if e != nil || same.Id != j.Id || p.starts != 1 {
		t.Fatalf("duplicate %#v err=%v starts=%d", same, e, p.starts)
	}
	reopened := New(store, snap, nil, p)
	got, e := reopened.Job(context.Background(), "repo", "fixture", "", "request", "get", 0)
	if e != nil || got.Id != j.Id || p.starts != 1 {
		t.Fatalf("reattach %#v err=%v starts=%d", got, e, p.starts)
	}
	_, e = reopened.Job(context.Background(), "other", "fixture", j.Id, "", "get", 0)
	if !errors.Is(e, ErrNotFound) {
		t.Fatalf("cross repo %v", e)
	}
	_, e = s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "request", OperatorNote: "different"})
	if !errors.Is(e, ErrConflict) {
		t.Fatalf("key conflict %v", e)
	}
}

func TestStagedDriftRetainsResultAndDoesNotRestart(t *testing.T) {
	s, _, snap, p := serviceFixture(t)
	_, j := admission(t, s, "drift")
	p.complete("succeeded", validResult())
	snap.drift()
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Result == nil || got.Applicable || got.Status != "succeeded" || p.starts != 1 {
		t.Fatalf("drift %#v err=%v starts=%d", got, e, p.starts)
	}
}

func TestCancellationRemainsPendingUntilOwnerConfirmationAndRejectsLateResult(t *testing.T) {
	s, _, _, p := serviceFixture(t)
	_, j := admission(t, s, "cancel")
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "cancel", 0)
	if e != nil || got.Status != "cancelling" || !got.CancelRequested {
		t.Fatalf("cancel %#v %v", got, e)
	}
	p.complete("succeeded", validResult())
	got, e = s.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Result != nil || got.Status != "cancelled" {
		t.Fatalf("late result %#v %v", got, e)
	}
	p.complete("cancelled", nil)
	got, e = s.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Status != "cancelled" || got.Result != nil {
		t.Fatalf("owner cancel %#v %v", got, e)
	}
}

func TestCancelWinsAgainstConcurrentWaitCompletion(t *testing.T) {
	s, _, _, p := serviceFixture(t)
	_, j := admission(t, s, "race")
	p.entered = make(chan struct{})
	p.release = make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "wait", time.Second)
		done <- e
	}()
	<-p.entered
	if _, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "cancel", 0); e != nil {
		t.Fatal(e)
	}
	p.complete("succeeded", validResult())
	close(p.release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Result != nil || got.Status != "cancelled" {
		t.Fatalf("raced result %#v %v", got, e)
	}
}

func TestMalformedStructuredResultFailsWithoutTranscriptFallback(t *testing.T) {
	s, _, _, p := serviceFixture(t)
	_, j := admission(t, s, "invalid")
	p.complete("succeeded", &v1.StagedMessageSuggestion{Title: "fake\nbody"})
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Status != "failed" || got.Result != nil {
		t.Fatalf("invalid %#v %v", got, e)
	}
}

func TestStaleCollectionCannotStartAgent(t *testing.T) {
	s, _, snap, p := serviceFixture(t)
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	snap.drift()
	_, e = s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "stale"})
	if !errors.Is(e, ErrConflict) || p.starts != 0 {
		t.Fatalf("stale admitted %v starts=%d", e, p.starts)
	}
}

type lostAckProvider struct {
	fakeProvider
	muAck   sync.Mutex
	keys    []string
	ackLost bool
}

func (p *lostAckProvider) Start(ctx context.Context, input DraftInput, key string) (Execution, error) {
	p.muAck.Lock()
	defer p.muAck.Unlock()
	p.keys = append(p.keys, key)
	if !p.ackLost {
		p.ackLost = true
		return Execution{}, errors.New("connection lost after owner accepted request")
	}
	return Execution{ID: "same-owner-execution", Status: "running"}, nil
}

func TestLostAdmissionAcknowledgementReconcilesSameDurableKey(t *testing.T) {
	s, store, snap, _ := serviceFixture(t)
	provider := &lostAckProvider{}
	s.provider = provider
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	first, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "lost-ack"})
	if e != nil || first.Status == "failed" || first.ExecutionId != "" {
		t.Fatalf("lost ack %#v %v", first, e)
	}
	reopened := New(store, snap, nil, provider)
	got, e := reopened.Job(context.Background(), "repo", "fixture", "", "lost-ack", "get", 0)
	if e != nil || got.ExecutionId != "same-owner-execution" || len(provider.keys) != 2 || provider.keys[0] != provider.keys[1] {
		t.Fatalf("reattach %#v err=%v keys=%v", got, e, provider.keys)
	}
}

type uncertainAdmissionProvider struct {
	fakeProvider
	deny     bool
	keys     []string
	accepted map[string]bool
}

func (p *uncertainAdmissionProvider) Start(_ context.Context, _ DraftInput, requestKey string) (Execution, error) {
	p.keys = append(p.keys, requestKey)
	if p.deny {
		return Execution{}, connect.NewError(connect.CodeUnauthenticated, errors.New("caller credential expired"))
	}
	if !p.accepted[requestKey] {
		p.accepted[requestKey] = true
		return Execution{}, errors.New("connection lost after owner admission")
	}
	return Execution{ID: "original-owner-execution", Status: "running"}, nil
}

func TestLostAdmissionAcknowledgementSurvivesLaterAuthenticationFailure(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	p := &uncertainAdmissionProvider{accepted: map[string]bool{}}
	s.provider = p
	_, first := admission(t, s, "uncertain-admission")
	if first.Status != "running" || first.ExecutionId != "" {
		t.Fatalf("initial uncertainty was lost: %v", first)
	}
	p.deny = true
	reopened := New(store, snapshot, nil, p)
	denied, err := reopened.Job(t.Context(), "repo", "fixture", "", "uncertain-admission", "get", 0)
	if err != nil || denied.Status != "running" || denied.Id != first.Id || denied.ExecutionId != "" || denied.Error == "" {
		t.Fatalf("later auth failure invented definitive rejection: %v %v", denied, err)
	}
	// Provider absence after another restart is also not evidence that the
	// original owner admission never happened.
	unavailable, err := New(store, snapshot, nil, nil).Job(t.Context(), "repo", "fixture", "", "uncertain-admission", "get", 0)
	if err != nil || unavailable.Status != "running" || unavailable.Id != first.Id {
		t.Fatalf("provider outage discarded uncertain admission: %v %v", unavailable, err)
	}
	p.deny = false
	restored, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "uncertain-admission", "get", 0)
	if err != nil || restored.Status != "running" || restored.ExecutionId != "original-owner-execution" || restored.Id != first.Id || len(p.accepted) != 1 || len(p.keys) != 3 {
		t.Fatalf("credential restore did not recover single original execution: %v %v keys=%v", restored, err, p.keys)
	}
	for _, requestKey := range p.keys {
		if requestKey != key("repo", "uncertain-admission") {
			t.Fatal("reconciliation changed original owner key")
		}
	}
}

func TestFirstDefinitiveAdmissionRejectionRemainsTerminal(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	p := &uncertainAdmissionProvider{deny: true, accepted: map[string]bool{}}
	s.provider = p
	_, job := admission(t, s, "first-auth-rejection")
	if job.Status != "failed" || job.ExecutionId != "" || len(p.accepted) != 0 || len(p.keys) != 1 {
		t.Fatalf("first definitive refusal not preserved: %v", job)
	}
	if got, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "first-auth-rejection", "get", 0); err != nil || got.Status != "failed" || len(p.keys) != 1 {
		t.Fatalf("retry contacted owner after definitive refusal: %v %v", got, err)
	}
}

type interleavedAdmissionProvider struct {
	fakeProvider
	admissionMu                sync.Mutex
	keys                       []string
	accepted                   map[string]bool
	firstEntered, firstRelease chan struct{}
	retryEntered, retryRelease chan struct{}
}

func (p *interleavedAdmissionProvider) Start(ctx context.Context, _ DraftInput, requestKey string) (Execution, error) {
	p.admissionMu.Lock()
	p.keys = append(p.keys, requestKey)
	call := len(p.keys)
	if call == 2 {
		p.accepted[requestKey] = true
	}
	p.admissionMu.Unlock()
	if call == 1 {
		close(p.firstEntered)
		select {
		case <-p.firstRelease:
		case <-ctx.Done():
			return Execution{}, ctx.Err()
		}
		return Execution{}, connect.NewError(connect.CodeUnauthenticated, errors.New("first caller credential expired"))
	}
	if call == 2 {
		close(p.retryEntered)
		select {
		case <-p.retryRelease:
		case <-ctx.Done():
			return Execution{}, ctx.Err()
		}
		return Execution{}, errors.New("retry connection lost after owner accepted request")
	}
	return Execution{ID: "original-interleaved-execution", Status: "running"}, nil
}

func TestConcurrentRetryAdmissionCannotBeTerminatedByOlderAuthenticationRefusal(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	p := &interleavedAdmissionProvider{accepted: map[string]bool{}, firstEntered: make(chan struct{}), firstRelease: make(chan struct{}), retryEntered: make(chan struct{}), retryRelease: make(chan struct{})}
	s.provider = p
	c, err := s.Collect(t.Context(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	type response struct {
		job *v1.StagedMessageJob
		err error
	}
	first := make(chan response, 1)
	go func() {
		job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "interleaved"})
		first <- response{job, err}
	}()
	select {
	case <-p.firstEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("first admission did not reach owner")
	}
	retries := make(chan response, 1)
	go func() {
		job, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "interleaved", "get", 0)
		retries <- response{job, err}
	}()
	select {
	case <-p.retryEntered:
	case <-time.After(3 * time.Second):
		close(p.firstRelease)
		close(p.retryRelease)
		t.Fatal("retry did not reach owner")
	}
	close(p.firstRelease)
	var firstReply response
	select {
	case firstReply = <-first:
	case <-time.After(3 * time.Second):
		close(p.retryRelease)
		t.Fatal("first admission failed to finish")
	}
	close(p.retryRelease)
	var retryReply response
	select {
	case retryReply = <-retries:
	case <-time.After(3 * time.Second):
		t.Fatal("retry failed to finish")
	}
	if firstReply.err != nil || firstReply.job.Status != "running" {
		t.Fatalf("older refusal erased accepted retry: %v %v", firstReply.job, firstReply.err)
	}
	retry := retryReply.job
	if retryReply.err != nil || retry.Status != "running" || retry.Id != firstReply.job.Id || retry.ExecutionId != "" {
		t.Fatalf("accepted retry uncertainty lost: %v %v", retry, retryReply.err)
	}
	restored, err := New(store, snapshot, nil, p).Job(t.Context(), "repo", "fixture", "", "interleaved", "get", 0)
	if err != nil || restored.Id != retry.Id || restored.ExecutionId != "original-interleaved-execution" || restored.Status != "running" {
		t.Fatalf("did not recover original execution: %v %v", restored, err)
	}
	p.admissionMu.Lock()
	defer p.admissionMu.Unlock()
	if len(p.accepted) != 1 || len(p.keys) != 3 {
		t.Fatalf("unexpected owner admissions: accepted=%v calls=%d", p.accepted, len(p.keys))
	}
	for _, requestKey := range p.keys {
		if requestKey != key("repo", "interleaved") {
			t.Fatal("retry replaced original owner identity")
		}
	}
}

func TestResultCoverageRejectsMissingDuplicatedAndInventedScope(t *testing.T) {
	input := DraftInput{Snapshot: &v1.StagedMessageSnapshot{Files: []*v1.StagedMessageFile{{Path: "file", EditCount: 1}}}}
	for _, test := range []struct {
		name   string
		change func(*v1.StagedMessageSuggestion)
	}{
		{"missing file", func(r *v1.StagedMessageSuggestion) { r.Coverage.CoveredPaths = nil }},
		{"duplicate file", func(r *v1.StagedMessageSuggestion) {
			r.Coverage.UncoveredPaths = []string{"file"}
			r.Coverage.Partial = true
		}},
		{"invented path", func(r *v1.StagedMessageSuggestion) { r.Coverage.CoveredPaths = []string{"other"} }},
		{"invalid count", func(r *v1.StagedMessageSuggestion) { r.Coverage.MatchedEdits = 2 }},
		{"invalid trailer", func(r *v1.StagedMessageSuggestion) {
			r.Trailers = []*v1.StagedMessageTrailer{{Key: "bad:key", Value: "value"}}
		}},
		{"inferred attribution", func(r *v1.StagedMessageSuggestion) {
			r.Trailers = []*v1.StagedMessageTrailer{{Key: "Vrooli-Run", Value: "invented"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := validResult()
			test.change(r)
			if e := validateResult(input, r); e == nil {
				t.Fatal("invalid structured result accepted")
			}
		})
	}
}

type concurrentOwner struct {
	fakeProvider
	admissionMu sync.Mutex
	keys        map[string]bool
	entered     chan struct{}
	release     chan struct{}
}

func (p *concurrentOwner) Start(ctx context.Context, _ DraftInput, key string) (Execution, error) {
	p.admissionMu.Lock()
	p.keys[key] = true
	p.admissionMu.Unlock()
	p.entered <- struct{}{}
	select {
	case <-p.release:
		return Execution{ID: "one-owner-execution", Status: "running"}, nil
	case <-ctx.Done():
		return Execution{}, ctx.Err()
	}
}

func TestConcurrentAdmissionRetainsOneJobAndOneOwnerIdempotencyIdentity(t *testing.T) {
	s, _, _, _ := serviceFixture(t)
	p := &concurrentOwner{keys: map[string]bool{}, entered: make(chan struct{}, 4), release: make(chan struct{})}
	s.provider = p
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	type reply struct {
		job *v1.StagedMessageJob
		err error
	}
	replies := make(chan reply, 4)
	call := func() {
		j, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "concurrent"})
		replies <- reply{j, e}
	}
	go call()
	select {
	case <-p.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first admission did not reach owner")
	}
	for i := 0; i < 3; i++ {
		go call()
	}
	// Competing row claims may elide redundant owner calls. The contract is
	// one durable job and owner identity, not one HTTP call per observer.
	close(p.release)
	want := ""
	for i := 0; i < 4; i++ {
		r := <-replies
		if r.err != nil {
			t.Fatal(r.err)
		}
		if want == "" {
			want = r.job.Id
		}
		if r.job.Id != want {
			t.Fatal("duplicate durable jobs")
		}
	}
	p.admissionMu.Lock()
	defer p.admissionMu.Unlock()
	if len(p.keys) != 1 {
		t.Fatalf("created new owner identities: %v", p.keys)
	}
}

func TestWaitOnLostAdmissionDoesNotSpinOnFastTransportFailure(t *testing.T) {
	s, _, _, _ := serviceFixture(t)
	p := &alwaysUnavailable{}
	s.provider = p
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, RequestKey: "outage"})
	if e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "wait", 25*time.Millisecond)
	if e != nil || got.Status != "running" || got.Error == "" || time.Since(start) < 20*time.Millisecond {
		t.Fatalf("unpaced/false-terminal wait %#v %v elapsed=%v", got, e, time.Since(start))
	}
}

type alwaysUnavailable struct{ fakeProvider }

func (*alwaysUnavailable) Start(context.Context, DraftInput, string) (Execution, error) {
	return Execution{}, errors.New("owner transport unavailable")
}

func TestDraftInputRetainsCanonicalEvidenceAboveProviderPromptBound(t *testing.T) {
	s, _, snapshot, p := serviceFixture(t)
	snapshot.patch = strings.Repeat("x", 100000)
	var selected []string
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("context-%d", i)
		selected = append(selected, id)
		p.contexts = append(p.contexts, &v1.StagedMessageContext{Id: id, Revision: "immutable", Body: strings.Repeat("y", 8100)})
	}
	c, e := s.Collect(context.Background(), "repo", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, SelectedContextIds: selected, RequestKey: "bounded"})
	if e != nil {
		t.Fatal(e)
	}
	raw, err := json.Marshal(p.lastInput)
	if err != nil || len(raw) <= legacyProviderInputBytes || len(raw) > MaxDraftInputBytes || p.lastInput.FormatVersion != 2 || len(p.lastInput.Snapshot.Files[0].Patch) != 100000 || len(c.Snapshot.Files[0].Patch) != 100000 {
		t.Fatal("canonical durable input was constrained by the separate provider packaging bound", err)
	}
}

func TestLegacyAdmissionRetryUsesOriginalDurableInputAndRejectsChangedIntent(t *testing.T) {
	s, store, snapshot, provider := serviceFixture(t)
	snapshot.patch = strings.Repeat("x", 100000)
	provider.contexts = []*v1.StagedMessageContext{{Id: "source@1", Revision: "1", Body: "retained context"}}
	c, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	oldSnapshot := proto.Clone(c.Snapshot).(*v1.StagedMessageSnapshot)
	oldSnapshot.Files[0].Patch = ""
	oldSnapshot.Files[0].OmissionReason = "patch omitted by immutable draft input byte bound"
	old := DraftInput{Snapshot: oldSnapshot, Contexts: provider.contexts, OperatorNote: "same intent"}
	raw, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(context.Background(), reviewjobs.Job{ID: "legacy-job", IdempotencyKey: key("repo", "legacy-key"), InputDigest: digest(raw), InputJSON: string(raw), ResultRef: c.Id, Operation: operation, RepositoryID: "repo", ExecutionID: "legacy-owner", State: reviewjobs.Running})
	if err != nil {
		t.Fatal(err)
	}
	req := &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: c.Snapshot.Digest, SelectedContextIds: []string{"source@1"}, OperatorNote: "same intent", RequestKey: "legacy-key"}
	got, err := s.Start(context.Background(), "repo", "fixture", req)
	if err != nil || got.Id != job.ID || provider.starts != 0 || got.Snapshot.Files[0].Patch != "" {
		t.Fatalf("legacy retry did not reattach original input: %v %v starts=%d", got, err, provider.starts)
	}
	retained, err := store.Get(context.Background(), job.ID)
	if err != nil || retained.InputJSON != string(raw) {
		t.Fatal("legacy immutable input was repackaged", err)
	}
	for _, change := range []string{"note", "sources", "collection", "subject"} {
		t.Run(change, func(t *testing.T) {
			altered := proto.Clone(req).(*v1.StartStagedMessageRequest)
			switch change {
			case "note":
				altered.OperatorNote = "different intent"
			case "sources":
				altered.SelectedContextIds = nil
			case "collection":
				other, err := s.Collect(context.Background(), "repo", "fixture")
				if err != nil {
					t.Fatal(err)
				}
				altered.CollectionId = other.Id
			case "subject":
				altered.SnapshotDigest = "different subject"
			}
			if _, err := s.Start(context.Background(), "repo", "fixture", altered); !errors.Is(err, ErrConflict) {
				t.Fatalf("changed %s must conflict: %v", change, err)
			}
		})
	}
}

type retainedFixture []RetainedMessage

func (r retainedFixture) Messages(context.Context, string, *v1.StagedMessageSnapshot) ([]RetainedMessage, []*v1.Omission, error) {
	return r, nil, nil
}

type immutableSnapshotFixture struct{ captured Captured }

func (f immutableSnapshotFixture) Capture(context.Context, string, string) (Captured, error) {
	return f.captured, nil
}

func (f immutableSnapshotFixture) Identity(context.Context, string, string) (string, error) {
	return f.captured.Snapshot.Digest, nil
}

func TestCollectionPersists4096LongPathsAndBudgetsExplanatoryDuplicates(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	snapshot := &v1.StagedMessageSnapshot{RepositoryId: "repo", Digest: "sha256:complete-large-manifest"}
	files := make([]RetainedFile, 0, MaxManifestFiles)
	for i := 0; i < MaxManifestFiles; i++ {
		// Roughly1KiB paths fit ordinary filesystem component lengths; the full
		// selection identity remains much larger than the provider prompt budget.
		path := fmt.Sprintf("scenarios/scope-%02d/%sfile-%04d.go", i%32, strings.Repeat("ordinary-directory/", 55), i)
		snapshot.Files = append(snapshot.Files, &v1.StagedMessageFile{Path: path, OldPath: path, OldBlobId: "old-blob", NewBlobId: "new-blob", OldMode: "100644", NewMode: "100644", OmissionReason: "representative detail omitted"})
		files = append(files, RetainedFile{Path: path, BaseBlobID: "old-blob", BlobID: "new-blob", OldMode: "100644", NewMode: "100644", Captured: true})
	}
	var messages retainedFixture
	for i := 0; i < 100; i++ {
		messages = append(messages, RetainedMessage{ID: fmt.Sprintf("proposal-%03d", i), Revision: "1", Title: "Retained scoped update", Body: strings.Repeat("context ", 1000), Files: files})
	}
	s.snapshots = immutableSnapshotFixture{Captured{Snapshot: snapshot}}
	s.messages = messages
	c, err := s.Collect(context.Background(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := protojson.Marshal(c)
	if err != nil || len(encoded) <= 2<<20 || len(encoded) > MaxCollectionBytes || len(c.Snapshot.Files) != MaxManifestFiles || len(c.Suggestions) != 1 || len(c.Contexts) != 100 {
		t.Fatalf("large collection not bounded: bytes=%d files=%d suggestions=%d contexts=%d err=%v", len(encoded), len(c.Snapshot.Files), len(c.Suggestions), len(c.Contexts), err)
	}
	if len(c.Suggestions[0].Coverage.CoveredPaths) != MaxManifestFiles || c.Suggestions[0].Coverage.Partial {
		t.Fatal("complete deterministic coverage was truncated")
	}
	if provider.starts != 0 {
		t.Fatal("large collection started an agent")
	}
	job, err := s.Start(context.Background(), "repo", "fixture", &v1.StartStagedMessageRequest{CollectionId: c.Id, SnapshotDigest: snapshot.Digest, SelectedContextIds: []string{c.Contexts[0].Id}, RequestKey: "large-selection"})
	if err != nil {
		t.Fatal(err)
	}
	retained, err := store.Get(context.Background(), job.Id)
	if err != nil || len(retained.InputJSON) <= legacyProviderInputBytes || len(provider.lastInput.Snapshot.Files) != MaxManifestFiles {
		t.Fatalf("complete durable admission lost: %v input bytes=%d", err, len(retained.InputJSON))
	}
	reattached, err := New(store, s.snapshots, nil, provider).Job(context.Background(), "repo", "fixture", "", "large-selection", "get", 0)
	if err != nil || reattached.Id != job.Id || len(reattached.Snapshot.Files) != MaxManifestFiles || reattached.Snapshot.Digest != snapshot.Digest {
		t.Fatal("restart lost canonical manifest", err)
	}
	t.Logf("collection JSON=%d bytes; durable job input=%d bytes; manifest=%d; contexts=%d; complete suggestions=%d", len(encoded), len(retained.InputJSON), len(c.Snapshot.Files), len(c.Contexts), len(c.Suggestions))
}

type expiredReadProvider struct{ fakeProvider }

func (*expiredReadProvider) Get(context.Context, string, DraftInput) (Execution, error) {
	return Execution{}, connect.NewError(connect.CodeUnauthenticated, errors.New("owner read credential expired"))
}

func TestOwnerReadAuthenticationFailureDoesNotInventTerminalExecution(t *testing.T) {
	s, store, snapshot, _ := serviceFixture(t)
	_, j := admission(t, s, "auth-expiry")
	s.provider = &expiredReadProvider{}
	got, e := s.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Status != "running" || got.ExecutionId == "" || got.Error == "" {
		t.Fatalf("read failure changed authoritative phase %#v %v", got, e)
	}
	reopened := New(store, snapshot, nil, nil)
	got, e = reopened.Job(context.Background(), "repo", "fixture", j.Id, "", "get", 0)
	if e != nil || got.Status != "running" || got.ExecutionId == "" {
		t.Fatalf("provider outage invented termination %#v %v", got, e)
	}
}
