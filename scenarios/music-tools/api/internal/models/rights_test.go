package models

import (
	"context"
	"testing"

	"github.com/vrooli/ai-provider-go/policy"
)

func TestSharedCurrentRightsEvaluatorFollowsContextAcrossMusicSelection(t *testing.T) {
	current := policy.RightsContext{
		ID: "operator-business-context/current", Version: "context-under-threshold",
		EntityScope: "operator-group", RevenueStatus: "pre_revenue", AsOf: "2026-11-14", Source: "operator_statement",
	}
	shared := policy.NewRightsEvaluator(func(context.Context) (policy.RightsContext, error) { return current, nil })
	evaluator := AdaptRightsEvaluator(shared)
	registry, err := LoadSeed(false)
	if err != nil {
		t.Fatal(err)
	}
	selectSA3 := func() (Model, error) {
		return registry.SelectCompositionWithRights(context.Background(), "stable-audio-3-medium", false, evaluator)
	}
	allowed, err := selectSA3()
	if err != nil || allowed.ID != "stabilityai/stable-audio-3-medium" {
		t.Fatalf("under-threshold music selection = %q, %v", allowed.ID, err)
	}

	// Exercise a crossing of the source grant's annual-revenue condition. This
	// is a hypothetical policy regression fixture, not a claim about operator
	// revenue; the trusted production context remains operator-owned.
	current.Version = "context-at-threshold"
	current.RevenueStatus = "at_or_above_threshold"
	if _, err := selectSA3(); err == nil {
		t.Fatal("music selection remained available after the grant threshold was crossed")
	}
	fallback, err := registry.SelectCompositionWithRights(context.Background(), "", false, evaluator)
	if err != nil || fallback.ID != "ACE-Step/Ace-Step1.5:acestep-v15-turbo" {
		t.Fatalf("fallback after SA3 refusal = %q, %v; want rights-eligible ACE-Step", fallback.ID, err)
	}

	current.Version = "context-under-threshold-restored"
	current.RevenueStatus = "pre_revenue"
	restored, err := selectSA3()
	if err != nil || restored.ID != allowed.ID {
		t.Fatalf("restored music selection = %q, %v; want %q", restored.ID, err, allowed.ID)
	}
}
