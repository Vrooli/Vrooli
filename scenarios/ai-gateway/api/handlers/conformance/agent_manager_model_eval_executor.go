package conformance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

var modelEvalSourceUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	agentManagerModelEvalPath  = "/api/v1/model-eval/batch"
	agentManagerModelEvalID    = "LM2-INFERENCE-ADMISSION-20261116"
	agentManagerModelEvalScope = "one local-only, sequential batch of up to 20 fixed Agent Manager caller cases using the already-installed chat.default model; metadata-only receipts; no acquisition, remote inference, promotion, or deletion"
	agentManagerModelEvalMax   = 20
	agentManagerMinClaimBytes  = int64(13_541_310_464)
	agentManagerMinDiskBytes   = int64(120 * 1024 * 1024 * 1024)
)

// DiscoveredAgentManagerModelEvalExecutor delegates source selection, source
// authentication, admission, and dispatch to Agent Manager. Only the owner's
// metadata-only response is returned to the Test Genie phase.
type DiscoveredAgentManagerModelEvalExecutor struct {
	HTTPClient *http.Client
	ResolveURL func(context.Context) (string, error)
}

type agentManagerModelEvalRequest struct {
	DecisionID string `json:"decision_id"`
	MaxCases   int    `json:"max_cases"`
}

type agentManagerModelEvalResponse struct {
	State       string                        `json:"state"`
	FailureCode string                        `json:"failure_code,omitempty"`
	Receipt     *agentManagerModelEvalReceipt `json:"receipt,omitempty"`
}

type agentManagerModelEvalReceipt struct {
	DecisionID          string                             `json:"decision_id"`
	DecisionScope       string                             `json:"decision_scope"`
	CaseSetSHA256       string                             `json:"case_set_sha256"`
	Denominator         int                                `json:"denominator"`
	Succeeded           int                                `json:"succeeded"`
	Failed              int                                `json:"failed"`
	Refused             int                                `json:"refused"`
	Unattempted         int                                `json:"unattempted"`
	EffectiveModel      string                             `json:"effective_model,omitempty"`
	DiskFreeBefore      int64                              `json:"disk_free_bytes_before,omitempty"`
	DiskFreeAfter       int64                              `json:"disk_free_bytes_after,omitempty"`
	ArtifactBytesBefore int64                              `json:"artifact_bytes_before,omitempty"`
	ArtifactBytesAfter  int64                              `json:"artifact_bytes_after,omitempty"`
	ArtifactHashBefore  string                             `json:"artifact_sha256_before,omitempty"`
	ArtifactHashAfter   string                             `json:"artifact_sha256_after,omitempty"`
	ReceiptBytes        int64                              `json:"metadata_receipt_bytes,omitempty"`
	ReceiptSHA256       string                             `json:"metadata_receipt_sha256,omitempty"`
	Cases               []agentManagerModelEvalCaseReceipt `json:"cases"`
}

type agentManagerModelEvalCaseReceipt struct {
	SourceEventID   string `json:"source_event_id"`
	SourceRunID     string `json:"source_run_id"`
	SourceSequence  int64  `json:"source_sequence"`
	EventTimestamp  string `json:"event_timestamp,omitempty"`
	RequestID       string `json:"request_id"`
	TaskRole        string `json:"task_role"`
	PolicyRole      string `json:"policy_role"`
	Consumer        string `json:"consumer"`
	GatewayEventID  string `json:"gateway_event_id,omitempty"`
	EffectiveModel  string `json:"effective_model,omitempty"`
	Locality        string `json:"locality,omitempty"`
	LatencyMillis   int64  `json:"latency_ms,omitempty"`
	RightsResult    string `json:"rights_result"`
	CapacityClaimID string `json:"capacity_claim_id,omitempty"`
	RequiredBytes   int64  `json:"capacity_required_bytes,omitempty"`
	GrantedBytes    int64  `json:"capacity_granted_bytes,omitempty"`
	Status          string `json:"status"`
	FailureCode     string `json:"failure_code,omitempty"`
}

func (e *DiscoveredAgentManagerModelEvalExecutor) ExecuteAgentManagerModelEval(ctx context.Context) (ModelEvalExecutionReceipt, error) {
	if e == nil {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval executor unavailable")
	}
	resolve := e.ResolveURL
	if resolve == nil {
		resolve = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
		}
	}
	baseURL, err := resolve(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval owner discovery failed")
	}
	body, err := json.Marshal(agentManagerModelEvalRequest{DecisionID: agentManagerModelEvalID, MaxCases: agentManagerModelEvalMax})
	if err != nil {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval request encoding failed")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+agentManagerModelEvalPath, bytes.NewReader(body))
	if err != nil {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval request could not be created")
	}
	request.Header.Set("Content-Type", "application/json")
	client := e.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Minute}
	}
	response, err := client.Do(request)
	if err != nil {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval owner request failed")
	}
	defer response.Body.Close()
	var result agentManagerModelEvalResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&result); err != nil {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval owner response was invalid")
	}
	if result.Receipt == nil {
		return ModelEvalExecutionReceipt{}, fmt.Errorf("Agent Manager model-eval owner returned no metadata receipt (state=%s)", result.State)
	}
	if err := validateAgentManagerModelEvalReceipt(*result.Receipt, result.State); err != nil {
		return ModelEvalExecutionReceipt{}, errors.New("Agent Manager model-eval owner receipt violated the bounded metadata contract")
	}
	receipt := mapAgentManagerModelEvalReceipt(*result.Receipt)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices || result.State != "succeeded" {
		return receipt, errors.New("Agent Manager refused or failed the bounded model-eval batch")
	}
	return receipt, nil
}

func validateAgentManagerModelEvalReceipt(receipt agentManagerModelEvalReceipt, state string) error {
	if (state != "succeeded" && state != "failed" && state != "refused") ||
		receipt.DecisionID != agentManagerModelEvalID || receipt.DecisionScope != agentManagerModelEvalScope ||
		receipt.Denominator < 1 || receipt.Denominator > agentManagerModelEvalMax || len(receipt.Cases) != receipt.Denominator ||
		!validModelEvalSHA256(receipt.CaseSetSHA256) || receipt.Succeeded+receipt.Failed+receipt.Refused+receipt.Unattempted != receipt.Denominator ||
		receipt.ReceiptBytes <= 0 || receipt.ReceiptBytes > 10<<20 || !validModelEvalSHA256(receipt.ReceiptSHA256) {
		return errors.New("top-level model-eval receipt contract is invalid")
	}
	unsigned := receipt
	unsigned.ReceiptBytes = 0
	unsigned.ReceiptSHA256 = ""
	unsignedBytes, err := json.Marshal(unsigned)
	if err != nil {
		return errors.New("owner receipt digest could not be recomputed")
	}
	unsignedDigest := sha256.Sum256(unsignedBytes)
	completeBytes, err := json.Marshal(receipt)
	if err != nil || int64(len(completeBytes)) != receipt.ReceiptBytes || hex.EncodeToString(unsignedDigest[:]) != strings.ToLower(receipt.ReceiptSHA256) {
		return errors.New("owner receipt bytes or digest do not match its metadata")
	}
	caseSetDigest, err := ownerCaseSetDigest(receipt.Cases)
	if err != nil || caseSetDigest != strings.ToLower(receipt.CaseSetSHA256) {
		return errors.New("owner case-set digest does not match the fixed case identities")
	}
	counts := map[string]int{"succeeded": 0, "failed": 0, "refused": 0, "unattempted": 0}
	for _, item := range receipt.Cases {
		if _, ok := counts[item.Status]; !ok || !modelEvalSourceUUID.MatchString(item.SourceEventID) || !modelEvalSourceUUID.MatchString(item.SourceRunID) || item.SourceSequence < 0 || item.EventTimestamp == "" || item.RequestID == "" || item.TaskRole == "" || item.PolicyRole != "chat.default" || item.Consumer != "agent-manager" {
			return errors.New("case identity or status is invalid")
		}
		counts[item.Status]++
		if item.Status == "succeeded" && (item.Locality != "local" || item.EffectiveModel == "" || item.RightsResult != "allowed" || item.CapacityClaimID == "" || item.RequiredBytes < agentManagerMinClaimBytes || item.GrantedBytes < agentManagerMinClaimBytes) {
			return errors.New("successful case lacks local route, rights, or capacity proof")
		}
	}
	if counts["succeeded"] != receipt.Succeeded || counts["failed"] != receipt.Failed || counts["refused"] != receipt.Refused || counts["unattempted"] != receipt.Unattempted {
		return errors.New("case outcomes do not match receipt counts")
	}
	if receipt.ArtifactBytesBefore < 0 || receipt.ArtifactBytesBefore > 10<<20 || receipt.ArtifactBytesAfter < 0 || receipt.ArtifactBytesAfter > 10<<20 ||
		(receipt.ArtifactBytesBefore > 0 && !validModelEvalSHA256(receipt.ArtifactHashBefore)) ||
		(receipt.ArtifactBytesAfter > 0 && !validModelEvalSHA256(receipt.ArtifactHashAfter)) ||
		(receipt.DiskFreeBefore > 0 && !validModelEvalSHA256(receipt.ArtifactHashBefore)) ||
		(receipt.DiskFreeAfter > 0 && !validModelEvalSHA256(receipt.ArtifactHashAfter)) {
		return errors.New("task artifact measurements are invalid")
	}
	if state == "succeeded" && (receipt.Succeeded != receipt.Denominator || receipt.Failed != 0 || receipt.Refused != 0 || receipt.Unattempted != 0 ||
		receipt.EffectiveModel == "" || receipt.DiskFreeBefore < agentManagerMinDiskBytes || receipt.DiskFreeAfter < agentManagerMinDiskBytes ||
		!validModelEvalSHA256(receipt.ArtifactHashBefore) || !validModelEvalSHA256(receipt.ArtifactHashAfter)) {
		return errors.New("successful batch lacks complete postconditions")
	}
	return nil
}

func ownerCaseSetDigest(cases []agentManagerModelEvalCaseReceipt) (string, error) {
	type sourceIdentity struct {
		EventID    string `json:"event_id"`
		RunID      string `json:"run_id"`
		Sequence   int64  `json:"sequence"`
		Timestamp  string `json:"event_timestamp"`
		TaskRole   string `json:"task_role"`
		RequestID  string `json:"request_id"`
		PolicyRole string `json:"policy_role"`
		Consumer   string `json:"consumer"`
	}
	identities := make([]sourceIdentity, 0, len(cases))
	for _, item := range cases {
		identities = append(identities, sourceIdentity{
			EventID: item.SourceEventID, RunID: item.SourceRunID, Sequence: item.SourceSequence,
			Timestamp: item.EventTimestamp, TaskRole: item.TaskRole, RequestID: item.RequestID,
			PolicyRole: item.PolicyRole, Consumer: item.Consumer,
		})
	}
	data, err := json.Marshal(identities)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func validModelEvalSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F') {
			return false
		}
	}
	return true
}

func mapAgentManagerModelEvalReceipt(source agentManagerModelEvalReceipt) ModelEvalExecutionReceipt {
	receipt := ModelEvalExecutionReceipt{
		DecisionID: source.DecisionID, DecisionScope: source.DecisionScope,
		CaseSetSHA256: source.CaseSetSHA256, Denominator: source.Denominator,
		Succeeded: source.Succeeded, Failed: source.Failed, Refused: source.Refused,
		Unattempted: source.Unattempted, EffectiveModel: source.EffectiveModel,
		DiskFreeBytesBefore: source.DiskFreeBefore, DiskFreeBytesAfter: source.DiskFreeAfter,
		ArtifactBytesBefore: source.ArtifactBytesBefore, ArtifactBytesAfter: source.ArtifactBytesAfter,
		ArtifactSHA256Before: source.ArtifactHashBefore, ArtifactSHA256After: source.ArtifactHashAfter,
		MetadataReceiptBytes: source.ReceiptBytes, MetadataReceiptSHA256: source.ReceiptSHA256,
		Cases: make([]ModelEvalExecutionCase, 0, len(source.Cases)),
	}
	for _, item := range source.Cases {
		receipt.Cases = append(receipt.Cases, ModelEvalExecutionCase{
			SourceEventID: item.SourceEventID, SourceRunID: item.SourceRunID,
			SourceSequence: item.SourceSequence, EventTimestamp: item.EventTimestamp,
			RequestID: item.RequestID, TaskRole: item.TaskRole, PolicyRole: item.PolicyRole,
			Consumer: item.Consumer, GatewayEventID: item.GatewayEventID,
			EffectiveModel: item.EffectiveModel, Locality: item.Locality,
			LatencyMillis: item.LatencyMillis, CapacityClaimID: item.CapacityClaimID,
			RightsResult:          item.RightsResult,
			CapacityRequiredBytes: item.RequiredBytes, CapacityGrantedBytes: item.GrantedBytes,
			Status: item.Status, FailureCode: item.FailureCode,
		})
	}
	return receipt
}
