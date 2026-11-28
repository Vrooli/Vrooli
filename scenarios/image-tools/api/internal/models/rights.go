package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vrooli/ai-provider-go/policy"
)

// Rights uses mirror ai-provider-go/policy. Local inference is normal use;
// the others stay license-bound.
const (
	RightsUseLocalInference      = "local_inference"
	RightsUseMonetizedUse        = "monetized_use"
	RightsUseHostedInference     = "hosted_inference"
	RightsUseRedistributeWeights = "redistribute_weights"
)

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

// ErrLocalUseNotPermitted marks a model that the rights owner does not allow
// for local inference (enable, install or pin).
var ErrLocalUseNotPermitted = errors.New("model rights do not permit local use")

// RequireLocalUsePermitted is the single lifecycle gate for enable, install
// and default-pin actions: normal local use. A verified commercial label
// passes as before; everything else, including non-commercial and unverified
// models, is decided by the shared rights owner against the operator's current
// trusted context. Operator stance 2026-11-21: always use the best models
// regardless of licensing for normal use, so the standing normal-use answer in
// that context allows every license class here exactly as it does at selection
// and execution. Without it, the gate fails closed.
func RequireLocalUsePermitted(ctx context.Context, evaluator RightsEvaluator, model Model) error {
	if model.CapabilityLabels.CommercialUse == CommercialUseYes {
		return nil
	}
	// A model the operator brought themselves carries no rights record for the
	// shared evaluator to read (the licence is whatever the source declared).
	// Running it locally is normal use, so it is permitted; every use beyond that
	// still goes through RequireUsePermitted, which refuses it without a record.
	if model.CapabilityLabels.Provenance == ProvenanceUserImported && !hasRightsRecord(model) {
		return nil
	}
	return requireUse(ctx, evaluator, model, RightsUseLocalInference)
}

// hasRightsRecord reports whether the model carries rights facts for the shared
// evaluator. A stored entry with none round-trips its absent record as JSON null,
// which is not a record.
func hasRightsRecord(m Model) bool {
	trimmed := bytes.TrimSpace(m.Rights)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}

// RequireUsePermitted gates every use beyond normal local inference:
// RightsUseMonetizedUse, RightsUseHostedInference and the redistribution uses.
// License evidence always decides them, so a non-commercial or unverified model
// is refused regardless of the operator's normal-use answer, and a commercial
// label alone never stands in for a redistribution attestation.
func RequireUsePermitted(ctx context.Context, evaluator RightsEvaluator, model Model, use string) error {
	if use == RightsUseLocalInference {
		return RequireLocalUsePermitted(ctx, evaluator, model)
	}
	return requireUse(ctx, evaluator, model, use)
}

func requireUse(ctx context.Context, evaluator RightsEvaluator, model Model, use string) error {
	decision, err := EvaluateCurrentModelRights(ctx, evaluator, model, use)
	if err != nil || decision.Status != "allowed" {
		return fmt.Errorf("%w: %q use %q (commercial use %q): the rights owner returned %s (%s)", ErrLocalUseNotPermitted, model.ID, use, model.CapabilityLabels.CommercialUse, decision.Status, decision.Reason)
	}
	return nil
}
