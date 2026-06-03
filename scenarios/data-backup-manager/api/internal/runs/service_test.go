package runs_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/runs"
	runsmocks "data-backup-manager/internal/runs/mocks"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/db"
	"data-backup-manager/internal/testutil/mocks"
)

func newRunsDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(runs.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

// buildService wires the run service against a real sqlite repo plus fakes for
// every other seam. Returns the service and the fakes the test asserts on.
func buildService(t *testing.T, plan runs.PlanForRun, targets map[string]runs.TargetForRun, blockFn func(string, int64) (bool, string, error)) (runs.Service, *mocks.FakeKopiaEngine, *runsmocks.FakeEventSink, *sourcesmocks.FakeCapturer) {
	t.Helper()
	repo := runs.NewSQLiteRepository(newRunsDB(t), mocks.NewFakeClock(time.Time{}))
	eng := &mocks.FakeKopiaEngine{}
	capt := &sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}
	registry := sources.NewRegistry(capt)
	events := &runsmocks.FakeEventSink{}

	dests := map[string]runs.DestinationForRun{"dst-1": {ID: "dst-1", Name: "nightly"}}
	svc := runs.NewService(runs.Deps{
		Repo:         repo,
		Plans:        &runsmocks.FakePlanLookup{Plans: map[string]runs.PlanForRun{plan.ID: plan}},
		Targets:      &runsmocks.FakeTargetLookup{Targets: targets},
		Destinations: &runsmocks.FakeDestinationLookup{Destinations: dests, BlockFn: blockFn},
		Engine:       eng,
		Sources:      registry,
		Events:       events,
		Clock:        mocks.NewFakeClock(time.Time{}),
		StagingRoot:  t.TempDir(),
		// Inline executor: TriggerRun runs the job to completion synchronously so
		// the test can GetRun the terminal run without polling.
		Executor: runsmocks.NewSyncExecutor(),
	})
	return svc, eng, events, capt
}

// TestCatalog_ListAndLastSuccess proves DBM-CAT-001: after a successful run,
// ListRuns shows history and ListTargetStatus shows last-success per target.
func TestCatalog_ListAndLastSuccess(t *testing.T) {
	ctx := context.Background()
	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"dst-1"}, KeepLatest: 3}
	targets := map[string]runs.TargetForRun{
		"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "a"},
		"t2": {ID: "t2", Kind: sources.KindFilesystem, Locator: "b"},
	}
	svc, eng, events, _ := buildService(t, plan, targets, nil)

	pending, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}
	// TriggerRun returns immediately with a non-terminal run; the inline
	// executor has driven it to terminal by the time Submit returned.
	if pending.Status != runs.RunPending {
		t.Fatalf("triggered run status = %s, want pending (async contract)", pending.Status)
	}
	run, err := svc.GetRun(ctx, pending.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.Status != runs.RunCompleted {
		t.Fatalf("run status = %s, want completed", run.Status)
	}
	if len(run.Outcomes) != 2 {
		t.Fatalf("outcomes = %d, want 2", len(run.Outcomes))
	}
	// Engine snapshotted both targets.
	var snaps int
	for _, c := range eng.Calls {
		if len(c) >= 14 && c[:14] == "SnapshotCreate" {
			snaps++
		}
	}
	if snaps != 2 {
		t.Fatalf("SnapshotCreate calls = %d, want 2 (%v)", snaps, eng.Calls)
	}
	// Event emitted.
	if len(events.Events) != 1 || events.Events[0].Status != runs.RunCompleted || events.Events[0].Succeeded != 2 {
		t.Fatalf("event wrong: %+v", events.Events)
	}

	// Catalog: one run in history.
	list, err := svc.ListRuns(ctx, "plan-1", 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListRuns: n=%d err=%v", len(list), err)
	}
	// Last-success per target recorded for both.
	statuses, err := svc.ListTargetStatus(ctx, nil)
	if err != nil {
		t.Fatalf("ListTargetStatus: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("target statuses = %d, want 2", len(statuses))
	}
	for _, s := range statuses {
		if s.LastSuccessAt.IsZero() {
			t.Errorf("target %s missing last_success", s.TargetID)
		}
		if s.LastRunStatus != runs.RunCompleted {
			t.Errorf("target %s last_run_status = %s", s.TargetID, s.LastRunStatus)
		}
	}
}

// TestSnapshotMetadataStampsOwnerNameRunDestination proves each snapshot is
// created with self-identifying kopia metadata (override-source, description,
// owner/name/kind/run/destination tags) and that the target locator never
// leaks into tags/description/override-source.
func TestSnapshotMetadataStampsOwnerNameRunDestination(t *testing.T) {
	ctx := context.Background()
	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: []string{"t1"}, DestinationIDs: []string{"dst-1"}}
	const secretLocator = "/very/sensitive/path"
	targets := map[string]runs.TargetForRun{
		"t1": {ID: "t1", Owner: "acme", Name: "db", Kind: sources.KindFilesystem, Locator: secretLocator},
	}
	svc, eng, _, _ := buildService(t, plan, targets, nil)

	run, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}
	if len(eng.SnapshotMetas) != 1 {
		t.Fatalf("snapshot metas = %d, want 1", len(eng.SnapshotMetas))
	}
	meta := eng.SnapshotMetas[0]
	if meta.OverrideSource != "dbm://acme/db" {
		t.Errorf("override-source = %q, want dbm://acme/db", meta.OverrideSource)
	}
	wantTags := map[string]bool{
		"dbm:true": true, "dbm.target_id:t1": true, "dbm.kind:filesystem": true,
		"dbm.run_id:" + run.ID: true, "dbm.destination_id:dst-1": true,
		"dbm.owner:acme": true, "dbm.name:db": true,
	}
	for _, tag := range meta.Tags {
		delete(wantTags, tag)
	}
	if len(wantTags) != 0 {
		t.Errorf("missing tags: %v (got %v)", wantTags, meta.Tags)
	}
	// Locator must never appear anywhere in the metadata.
	for _, tag := range meta.Tags {
		if strings.Contains(tag, secretLocator) {
			t.Errorf("locator leaked into tag %q", tag)
		}
	}
	if strings.Contains(meta.Description, secretLocator) || strings.Contains(meta.OverrideSource, secretLocator) {
		t.Errorf("locator leaked into description/override-source: %q / %q", meta.Description, meta.OverrideSource)
	}
}

func TestListTargetStatus_DefaultsToActiveCatalogTargets(t *testing.T) {
	ctx := context.Background()
	repo := runsmocks.NewFakeRepository()
	now := time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)
	seedRun := func(runID, planID, targetID string, startedAt time.Time) {
		t.Helper()
		if _, err := repo.CreateRun(ctx, runs.Run{ID: runID, PlanID: planID, Status: runs.RunPending, StartedAt: startedAt}); err != nil {
			t.Fatalf("seed run %s: %v", runID, err)
		}
		if err := repo.SaveOutcome(ctx, runID, runs.TargetOutcome{TargetID: targetID, Status: runs.OutcomeSucceeded, FinishedAt: startedAt}); err != nil {
			t.Fatalf("seed outcome %s: %v", runID, err)
		}
		if err := repo.FinishRun(ctx, runID, runs.RunCompleted, "", startedAt); err != nil {
			t.Fatalf("finish run %s: %v", runID, err)
		}
	}
	seedRun("run-old", "plan-deleted", "deleted-target", now.Add(-time.Hour))
	seedRun("run-current", "plan-current", "current-target", now)

	svc := runs.NewService(runs.Deps{
		Repo:          repo,
		ActiveTargets: &runsmocks.FakeActiveTargetLookup{TargetIDs: []string{"current-target"}},
		Clock:         mocks.NewFakeClock(now),
		Executor:      runsmocks.NewSyncExecutor(),
	})
	statuses, err := svc.ListTargetStatus(ctx, nil)
	if err != nil {
		t.Fatalf("ListTargetStatus: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("statuses = %+v, want only current target", statuses)
	}
	if statuses[0].TargetID != "current-target" {
		t.Fatalf("target id = %q, want current-target", statuses[0].TargetID)
	}

	history, err := svc.ListTargetStatus(ctx, []string{"deleted-target"})
	if err != nil {
		t.Fatalf("ListTargetStatus explicit deleted target: %v", err)
	}
	if len(history) != 1 || history[0].TargetID != "deleted-target" {
		t.Fatalf("explicit target lookup = %+v, want deleted-target history", history)
	}
}

// TestRun_PartialFailure proves a single target's capture failure does not
// abort the others: the run is partial_failed with one success + one failure.
func TestRun_PartialFailure(t *testing.T) {
	ctx := context.Background()
	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: []string{"good", "bad"}, DestinationIDs: []string{"dst-1"}}
	targets := map[string]runs.TargetForRun{
		"good": {ID: "good", Kind: sources.KindFilesystem, Locator: "good"},
		"bad":  {ID: "bad", Kind: sources.KindFilesystem, Locator: "bad"},
	}
	svc, _, events, capt := buildService(t, plan, targets, nil)
	capt.CaptureFn = func(_ context.Context, spec sources.CaptureSpec) (sources.Artifact, error) {
		if spec.Locator == "bad" {
			return sources.Artifact{}, errors.New("capture boom")
		}
		return sources.Artifact{Path: spec.StageDir, Bytes: 10}, nil
	}

	pending, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}
	run, err := svc.GetRun(ctx, pending.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.Status != runs.RunPartialFailed {
		t.Fatalf("status = %s, want partial_failed", run.Status)
	}
	var ok, failed int
	for _, o := range run.Outcomes {
		switch o.Status {
		case runs.OutcomeSucceeded:
			ok++
		case runs.OutcomeFailed:
			failed++
		}
	}
	if ok != 1 || failed != 1 {
		t.Fatalf("outcomes ok=%d failed=%d, want 1/1", ok, failed)
	}
	if events.Events[0].Status != runs.RunPartialFailed {
		t.Fatalf("event status = %s", events.Events[0].Status)
	}
}

// TestRun_CapBlock proves the storage-limit block: when the destination would
// exceed its cap, the target is BLOCKED with no snapshot written (never
// evicts). With the only target blocked, the run is failed.
func TestRun_CapBlock(t *testing.T) {
	ctx := context.Background()
	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: []string{"t1"}, DestinationIDs: []string{"dst-1"}}
	targets := map[string]runs.TargetForRun{"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "a"}}
	blockAll := func(string, int64) (bool, string, error) { return true, "over cap", nil }
	svc, eng, _, _ := buildService(t, plan, targets, blockAll)

	pending, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}
	run, err := svc.GetRun(ctx, pending.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.Status != runs.RunFailed {
		t.Fatalf("status = %s, want failed", run.Status)
	}
	if len(run.Outcomes) != 1 || run.Outcomes[0].Status != runs.OutcomeBlocked {
		t.Fatalf("outcome = %+v, want blocked", run.Outcomes)
	}
	// No snapshot was written and nothing was deleted — block, never evict.
	for _, c := range eng.Calls {
		if len(c) >= 14 && c[:14] == "SnapshotCreate" {
			t.Fatalf("SnapshotCreate must not be called when cap-blocked: %v", eng.Calls)
		}
	}
}
