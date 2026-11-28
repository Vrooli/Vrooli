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
	SourceEventID         string `json:"source_event_id"`
	SourceRunID           string `json:"source_run_id"`
	SourceSequence        int64  `json:"source_sequence"`
	EventTimestamp        string `json:"event_timestamp,omitempty"`
	RequestID             string `json:"request_id"`
	TaskRole              string `json:"task_role"`
	PolicyRole            string `json:"policy_role"`
	Consumer              string `json:"consumer"`
	GatewayEventID        string `json:"gateway_event_id,omitempty"`
	EffectiveModel        string `json:"effective_model,omitempty"`
	Locality              string `json:"locality,omitempty"`
	LatencyMillis         int64  `json:"latency_ms,omitempty"`
	RightsResult          string `json:"rights_result"`
	CapacityClaimID       string `json:"capacity_claim_id,omitempty"`
	CapacityRequiredBytes int64  `json:"capacity_required_bytes,omitempty"`
	CapacityGrantedBytes  int64  `json:"capacity_granted_bytes,omitempty"`
	Status                string `json:"status"`
	FailureCode           string `json:"failure_code,omitempty"`
}

type ModelEvalExecutionReceipt struct {
	DecisionID            string                   `json:"decision_id"`
	DecisionScope         string                   `json:"decision_scope"`
	CaseSetSHA256         string                   `json:"case_set_sha256"`
	Denominator           int                      `json:"denominator"`
	Succeeded             int                      `json:"succeeded"`
	Failed                int                      `json:"failed"`
	Refused               int                      `json:"refused"`
	Unattempted           int                      `json:"unattempted"`
	EffectiveModel        string                   `json:"effective_model,omitempty"`
	DiskFreeBytesBefore   int64                    `json:"disk_free_bytes_before,omitempty"`
	DiskFreeBytesAfter    int64                    `json:"disk_free_bytes_after,omitempty"`
	ArtifactBytesBefore   int64                    `json:"artifact_bytes_before,omitempty"`
	ArtifactBytesAfter    int64                    `json:"artifact_bytes_after,omitempty"`
	ArtifactSHA256Before  string                   `json:"artifact_sha256_before,omitempty"`
	ArtifactSHA256After   string                   `json:"artifact_sha256_after,omitempty"`
	MetadataReceiptBytes  int64                    `json:"metadata_receipt_bytes,omitempty"`
	MetadataReceiptSHA256 string                   `json:"metadata_receipt_sha256,omitempty"`
	Cases                 []ModelEvalExecutionCase `json:"cases"`
}
