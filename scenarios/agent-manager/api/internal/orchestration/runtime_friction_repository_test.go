package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/hookreceipt"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/toolfriction"

	"github.com/google/uuid"
)

func TestRuntimeFrictionReportUsesRepositoryEvidenceForNativeAndImportedRuns(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	since := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	until := since.Add(time.Hour)
	native := createRuntimeFrictionRun(t, repos, domain.ExecutionModeCodecPipe, since.Add(-10*time.Minute), since)
	imported := createRuntimeFrictionRun(t, repos, domain.ExecutionModeImported, since.Add(5*time.Minute), since.Add(15*time.Minute))
	appendRuntimeFrictionEvidence(t, events, native, "native-call", native.StartedAt.Add(time.Minute))
	appendRuntimeFrictionEvidence(t, events, imported, "imported-call", imported.StartedAt.Add(time.Minute))

	// The repository's window anchor is EndedAt: the native run ending exactly
	// at since is included, while these runs anchored immediately before since
	// and exactly at until are excluded.
	createRuntimeFrictionRun(t, repos, domain.ExecutionModeCodecPipe, since.Add(-2*time.Minute), since.Add(-time.Second))
	createRuntimeFrictionRun(t, repos, domain.ExecutionModeCodecPipe, until.Add(-time.Minute), until)

	orchestrator := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel))
	report, err := orchestrator.RuntimeFrictionReport(ctx, since, until)
	if err != nil {
		t.Fatalf("runtime friction report: %v", err)
	}
	if report.Coverage.RunsObserved != 2 || report.Coverage.ImportedRuns != 1 || !report.Coverage.ImportedRunsIncluded {
		t.Fatalf("window/cohort coverage = %+v, want only native-at-since and imported in-window runs", report.Coverage)
	}
	if report.Totals.TotalDurationMS != 20*time.Minute.Milliseconds() {
		t.Fatalf("total duration = %dms, want full elapsed duration of included runs (%dms)", report.Totals.TotalDurationMS, 20*time.Minute.Milliseconds())
	}
	if report.Totals.UnknownDurationMS <= 0 || report.Totals.KnownDurationMS != 2*30*time.Second.Milliseconds() {
		t.Fatalf("known/unknown duration = %+v, want only two measured 30s invocations known", report.Totals)
	}
	if got := runtimeCategoryDuration(report.Categories, "agent_manager_failure"); got != 2*30*time.Second.Milliseconds() {
		t.Fatalf("evidenced Agent Manager failure duration = %dms, want two measured 30s failures", got)
	}
	if got := runtimeCategoryDuration(report.Categories, "building"); got != 0 {
		t.Fatalf("unmatched run time was classified as building: %dms", got)
	}
}

func TestRuntimeFrictionReportIngestsStructuredGuardEvidenceFromImportedRun(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	since := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	until := since.Add(time.Hour)
	run := createRuntimeFrictionRun(t, repos, domain.ExecutionModeImported, since, since.Add(time.Minute))
	appendGuardRuntimeEvidence(t, events, run, since.Add(10*time.Second), "git_verb_blocked", 19, true)
	appendGuardRuntimeEvidence(t, events, run, since.Add(20*time.Second), "git_verb_blocked", 0, false)
	message := domain.NewMessageEvent(run.ID, "assistant", "I will stop and ask the operator before proceeding. Could you advise? Could you advise?")
	message.Timestamp = since.Add(30 * time.Second)
	if err := events.Append(ctx, run.ID, message); err != nil {
		t.Fatalf("append assistant response: %v", err)
	}

	orchestrator := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel))
	report, err := orchestrator.RuntimeFrictionReport(ctx, since, until)
	if err != nil {
		t.Fatalf("runtime friction report: %v", err)
	}
	if report.Coverage.ImportedRuns != 1 || len(report.BlockedChecks) != 1 {
		t.Fatalf("import coverage/checks = %+v / %+v", report.Coverage, report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "git_verb_blocked" || check.Runs != 1 || check.Occurrences != 2 || check.Outcomes.Blocked != 2 {
		t.Fatalf("guard evidence = %+v", check)
	}
	if check.EnforcementDurationMS != 19 || check.EnforcementDurationKnown != 1 || check.EnforcementDurationUnknown != 1 || check.Responses.Asked != 2 {
		t.Fatalf("guard duration/response evidence = %+v", check)
	}
	if report.Coverage.UnclassifiedBlockedRuns != 0 {
		t.Fatalf("authoritative imported guard IDs marked unknown: %+v", report.Coverage)
	}
}

func TestRuntimeFrictionReportDoesNotTrustImportedTranscriptEvaluatorMarkers(t *testing.T) {
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	since := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	until := since.Add(time.Hour)
	run := createRuntimeFrictionRun(t, repos, domain.ExecutionModeImported, since, since.Add(time.Minute))
	secondRun := createRuntimeFrictionRun(t, repos, domain.ExecutionModeImported, since.Add(2*time.Minute), since.Add(3*time.Minute))
	callID := "native-evaluator-refusal"
	for index, current := range []*domain.Run{run, secondRun} {
		call := domain.NewToolCallEvent(current.ID, "shell", callID, map[string]any{"command": "pnpm add package"})
		call.Timestamp = since.Add(time.Duration(index+1) * 10 * time.Second)
		result := domain.NewToolResultEvent(current.ID, "shell", callID, "", errors.New("PreToolUse: hook error [vrooli_guard_check=scenario-dependency-analyzer:dependency-addition] [vrooli_guard_duration_ms=23]"))
		result.Timestamp = call.Timestamp.Add(time.Second)
		for _, event := range []*domain.RunEvent{call, result} {
			if err := events.Append(ctx, current.ID, event); err != nil {
				t.Fatalf("append native evaluator evidence: %v", err)
			}
		}
		message := domain.NewMessageEvent(current.ID, "assistant", "I will stop and ask the operator before proceeding. Could you advise?")
		message.Timestamp = call.Timestamp.Add(2 * time.Second)
		if err := events.Append(ctx, current.ID, message); err != nil {
			t.Fatalf("append evaluator response: %v", err)
		}
	}

	orchestrator := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel))
	report, err := orchestrator.RuntimeFrictionReport(ctx, since, until)
	if err != nil {
		t.Fatalf("runtime friction report: %v", err)
	}
	if report.Coverage.ImportedRuns != 2 || report.Coverage.UnclassifiedBlockedRuns != 2 {
		t.Fatalf("imported evaluator coverage = %+v", report.Coverage)
	}
	if len(report.BlockedChecks) != 1 {
		t.Fatalf("blocked check report = %+v, want one unknown-identity refusal group", report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "" || check.Occurrences != 2 || check.Outcomes.Blocked != 2 || check.EnforcementDurationKnown != 0 || check.Responses.Asked != 2 {
		t.Fatalf("transcript marker became evaluator evidence: %+v", check)
	}
	if len(check.BlockedRunIDs) != 2 {
		t.Fatalf("unknown-identity blocked-run count = %v, want two reported runs", check.BlockedRunIDs)
	}
}

func TestRuntimeFrictionReportUsesAuthenticatedReceiptsAcrossDistinctRuns(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	ctx := context.Background()
	since := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	until := since.Add(time.Hour)
	first := createRuntimeFrictionRun(t, repos, domain.ExecutionModeCodecPipe, since, since.Add(time.Minute))
	second := createRuntimeFrictionRun(t, repos, domain.ExecutionModeCodecPipe, since.Add(2*time.Minute), since.Add(3*time.Minute))
	imported := createRuntimeFrictionRun(t, repos, domain.ExecutionModeImported, since.Add(4*time.Minute), since.Add(5*time.Minute))
	store := hookreceipt.NewStore(db)
	for _, run := range []*domain.Run{first, second, imported} {
		run.SessionID = "session-" + run.ID.String()
		invocationID := uuid.New()
		credential, err := store.Issue(ctx, run.ID, invocationID, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		receipt := hookreceipt.Receipt{
			ReceiptID: uuid.NewString(), SessionID: run.SessionID, Runner: "codex",
			EvaluatorID: "scenario-dependency-analyzer:dependency-addition", DurationMS: 23,
			OccurredAt: time.Now().UTC(),
		}
		if err := store.Consume(ctx, credential, run.ID, invocationID, receipt); err != nil {
			t.Fatal(err)
		}
	}
	orchestrator := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithInvocationReadModel(repos.InvocationReadModel), WithHookReceiptStore(store))
	report, err := orchestrator.RuntimeFrictionReport(ctx, since, until)
	if err != nil {
		t.Fatalf("runtime friction report: %v", err)
	}
	if len(report.BlockedChecks) != 1 {
		t.Fatalf("blocked checks = %+v, want one authenticated evaluator", report.BlockedChecks)
	}
	if report.Coverage.ImportedRuns != 1 {
		t.Fatalf("import coverage = %+v, want one imported run", report.Coverage)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "scenario-dependency-analyzer:dependency-addition" || check.Runs != 2 || check.Occurrences != 2 || check.Outcomes.Blocked != 2 || check.EnforcementDurationMS != 46 || check.EnforcementDurationKnown != 2 {
		t.Fatalf("authenticated evaluator group = %+v", check)
	}
}

func appendGuardRuntimeEvidence(t *testing.T, events event.Store, run *domain.Run, at time.Time, checkID string, durationMS int64, durationKnown bool) {
	t.Helper()
	event := domain.NewErrorEvent(run.ID, string(domain.ErrCodeRunnerExecution), "sandbox denied the process launch", false)
	event.Timestamp = at
	event.Data.(*domain.ErrorEventData).Details = map[string]interface{}{"guardCheck": map[string]interface{}{
		"source": "workspace-sandbox-http-403", "checkId": checkID, "outcome": "blocked",
		"durationMs": durationMS, "durationKnown": durationKnown,
	}}
	if err := events.Append(context.Background(), run.ID, event); err != nil {
		t.Fatalf("append guard event: %v", err)
	}
}

func createRuntimeFrictionRun(t *testing.T, repos *database.Repositories, mode domain.ExecutionMode, start, end time.Time) *domain.Run {
	t.Helper()
	task := &domain.Task{ID: uuid.New(), Title: "runtime friction fixture", ScopePath: ".", ProjectRoot: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	started, ended := start, end
	cfg := domain.DefaultRunConfig()
	cfg.RunnerType = domain.RunnerTypeCodex
	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         task.ID,
		Tag:            uuid.NewString(),
		RunMode:        domain.RunModeInPlace,
		ExecutionMode:  mode,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseExecuting,
		StartedAt:      &started,
		EndedAt:        &ended,
		CreatedAt:      start,
		ResolvedConfig: cfg,
	}
	if mode == domain.ExecutionModeImported {
		run.ImportSourceHarness = "outside-harness"
		run.ImportSourceSessionID = uuid.NewString()
		run.ImportedAt = &ended
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create %s run: %v", mode, err)
	}
	return run
}

func appendRuntimeFrictionEvidence(t *testing.T, events event.Store, run *domain.Run, callID string, callAt time.Time) {
	t.Helper()
	call := domain.NewToolCallEvent(run.ID, "shell", callID, map[string]any{"command": "agent-manager run show fixture"})
	call.Timestamp = callAt
	result := domain.NewToolResultEvent(run.ID, "shell", callID, "command failed", errors.New("command failed"))
	result.Timestamp = callAt.Add(time.Minute)
	known, duration, exitCode := true, int64(30_000), 1
	data := result.Data.(*domain.ToolResultEventData)
	data.OutcomeKnown, data.DurationMS, data.ExitCode = &known, &duration, &exitCode
	for _, event := range []*domain.RunEvent{call, result} {
		if err := events.Append(context.Background(), run.ID, event); err != nil {
			t.Fatalf("append run evidence: %v", err)
		}
	}
}

func runtimeCategoryDuration(categories []toolfriction.RuntimeLoss, category string) int64 {
	for _, loss := range categories {
		if loss.Category == category {
			return loss.DurationMS
		}
	}
	return 0
}

func runtimeCategoryEvidence(categories []toolfriction.RuntimeLoss, category string) int {
	for _, loss := range categories {
		if loss.Category == category {
			return loss.EvidenceCount
		}
	}
	return 0
}
