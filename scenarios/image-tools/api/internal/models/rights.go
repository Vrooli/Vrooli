package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vrooli/ai-provider-go/policy"
)

const RightsUseLocalInference = "local_inference"

type RightsDecision struct {
	Status         string   `json:"status"`
	Reason         string   `json:"reason"`
	DecisionSHA256 string   `json:"decision_sha256,omitempty"`
	Obligations    []string `json:"obligations,omitempty"`
}

// RightsEvaluator is an owner injection seam implemented by the shared
// ai-provider-go/policy adapter. The adapter must load current trusted context
// itself and decode these facts into policy.ArtifactRights. No request field is
// a trusted context source.
type RightsEvaluator func(context.Context, json.RawMessage, string) (RightsDecision, error)

// AdaptRightsEvaluator delegates image registry facts and trusted context
// evaluation to ai-provider-go. Malformed facts remain unknown and fail closed.
func AdaptRightsEvaluator(evaluator *policy.CurrentRightsEvaluator) RightsEvaluator {
	return func(ctx context.Context, raw json.RawMessage, use string) (RightsDecision, error) {
		var artifact policy.ArtifactRights
		if err := json.Unmarshal(raw, &artifact); err != nil {
			return RightsDecision{Status: string(policy.RightsUnknown), Reason: "model_rights_facts_are_invalid"}, err
		}
		decision, err := evaluator.Evaluate(ctx, &artifact, policy.RightsUse(use))
		return RightsDecision{
			Status: string(decision.Status), Reason: decision.Reason,
			DecisionSHA256: decision.DecisionSHA256,
			Obligations:    append([]string(nil), decision.Obligations...),
		}, err
	}
}

func EvaluateCurrentModelRights(ctx context.Context, evaluator RightsEvaluator, model Model, use string) (RightsDecision, error) {
	unknown := RightsDecision{Status: "unknown", Reason: "shared_model_rights_evaluator_unavailable"}
	if evaluator == nil {
		if len(model.Rights) == 0 {
			unknown.Reason = "model_rights_facts_missing"
		}
		return unknown, nil
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
