package learning

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	source "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	srpc "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	scopes "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	scrpc "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	"google.golang.org/protobuf/types/known/timestamppb"

	learningdata "vrooli-memory/internal/learning"
	"vrooli-memory/internal/ledgerclient"
)

type journalFake struct {
	srpc.JournalServiceClient
	stored   *source.Entry
	writes   int
	reads    int
	scope    string
	err      error
	page     *source.ListEntriesResponse
	resolved *source.Entry
	lookup   *source.GetEntryRequest
}

func (f *journalFake) AppendEntry(_ context.Context, r *connect.Request[source.AppendEntryRequest]) (*connect.Response[source.AppendEntryResponse], error) {
	f.writes++
	f.scope = r.Msg.Scope
	if f.stored != nil {
		return connect.NewResponse(&source.AppendEntryResponse{Entry: f.stored, Existing: true}), nil
	}
	f.stored = &source.Entry{Id: "entry", Body: r.Msg.Body}
	return connect.NewResponse(&source.AppendEntryResponse{Entry: f.stored}), nil
}
func (f *journalFake) GetEntry(_ context.Context, r *connect.Request[source.GetEntryRequest]) (*connect.Response[source.GetEntryResponse], error) {
	f.scope = r.Msg.Scope
	f.lookup = r.Msg
	if f.resolved != nil {
		return connect.NewResponse(&source.GetEntryResponse{Entry: f.resolved}), f.err
	}
	return connect.NewResponse(&source.GetEntryResponse{Entry: &source.Entry{Id: r.Msg.Id, CreatedAt: timestamppb.New(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))}}), f.err
}

func TestObservationResolvesStableAttemptIdentityNotLedgerUUID(t *testing.T) {
	a := request().Msg.Attempt
	body, err := learningdata.Encode(a)
	if err != nil {
		t.Fatal(err)
	}
	f := &journalFake{resolved: &source.Entry{Id: "ledger-uuid-distinct-from-attempt", Kind: "task-record", Body: body}}
	o := &pb.Observation{ObservationId: "feedback-1", AttemptId: a.AttemptId, Disposition: "supported", EvidenceRefs: []string{"test:receipt"}, Provenance: "test", ObservedAt: a.FinishedAt}
	result, err := handler(f, true).RecordObservation(context.Background(), connect.NewRequest(&pb.RecordObservationRequest{Scope: "owner-usage", Observation: o}))
	if err != nil {
		t.Fatal(err)
	}
	if result.Msg.EntryId == "" || f.writes != 1 || f.lookup.Id != "" || f.lookup.Scope != "owner-usage" {
		t.Fatal(result, f.lookup)
	}
	p := f.lookup.ImportProvenance
	if p == nil || p.Runtime != "vrooli-memory.learning/v1" || p.SourceLocator != a.AttemptId || p.ContentHash != "immutable-attempt" {
		t.Fatal(p)
	}
}
func (f *journalFake) ListEntries(_ context.Context, r *connect.Request[source.ListEntriesRequest]) (*connect.Response[source.ListEntriesResponse], error) {
	f.scope = r.Msg.Scope
	f.reads++
	if f.err != nil {
		return nil, f.err
	}
	if f.page != nil {
		return connect.NewResponse(f.page), nil
	}
	return connect.NewResponse(&source.ListEntriesResponse{}), nil
}

type scopesFake struct {
	scrpc.ScopesServiceClient
	exists bool
}

func (f scopesFake) ListScopes(_ context.Context, r *connect.Request[scopes.ListScopesRequest]) (*connect.Response[scopes.ListScopesResponse], error) {
	out := &scopes.ListScopesResponse{}
	if f.exists {
		out.Scopes = []*scopes.Scope{{Id: r.Msg.Scope}}
	}
	return connect.NewResponse(out), nil
}
func handler(f *journalFake, exists bool) *Handler {
	return &Handler{client: &ledgerclient.Client{Journal: f, Scopes: scopesFake{exists: exists}}, now: func() time.Time { return time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC) }}
}
func request() *connect.Request[pb.RecordAttemptRequest] {
	return connect.NewRequest(&pb.RecordAttemptRequest{Scope: "owner-usage", Attempt: &pb.Attempt{AttemptId: "a", TaskId: "t", Operation: "inspect", ContextKey: "linux-v1", TaskStartedAt: "2026-09-01T00:00:00Z", StartedAt: "2026-09-01T00:00:00Z", FinishedAt: "2026-09-01T00:01:00Z", AttemptNumber: 1, Outcome: "verified_success", EvidenceRefs: []string{"run:1"}, RecallStatus: "no_match", Provenance: "operator", Trigger: "task", Approach: "inspect"}})
}
func TestCaptureReplayAndConflict(t *testing.T) {
	f := &journalFake{}
	h := handler(f, true)
	r := request()
	a, e := h.RecordAttempt(context.Background(), r)
	if e != nil || a.Msg.Existing || f.scope != "owner-usage" {
		t.Fatal(a, e)
	}
	a, e = h.RecordAttempt(context.Background(), r)
	if e != nil || !a.Msg.Existing {
		t.Fatal(a, e)
	}
	r.Msg.Attempt.Approach = "changed"
	_, e = h.RecordAttempt(context.Background(), r)
	if connect.CodeOf(e) != connect.CodeAlreadyExists {
		t.Fatal(e)
	}
}
func TestMissingScopeAndInvalidClaimsDoNotAppend(t *testing.T) {
	f := &journalFake{}
	_, e := handler(f, false).RecordAttempt(context.Background(), request())
	if connect.CodeOf(e) != connect.CodeNotFound || f.writes != 0 {
		t.Fatal(e)
	}
	r := request()
	r.Msg.Attempt.EvidenceRefs = nil
	_, e = handler(f, true).RecordAttempt(context.Background(), r)
	if connect.CodeOf(e) != connect.CodeInvalidArgument || f.writes != 0 {
		t.Fatal(e)
	}
}
func TestMeasureOutageIsNotEmptySuccess(t *testing.T) {
	f := &journalFake{err: connect.NewError(connect.CodeUnavailable, context.DeadlineExceeded)}
	_, e := handler(f, true).MeasureLearning(context.Background(), connect.NewRequest(&pb.MeasureLearningRequest{Scope: "owner-usage"}))
	if e == nil {
		t.Fatal("outage became success")
	}
}
func TestRepeatedCursorStopsRead(t *testing.T) {
	f := &journalFake{page: &source.ListEntriesResponse{NextCursor: "same"}}
	_, e := handler(f, true).MeasureLearning(context.Background(), connect.NewRequest(&pb.MeasureLearningRequest{Scope: "owner-usage"}))
	if e == nil || f.reads != 2 {
		t.Fatal(e, f.reads)
	}
}
func TestEmptyWindowReportsUnknown(t *testing.T) {
	f := &journalFake{}
	r, e := handler(f, true).MeasureLearning(context.Background(), connect.NewRequest(&pb.MeasureLearningRequest{Scope: "owner-usage"}))
	if e != nil || r.Msg.Reliable || r.Msg.Reason != "unreliable:no_eligible_attempts" {
		t.Fatal(r, e)
	}
}
