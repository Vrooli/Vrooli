package workflowruntime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

func TestValidateEngagementGrantCannotWidenWorkflowDeclaration(t *testing.T) {
	declared := domain.WorkflowBudgets{MaxTokens: 100, WallTimeSeconds: 60, MaxChargeMicroUSD: 1000}
	valid := domain.WorkflowEngagementGrant{MaxTokens: 50, MaxWallTimeSeconds: 30, MaxChargeMicroUSD: 500}
	if err := validateEngagementGrant(valid, declared); err != nil {
		t.Fatalf("bounded grant rejected: %v", err)
	}
	for name, grant := range map[string]domain.WorkflowEngagementGrant{
		"tokens": {MaxTokens: 101, MaxWallTimeSeconds: 30},
		"wall":   {MaxTokens: 50, MaxWallTimeSeconds: 61},
		"charge": {MaxTokens: 50, MaxWallTimeSeconds: 30, MaxChargeMicroUSD: 1001},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateEngagementGrant(grant, declared); err == nil {
				t.Fatal("widening grant unexpectedly accepted")
			}
		})
	}
}

func TestGrantCapacityRequiresBindingAndPreservesUngrantDefaults(t *testing.T) {
	definition := baseDefinition()
	capacity := definition.Budgets
	capacity.MaxTokens *= 2
	capacity.WallTimeSeconds *= 2
	definition.GrantCapacity = &capacity
	engine, _, _ := testEngine(t, definition)
	r := revision(definition)
	grant := &domain.WorkflowEngagementGrant{MaxTokens: capacity.MaxTokens, MaxWallTimeSeconds: capacity.WallTimeSeconds}
	if _, err := engine.StartWithGrant(context.Background(), r, []byte(`{"topic":"A"}`), "unbound-capacity", grant); err == nil {
		t.Fatal("capacity admitted an unbound grant")
	}
	if _, err := engine.StartWithGrant(context.Background(), r, []byte(`{"topic":"A"}`), "bound-capacity", grant, ExecutionBinding{ApprovalDigest: "sha256:approval", GrantDigest: "sha256:grant"}); err != nil {
		t.Fatal(err)
	}
	if got := applyEngagementGrant(r, grant).Definition.Budgets; got.MaxTokens != capacity.MaxTokens || got.WallTimeSeconds != capacity.WallTimeSeconds {
		t.Fatalf("bound explicit allowance was clipped to legacy defaults: %+v", got)
	}
	if got := applyEngagementGrant(r, nil).Definition.Budgets; got != definition.Budgets {
		t.Fatalf("capacity widened default authority: %+v", got)
	}
	grant.MaxTokens++
	if _, err := engine.StartWithGrant(context.Background(), r, []byte(`{"topic":"A"}`), "exceeds-capacity", grant, ExecutionBinding{ApprovalDigest: "sha256:approval", GrantDigest: "sha256:other"}); err == nil {
		t.Fatal("grant exceeded the declared technical capacity")
	}
}

func TestApplyEngagementGrantNarrowsEveryDeclaredDimension(t *testing.T) {
	revision := &domain.WorkflowRevision{Definition: domain.WorkflowDefinition{Budgets: domain.WorkflowBudgets{
		MaxTurns: 10, MaxTokens: 1000, WallTimeSeconds: 60, MaxChargeMicroUSD: 1000,
		MaxNodeAttempts: 10, MaxChildren: 5, MaxConcurrency: 3, MaxRecursion: 2, MaxRetries: 4, MaxWaitSeconds: 120,
	}}}
	grant := &domain.WorkflowEngagementGrant{MaxTurns: 2, MaxTokens: 200, MaxWallTimeSeconds: 30, MaxChargeMicroUSD: 250, MaxNodeAttempts: 4, MaxChildren: 2, MaxConcurrency: 1, MaxRecursion: 1, MaxRetries: 1, MaxWaitSeconds: 20}
	narrowed := applyEngagementGrant(revision, grant)
	if narrowed == revision || narrowed.Definition.Budgets.MaxTokens != 200 || narrowed.Definition.Budgets.WallTimeSeconds != 30 || narrowed.Definition.Budgets.MaxChargeMicroUSD != 250 || narrowed.Definition.Budgets.MaxChildren != 2 || narrowed.Definition.Budgets.MaxConcurrency != 1 || narrowed.Definition.Budgets.MaxWaitSeconds != 20 {
		t.Fatalf("narrowed budgets=%+v", narrowed.Definition.Budgets)
	}
	if revision.Definition.Budgets.MaxTokens != 1000 {
		t.Fatal("apply mutated catalog revision")
	}
}

func TestExplicitZeroRetryGrantSurvivesPersistenceAndDoesNotRestoreDefaults(t *testing.T) {
	d := budgetAdmissionDefinition()
	e, store, _ := testEngine(t, d)
	grant := &domain.WorkflowEngagementGrant{MaxTokens: 200, MaxWallTimeSeconds: 30, RetryLimitSet: true}
	x, err := e.StartWithGrant(t.Context(), revision(d), []byte(`{}`), "zero-retries", grant)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Get(t.Context(), x.ID)
	if err != nil || !reloaded.EngagementGrant.RetryLimitSet {
		t.Fatalf("retry presence lost: %+v %v", reloaded, err)
	}
	if got := applyEngagementGrant(revision(d), reloaded.EngagementGrant).Definition.Budgets.MaxRetries; got != 0 {
		t.Fatalf("exhausted retry allowance restored %d retries", got)
	}
	legacy := cloneGrant(grant)
	legacy.RetryLimitSet = false
	if got := applyEngagementGrant(revision(d), legacy).Definition.Budgets.MaxRetries; got != d.Budgets.MaxRetries {
		t.Fatalf("legacy default changed: %d", got)
	}
	if sameGrant(grant, legacy) {
		t.Fatal("explicit zero is not the same authority as an omitted retry limit")
	}
	reloaded.Status = domain.WorkflowExecutionFailed
	reloaded.Version++
	if ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: reloaded.Version - 1, Execution: reloaded}); err != nil || !ok {
		t.Fatalf("persist failure: %t %v", ok, err)
	}
	if _, _, err := e.Retry(t.Context(), reloaded.ID, "forbidden-owner-retry", reloaded.Version); err == nil {
		t.Fatal("owner Retry ignored the explicit zero allowance and reused the catalog default")
	}
}

func TestChildWorkflowInheritsRemainingParentGrantBeforeItsFirstRun(t *testing.T) {
	d := budgetAdmissionDefinition()
	e, store, children := testEngine(t, d)
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	grant := &domain.WorkflowEngagementGrant{MaxTurns: 5, MaxTokens: 500, MaxWallTimeSeconds: 60, MaxChargeMicroUSD: 9, MaxNodeAttempts: 5, MaxChildren: 5, MaxConcurrency: 1, MaxRecursion: 2, MaxRetries: 1, RetryLimitSet: true, MaxWaitSeconds: 30, AllowedEffects: []string{"filesystem.read[paths=scenarios/fixture/**]"}}
	parent, err := e.StartWithGrant(t.Context(), revision(d), []byte(`{}`), "parent", grant, ExecutionBinding{ApprovalDigest: "reviewed-item", GrantDigest: "parent-grant"})
	if err != nil {
		t.Fatal(err)
	}
	parent.BudgetUsage = domain.WorkflowBudgetUsage{Turns: 4, Tokens: 450, ChargeMicroUSD: 8, ChargeMeasured: true, AccountingComplete: true, Children: 3, NodeAttempts: 4, Retries: 1}
	parent.Version++
	if ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: parent.Version - 1, Execution: parent}); err != nil || !ok {
		t.Fatalf("persist parent usage: %t %v", ok, err)
	}
	now = now.Add(50 * time.Second)
	attemptID := uuid.New()
	child, err := e.StartChild(t.Context(), revision(d), []byte(`{}`), "nested-review", parent.ID, attemptID, 1)
	if err != nil {
		t.Fatal(err)
	}
	g := child.EngagementGrant
	if g == nil || g.MaxTurns != 1 || g.MaxTokens != 50 || g.MaxWallTimeSeconds != 10 || g.MaxChargeMicroUSD != 1 || g.MaxNodeAttempts != 1 || g.MaxChildren != 1 || g.MaxRetries != 0 || !g.RetryLimitSet || g.MaxConcurrency != 1 {
		t.Fatalf("nested review received fresh authority: %+v", g)
	}
	if child.ApprovalDigest != parent.ApprovalDigest || child.GrantDigest == "" || child.GrantDigest == parent.GrantDigest {
		t.Fatalf("child authority not bound to parent: %+v", child)
	}
	// Reload from durable state before dispatch, as recovery does.
	child, err = store.Get(t.Context(), child.ID)
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, e, child.ID)
	mustAdvance(t, e, child.ID)
	if len(children.requests) != 1 || children.requests[0].maxTurns != 1 || children.requests[0].timeout != 10*time.Second || len(children.requests[0].effects) != 1 {
		t.Fatalf("bounded child grant did not reach its run: %+v", children.requests)
	}
	now = now.Add(3 * time.Second)
	replayed, err := e.StartChild(t.Context(), revision(d), []byte(`{}`), "nested-review", parent.ID, attemptID, 1)
	if err != nil || replayed.ID != child.ID || replayed.GrantDigest != child.GrantDigest || replayed.EngagementGrant.MaxWallTimeSeconds != 10 {
		t.Fatalf("lost response changed reservation: %+v %v", replayed, err)
	}
}

func TestChildWorkflowCannotDispatchAgainstExhaustedOrUnknownParentAllowance(t *testing.T) {
	for _, test := range []struct {
		name       string
		usage      domain.WorkflowBudgetUsage
		wantBudget string
	}{
		{"tokens", domain.WorkflowBudgetUsage{Tokens: 500, AccountingComplete: true}, "tokens"},
		{"last child slot", domain.WorkflowBudgetUsage{Children: 4, AccountingComplete: true}, "children"},
		{"unknown usage", domain.WorkflowBudgetUsage{Tokens: 1}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := budgetAdmissionDefinition()
			e, store, children := testEngine(t, d)
			grant := &domain.WorkflowEngagementGrant{MaxTokens: 500, MaxWallTimeSeconds: 60, MaxChildren: 5, MaxConcurrency: 1}
			parent, err := e.StartWithGrant(t.Context(), revision(d), json.RawMessage(`{}`), "parent", grant)
			if err != nil {
				t.Fatal(err)
			}
			parent.BudgetUsage = test.usage
			parent.Version++
			if ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: parent.Version - 1, Execution: parent}); err != nil || !ok {
				t.Fatalf("save usage: %v", err)
			}
			child, err := e.StartChild(t.Context(), revision(d), []byte(`{}`), "forbidden-child", parent.ID, uuid.New(), 1)
			if err == nil || child != nil || len(children.requests) != 0 {
				t.Fatalf("exhausted child admitted: %+v %v", child, err)
			}
			if test.wantBudget != "" {
				var exhausted *inheritedBudgetExhausted
				if !errors.As(err, &exhausted) || exhausted.dimension != test.wantBudget {
					t.Fatalf("wrong exhaustion: %v", err)
				}
			}
			if stored, _ := store.GetByIdempotencyKey(t.Context(), "forbidden-child"); stored != nil {
				t.Fatal("rejected child persisted a new reservation")
			}
		})
	}
}
