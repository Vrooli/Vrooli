package models

import (
	"context"
	"encoding/json"
	"fmt"
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
