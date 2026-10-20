package phases

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

type setupConflictRuns struct {
	repository.RunRepository
	current *domain.Run
}

func (r setupConflictRuns) Get(context.Context, uuid.UUID) (*domain.Run, error) {
	copy := *r.current
	return &copy, nil
}

func (setupConflictRuns) Update(context.Context, *domain.Run) error {
	return errors.New("run update lost lifecycle race")
}

func TestCreateSandboxWorkspace_CleansOnlyKnownOwnedSandboxAfterCancellationWins(t *testing.T) {
	cases := []struct {
		name                string
		owner               string
		missingMetadata     bool
		cancelled           bool
		unchangedVersion    bool
		changedOwnerEpoch   bool
		currentSandboxID    *uuid.UUID
		createErr           error
		deleteOn            []domain.SandboxLifecycleEvent
		stopOn              []domain.SandboxLifecycleEvent
		autoApply           *bool
		finalizationPending bool
		wantDelete          int
		wantStop            int
	}{
		{name: "owned cancellation applies configured delete without required apply", cancelled: true, autoApply: boolPtr(false), deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}, wantDelete: 1},
		{name: "owned cancellation preserves when policy says retain", cancelled: true},
		{name: "owned cancellation applies configured stop without required apply", cancelled: true, autoApply: boolPtr(false), stopOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}, wantStop: 1},
		{name: "required apply finalization preserves sandbox", cancelled: true, autoApply: boolPtr(true), finalizationPending: true, deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "wrong owner metadata is untouched", owner: uuid.NewString(), cancelled: true, deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "missing owner metadata is untouched", missingMetadata: true, cancelled: true, deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "non-cancellation update race is untouched", cancelled: false, deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "unchanged lifecycle version is not cancellation proof", cancelled: true, unchangedVersion: true, deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "changed owner epoch is preserved", cancelled: true, changedOwnerEpoch: true, deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "replacement sandbox is preserved", cancelled: true, currentSandboxID: ptrUUID(uuid.New()), deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
		{name: "provider error without returned ID does not guess cleanup", cancelled: true, createErr: errors.New("provider response outcome uncertain"), deleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunCancelled}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			task := fixtures.NewTask(t)
			run := fixtures.NewRun(t, task.ID, uuid.Nil)
			run.RunMode = domain.RunModeSandboxed
			run.LifecycleVersion = 4
			run.Status = domain.RunStatusRunning
			run.SandboxConfig = &domain.SandboxConfig{AutoApply: tc.autoApply, Lifecycle: domain.SandboxLifecycleConfig{DeleteOn: tc.deleteOn, StopOn: tc.stopOn}}
			if tc.finalizationPending {
				run.FinalizationStatus = domain.RunFinalizationStatusPending
			}
			if err := repos.Tasks.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}

			current := *run
			if !tc.unchangedVersion {
				current.LifecycleVersion++
			}
			if tc.changedOwnerEpoch {
				current.OwnerEpoch++
			}
			if tc.currentSandboxID != nil {
				current.SandboxID = tc.currentSandboxID
			}
			if tc.cancelled {
				current.Status = domain.RunStatusCancelled
				at := time.Now()
				current.CancelRequestedAt = &at
			} else {
				current.Status = domain.RunStatusRunning
			}
			createdID := uuid.New()
			owner := run.ID.String()
			if tc.owner != "" {
				owner = tc.owner
			}
			provider := mocks.NewFakeSandboxProvider()
			provider.CreateFunc = func(_ context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
				if tc.createErr != nil {
					return nil, tc.createErr
				}
				metadata := map[string]string{sandbox.AgentManagerRunIDMetadataKey: owner}
				if tc.missingMetadata {
					metadata = nil
				}
				return &sandbox.Sandbox{ID: createdID, Status: sandbox.SandboxStatusActive, Metadata: metadata}, nil
			}

			_, err := CreateSandboxWorkspace(ctx, SetupWorkspaceInput{
				Deps: Deps{Levers: config.DefaultLevers(), Runs: setupConflictRuns{RunRepository: repos.Runs, current: &current}},
				Run:  run, Task: task, Profile: fixtures.NewAgentProfile(t), Sandbox: provider,
			})
			if err == nil {
				t.Fatal("expected setup failure")
			}
			if got := provider.DeleteCallCount(); got != tc.wantDelete {
				t.Fatalf("Delete calls = %d, want %d", got, tc.wantDelete)
			}
			if got := provider.StopCallCount(); got != tc.wantStop {
				t.Fatalf("Stop calls = %d, want %d", got, tc.wantStop)
			}
		})
	}
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func boolPtr(value bool) *bool {
	return &value
}
