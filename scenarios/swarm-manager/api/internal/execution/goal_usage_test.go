package execution

import (
	"context"
	"errors"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitions"
	"swarm-manager/internal/workflowcontract"
)

func completeGoalUsage() *workflowcontract.Usage {
	return &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 400000, Turns: 50, WallSeconds: 3600, ChargeMicroUSD: 2000000}
}

// goalGrantFixture returns an accepted item and a goal prior that holds a
// reservation but carries no settled usage: the state every goal run ended in
// before goal runs had owner accounting.
func goalGrantFixture(t *testing.T) (backlogItem, Record, func() Record) {
	t.Helper()
	limits := reviewedLimits()
	item := backlogItem{Kind: "execute", Name: "goal-item", ExecutionLimits: limits, PlanAcceptance: &planAcceptance{SubjectVersion: "subject", PlanContentHash: "plan"}}
	approval := digestStrings("subject", "plan")
	newRecord := func() Record {
		return Record{ExecutionID: "next", BacklogKind: item.Kind, BacklogName: item.Name, ExecutionMode: transitions.ExecutionModeGoal, ExecutionLimits: limits.Clone(), ApprovalDigest: approval}
	}
	prior := newRecord()
	prior.ExecutionID = "prior"
	prior.RunID = "run-prior"
	prior.Status = StatusInterrupted
	if err := (&Service{}).prepareExecutionGrantLocked(t.Context(), nil, &prior, item); err != nil {
		t.Fatal(err)
	}
	return item, prior, newRecord
}

func TestGoalPriorSettlesFromRunAccountingAndGrantsOnlyTheRemainder(t *testing.T) {
	item, prior, newRecord := goalGrantFixture(t)
	usage := completeGoalUsage()
	service := &Service{goalRunReader: stubGoalRunReader{usage: usage, terminal: true}}
	records := []Record{prior}
	next := newRecord()
	if err := service.prepareExecutionGrantLocked(t.Context(), records, &next, item); err != nil {
		t.Fatalf("a goal prior with a complete owner receipt blocked the next run: %v", err)
	}
	limits := item.ExecutionLimits
	grant := next.WorkflowGrant
	if grant == nil || grant.MaxTokens != limits.MaxTokens-usage.Tokens || grant.MaxTurns != limits.MaxTurns-int(usage.Turns) ||
		grant.MaxWallTimeSeconds != limits.MaxWallSeconds-usage.WallSeconds || grant.MaxChargeMicroUSD != limits.MaxChargeMicroUSD-usage.ChargeMicroUSD {
		t.Fatalf("grant did not spend the goal run's usage: %+v", grant)
	}
	if records[0].SettledUsage == nil || records[0].SettledUsage.Tokens != usage.Tokens {
		t.Fatalf("collected goal usage was not retained on the prior: %+v", records[0].SettledUsage)
	}
}

func TestGoalPriorWithoutACompleteReceiptKeepsItsReservation(t *testing.T) {
	cases := map[string]stubGoalRunReader{
		"tokens unknown":    {usage: &workflowcontract.Usage{ChargeMeasured: true, Tokens: 10, WallSeconds: 5}, terminal: true},
		"charge unmeasured": {usage: &workflowcontract.Usage{TokensKnown: true, Tokens: 10, WallSeconds: 5}, terminal: true},
		"run still live":    {usage: completeGoalUsage(), terminal: false},
		"owner unavailable": {usageErr: errors.New("agent-manager unavailable")},
	}
	for name, reader := range cases {
		t.Run(name, func(t *testing.T) {
			item, prior, newRecord := goalGrantFixture(t)
			records := []Record{prior}
			next := newRecord()
			err := (&Service{goalRunReader: reader}).prepareExecutionGrantLocked(t.Context(), records, &next, item)
			if err == nil || next.WorkflowGrant != nil {
				t.Fatalf("unknown goal usage released the reservation: grant=%+v err=%v", next.WorkflowGrant, err)
			}
			if records[0].SettledUsage != nil {
				t.Fatalf("incomplete goal usage was recorded as settled: %+v", records[0].SettledUsage)
			}
		})
	}
}

func TestReconciledGoalRunSettlesOnlyACompleteReceipt(t *testing.T) {
	cases := []struct {
		name        string
		reader      stubGoalRunReader
		wantSettled bool
	}{
		{"complete receipt", stubGoalRunReader{usage: completeGoalUsage(), terminal: true}, true},
		{"pending receipt", stubGoalRunReader{usage: &workflowcontract.Usage{Tokens: 5, WallSeconds: 5}, terminal: true}, false},
		{"accounting unavailable", stubGoalRunReader{usageErr: errors.New("accounting unavailable")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestPollingService(t)
			reader := tc.reader
			reader.state = agentmanager.GoalRunState{TerminalClass: "interruption", StopReason: "timeout"}
			svc.goalRunReader = reader
			rec := Record{ExecutionID: "exec-goal", BacklogKind: "execute", BacklogName: "goal-item", RunID: "run-goal", ExecutionMode: transitions.ExecutionModeGoal, Status: StatusRunning, CreatedAt: nowRFC3339(), UpdatedAt: nowRFC3339()}
			if err := svc.store.Save([]Record{rec}); err != nil {
				t.Fatal(err)
			}
			changed, err := svc.applyReconciledGoalRun(context.Background(), "exec-goal")
			if err != nil || !changed {
				t.Fatalf("applyReconciledGoalRun: changed=%v err=%v", changed, err)
			}
			records, err := svc.store.Load()
			if err != nil {
				t.Fatal(err)
			}
			got := records[0]
			if got.Status != StatusInterrupted {
				t.Fatalf("accounting changed the terminal transition: status=%q", got.Status)
			}
			if (got.SettledUsage != nil) != tc.wantSettled {
				t.Fatalf("settled usage = %+v, want settled=%v", got.SettledUsage, tc.wantSettled)
			}
		})
	}
}

// Regression for bug knw-1789322116469984811: an interrupted goal run has no
// workflow receipt, so its continuation used to be refused as an unresolved
// dispatch reservation. The child must admit against the parent's run usage.
func TestInterruptedGoalContinuationAdmitsAgainstTheParentRunAccounting(t *testing.T) {
	service, _ := continuationTestService(t, "until-allowance")
	usage := &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 100, Turns: 2, WallSeconds: 10, ChargeMicroUSD: 50}
	service.goalRunReader = stubGoalRunReader{usage: usage, terminal: true}
	item := backlogItem{Kind: "execute", Name: "fixture", PlanAcceptance: &planAcceptance{SubjectVersion: "subject", PlanContentHash: "plan"}}
	parent := continuationParent()
	parent.Status = StatusInterrupted
	parent.ExecutionMode = transitions.ExecutionModeGoal
	parent.StopReason = "timeout"
	parent.RunID = "run-parent"
	parent.ApprovalDigest = digestStrings("subject", "plan")
	parent.SettledUsage = nil
	if err := service.store.Save([]Record{parent}); err != nil {
		t.Fatal(err)
	}

	service.continueExhaustedLocked(context.Background())

	records, err := service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("interrupted goal parent produced %d records, want parent and one child", len(records))
	}
	if records[0].SettledUsage == nil {
		t.Fatal("the sweeper did not settle the parent's run accounting")
	}
	child := records[1]
	if err := service.prepareExecutionGrantLocked(t.Context(), records, &child, item); err != nil {
		t.Fatalf("goal continuation was refused: %v", err)
	}
	if child.WorkflowGrant == nil || child.WorkflowGrant.MaxTokens != parent.ExecutionLimits.MaxTokens-usage.Tokens {
		t.Fatalf("continuation grant did not spend the parent's usage: %+v", child.WorkflowGrant)
	}
}
