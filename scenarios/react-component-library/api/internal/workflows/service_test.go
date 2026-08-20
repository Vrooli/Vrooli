package workflows

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	adoptionmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	componentmocks "react-component-library/internal/components/mocks"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

type fakeDispatcher struct {
	dispatchResult DispatchResult
	dispatchErr    error
	waitResult     DispatchResult
	waitErr        error
	snapshot       RunSnapshot
	snapshotErr    error
	stopped        RunSnapshot
	stopErr        error
	dispatches     []StartInput
	refreshRuns    []string
	stoppedRuns    []string
}

func TestPromotionReadinessRequiresParityExamplesAndCleanOriginReplacement(t *testing.T) {
	ctx := context.Background()
	componentRepo := componentmocks.NewFakeRepository()
	componentSvc := components.NewService(componentRepo)
	component, err := componentRepo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{LibraryID: "react-component-library:DrawerShell", Slug: "drawer-shell", DisplayName: "DrawerShell", LatestVersion: "1.0.0", DraftVersion: "1.0.0-draft.1", Dependencies: []components.AssetDependency{{LibraryID: "react-component-library:useFocusTrap", Version: "1.0.0"}}},
		Versions: []components.ComponentVersion{{Version: "1.0.0-draft.1", ParityReport: &components.IngestParityReport{OriginFiles: []string{"DrawerShell.tsx", "useFocusTrap.ts"}}}},
		Stories:  []components.ComponentStory{{Version: "1.0.0-draft.1", SchemaVersion: 1, Kind: components.StoryKindComponent, ContractJSON: `{"schemaVersion":1,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","args":{}}]}`}},
	})
	require.NoError(t, err)
	adoptionRepo := adoptionmocks.NewFakeRepository()
	adoptionRepo.Seed(adoptions.Adoption{ID: "origin", ComponentID: component.ID, Scenario: "web-console", AdoptedVersion: "1.0.0-draft.1", LibraryVersionStatus: adoptions.LibraryVersionStatusCurrent, LocalStatus: adoptions.LocalStatusClean})
	adoptionSvc := adoptions.NewService(adoptionRepo, nil, nil, scheduletest.New(time.Now()))
	reader := NewPromotionReadinessReader(componentSvc, adoptionSvc)

	got, err := reader.PromotionReadiness(ctx, PromotionReadinessInput{AssetID: component.ID, OriginScenario: "web-console"})
	require.NoError(t, err)
	require.True(t, got.Ready)
	require.Equal(t, []string{"react-component-library:useFocusTrap"}, got.DependencyLibraryIDs)
	require.True(t, got.OriginReplacementPresent)
	require.True(t, got.OriginReplacementClean)

	blocked, err := reader.PromotionReadiness(ctx, PromotionReadinessInput{AssetID: component.ID, OriginScenario: "other"})
	require.NoError(t, err)
	require.False(t, blocked.Ready)
	require.Contains(t, blocked.Blockers, "origin scenario has no recorded replacement adoption at selected version")
}

func (f *fakeDispatcher) Start(_ context.Context, in StartInput) (DispatchResult, error) {
	f.dispatches = append(f.dispatches, in)
	return f.dispatchResult, f.dispatchErr
}

func (f *fakeDispatcher) Wait(_ context.Context, _ string) (DispatchResult, error) {
	if f.waitErr != nil || f.waitResult.ExecutionID != "" || f.waitResult.Status != "" {
		return f.waitResult, f.waitErr
	}
	return f.dispatchResult, f.dispatchErr
}

func (f *fakeDispatcher) Snapshot(_ context.Context, runID string, _ int64) (RunSnapshot, error) {
	f.refreshRuns = append(f.refreshRuns, runID)
	return f.snapshot, f.snapshotErr
}

func (f *fakeDispatcher) Stop(_ context.Context, runID string) (RunSnapshot, error) {
	f.stoppedRuns = append(f.stoppedRuns, runID)
	return f.stopped, f.stopErr
}

func newWorkflowService(t *testing.T, dispatcher *fakeDispatcher) (Service, Repository, *scheduletest.FakeClock) {
	t.Helper()
	database := db.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	clk := scheduletest.New(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepository(database, clk)
	return NewService(repo, dispatcher), repo, clk
}

func extractInput(key string) StartInput {
	return StartInput{Kind: KindExtract, SourceScenario: "sample", SourcePath: "ui/src/Card.tsx", IdempotencyKey: key}
}

func TestServiceStartPersistsDispatcherIdentityAndDeduplicatesActiveWork(t *testing.T) {
	dispatcher := &fakeDispatcher{dispatchResult: DispatchResult{ExecutionID: "execution-1", Status: StatusRunning}}
	svc, _, clk := newWorkflowService(t, dispatcher)

	first, depth, err := svc.Start(context.Background(), extractInput("extract:sample:card"))
	require.NoError(t, err)
	require.Zero(t, depth)
	require.Equal(t, StatusRunning, first.Status)
	require.Equal(t, "execution-1", first.AgentManagerExecutionID)
	require.Equal(t, clk.Now(), first.CreatedAt)

	duplicate, duplicateDepth, err := svc.Start(context.Background(), extractInput("extract:sample:card"))
	require.NoError(t, err)
	require.Equal(t, first.ID, duplicate.ID)
	require.Zero(t, duplicateDepth)
	require.Len(t, dispatcher.dispatches, 1)
}

func TestServiceStartRecordsUnavailableDispatcherWithoutPretendingSuccess(t *testing.T) {
	dispatcher := &fakeDispatcher{dispatchErr: errors.New("agent-manager is unavailable")}
	svc, repo, _ := newWorkflowService(t, dispatcher)

	w, depth, err := svc.Start(context.Background(), extractInput("extract:offline"))
	require.NoError(t, err)
	require.Zero(t, depth)
	require.Equal(t, StatusUnavailable, w.Status)
	require.Equal(t, "agent-manager is unavailable", w.Error)
	require.NotZero(t, w.CompletedAt)

	persisted, err := repo.Get(context.Background(), w.ID)
	require.NoError(t, err)
	require.Equal(t, StatusUnavailable, persisted.Status)
}

func TestServiceStartKeepsExecutionReferenceWhenWaitDisconnects(t *testing.T) {
	dispatcher := &fakeDispatcher{dispatchResult: DispatchResult{ExecutionID: "execution-1", Status: StatusRunning}, waitErr: errors.New("wait disconnected")}
	svc, repo, _ := newWorkflowService(t, dispatcher)

	w, _, err := svc.Start(context.Background(), extractInput("extract:wait-disconnected"))
	require.NoError(t, err)
	require.Equal(t, "execution-1", w.AgentManagerExecutionID)
	require.Equal(t, StatusUnavailable, w.Status)
	require.ErrorContains(t, errors.New(w.Error), "wait disconnected")
	persisted, err := repo.Get(context.Background(), w.ID)
	require.NoError(t, err)
	require.Equal(t, "execution-1", persisted.AgentManagerExecutionID)
}

func TestServiceRefreshStopAndRetryUseDurableWorkflowState(t *testing.T) {
	dispatcher := &fakeDispatcher{dispatchResult: DispatchResult{ExecutionID: "execution-1", Status: StatusRunning}}
	svc, _, clk := newWorkflowService(t, dispatcher)

	w, _, err := svc.Start(context.Background(), extractInput("extract:refresh"))
	require.NoError(t, err)

	clk.Advance(time.Minute)
	dispatcher.snapshot = RunSnapshot{Status: StatusParked, Summary: "needs review", LastEventSequence: 4}
	w, err = svc.Refresh(context.Background(), w.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRunning, w.Status)
	require.Empty(t, dispatcher.refreshRuns)

	clk.Advance(time.Minute)
	dispatcher.stopped = RunSnapshot{Status: StatusStopped, Summary: "stopped by user", LastEventSequence: 5}
	w, err = svc.Stop(context.Background(), w.ID)
	require.NoError(t, err)
	require.Equal(t, StatusStopped, w.Status)
	require.Equal(t, int64(5), w.LastEventSequence)
	require.NotZero(t, w.CompletedAt)
	require.Equal(t, []string{"execution-1"}, dispatcher.stoppedRuns)

	dispatcher.dispatchResult = DispatchResult{ExecutionID: "execution-2", Status: StatusQueued}
	retry, depth, err := svc.Retry(context.Background(), w.ID, "extract:refresh:retry")
	require.NoError(t, err)
	require.Zero(t, depth)
	require.NotEqual(t, w.ID, retry.ID)
	require.Equal(t, "execution-2", retry.AgentManagerExecutionID)
	require.Equal(t, StatusQueued, retry.Status)
}

func TestServiceRejectsIncompleteOrUnspecifiedWorkflowRequests(t *testing.T) {
	svc, _, _ := newWorkflowService(t, &fakeDispatcher{})

	_, _, err := svc.Start(context.Background(), StartInput{IdempotencyKey: "bad"})
	require.ErrorContains(t, err, "kind")
	_, _, err = svc.Start(context.Background(), StartInput{Kind: KindExtract, SourceScenario: "sample", IdempotencyKey: "bad"})
	require.ErrorContains(t, err, "source_scenario and source_path")
	_, _, err = svc.Start(context.Background(), StartInput{Kind: KindAdopt, AssetID: "asset", IdempotencyKey: "bad"})
	require.ErrorContains(t, err, "asset_id and target_scenario")
}

func TestEnsureSchemaMigrationsPreservesExistingAssistedWorkflowRows(t *testing.T) {
	database := db.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), `CREATE TABLE assisted_workflows (id TEXT PRIMARY KEY, status TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = database.ExecContext(context.Background(), `INSERT INTO assisted_workflows (id, status) VALUES ('legacy', 'succeeded')`)
	require.NoError(t, err)

	require.NoError(t, EnsureSchemaMigrations(context.Background(), database))
	require.NoError(t, EnsureSchemaMigrations(context.Background(), database), "migration must be boot-idempotent")
	var executionID string
	var status string
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT agent_manager_execution_id, status FROM assisted_workflows WHERE id='legacy'`).Scan(&executionID, &status))
	require.Empty(t, executionID)
	require.Equal(t, "succeeded", status)
}
