package main

import (
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	"fmt"
	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	j "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	jc "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type walkLedgerFake struct {
	jc.UnimplementedJournalServiceHandler
	mu      sync.Mutex
	entries []*j.Entry
	keys    map[string]*j.Entry
}

func (f *walkLedgerFake) GetEntry(_ context.Context, r *connect.Request[j.GetEntryRequest]) (*connect.Response[j.GetEntryResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := f.keys[r.Msg.RequestKey]; e != nil {
		return connect.NewResponse(&j.GetEntryResponse{Entry: e}), nil
	}
	for _, e := range f.entries {
		if e.Id == r.Msg.Id {
			return connect.NewResponse(&j.GetEntryResponse{Entry: e}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("missing"))
}
func (f *walkLedgerFake) ListEntries(_ context.Context, r *connect.Request[j.ListEntriesRequest]) (*connect.Response[j.ListEntriesResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := &j.ListEntriesResponse{}
	for i := len(f.entries) - 1; i >= 0; i-- {
		if f.entries[i].Kind == r.Msg.Kind {
			out.Entries = append(out.Entries, f.entries[i])
			break
		}
	}
	return connect.NewResponse(out), nil
}
func (f *walkLedgerFake) AppendEntry(_ context.Context, r *connect.Request[j.AppendEntryRequest]) (*connect.Response[j.AppendEntryResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := f.keys[r.Msg.RequestKey]; e != nil {
		if e.Body != r.Msg.Body {
			return nil, conflictWalk("key")
		}
		return connect.NewResponse(&j.AppendEntryResponse{Entry: e, Existing: true}), nil
	}
	last := ""
	for _, e := range f.entries {
		if e.Kind == r.Msg.Kind {
			last = e.Id
		}
	}
	if last != r.Msg.GetExpectedLatestId() {
		return nil, conflictWalk("predecessor")
	}
	e := &j.Entry{Id: fmt.Sprint(len(f.entries) + 1), Body: r.Msg.Body, Kind: r.Msg.Kind, CreatedAt: timestamppb.Now()}
	f.entries = append(f.entries, e)
	f.keys[r.Msg.RequestKey] = e
	return connect.NewResponse(&j.AppendEntryResponse{Entry: e}), nil
}
func walkTestService(t *testing.T) walkConnectService {
	f := &walkLedgerFake{keys: map[string]*j.Entry{}}
	_, h := jc.NewJournalServiceHandler(f)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return walkConnectService{ledger: jc.NewJournalServiceClient(srv.Client(), srv.URL)}
}
func TestWalkCheckpointTransitionsReplayAndChannels(t *testing.T) { // [REQ:CC-P0-016]
	s := walkTestService(t)
	ctx := context.Background()
	req := &walkv1.CheckpointRequest{Channel: "test", RequestKey: "a", WalkId: "w", State: "active", ResumePhase: "5.5", Content: "Exact ✓\n"}
	a, e := s.Checkpoint(ctx, connect.NewRequest(req))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Checkpoint(ctx, connect.NewRequest(&walkv1.CheckpointRequest{Channel: "test", RequestKey: "b", WalkId: "other", State: "active", ResumePhase: "1", Content: "wrong", ExpectedPreviousId: a.Msg.EntryId}))
	if connect.CodeOf(e) != connect.CodeAborted {
		t.Fatalf("other walk: %v", e)
	}
	b, e := s.Checkpoint(ctx, connect.NewRequest(&walkv1.CheckpointRequest{Channel: "test", RequestKey: "done", WalkId: "w", State: "completed", ExpectedPreviousId: a.Msg.EntryId}))
	if e != nil {
		t.Fatal(e)
	}
	replay, e := s.Checkpoint(ctx, connect.NewRequest(req))
	if e != nil || replay.Msg.EntryId != a.Msg.EntryId || !replay.Msg.Existing {
		t.Fatalf("replay: %v %v", replay, e)
	}
	req.RequestKey = "resurrect"
	req.ExpectedPreviousId = b.Msg.EntryId
	_, e = s.Checkpoint(ctx, connect.NewRequest(req))
	if connect.CodeOf(e) != connect.CodeAborted {
		t.Fatalf("resurrection: %v", e)
	}
	state, e := s.State(ctx, connect.NewRequest(&walkv1.StateRequest{Channel: "operator"}))
	if e != nil || state.Msg.Checkpoint != nil {
		t.Fatal("test leaked to operator")
	}
}
func TestWalkPublishValidatesAndReturnsSameReceipt(t *testing.T) { // [REQ:CC-P0-016]
	s := walkTestService(t)
	ph := []map[string]string{}
	for _, p := range walkPhases {
		ph = append(ph, map[string]string{"phase": p})
	}
	raw, _ := json.Marshal(map[string]any{"program": "command-center.vision-walk-prep", "status": "partial", "signals": map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339Nano), "phases": ph, "checkpoint": map[string]string{"status": "none"}}})
	r := &walkv1.PublishRequest{Channel: "test", RequestKey: "p", ProgramId: "prog_test", EnvelopeJson: string(raw), Briefing: "Verified partial rehearsal", FleetHealthJson: `{"status":"unavailable","reason":"test"}`}
	a, e := s.Publish(context.Background(), connect.NewRequest(r))
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.Publish(context.Background(), connect.NewRequest(r))
	if e != nil || a.Msg.EntryId != b.Msg.EntryId || !b.Msg.Existing {
		t.Fatal("replay failed", e)
	}
	r.Briefing = "different"
	_, e = s.Publish(context.Background(), connect.NewRequest(r))
	if connect.CodeOf(e) != connect.CodeAborted {
		t.Fatal("conflicting replay accepted", e)
	}
	r.RequestKey = "bad"
	r.EnvelopeJson = `{}`
	_, e = s.Publish(context.Background(), connect.NewRequest(r))
	if connect.CodeOf(e) != connect.CodeInvalidArgument {
		t.Fatal("bad envelope accepted", e)
	}
}
