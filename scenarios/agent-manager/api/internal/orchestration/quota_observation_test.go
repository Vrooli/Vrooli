package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/transcriptindex"

	"github.com/sirupsen/logrus"

	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"

	"github.com/google/uuid"
)

type recordingQuotaObservationStore struct {
	rows []pricing.QuotaObservation
}

func TestTypedNilQuotaOwnerIsUnavailable(t *testing.T) {
	var owner *recordingQuotaObservationStore
	o := &Orchestrator{}
	WithQuotaObservationStore(owner)(o)
	if o.quotaObservations != nil || o.quotaObservationsDisabled {
		t.Fatal("typed-nil owner must remain unavailable, not enabled or disabled")
	}
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

// runEventSinkPaths builds each live run-event sink the way its production
// owner does: the fresh headless executor, the interactive/continuation
// coordinator (CoordinatorDeps.NewSink is o.runEventSink) and the recovery
// reconciler wired through SetReconciler.
func runEventSinkPaths(t *testing.T, o *Orchestrator) map[string]func(*domain.Run) runner.EventSink {
	t.Helper()
	return map[string]func(*domain.Run) runner.EventSink{
		"headless executor": func(run *domain.Run) runner.EventSink {
			return o.newRunExecutor(t.Context(), run, nil, nil, "", "", "", nil).createEventSink()
		},
		"interactive tail": func(run *domain.Run) runner.EventSink { return o.runEventSink(run.ID) },
		"recovery": func(run *domain.Run) runner.EventSink {
			r := NewReconciler(o.runs, o.runners, WithReconcilerEvents(o.events))
			o.SetReconciler(r)
			return r.recoveryEventSink(run.ID)
		},
	}
}

// codexQuotaEvent is one parsed Codex token_count rate-limit frame.
func codexQuotaEvent(t *testing.T, runID uuid.UUID, percent int) *domain.RunEvent {
	t.Helper()
	line := fmt.Sprintf(`{"timestamp":"2026-10-09T07:33:59Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":%d,"window_minutes":10080,"resets_at":1791948560}}}}`, percent)
	got := codecs.NewCodexForTest().NewTranscriptParser().ParseTranscriptLine(runID, line)
	if got.Err != nil || len(got.Events) != 1 || got.Events[0].EventType != domain.EventTypeQuota {
		t.Fatalf("codec quota frame = %+v", got)
	}
	return got.Events[0]
}

func TestEveryLiveSinkPathRecordsProviderQuotaObservations(t *testing.T) {
	for _, name := range []string{"headless executor", "interactive tail", "recovery"} {
		t.Run(name, func(t *testing.T) {
			repos, events, run := tokenCapFixture(t, "gpt-6-luna")
			quota := &recordingQuotaObservationStore{}
			o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithQuotaObservationStore(quota), WithClock(func() time.Time { return time.Date(2026, 10, 9, 7, 40, 0, 0, time.UTC) }))
			sink := runEventSinkPaths(t, o)[name](run)
			frame := codexQuotaEvent(t, run.ID, 42)
			if err := sink.Emit(frame); err != nil {
				t.Fatal(err)
			}
			// Re-appending the same event is a persistence replay: no second hook run.
			if err := sink.Emit(frame); err != nil {
				t.Fatal(err)
			}
			if len(quota.rows) != 1 {
				t.Fatalf("observations = %+v, want one per persisted frame", quota.rows)
			}
			row := quota.rows[0]
			if row.Provider != "openai" || row.Pool != "primary" || row.Window != "10080m" || row.UsedPercent == nil || *row.UsedPercent != 42 || row.Standing != pricing.QuotaStandingAvailable {
				t.Fatalf("observation = %+v", row)
			}
			if row.SourceRunID != run.ID.String() || !strings.Contains(row.EvidenceRef, frame.ID.String()) || row.Freshness != pricing.ObservationFresh || row.ID != pricing.QuotaObservationID(&row) {
				t.Fatalf("observation lineage/freshness/identity = %+v", row)
			}
			if row.Used != nil || row.Limit != nil {
				t.Fatalf("observation attributed unreported amount = %+v", row)
			}
		})
	}
}

func TestImportRecordsQuotaButNeverActsOnImportedRuns(t *testing.T) {
	for _, recall := range []bool{true, false} {
		t.Run(fmt.Sprintf("recall=%v", recall), func(t *testing.T) {
			repos, events, run := tokenCapFixture(t, "gpt-6-luna")
			quota := &recordingQuotaObservationStore{}
			policy := DefaultTokenCapPolicy()
			policy.Mode, policy.Caps = TokenCapEnforce, map[string]int64{tokenCapProfile: 1}
			o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithQuotaObservationStore(quota), WithTokenCapPolicy(policy))
			host := &transcriptIndexHost{o: o, redact: map[string]bool{}}
			frame := codexQuotaEvent(t, run.ID, 7)
			if err := host.Record(t.Context(), run.ID.String(), []*domain.RunEvent{frame, usageEvent(run.ID, "gpt-6-luna", 5000, 1)}, recall); err != nil {
				t.Fatal(err)
			}
			if len(quota.rows) != 1 || *quota.rows[0].UsedPercent != 7 {
				t.Fatalf("imported quota observations = %+v", quota.rows)
			}
			wantRef := "agent-manager://runs/" + run.ID.String() + "/events/" + frame.ID.String()
			if !recall {
				wantRef = "agent-manager://runs/" + run.ID.String()
			}
			if quota.rows[0].EvidenceRef != wantRef {
				t.Fatalf("evidence ref = %q, want %q (an unretained event is never referenced)", quota.rows[0].EvidenceRef, wantRef)
			}
			if _, tracked := o.tokenCaps.runs.Load(run.ID); tracked {
				t.Fatal("token cap observed usage on an imported run")
			}
		})
	}
}

func TestImportTranscriptRecordsCodexQuotaFrames(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatal(err)
	}
	quota := &recordingQuotaObservationStore{}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(eventStore), WithRunners(registry), WithRunStateRoot(t.TempDir()), WithQuotaObservationStore(quota),
		WithConfig(OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}), WithInvocationReadModel(repos.InvocationReadModel))
	NewTranscriptIndex(o, repos.TranscriptIndex, transcriptindex.Config{Roots: transcriptindex.RootConfig{Home: t.TempDir()}, RecallMaxBytes: transcriptindex.DefaultRecallMaxBytes})
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	body := `{"timestamp":"2026-10-09T07:00:00Z","type":"session_meta","payload":{"id":"quota-import"}}` + "\n" +
		`{"timestamp":"2026-10-09T07:33:59Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":20,"window_minutes":10080,"resets_at":1791948560}}}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := o.ImportTranscript(t.Context(), ImportTranscriptRequest{Path: path, RunnerType: domain.RunnerTypeCodex, SourceHarness: "codex", SourceSessionID: "quota-import"}); err != nil {
		t.Fatal(err)
	}
	if len(quota.rows) != 1 || quota.rows[0].Provider != "openai" || *quota.rows[0].UsedPercent != 20 {
		t.Fatalf("import observations = %+v", quota.rows)
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
				if err := os.WriteFile(path, []byte(line+later), 0o600); err != nil {
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
	if err := os.WriteFile(path, []byte(line+line), 0o600); err != nil {
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
