package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"scenario-to-cloud/identity"
)

// OperationState is the durable state of a cloud operation.
type OperationState string

// Operation states. Terminal states are succeeded, failed, failed_recovery
// and cancelled.
const (
	OperationAdmitted        OperationState = "admitted"
	OperationWaitingInput    OperationState = "waiting_input"
	OperationRunning         OperationState = "running"
	OperationVerifying       OperationState = "verifying"
	OperationReconciling     OperationState = "reconciling"
	OperationRecovering      OperationState = "recovering"
	OperationCancelRequested OperationState = "cancel_requested"
	OperationSucceeded       OperationState = "succeeded"
	OperationFailed          OperationState = "failed"
	OperationFailedRecovery  OperationState = "failed_recovery"
	OperationCancelled       OperationState = "cancelled"
)

// IsTerminal reports whether the state admits no further transitions.
func (s OperationState) IsTerminal() bool {
	switch s {
	case OperationSucceeded, OperationFailed, OperationFailedRecovery, OperationCancelled:
		return true
	}
	return false
}

// CloudOperation is one admitted executable plan against a deployment. The
// (deployment_id, request_key) pair is unique: the same key with the same
// plan digest returns the existing operation, a different digest is a typed
// request_key_conflict.
type CloudOperation struct {
	ID              string          `json:"id"`
	DeploymentID    string          `json:"deployment_id"`
	RequestKey      string          `json:"request_key"`
	PlanDigest      string          `json:"plan_digest"`
	Plan            json.RawMessage `json:"plan,omitempty"`
	State           OperationState  `json:"state"`
	Fence           uint64          `json:"fence"`
	WorkerID        string          `json:"worker_id,omitempty"`
	LeaseExpiresAt  *time.Time      `json:"lease_expires_at,omitempty"`
	HeartbeatAt     *time.Time      `json:"heartbeat_at,omitempty"`
	CancelRequested bool            `json:"cancel_requested"`
	UnknownEffects  json.RawMessage `json:"unknown_effects,omitempty"`
	StepReceipts    json.RawMessage `json:"step_receipts,omitempty"`
	ActiveStep      json.RawMessage `json:"active_step,omitempty"`
	Result          json.RawMessage `json:"result,omitempty"`
	Error           json.RawMessage `json:"error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	TerminalAt      *time.Time      `json:"terminal_at,omitempty"`
}

// Ref returns the operation identity.
func (o *CloudOperation) Ref() identity.OperationRef {
	return identity.OperationRef{ID: o.ID, RequestKey: o.RequestKey, PlanDigest: o.PlanDigest, Fence: o.Fence}
}

// OperationResult is the terminal outcome envelope stored in
// cloud_operations.result. RecoveryOutcome keeps operation success distinct
// from recovery success: a failed change may still have restored service.
type OperationResult struct {
	Outcome         string `json:"outcome"`
	RecoveryOutcome string `json:"recovery_outcome,omitempty"`
	CompletedSteps  int    `json:"completed_steps"`
	Message         string `json:"message,omitempty"`
}

// StepOutcome is the durable per-step verdict.
type StepOutcome string

// Step outcomes. Unknown is recorded explicitly, never inferred away.
const (
	StepSucceeded StepOutcome = "succeeded"
	StepFailed    StepOutcome = "failed"
	StepSkipped   StepOutcome = "skipped"
	StepUnchanged StepOutcome = "unchanged"
	StepUnknown   StepOutcome = "unknown"
)

// StepReceipt is one committed step result. Fence is the fence the worker
// held when it committed; Source says whether the worker observed the
// outcome itself or read it back from the target receipt.
type StepReceipt struct {
	Step           string      `json:"step"`
	OwnerOperation string      `json:"owner_operation,omitempty"`
	Outcome        StepOutcome `json:"outcome"`
	Fence          uint64      `json:"fence"`
	Replayed       bool        `json:"replayed,omitempty"`
	Source         string      `json:"source"`
	Detail         string      `json:"detail,omitempty"`
	Error          string      `json:"error,omitempty"`
	StartedAt      string      `json:"started_at,omitempty"`
	CompletedAt    string      `json:"completed_at"`
}

// UnknownEffect records a remote effect whose outcome could not be observed.
// It is the explicit alternative to inferring failure from a timeout.
type UnknownEffect struct {
	Step       string `json:"step"`
	Fence      uint64 `json:"fence"`
	Reason     string `json:"reason"`
	Retry      string `json:"retry"`
	NextAction string `json:"next_action"`
	RecordedAt string `json:"recorded_at"`
}

// ActiveStep is the step the worker was executing when it last wrote. It is
// stored alongside the receipts so reconciliation knows which step was in
// flight when ownership was lost.
type ActiveStep struct {
	Step      string `json:"step"`
	Fence     uint64 `json:"fence"`
	StartedAt string `json:"started_at"`
}

// Receipts decodes the committed step receipts (empty when none).
func (o *CloudOperation) Receipts() ([]StepReceipt, error) {
	var out []StepReceipt
	if len(o.StepReceipts) == 0 || string(o.StepReceipts) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(o.StepReceipts, &out); err != nil {
		return nil, fmt.Errorf("decode step receipts of %s: %w", o.ID, err)
	}
	return out, nil
}

// Receipt returns the latest committed receipt for a step, if any.
func (o *CloudOperation) Receipt(step string) (*StepReceipt, bool) {
	receipts, err := o.Receipts()
	if err != nil {
		return nil, false
	}
	for i := len(receipts) - 1; i >= 0; i-- {
		if receipts[i].Step == step {
			return &receipts[i], true
		}
	}
	return nil, false
}

// UnknownEffectList decodes the recorded unknown effects.
func (o *CloudOperation) UnknownEffectList() ([]UnknownEffect, error) {
	var out []UnknownEffect
	if len(o.UnknownEffects) == 0 || string(o.UnknownEffects) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(o.UnknownEffects, &out); err != nil {
		return nil, fmt.Errorf("decode unknown effects of %s: %w", o.ID, err)
	}
	return out, nil
}

// ActiveStepMarker decodes the in-flight step marker, or nil.
func (o *CloudOperation) ActiveStepMarker() *ActiveStep {
	if len(o.ActiveStep) == 0 || string(o.ActiveStep) == "null" {
		return nil
	}
	var marker ActiveStep
	if err := json.Unmarshal(o.ActiveStep, &marker); err != nil || marker.Step == "" {
		return nil
	}
	return &marker
}

// ResultValue decodes the terminal result, or nil.
func (o *CloudOperation) ResultValue() *OperationResult {
	if len(o.Result) == 0 || string(o.Result) == "null" {
		return nil
	}
	var result OperationResult
	if err := json.Unmarshal(o.Result, &result); err != nil {
		return nil
	}
	return &result
}
