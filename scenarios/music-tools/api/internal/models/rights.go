package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vrooli/ai-provider-go/policy"
)

const RightsUseLocalInference = "local_inference"

type RightsDecision struct {
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	DecisionSHA256 string `json:"decision_sha256,omitempty"`
}

// RightsEvaluator is implemented by the shared policy adapter, which loads
// trusted current context instead of accepting context from request data.
type RightsEvaluator func(context.Context, json.RawMessage, string) (RightsDecision, error)

// NewCurrentRightsEvaluator adapts the shared policy evaluator to the music
// registry contract. The shared evaluator loads trusted context for every
// call; the request cannot provide or cache that context.
func NewCurrentRightsEvaluator() RightsEvaluator {
	return AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(nil))
}

// AdaptRightsEvaluator keeps policy evaluation and trusted-context loading in
// ai-provider-go while allowing callers to inject its context source in tests.
func AdaptRightsEvaluator(evaluator *policy.CurrentRightsEvaluator) RightsEvaluator {
	return func(ctx context.Context, raw json.RawMessage, use string) (RightsDecision, error) {
		if evaluator == nil {
			return RightsDecision{Status: string(policy.RightsUnknown), Reason: "shared_model_rights_evaluator_unavailable"}, fmt.Errorf("shared model rights evaluator is unavailable")
		}
		var artifact policy.ArtifactRights
		if err := json.Unmarshal(raw, &artifact); err != nil {
			return RightsDecision{Status: string(policy.RightsUnknown), Reason: "model_rights_facts_are_invalid"}, err
		}
		decision, err := evaluator.Evaluate(ctx, &artifact, policy.RightsUse(use))
		return RightsDecision{Status: string(decision.Status), Reason: decision.Reason, DecisionSHA256: decision.DecisionSHA256}, err
	}
}

func EvaluateCurrentRights(ctx context.Context, evaluator RightsEvaluator, model Model, use string) (RightsDecision, error) {
	if evaluator == nil {
		return RightsDecision{Status: "unknown", Reason: "shared_model_rights_evaluator_unavailable"}, nil
	}
	decision, err := evaluator(ctx, append(json.RawMessage(nil), model.Rights...), use)
	if err != nil {
		return RightsDecision{Status: "unknown", Reason: fmt.Sprintf("shared_model_rights_evaluation_failed: %v", err)}, err
	}
	if decision.Status == "" {
		return RightsDecision{Status: "unknown", Reason: "shared_model_rights_evaluator_returned_no_status"}, nil
	}
	return decision, nil
}
