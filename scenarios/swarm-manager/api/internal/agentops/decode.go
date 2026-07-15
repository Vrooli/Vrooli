package agentops

import (
	"encoding/json"
	"fmt"
)

// Decode helpers return the typed value for a raw document. They perform only
// structural JSON decoding; callers that need the semantic guarantees must run
// the matching Validate* function first (Load paths do). They exist so consumers
// outside this package (the catalog loader, the runner) obtain typed values
// without re-implementing the json tags.

// DecodeOperationContract decodes a raw operation-contract document.
func DecodeOperationContract(raw []byte) (OperationContract, error) {
	var oc OperationContract
	if err := json.Unmarshal(raw, &oc); err != nil {
		return OperationContract{}, fmt.Errorf("decode operation contract: %w", err)
	}
	return oc, nil
}

// DecodeBinding decodes a raw operation-binding document.
func DecodeBinding(raw []byte) (OperationBinding, error) {
	var b OperationBinding
	if err := json.Unmarshal(raw, &b); err != nil {
		return OperationBinding{}, fmt.Errorf("decode operation binding: %w", err)
	}
	return b, nil
}

// DecodeTransitionPolicy decodes a raw transition-policy document.
func DecodeTransitionPolicy(raw []byte) (TransitionPolicy, error) {
	var p TransitionPolicy
	if err := json.Unmarshal(raw, &p); err != nil {
		return TransitionPolicy{}, fmt.Errorf("decode transition policy: %w", err)
	}
	return p, nil
}

// DecodeWorkflowInstance decodes a raw workflow-instance document.
func DecodeWorkflowInstance(raw []byte) (WorkflowInstance, error) {
	var w WorkflowInstance
	if err := json.Unmarshal(raw, &w); err != nil {
		return WorkflowInstance{}, fmt.Errorf("decode workflow instance: %w", err)
	}
	return w, nil
}

// DecodeProvenance decodes a raw execution-provenance document.
func DecodeProvenance(raw []byte) (ExecutionProvenance, error) {
	var p ExecutionProvenance
	if err := json.Unmarshal(raw, &p); err != nil {
		return ExecutionProvenance{}, fmt.Errorf("decode execution provenance: %w", err)
	}
	return p, nil
}
