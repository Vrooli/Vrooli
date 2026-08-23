package execution

import (
	"context"
	"testing"

	"swarm-manager/internal/planclient"
)

func TestUntilDrainEligibilityRejectsIrreversiblePlan(t *testing.T) {
	service := &Service{planRenderer: &fakeMarkdownRenderer{result: planclient.RenderMarkdownResult{
		Markdown: "# Execution\n\nThis phase performs an irreversible migration.",
	}}}
	reason, err := service.untilDrainEligibility(context.Background(), backlogItem{
		Kind: "execute", Name: "plan",
		PlanRef: &planRef{Provider: planRefProviderPlanManager, PlanID: "plan-1", Role: planRefRoleExecutionSpec},
	})
	if err != nil {
		t.Fatalf("untilDrainEligibility() error = %v", err)
	}
	if reason == "" {
		t.Fatal("until-drain eligibility accepted an irreversible plan")
	}
}

func TestUntilDrainEligibilityAcceptsReviewedPlan(t *testing.T) {
	service := &Service{planRenderer: &fakeMarkdownRenderer{result: planclient.RenderMarkdownResult{
		Markdown: "# Execution\n\nImplement the reviewed changes and record evidence.",
	}}}
	reason, err := service.untilDrainEligibility(context.Background(), backlogItem{
		Kind: "execute", Name: "plan",
		PlanRef: &planRef{Provider: planRefProviderPlanManager, PlanID: "plan-1", Role: planRefRoleExecutionSpec},
	})
	if err != nil {
		t.Fatalf("untilDrainEligibility() error = %v", err)
	}
	if reason != "" {
		t.Fatalf("until-drain eligibility rejected reviewed plan: %s", reason)
	}
}
