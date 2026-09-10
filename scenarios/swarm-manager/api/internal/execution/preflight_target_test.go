package execution

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPlanBackedPreflightDoesNotSuggestGeneratingWorkItemName(t *testing.T) {
	service := &Service{repoRoot: filepath.Join(t.TempDir(), "swarm-manager")}
	for _, name := range []string{"audio-tools-portable-voice-development", "tech-tree-designer-ecosystem-development"} {
		t.Run(name, func(t *testing.T) {
			item := backlogItem{
				Kind: "execute", Name: name, Status: backlogStatusBacklog,
				ExecutionStrategy: adaptiveImprovementStrategy,
				PlanRef:           &planRef{Provider: planRefProviderPlanManager, PlanID: "canonical-plan", Role: planRefRoleExecutionSpec},
			}
			got := service.processPreflightForItem(context.Background(), item, false)
			if got.ResolvedTargetScenarioID != "" || got.ArchivedRevival || got.SuggestedOperation != "plan.execute" || got.SuggestedSteerProfileID != "" {
				t.Fatalf("plan-backed preflight invented scenario generation: %+v", got)
			}
			if got.Ready || len(got.BlockingDetails) != 1 || got.BlockingDetails[0].Code != "plan_not_accepted" {
				t.Fatalf("correcting display must preserve the explicit approval gate: %+v", got)
			}
		})
	}
}

func TestUnplannedPreflightPreservesScenarioGenerationHint(t *testing.T) {
	service := &Service{repoRoot: filepath.Join(t.TempDir(), "swarm-manager")}
	got := service.processPreflightForItem(context.Background(), backlogItem{Kind: "idea", Name: "new-scenario", Status: backlogStatusBacklog}, false)
	if got.ResolvedTargetScenarioID != "new-scenario" || got.SuggestedOperation != "generator" || got.SuggestedSteerProfileID != "rapid-mvp" {
		t.Fatalf("unplanned scenario guidance changed: %+v", got)
	}
}
