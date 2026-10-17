package orchestration

import (
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/protoconv"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/pricing"

	"github.com/google/uuid"
)

type recordingQuotaObservationStore struct {
	rows []pricing.QuotaObservation
}

func (s *recordingQuotaObservationStore) RecordQuotaObservation(_ context.Context, observation *pricing.QuotaObservation) error {
	s.rows = append(s.rows, *observation)
	return nil
}

func (s *recordingQuotaObservationStore) ListQuotaObservations(_ context.Context, provider, pool, window string, limit int) ([]pricing.QuotaObservation, error) {
	result := make([]pricing.QuotaObservation, 0, len(s.rows))
	for i := len(s.rows) - 1; i >= 0 && len(result) < limit; i-- {
		row := s.rows[i]
		if (provider != "" && row.Provider != provider) || (pool != "" && row.Pool != pool) || (window != "" && row.Window != window) {
			continue
		}
		result = append(result, row)
	}
	return result, nil
}

func TestRunEventSinkPersistsProviderQuotaObservationAfterEventEvidence(t *testing.T) {
	runID := uuid.New()
	observedAt := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	store := mocks.NewFakeEventStore()
	quota := &recordingQuotaObservationStore{}
	o := &Orchestrator{
		events:            store,
		quotaObservations: quota,
		clock:             func() time.Time { return observedAt.Add(time.Minute) },
	}
	percent := 42.5
	event := domain.NewRateLimitEvent(runID, "5h", "native quota", nil, 0)
	event.Timestamp = observedAt
	data := event.Data.(*domain.RateLimitEventData)
	data.Provider = "openai"
	data.Pool = "primary"
	data.UsedPercent = &percent
	data.WindowMinutes = 300
	data.Provenance = "codex:event_msg.token_count.rate_limits"

	if err := o.runEventSink(runID).Emit(event); err != nil {
		t.Fatal(err)
	}
	rows, err := quota.ListQuotaObservations(context.Background(), "openai", "primary", "300m", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("quota rows=%+v err=%v", rows, err)
	}
	row := rows[0]
	if row.SourceRunID != runID.String() || row.EvidenceRef == "" || row.Freshness != pricing.ObservationFresh {
		t.Fatalf("observation lineage/freshness=%+v", row)
	}
	if row.Used != nil || row.Limit != nil || row.UsedPercent == nil || *row.UsedPercent != percent {
		t.Fatalf("observation attributed unreported amount=%+v", row)
	}
}

func TestNativeQuotaCodecSinkDurableObservationAndPublicEvent(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, eventStore, _ := testutil.SetupTestReposWithDB(t, db)
	task := &domain.Task{ID: uuid.New(), Title: "native quota", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	quota := database.NewPricingRepository(db, logrus.New()).(pricing.QuotaObservationRepository)
	o := &Orchestrator{events: eventStore, quotaObservations: quota, clock: func() time.Time { return time.Date(2026, 10, 9, 7, 34, 0, 0, time.UTC) }}
	p := codecs.NewCodexForTest().NewTranscriptParser()
	got := p.ParseTranscriptLine(run.ID, `{"timestamp":"2026-10-09T07:33:59Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":20,"window_minutes":10080,"resets_at":1791948560}}}}`)
	if got.Err != nil || len(got.Events) != 1 {
		t.Fatalf("codec=%+v", got)
	}
	evt := got.Events[0]
	if err := o.runEventSink(run.ID).Emit(evt); err != nil {
		t.Fatal(err)
	}
	// This check exercises quota observation identity directly; event replay is qualified below.
	// Qualify observation replay through its existing immutable-row owner below.
	rows, err := quota.ListQuotaObservations(t.Context(), "openai", "primary", "10080m", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("observations=%+v error=%v", rows, err)
	}
	row := rows[0]
	if err := quota.RecordQuotaObservation(t.Context(), &row); err != nil {
		t.Fatal(err)
	}
	replayed, err := quota.ListQuotaObservations(t.Context(), "openai", "primary", "10080m", 10)
	if err != nil || len(replayed) != 1 || replayed[0].ID != row.ID {
		t.Fatalf("replay=%+v error=%v", replayed, err)
	}
	reset := time.Date(2026, 10, 14, 3, 29, 20, 0, time.UTC)
	if row.SourceRunID != run.ID.String() || !strings.Contains(row.EvidenceRef, evt.ID.String()) || row.UsedPercent == nil || *row.UsedPercent != 20 || row.ResetAt == nil || !row.ResetAt.Equal(reset) || row.Used != nil || row.Limit != nil || row.Remaining != nil || row.Standing == pricing.QuotaStandingExhausted {
		t.Fatalf("observation=%+v", row)
	}
	durable, err := eventStore.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(durable) == 0 {
		t.Fatalf("durable=%+v error=%v", durable, err)
	}
	public := protoconv.RunEventToProto(durable[0]).GetRateLimit()
	if public == nil || public.UsedPercent == nil || public.GetUsedPercent() != 20 || public.GetProvider() != "openai" || public.GetPool() != "primary" || public.GetWindowMinutes() != 10080 || public.GetResetTime() == nil || !public.GetResetTime().AsTime().Equal(reset) {
		t.Fatalf("public=%+v", public)
	}
}

type warningSinkFailure struct{ runner.EventSink }

func (s warningSinkFailure) Emit(evt *domain.RunEvent) error {
	if data, ok := evt.Data.(*domain.LogEventData); ok && data.Level == "warning" {
		return errors.New("warning persistence refused")
	}
	return s.EventSink.Emit(evt)
}

func TestCodexWarningPersistencePrecedesCursorAndReplayContinues(t *testing.T) {
	for _, dialect := range []string{"rollout", "flat"} {
		for _, failure := range []string{"none", "warning sink", "cursor"} {
			t.Run(dialect+"/"+failure, func(t *testing.T) {
				repos, eventStore, cleanup := testutil.SetupTestRepos(t)
				t.Cleanup(cleanup)
				runID := uuid.New()
				task := &domain.Task{ID: uuid.New(), Title: "warning cursor", ScopePath: ".", Status: domain.TaskStatusQueued}
				if err := repos.Tasks.Create(t.Context(), task); err != nil {
					t.Fatal(err)
				}
				run := &domain.Run{ID: runID, TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting}
				if err := repos.Runs.Create(t.Context(), run); err != nil {
					t.Fatal(err)
				}
				broadcaster := &recordingBroadcaster{}
				o := &Orchestrator{events: eventStore, broadcaster: broadcaster}
				var sink runner.EventSink = o.runEventSink(runID)
				if failure == "warning sink" {
					sink = warningSinkFailure{sink}
				}
				line := `{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"mixed","output":[{"type":"input_text","text":"first"},{"type":"image","future":true}]}}` + "\n"
				later := `{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"later","output":"valid later"}}` + "\n" + `{"type":"event_msg","payload":{"type":"task_complete","turn_id":"after-warning"}}` + "\n"
				if dialect == "flat" {
					line = `{"type":"item.completed","item":{"type":"tool_result","id":"mixed","name":"exec","output":"{\"metadata\":{\"exit_code\":\"bad\"}}"}}` + "\n"
				}
				path := filepath.Join(t.TempDir(), "rollout.jsonl")
				if err := os.WriteFile(path, []byte(line+later), 0600); err != nil {
					t.Fatal(err)
				}
				p := codecs.NewCodexForTest().NewTranscriptParser()
				persistedCursor := int64(0)
				advance := func(cursor, lastSeq int64) error {
					events, err := eventStore.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
					if err != nil {
						return err
					}
					found := false
					for _, evt := range events {
						if log, ok := evt.Data.(*domain.LogEventData); ok && log.Level == "warning" {
							found = true
						}
					}
					if !found {
						return errors.New("cursor advanced without persisted warning")
					}
					if failure == "cursor" {
						return errors.New("cursor persistence refused")
					}
					persistedCursor = cursor
					return nil
				}
				localCursor, terminal, err := runner.Consume(t.Context(), runner.ConsumeArgs{RunID: runID, Transcript: path, EndAt: int64(len(line)), ParseFn: p.ParseTranscriptLine, EventSink: sink, OnAdvance: advance})
				events, readErr := eventStore.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
				if readErr != nil {
					t.Fatal(readErr)
				}
				if terminal != nil || localCursor != int64(len(line)) {
					t.Fatalf("cursor=%d terminal=%+v", localCursor, terminal)
				}
				if failure != "none" {
					if err == nil || persistedCursor != 0 {
						t.Fatalf("err=%v durable cursor=%d", err, persistedCursor)
					}
					if failure == "warning sink" && len(events) != 1 {
						t.Fatalf("events=%+v", events)
					}
					if failure == "cursor" {
						var cursorError *runner.TranscriptCursorAdvanceError
						if !errors.As(err, &cursorError) || len(events) != 2 {
							t.Fatalf("events=%+v err=%v", events, err)
						}
					}
					recovered := codecs.NewCodexForTest().NewTranscriptParser()
					_, replayedTerminal, replayErr := runner.Consume(t.Context(), runner.ConsumeArgs{RunID: runID, Transcript: path, StartAt: persistedCursor, ParseFn: recovered.ParseTranscriptLine, EventSink: o.runEventSink(runID)})
					replayed, readErr := eventStore.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
					if replayErr != nil || readErr != nil || replayedTerminal == nil || !replayedTerminal.Success || len(replayed) != 3 {
						t.Fatalf("replayed=%+v terminal=%+v error=%v read=%v", replayed, replayedTerminal, replayErr, readErr)
					}
					if broadcaster.eventCount() != 3 {
						t.Fatalf("replayed broadcast count=%d", broadcaster.eventCount())
					}
					for index, evt := range replayed {
						if evt.Sequence != int64(index) {
							t.Fatalf("sequence=%d want=%d", evt.Sequence, index)
						}
					}

					return
				}
				if err != nil || persistedCursor != int64(len(line)) || len(events) != 2 {
					t.Fatalf("events=%+v cursor=%d err=%v", events, persistedCursor, err)
				}
				// Owner restart restores parser correlation by priming without a sink; it
				// must neither persist diagnostics twice nor bypass the durable cursor.
				fresh := codecs.NewCodexForTest().NewTranscriptParser()
				fresh.ParseTranscriptLine(runID, strings.TrimSuffix(line, "\n"))
				_, terminal, err = runner.Consume(t.Context(), runner.ConsumeArgs{RunID: runID, Transcript: path, StartAt: persistedCursor, ParseFn: fresh.ParseTranscriptLine, EventSink: sink})
				events, readErr = eventStore.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
				if err != nil || readErr != nil || terminal == nil || !terminal.Success || len(events) != 3 || events[2].Data.(*domain.ToolResultEventData).ToolCallID != "later" {
					t.Fatalf("events=%+v terminal=%+v err=%v read=%v", events, terminal, err, readErr)
				}
			})
		}
	}
}

func TestCodexIdenticalRecordsRetainDistinctSourceOccurrences(t *testing.T) {
	repos, store, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	id := uuid.New()
	task := &domain.Task{ID: uuid.New(), Title: "distinct occurrences", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Create(t.Context(), &domain.Run{ID: id, TaskID: task.ID, Status: domain.RunStatusRunning}); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"same","output":[{"type":"image","future":true}]}}` + "\n"
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte(line+line), 0600); err != nil {
		t.Fatal(err)
	}
	broadcaster := &recordingBroadcaster{}
	o := &Orchestrator{events: store, broadcaster: broadcaster}
	for attempt := 0; attempt < 2; attempt++ {
		p := codecs.NewCodexForTest().NewTranscriptParser()
		if _, _, err := runner.Consume(t.Context(), runner.ConsumeArgs{RunID: id, Transcript: path, ParseFn: p.ParseTranscriptLine, EventSink: o.runEventSink(id)}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.Get(t.Context(), id, event.GetOptions{AfterSequence: -1})
	if err != nil || len(rows) != 4 || rows[0].ID == rows[2].ID || rows[1].ID == rows[3].ID || broadcaster.eventCount() != 4 {
		t.Fatalf("rows=%+v err=%v broadcasts=%d", rows, err, broadcaster.eventCount())
	}
}
