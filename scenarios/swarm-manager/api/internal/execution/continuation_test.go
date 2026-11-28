package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/identity"
	"swarm-manager/internal/workflowcontract"
)

func TestContinueExhaustedCreatesOneInheritedChild(t *testing.T) {
	service, item := continuationTestService(t, "until-allowance")
	parent := continuationParent()
	if err := service.store.Save([]Record{parent}); err != nil {
		t.Fatal(err)
	}
	service.continueExhaustedLocked(context.Background())
	records, err := service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records=%d, want parent and one child", len(records))
	}
	child := records[1]
	if child.Status != StatusPending || child.Operation != continuationOperation || child.StartedBy != continuationStartedBy {
		t.Fatalf("child=%+v", child)
	}
	if child.ContinuationOf != parent.ExecutionID || child.ParentExecutionID != parent.ExecutionID {
		t.Fatalf("child parent links=%q/%q, want %q", child.ContinuationOf, child.ParentExecutionID, parent.ExecutionID)
	}
	if child.ExecutionMode != parent.ExecutionMode || child.ExecutionPreferences == nil || child.ExecutionPreferences.PreferredRunner != "codex" {
		t.Fatalf("child did not inherit execution selection: %+v", child)
	}
	if len(records[0].ContinuationChildIDs) != 1 || records[0].ContinuationChildIDs[0] != child.ExecutionID {
		t.Fatalf("parent child links=%v", records[0].ContinuationChildIDs)
	}
	service.continueExhaustedLocked(context.Background())
	records, err = service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("second sweep created a duplicate: %d records", len(records))
	}
	_ = item
}

func TestContinueExhaustedRespectsManualAndHalt(t *testing.T) {
	for _, policy := range []string{"manual", "halted"} {
		t.Run(policy, func(t *testing.T) {
			service, _ := continuationTestService(t, policy)
			parent := continuationParent()
			if err := service.store.Save([]Record{parent}); err != nil {
				t.Fatal(err)
			}
			if policy == "halted" {
				if err := service.HaltContinuation("execute/fixture", "operator"); err != nil {
					t.Fatal(err)
				}
			}
			service.continueExhaustedLocked(context.Background())
			records, err := service.store.Load()
			if err != nil {
				t.Fatal(err)
			}
			if len(records) != 1 {
				t.Fatalf("policy %s created %d child records", policy, len(records)-1)
			}
		})
	}
}

func TestContinueExhaustedStopsOnNoProgressAndExhaustedDimension(t *testing.T) {
	t.Run("no progress", func(t *testing.T) {
		service, _ := continuationTestService(t, "until-allowance")
		root := continuationParent()
		first := root
		first.ExecutionID = "first"
		first.Status = StatusBudgetExhausted
		first.ContinuationOf = root.ExecutionID
		second := first
		second.ExecutionID = "second"
		second.ContinuationOf = first.ExecutionID
		parent := second
		parent.ExecutionID = "third"
		parent.ContinuationOf = second.ExecutionID
		if err := service.store.Save([]Record{root, first, second, parent}); err != nil {
			t.Fatal(err)
		}
		service.continueExhaustedLocked(context.Background())
		var item backlogItem
		contents, err := os.ReadFile(filepath.Join(service.itemDir("execute", "fixture"), "spec.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(contents, &item); err != nil {
			t.Fatal(err)
		}
		if item.ContinuationStoppedReason != "no_progress" || item.ContinuationHaltedAt == "" {
			t.Fatalf("item continuation state=%+v", item)
		}
	})

	t.Run("tokens exhausted", func(t *testing.T) {
		service, _ := continuationTestService(t, "until-allowance")
		parent := continuationParent()
		parent.ExecutionLimits.MaxTokens = 1
		parent.SettledUsage.Tokens = 1
		if err := service.store.Save([]Record{parent}); err != nil {
			t.Fatal(err)
		}
		service.continueExhaustedLocked(context.Background())
		item, err := service.loadBacklogItem("execute", "fixture")
		if err != nil {
			t.Fatal(err)
		}
		if item.ContinuationStoppedReason != "tokens" {
			t.Fatalf("stop reason=%q, want tokens", item.ContinuationStoppedReason)
		}
	})
}

func continuationTestService(t *testing.T, policy string) (*Service, backlogItem) {
	t.Helper()
	root := t.TempDir()
	itemDir := filepath.Join(root, "execute", "fixture")
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		t.Fatal(err)
	}
	item := backlogItem{Name: "fixture", Kind: "execute", Status: "queued", Continuation: policy}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(itemDir, "spec.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return NewService(ServiceConfig{
		TransitionRegistry: testTransitionRegistry(t), DataRoot: root, StorePath: filepath.Join(root, "execution.json"),
	}), item
}

func continuationParent() Record {
	return Record{
		ExecutionID:          "parent",
		BacklogKind:          "execute",
		BacklogName:          "fixture",
		Status:               StatusBudgetExhausted,
		Mode:                 ModeManual,
		ExecutionMode:        "sliced",
		MaxSlices:            1,
		ExecutionLimits:      &identity.ExecutionLimits{MaxSlices: 4, MaxTokens: 1000, MaxTurns: 10, MaxWallSeconds: 100, MaxChargeMicroUSD: 1000, MaxChildren: 10, MaxNodeAttempts: 10, MaxRetries: 4},
		ExecutionPreferences: &ExecutionPreferences{PreferredRunner: "codex", Model: "gpt-5", Effort: "high"},
		ApprovalDigest:       "accepted",
		WorkflowGrant:        &workflowcontract.Grant{MaxTurns: 1},
		SettledUsage:         &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, WallSeconds: 1, Tokens: 10, Turns: 1, ChargeMicroUSD: 10, Slices: 1},
	}
}

func TestContinueExhaustedProgressResetsBrakeAtDepth(t *testing.T) {
	service, _ := continuationTestService(t, "until-allowance")
	chain := func(id, continuationOf string, progress, streak int) Record {
		record := continuationParent()
		record.ExecutionID = id
		record.ContinuationOf = continuationOf
		record.PlanManagerProgress = progress
		record.NoProgressStreak = streak
		record.PlanManagerExecutionID = "plan-exec-" + id
		// Prior chain links are not reservations: the brake and allowance are
		// exercised independently.
		record.WorkflowGrant = nil
		record.SettledUsage = nil
		return record
	}
	root := chain("root", "", 0, 0)
	first := chain("first", "root", 1, 0)
	second := chain("second", "first", 2, 0)
	parent := continuationParent()
	parent.ExecutionID = "parent"
	parent.ContinuationOf = "second"
	parent.PlanManagerProgress = 3
	parent.NoProgressStreak = 2
	parent.PlanManagerExecutionID = "plan-exec-parent"
	if err := service.store.Save([]Record{root, first, second, parent}); err != nil {
		t.Fatal(err)
	}
	// The read observes progress beyond the parent, so the brake resets even at
	// a chain depth the old chain-depth brake would have halted.
	service.SetPlanProgressReader(func(context.Context, string) (int, error) { return 4, nil })
	service.continueExhaustedLocked(context.Background())
	records, err := service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 5 {
		t.Fatalf("progressing chain halted: %d records, want parent plus one child", len(records))
	}
	child := records[len(records)-1]
	if child.NoProgressStreak != 0 || child.PlanManagerProgress != 4 {
		t.Fatalf("child brake fields=%d/%d, want 0/4", child.NoProgressStreak, child.PlanManagerProgress)
	}
}

func TestContinueExhaustedHaltsAfterThreeNoProgressResumes(t *testing.T) {
	service, _ := continuationTestService(t, "until-allowance")
	parent := continuationParent()
	parent.PlanManagerProgress = 5
	parent.NoProgressStreak = 2
	parent.PlanManagerExecutionID = "plan-exec-parent"
	if err := service.store.Save([]Record{parent}); err != nil {
		t.Fatal(err)
	}
	service.SetPlanProgressReader(func(context.Context, string) (int, error) { return 5, nil })
	service.continueExhaustedLocked(context.Background())
	records, err := service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("no-progress chain created a child: %d records", len(records))
	}
	item, err := service.loadBacklogItem("execute", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if item.ContinuationStoppedReason != "no_progress" || item.ContinuationHaltedAt == "" {
		t.Fatalf("item continuation state=%+v", item)
	}
}

func TestContinueResumesInterruptedGoalWithFreshRunChain(t *testing.T) {
	service, _ := continuationTestService(t, "until-allowance")
	parent := continuationParent()
	parent.Status = StatusInterrupted
	parent.ExecutionMode = "goal"
	parent.StopReason = "timeout"
	parent.LastHandoff = "earlier handoff text"
	parent.PlanManagerExecutionID = "plan-exec-parent"
	if err := service.store.Save([]Record{parent}); err != nil {
		t.Fatal(err)
	}
	service.continueExhaustedLocked(context.Background())
	records, err := service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("interrupted goal resume created %d records, want one child", len(records))
	}
	child := records[1]
	if child.ResumePath != "fresh_run" {
		t.Fatalf("resume path=%q, want fresh_run", child.ResumePath)
	}
	if child.ResumeOrdinal != 1 || child.ResumeReason != "timeout" {
		t.Fatalf("resume chain ordinal/reason=%d/%q, want 1/timeout", child.ResumeOrdinal, child.ResumeReason)
	}
	if child.LastHandoff != "earlier handoff text" {
		t.Fatalf("child lost the parent handoff: %q", child.LastHandoff)
	}
}

func TestContinueNeverResumesVerdicts(t *testing.T) {
	for _, status := range []Status{StatusNeedsAttention, StatusNeedsReview, StatusValidating, StatusCompleted, StatusAbstained, StatusFailed, StatusCanceled} {
		service, _ := continuationTestService(t, "until-allowance")
		parent := continuationParent()
		parent.Status = status
		if err := service.store.Save([]Record{parent}); err != nil {
			t.Fatal(err)
		}
		service.continueExhaustedLocked(context.Background())
		records, err := service.store.Load()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 1 {
			t.Fatalf("verdict status %s was resumed into %d records", status, len(records)-1)
		}
	}
}

func TestContinueIdleSweeperWritesNothing(t *testing.T) {
	service, _ := continuationTestService(t, "manual")
	parent := continuationParent()
	if err := service.store.Save([]Record{parent}); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(service.itemDir("execute", "fixture"), "spec.json")
	before, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	service.continueExhaustedLocked(context.Background())
	after, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("idle sweeper rewrote the item spec")
	}
}
