package conformance

import (
	"context"

	internalconformance "ai-gateway/internal/conformance"
	internalrouting "ai-gateway/internal/routing"
	"github.com/vrooli/maturity-go/assessment"
)

type Deps struct {
	Scanner             *internalconformance.Scanner
	MaturitySpec        *assessment.Spec
	ScenarioDir         string
	ModelPolicyPath     string
	TaskRoleCatalogPath string
	ModelEvalSource     internalconformance.ModelEvalReadinessSource
	ModelEvalConsumers  interface {
		ModelEvalConsumers(context.Context) ([]string, error)
	}
	ModelEvalRuntimeModels func(context.Context) ([]internalrouting.ModelEvalRouteModel, error)
	ModelEvalExecutor      ModelEvalExecutor
}

// ModelEvalExecutor is the phase-owned boundary to Agent Manager's
// approval-gated source operation. The returned shape intentionally contains
// only metadata; it has no prompt or output field.
type ModelEvalExecutor interface {
	ExecuteAgentManagerModelEval(context.Context) (ModelEvalExecutionReceipt, error)
}

type ModelEvalExecutionCase struct {
	SourceEventID   string `json:"source_event_id"`
	SourceRunID     string `json:"source_run_id"`
	SourceSequence  int64  `json:"source_sequence"`
	RequestID       string `json:"request_id"`
	TaskRole        string `json:"task_role"`
	PolicyRole      string `json:"policy_role"`
	Consumer        string `json:"consumer"`
	GatewayEventID  string `json:"gateway_event_id,omitempty"`
	EffectiveModel  string `json:"effective_model,omitempty"`
	Locality        string `json:"locality,omitempty"`
	CapacityClaimID string `json:"capacity_claim_id,omitempty"`
	Status          string `json:"status"`
	FailureCode     string `json:"failure_code,omitempty"`
}

type ModelEvalExecutionReceipt struct {
	DecisionID     string                   `json:"decision_id"`
	CaseSetSHA256  string                   `json:"case_set_sha256"`
	Denominator    int                      `json:"denominator"`
	Succeeded      int                      `json:"succeeded"`
	Failed         int                      `json:"failed"`
	Refused        int                      `json:"refused"`
	Unattempted    int                      `json:"unattempted"`
	EffectiveModel string                   `json:"effective_model,omitempty"`
	Cases          []ModelEvalExecutionCase `json:"cases"`
}
