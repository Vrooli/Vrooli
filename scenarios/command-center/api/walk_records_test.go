package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	j "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	jc "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	limit := int(r.Msg.GetLimit())
	if limit <= 0 {
		limit = 1
	}
	for i := len(f.entries) - 1; i >= 0 && len(out.Entries) < limit; i-- {
		if f.entries[i].Kind == r.Msg.Kind {
			out.Entries = append(out.Entries, f.entries[i])
		}
	}
	return connect.NewResponse(out), nil
}

// appendForeign mimics `source-ledger journal note --kind <owner kind>`: a writer holding
// the scope appends prose carrying an owner kind, without an owner request key.
func (f *walkLedgerFake) appendForeign(kind, body string) *j.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	e := &j.Entry{Id: fmt.Sprint(len(f.entries) + 1), Body: body, Kind: kind, CreatedAt: timestamppb.Now()}
	f.entries = append(f.entries, e)
	return e
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

func walkTestLedger(t *testing.T) (walkConnectService, *walkLedgerFake) {
	f := &walkLedgerFake{keys: map[string]*j.Entry{}}
	_, h := jc.NewJournalServiceHandler(f)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return walkConnectService{ledger: jc.NewJournalServiceClient(srv.Client(), srv.URL)}, f
}

func walkTestService(t *testing.T) walkConnectService {
	s, _ := walkTestLedger(t)
	return s
}

func walkPublishRequest(key, briefing string) *walkv1.PublishRequest {
	phases := []map[string]string{}
	for _, p := range walkPhases {
		phases = append(phases, map[string]string{"phase": p})
	}
	raw, _ := json.Marshal(map[string]any{"program": "command-center.vision-walk-prep", "status": "partial", "signals": map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano), "phases": phases, "checkpoint": map[string]string{"status": "none"},
	}})
	return &walkv1.PublishRequest{
		Channel: "test", RequestKey: key, ProgramId: "prog_test", EnvelopeJson: string(raw),
		Briefing: briefing, FleetHealthJson: `{"status":"unavailable","reason":"test"}`,
	}
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

func TestWalkPublishPreservesUnavailableEvidence(t *testing.T) { // [REQ:CC-P0-016]
	s := walkTestService(t)
	phases := []map[string]string{}
	for _, phase := range walkPhases {
		phases = append(phases, map[string]string{"phase": phase})
	}
	envelope := map[string]any{
		"program": "command-center.vision-walk-prep", "status": "unavailable",
		"errors": []map[string]string{{"class": "scenario_unreachable", "where": "outcomes"}},
		"signals": map[string]any{
			"generated_at": time.Now().UTC().Format(time.RFC3339Nano), "phases": phases,
			"checkpoint": map[string]string{"status": "unavailable", "reason": "source outage"},
		},
	}
	raw, _ := json.Marshal(envelope)
	req := &walkv1.PublishRequest{
		Channel: "test", RequestKey: "outage", ProgramId: "prog_outage",
		EnvelopeJson: string(raw), Briefing: "Preparation unavailable. Source outage; continuity unknown.",
		FleetHealthJson: `{"status":"unavailable","reason":"owner unreachable"}`,
	}
	receipt, err := s.Publish(context.Background(), connect.NewRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.State(context.Background(), connect.NewRequest(&walkv1.StateRequest{Channel: "test"}))
	if err != nil || state.Msg.Briefing.GetEntryId() != receipt.Msg.EntryId {
		t.Fatalf("unavailable evidence was not retained: %v %v", state, err)
	}
	// Formatting is not payload growth and must not force agents to run jq -c.
	req.EnvelopeJson = strings.Repeat("\n", 60001) + string(raw)
	replayed, err := s.Publish(context.Background(), connect.NewRequest(req))
	if err != nil || replayed.Msg.EntryId != receipt.Msg.EntryId {
		t.Fatalf("formatted envelope lost replay identity: %v", err)
	}
	// A malformed/failed program cannot masquerade as an unavailable briefing.
	envelope["status"] = "failed"
	raw, _ = json.Marshal(envelope)
	req.EnvelopeJson = string(raw)
	req.RequestKey = "failed"
	if _, err := s.Publish(context.Background(), connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("failed program accepted: %v", err)
	}
}

// A journal kind is a label any scope holder can write. Prose carrying an owner kind must
// never be read back as walk state, and it must not wedge the next publication. [REQ:CC-P0-016]
func TestWalkStateSkipsForeignEntriesCarryingOwnerKind(t *testing.T) {
	s, ledger := walkTestLedger(t)
	ctx := context.Background()

	published, err := s.Publish(ctx, connect.NewRequest(walkPublishRequest("first", "Own briefing, thirteen phases")))
	if err != nil {
		t.Fatal(err)
	}
	foreign := ledger.appendForeign("vision-walk-briefing-test", "Topic: vision-walk-record/2026-09-06/heartbeat-0900\n\nA member's prose note.")

	state, err := s.State(ctx, connect.NewRequest(&walkv1.StateRequest{Channel: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	if state.Msg.Briefing.GetEntryId() != published.Msg.EntryId {
		t.Fatalf("prose shadowed the briefing: got %q want %q", state.Msg.Briefing.GetEntryId(), published.Msg.EntryId)
	}

	// The predecessor the reader was handed still publishes, chaining past the foreign entry.
	next, err := s.Publish(ctx, connect.NewRequest(func() *walkv1.PublishRequest {
		r := walkPublishRequest("second", "Second own briefing")
		r.ExpectedPreviousId = state.Msg.Briefing.GetEntryId()
		return r
	}()))
	if err != nil {
		t.Fatalf("foreign entry wedged publication: %v", err)
	}
	if next.Msg.EntryId == foreign.Id {
		t.Fatal("publication reused the foreign entry id")
	}

	// A stale predecessor is still a conflict; the foreign id is never a valid one.
	stale := walkPublishRequest("third", "Third own briefing")
	stale.ExpectedPreviousId = published.Msg.EntryId
	if _, err := s.Publish(ctx, connect.NewRequest(stale)); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale predecessor accepted: %v", err)
	}
	byForeign := walkPublishRequest("fourth", "Fourth own briefing")
	byForeign.ExpectedPreviousId = foreign.Id
	if _, err := s.Publish(ctx, connect.NewRequest(byForeign)); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("foreign predecessor accepted: %v", err)
	}
}

// The same label problem reaches continuity: a foreign entry must not read as the prior
// checkpoint, nor block the transition after it. [REQ:CC-P0-016]
func TestWalkCheckpointSkipsForeignEntriesCarryingOwnerKind(t *testing.T) {
	s, ledger := walkTestLedger(t)
	ctx := context.Background()

	active, err := s.Checkpoint(ctx, connect.NewRequest(&walkv1.CheckpointRequest{
		Channel: "test", RequestKey: "open", WalkId: "w", State: "active", ResumePhase: "5.5", Content: "Phases 1-5 done",
	}))
	if err != nil {
		t.Fatal(err)
	}
	ledger.appendForeign("walk-checkpoint-test", "A member's prose about the walk.")

	state, err := s.State(ctx, connect.NewRequest(&walkv1.StateRequest{Channel: "test"}))
	if err != nil || state.Msg.Checkpoint.GetEntryId() != active.Msg.EntryId {
		t.Fatalf("prose shadowed the checkpoint: %v %v", state, err)
	}
	if _, err := s.Checkpoint(ctx, connect.NewRequest(&walkv1.CheckpointRequest{
		Channel: "test", RequestKey: "close", WalkId: "w", State: "completed", ExpectedPreviousId: active.Msg.EntryId,
	})); err != nil {
		t.Fatalf("foreign entry wedged continuity: %v", err)
	}
}
