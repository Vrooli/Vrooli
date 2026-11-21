package orchestration_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modules"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
)

// Exercise real schema, repositories, runner stream callbacks, event sink and
// terminal finalization. A passed static storage analysis cannot prove these
// asynchronous effects kept the request's exact leased generation.
func TestRunExecutorKeepsPinnedSQLThroughEventsAndFinalization(t *testing.T) {
	// Match the serving owner's WAL and bounded reader/writer pool. A bare
	// rollback-journal fixture adds unrelated lock contention to the handoff.
	dsn := func(name string) string {
		return "file:" + filepath.Join(t.TempDir(), name) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	}
	routed, err := coredb.Open(t.Context(), coredb.Config{Driver: coredb.DriverSQLite, DSN: dsn("primary.db"), MaxOpenConns: 5, MaxIdleConns: 5})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = routed.Close() })
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	db := database.NewRoutedDB(routed, logger)
	if err := db.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return coredb.EnsureSchemas(ctx, pool, modules.AllSchemas()...)
	})
	if err := routed.InstallTestPool(t.Context(), dsn("leased.db"), "executor-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx, err := routed.PinTestPool(t.Context(), "executor-lease")
	if err != nil {
		t.Fatal(err)
	}
	repos := database.NewRepositories(db, logger)
	events := event.NewSQLiteStore(db, logger)
	f := newInPlaceFixtures(t)
	f.task.ProjectRoot = t.TempDir()
	f.run.ResolvedConfig = &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, Model: "mock-model", MaxTurns: 1, Timeout: time.Minute}
	for _, create := range []func() error{
		func() error { return repos.Profiles.Create(ctx, f.profile) },
		func() error { return repos.Tasks.Create(ctx, f.task) },
		func() error { return repos.Runs.Create(ctx, f.run) },
	} {
		if err := create(); err != nil {
			t.Fatal(err)
		}
	}
	f.run, err = repos.Runs.Get(ctx, f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Update(ctx, f.run); err != nil {
		var dbErr *domain.DatabaseError
		if errors.As(err, &dbErr) {
			t.Fatalf("fixture run update: %v; cause=%v", err, dbErr.Cause)
		}
		t.Fatal(err)
	}
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		if !coredb.IsTestMode(ctx) {
			t.Error("runner lost its leased operation context")
		}
		if req.Transcript != nil {
			if err := req.Transcript.OnProcessStart(900001, 900001); err != nil {
				return nil, err
			}
			if err := req.Transcript.OnSessionID("leased-runner-session"); err != nil {
				return nil, err
			}
			if err := req.Transcript.OnAdvance(7, 1); err != nil {
				return nil, err
			}
		}
		if err := req.EventSink.Emit(domain.NewLogEvent(f.run.ID, "info", "LEASED_RUNNER_EVENT")); err != nil {
			return nil, err
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "LEASED_RUNNER_DONE"}}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mock)
	executor := orchestration.NewRunExecutor(&routingObservedRuns{RunRepository: repos.Runs, t: t}, registry, nil, events, f.run, f.task, f.profile, "fixture", "")
	executor.WithRunStateRoot(t.TempDir())
	executor.Execute(ctx)
	t.Logf("executor receipt: status=%s phase=%s error=%s", f.run.Status, f.run.Phase, f.run.ErrorMsg)
	stored, err := repos.Runs.Get(ctx, f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.RunStatusComplete || stored.Phase != domain.RunPhaseCompleted {
		t.Errorf("leased run did not reach its final receipt: status=%s phase=%s error=%s", stored.Status, stored.Phase, stored.ErrorMsg)
	}
	if stored.RunnerPID != 900001 || stored.SessionID != "leased-runner-session" || stored.TranscriptCursor != 7 {
		t.Errorf("stream binding lost: pid=%d session=%q cursor=%d", stored.RunnerPID, stored.SessionID, stored.TranscriptCursor)
	}
	leasedEvents, err := events.Get(ctx, f.run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(leasedEvents) == 0 {
		t.Fatalf("leased events missing: count=%d err=%v", len(leasedEvents), err)
	}
	for _, table := range []string{"runs", "run_events"} {
		var count int
		if err := routed.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Errorf("primary %s changed: count=%d err=%v", table, count, err)
		}
	}
}

type routingObservedRuns struct {
	repository.RunRepository
	t *testing.T
}

func (r *routingObservedRuns) Update(ctx context.Context, run *domain.Run) error {
	err := r.RunRepository.Update(ctx, run)
	var dbErr *domain.DatabaseError
	if errors.As(err, &dbErr) {
		r.t.Logf("run update failed: cause=%v", dbErr.Cause)
	}
	return err
}
