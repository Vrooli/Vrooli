package event_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSQLiteStore_QuotaFactsSurviveDurableReadAndPublicConversion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	runID := uuid.New()
	reset := time.Date(2026, 10, 14, 3, 29, 20, 7, time.UTC)
	zero := 0.0
	quota := domain.NewRateLimitEvent(runID, "weekly", "native quota", &reset, 0)
	q := quota.Data.(*domain.RateLimitEventData)
	q.Provider, q.Pool, q.UsedPercent, q.WindowMinutes, q.Provenance = "openai", "primary", &zero, 10080, "native"
	store := event.NewSQLiteStore(db, newTestLogger())
	if err := store.Append(ctx, runID, quota); err != nil {
		t.Fatal(err)
	}
	generic := domain.NewErrorEvent(runID, "RATE_LIMIT", "ordinary error", true)
	if err := store.Append(ctx, runID, generic); err != nil {
		t.Fatal(err)
	}

	// A fresh store reads owner-persisted JSON, rather than the appended pointer.
	store = event.NewSQLiteStore(db, newTestLogger())
	got, err := store.Get(ctx, runID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 2 {
		t.Fatalf("durable events=%+v err=%v", got, err)
	}
	data, ok := got[0].Data.(*domain.RateLimitEventData)
	if !ok || data.ResetTime == nil || !data.ResetTime.Equal(reset) || data.UsedPercent == nil || *data.UsedPercent != 0 || data.Provider != "openai" || data.Pool != "primary" || data.WindowMinutes != 10080 || data.Provenance != "native" || data.Limit != 0 || data.CurrentUsed != 0 {
		t.Fatalf("durable quota=%+v (%T)", got[0].Data, got[0].Data)
	}
	public := protoconv.RunEventToProto(got[0])
	wire, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	var rate map[string]json.RawMessage
	if err := json.Unmarshal(fields["rate_limit"], &rate); err != nil {
		t.Fatalf("public event=%s: %v", wire, err)
	}
	if string(rate["used_percent"]) != "0" || string(rate["provider"]) != `"openai"` || string(rate["pool"]) != `"primary"` || string(rate["window_minutes"]) != `"10080"` || string(rate["provenance"]) != `"native"` || public.GetRateLimit().GetResetTime() == nil || !public.GetRateLimit().GetResetTime().AsTime().Equal(reset) {
		t.Fatalf("public quota facts=%s", wire)
	}
	if e := protoconv.RunEventToProto(got[1]).GetError(); e == nil || e.Code != "RATE_LIMIT" || e.Message != "ordinary error" || !e.Retryable {
		t.Fatalf("generic error=%+v", e)
	}

	// Reader-stage proof uses literal tagged JSON independently of the writer constructor.
	tagged := `{"payloadKind":"quota","limitType":"weekly","message":"native quota","resetTime":"2026-10-14T03:29:20.000000007Z","provider":"openai","pool":"primary","usedPercent":0,"windowMinutes":10080,"provenance":"native"}`
	taggedID := uuid.New().String()
	if _, err := db.Exec(`INSERT INTO run_events (id,run_id,sequence,event_type,timestamp,data) VALUES (?,?,?,?,?,?)`, taggedID, runID.String(), 3, string(domain.EventTypeQuota), reset, tagged); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(ctx, runID, event.GetOptions{AfterSequence: 2, Limit: 1})
	if err != nil || len(got) != 1 {
		t.Fatalf("tagged read=%+v err=%v", got, err)
	}
	data, ok = got[0].Data.(*domain.RateLimitEventData)
	if !ok || data.ResetTime == nil || !data.ResetTime.Equal(reset) || data.UsedPercent == nil || *data.UsedPercent != 0 || data.Provider != "openai" || data.Pool != "primary" || data.WindowMinutes != 10080 || data.Provenance != "native" || data.Limit != 0 || data.CurrentUsed != 0 {
		t.Fatalf("tagged durable quota=%+v (%T)", got[0].Data, got[0].Data)
	}
	public = protoconv.RunEventToProto(got[0])
	wire, err = (protojson.MarshalOptions{UseProtoNames: true}).Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	fields = make(map[string]json.RawMessage)
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	rate = make(map[string]json.RawMessage)
	if err := json.Unmarshal(fields["rate_limit"], &rate); err != nil {
		t.Fatalf("tagged public event=%s: %v", wire, err)
	}
	if string(rate["used_percent"]) != "0" || string(rate["provider"]) != `"openai"` || string(rate["pool"]) != `"primary"` || string(rate["window_minutes"]) != `"10080"` || string(rate["provenance"]) != `"native"` || public.GetRateLimit().GetResetTime() == nil || !public.GetRateLimit().GetResetTime().AsTime().Equal(reset) {
		t.Fatalf("tagged public quota facts=%s", wire)
	}
	var taggedPersisted string
	if err := db.Get(&taggedPersisted, `SELECT data FROM run_events WHERE id=?`, taggedID); err != nil || taggedPersisted != tagged {
		t.Fatalf("tagged bytes=%q err=%v", taggedPersisted, err)
	}

	// Historical limit-only JSON stays byte-identical and has no invented native usage.
	legacy := `{"limitType":"token","message":"limit","currentUsed":25,"limit":100}`
	legacyID := uuid.New().String()
	if _, err := db.Exec(`INSERT INTO run_events (id,run_id,sequence,event_type,timestamp,data) VALUES (?,?,?,?,?,?)`, legacyID, runID.String(), 4, string(domain.EventTypeQuota), reset, legacy); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(ctx, runID, event.GetOptions{AfterSequence: 3, Limit: 1})
	if err != nil || len(got) != 1 {
		t.Fatalf("legacy read=%+v err=%v", got, err)
	}
	data, ok = got[0].Data.(*domain.RateLimitEventData)
	if !ok || data.Limit != 100 || data.CurrentUsed != 25 || data.UsedPercent != nil || data.Provider != "" || data.Pool != "" || data.WindowMinutes != 0 || data.ResetTime != nil || data.Provenance != "" {
		t.Fatalf("legacy facts=%+v (%T)", got[0].Data, got[0].Data)
	}
	public = protoconv.RunEventToProto(got[0])
	if public.GetRateLimit() == nil || public.GetRateLimit().UsedPercent != nil || public.GetRateLimit().GetLimit() != 100 {
		t.Fatalf("legacy public quota=%+v", public)
	}
	var persisted string
	if err := db.Get(&persisted, `SELECT data FROM run_events WHERE id=?`, legacyID); err != nil || persisted != legacy {
		t.Fatalf("historical bytes=%q err=%v", persisted, err)
	}
}

func TestSQLiteStoreMalformedKnownPayloadIsDatabaseError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, logrus.New())
	runID, eventID := uuid.New(), uuid.New()
	raw := `{"limitType":"weekly","usedPercent":"bad"}`
	_, err := db.Exec(`INSERT INTO run_events(id,run_id,sequence,event_type,timestamp,data) VALUES(?,?,?,?,?,?)`, eventID.String(), runID.String(), 0, "quota", time.Now(), raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), runID, event.GetOptions{AfterSequence: -1})
	var databaseError *domain.DatabaseError
	if !errors.As(err, &databaseError) || databaseError.Operation != "decode_event" || databaseError.EntityID != eventID.String() || got != nil {
		t.Fatalf("events=%+v error=%v", got, err)
	}
	var persisted string
	if err := db.Get(&persisted, `SELECT data FROM run_events WHERE id=?`, eventID.String()); err != nil || persisted != raw {
		t.Fatalf("persisted=%q error=%v", persisted, err)
	}
}

func TestSQLiteStoreToolTruthSurvivesPublicConversion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, logrus.New())
	runID := uuid.New()
	for index, exit := range []*int{nil, eventExitCode(0), eventExitCode(8)} {
		known := exit != nil
		var duration *int64
		if index > 0 {
			value := int64(0)
			if index == 2 {
				value = 125
			}
			duration = &value
		}
		evt := domain.NewToolResultEvent(runID, "exec", "call", "", nil)
		data := evt.Data.(*domain.ToolResultEventData)
		data.ExitCode = exit
		data.DurationMS = duration
		data.OutcomeKnown = &known
		data.Success = known && *exit == 0
		data.OutputBlocks = []domain.ToolOutputBlock{{Type: "input_text", Text: "one", RawJSON: `{"type":"input_text","text":"one"}`}, {Type: "image", RawJSON: `{"type":"image","future":true}`}}
		data.RawOutputJSON = `[ {"type":"input_text", "text":"one"}, {"type":"image", "future":true} ]`
		if err := store.Append(context.Background(), runID, evt); err != nil {
			t.Fatal(err)
		}
		rows, err := store.Get(context.Background(), runID, event.GetOptions{AfterSequence: int64(index) - 1, Limit: 1})
		if err != nil || len(rows) != 1 {
			t.Fatalf("read=%+v error=%v", rows, err)
		}
		public := protoconv.RunEventToProto(rows[0]).GetToolResult()
		if (public.ExitCode == nil) != (exit == nil) || exit != nil && int(public.GetExitCode()) != *exit || public.OutcomeKnown == nil || public.GetOutcomeKnown() != known || (public.DurationMs == nil) != (duration == nil) || duration != nil && public.GetDurationMs() != *duration || len(public.GetOutputBlocks()) != 2 || public.GetOutputBlocks()[1].GetRawJson() != `{"type":"image","future":true}` || public.GetRawOutputJson() != string(data.RawOutputJSON) {
			t.Fatalf("public=%+v", public)
		}
	}
}
func eventExitCode(n int) *int { return &n }

func TestSQLiteStoreImmutableReplaySuppressesNotificationsAndConflicts(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	run := uuid.New()
	stream, err := store.Stream(ctx, run, event.StreamOptions{BufferSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	original := domain.NewLogEvent(run, "warning", "same evidence")
	if err := store.Append(ctx, run, original); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stream:
	case <-time.After(time.Second):
		t.Fatal("initial notification missing")
	}
	replay := domain.NewLogEvent(run, "warning", "same evidence")
	replay.ID = original.ID
	if err := store.Append(ctx, run, replay); err != nil || !replay.PersistenceReplay || replay.Sequence != original.Sequence {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	select {
	case evt := <-stream:
		t.Fatalf("duplicate notification=%+v", evt)
	default:
	}
	conflict := domain.NewLogEvent(run, "warning", "changed evidence")
	conflict.ID = original.ID
	err = store.Append(ctx, run, conflict)
	var dbErr *domain.DatabaseError
	if !errors.As(err, &dbErr) || dbErr.Operation != "replay_event_conflict" || dbErr.EntityID != original.ID.String() {
		t.Fatalf("conflict=%v", err)
	}
	another := domain.NewLogEvent(run, "warning", "same evidence")
	if err := store.Append(ctx, run, another); err != nil || another.Sequence != 1 {
		t.Fatalf("distinct evidence=%+v err=%v", another, err)
	}
}

// A deterministic event re-read over a row stored with a random id is the same
// source line and replays it; deterministic twins from distinct lines stay.
func TestSQLiteStoreDeterministicEventReplaysItsLegacyTwin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := t.Context()
	run := uuid.New()
	at := time.Date(2026, 10, 29, 14, 19, 37, 247000000, time.UTC)
	legacy := domain.NewLogEvent(run, "info", "line")
	legacy.Timestamp = at
	if err := store.Append(ctx, run, legacy); err != nil {
		t.Fatal(err)
	}
	deterministic := func(name string) *domain.RunEvent {
		e := domain.NewLogEvent(run, "info", "line")
		e.Timestamp = at
		e.ID = uuid.NewSHA1(run, []byte(name))
		return e
	}
	reread := deterministic("offset-0")
	if err := store.Append(ctx, run, reread); err != nil || !reread.PersistenceReplay || reread.Sequence != legacy.Sequence {
		t.Fatalf("re-read over legacy row = %+v err=%v", reread, err)
	}
	other := domain.NewLogEvent(run, "info", "other line")
	other.Timestamp = at
	other.ID = uuid.NewSHA1(run, []byte("offset-9"))
	first, second := deterministic("offset-20"), deterministic("offset-40")
	for _, e := range []*domain.RunEvent{other, first, second} {
		if err := store.Append(ctx, run, e); err != nil {
			t.Fatal(err)
		}
	}
	// "line" at offset-20 also matches the legacy row, so only the first row exists for it.
	if count, _ := store.Count(ctx, run); count != 2 {
		t.Fatalf("events = %d, want the legacy row and the different line", count)
	}
	fresh1, fresh2 := domain.NewLogEvent(run, "info", "fresh"), domain.NewLogEvent(run, "info", "fresh")
	for i, e := range []*domain.RunEvent{fresh1, fresh2} {
		e.Timestamp, e.ID = at, uuid.NewSHA1(run, []byte(fmt.Sprintf("fresh-%d", i)))
		if err := store.Append(ctx, run, e); err != nil {
			t.Fatal(err)
		}
	}
	if count, _ := store.Count(ctx, run); count != 4 {
		t.Fatalf("events = %d, want deterministic twins of one line both kept", count)
	}
}
