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
	if child.ExecutionStrategy != parent.ExecutionStrategy || child.ExecutionPreferences == nil || child.ExecutionPreferences.PreferredRunner != "codex" {
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
	return NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "execution.json")}), item
}

func continuationParent() Record {
	return Record{
		ExecutionID:          "parent",
		BacklogKind:          "execute",
		BacklogName:          "fixture",
		Status:               StatusBudgetExhausted,
		Mode:                 ModeManual,
		ExecutionStrategy:    "adaptive-improvement",
		MaxSlices:            1,
		ExecutionLimits:      &identity.ExecutionLimits{MaxSlices: 4, MaxTokens: 1000, MaxTurns: 10, MaxWallSeconds: 100, MaxChargeMicroUSD: 1000, MaxChildren: 10, MaxNodeAttempts: 10, MaxRetries: 4},
		ExecutionPreferences: &ExecutionPreferences{PreferredRunner: "codex", Model: "gpt-5", Effort: "high"},
		ApprovalDigest:       "accepted",
		WorkflowGrant:        &workflowcontract.Grant{MaxTurns: 1},
		SettledUsage:         &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, WallSeconds: 1, Tokens: 10, Turns: 1, ChargeMicroUSD: 10, Slices: 1},
	}
}
