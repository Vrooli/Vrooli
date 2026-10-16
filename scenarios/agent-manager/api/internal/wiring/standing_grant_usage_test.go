package wiring

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
)

// ownerRun persists a non-terminal run for owner carrying an accumulated cost
// estimate, so the usage reader can attribute committed spend to that owner.
func ownerRun(t *testing.T, repos *testRepos, taskID uuid.UUID, owner string, costUSD float64) {
	t.Helper()
	run := &domain.Run{
		ID:           uuid.New(),
		TaskID:       taskID,
		Status:       domain.RunStatusPending,
		Phase:        domain.RunPhaseQueued,
		OwnerSubject: owner,
		Summary:      &domain.RunSummary{CostEstimate: costUSD},
	}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("create owner run: %v", err)
	}
}

type testRepos = database.Repositories

// G1: the usage reader populates CommittedBudgetUSD from the owner's live runs,
// so authn.AuthorizeStandingGrant actually enforces MaxRunBudgetUSD. This proves
// the previously-unconditional budget admission (the removed TODO) is closed:
//   - an over-budget owner is refused with ErrStandingGrantLimit (NEGATIVE path);
//   - an in-budget owner is admitted (POSITIVE path);
//   - concurrency enforcement still works unchanged.
func TestStandingGrantUsageEnforcesBudgetFromLiveRunSpend(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	o := orchestration.New(nil, repos.Tasks, repos.Runs, orchestration.WithEvents(events))
	reader := standingGrantUsageReader{orch: o}

	task := &domain.Task{ID: uuid.New(), Title: "budget-harness", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	const owner = "operator-owner"
	// Two live runs for this owner: committed spend $30 + $15 = $45.
	ownerRun(t, repos, task.ID, owner, 30)
	ownerRun(t, repos, task.ID, owner, 15)
	// A run for a DIFFERENT owner must not be attributed here.
	ownerRun(t, repos, task.ID, "someone-else", 1000)

	usage, err := reader.StandingGrantUsage(ctx, owner)
	if err != nil {
		t.Fatalf("usage read: %v", err)
	}
	if usage.ConcurrentRuns != 2 {
		t.Fatalf("concurrency should count only this owner's runs: got %d want 2", usage.ConcurrentRuns)
	}
	if usage.CommittedBudgetUSD != 45 {
		t.Fatalf("committed budget should sum this owner's live-run spend: got %v want 45", usage.CommittedBudgetUSD)
	}

	now := time.Now()
	grant := &requestidentity.OperationGrant{
		ID: "g", Approver: owner, Policy: "run-creation",
		Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"},
		ExpiresAt: now.Add(time.Hour),
	}

	// NEGATIVE: committed $45 is over a $40 budget → refuse with the standing-grant
	// limit error (never an approval; approving cannot fix an exhausted bound).
	if err := authn.AuthorizeStandingGrant(grant, authn.StandingGrantLimits{MaxRunBudgetUSD: 40}, usage, "run.create", "agent-manager:*", now); err != authn.ErrStandingGrantLimit {
		t.Fatalf("over-budget request not refused with ErrStandingGrantLimit: %v", err)
	}

	// POSITIVE: committed $45 is under a $100 budget → admit.
	if err := authn.AuthorizeStandingGrant(grant, authn.StandingGrantLimits{MaxRunBudgetUSD: 100}, usage, "run.create", "agent-manager:*", now); err != nil {
		t.Fatalf("in-budget request refused: %v", err)
	}

	// Concurrency still binds independently: 2 concurrent at a cap of 2 → refuse.
	if err := authn.AuthorizeStandingGrant(grant, authn.StandingGrantLimits{MaxConcurrentRuns: 2}, usage, "run.create", "agent-manager:*", now); err != authn.ErrStandingGrantLimit {
		t.Fatalf("concurrency over-limit not refused: %v", err)
	}
	// Concurrency under its cap admits.
	if err := authn.AuthorizeStandingGrant(grant, authn.StandingGrantLimits{MaxConcurrentRuns: 5}, usage, "run.create", "agent-manager:*", now); err != nil {
		t.Fatalf("within concurrency limit refused: %v", err)
	}
}
