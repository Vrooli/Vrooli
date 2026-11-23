package model_eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

const (
	AdmissionDecisionID = "LM2-INFERENCE-ADMISSION-20261116"
	AdmissionScope      = "one local-only, sequential batch of up to 20 fixed Agent Manager caller cases using the already-installed chat.default model; metadata-only receipts; no acquisition, remote inference, promotion, or deletion"
	MaximumCases        = 20
	MinimumDiskBytes    = int64(120 * 1024 * 1024 * 1024)
	MaximumArtifactSize = int64(10 * 1024 * 1024)
	RequiredVRAMBytes   = int64(13_541_310_464)
)

var ErrAdmissionRefused = errors.New("model-eval admission refused")

type Approval struct {
	DecisionID string
	Answer     string
	Scope      string
	RecordedAt time.Time
}

type ApprovalSource interface {
	ReadDecision(context.Context, string) (Approval, error)
}

// DispatchAdmissionClaimer atomically consumes this one-time approval only
// after every non-dispatch gate passes and immediately before the first call.
type DispatchAdmissionClaimer interface {
	ClaimDispatch(context.Context, Approval, string) error
}

// SourceRef contains no message body. It is the frozen identity sent from the
// model-eval coordinator; Agent Manager re-reads and verifies every source.
type SourceRef struct {
	EventID    string `json:"event_id"`
	RunID      string `json:"run_id"`
	Sequence   int64  `json:"sequence"`
	Timestamp  string `json:"event_timestamp"`
	TaskRole   string `json:"task_role"`
	RequestID  string `json:"request_id"`
	PolicyRole string `json:"policy_role"`
	Consumer   string `json:"consumer"`
}

type Request struct {
	DecisionID  string      `json:"decision_id"`
	CaseSetHash string      `json:"case_set_sha256"`
	MaxCases    int         `json:"max_cases,omitempty"`
	Cases       []SourceRef `json:"cases"`
}

type VerifiedCase struct {
	Ref       SourceRef
	Timestamp time.Time
	Source    string
	Prompt    string
}

type Source interface {
	VerifyAndLoad(context.Context, SourceRef) (VerifiedCase, error)
}

type CaseSelector interface {
	Select(context.Context, int) ([]SourceRef, error)
}

type PrivacyVerifier interface {
	ProveNoPromptOrOutputPersistence(context.Context) error
}

// GatewayPrivacyVerifier is a fail-closed latch for the complete deployed
// boundary. Source inspection and focused tests cover owner persistence, but
// the latch cannot be set until transport, provider and logging sinks are also
// verified for the serving deployment.
type GatewayPrivacyVerifier struct {
	// Complete is set only after the owner proves the deployed transport,
	// provider and logging sinks. It remains false in production until then.
	Complete bool
}

func (v GatewayPrivacyVerifier) ProveNoPromptOrOutputPersistence(context.Context) error {
	if !v.Complete {
		return errors.New("complete deployed prompt/output non-persistence proof is not available")
	}
	return nil
}

type DiskObservation struct {
	ObservedAt    time.Time
	FreeBytes     int64
	ArtifactBytes int64
	ArtifactHash  string
}

type DiskGate interface {
	Measure(context.Context) (DiskObservation, error)
}

type CapacityObservation struct {
	ObservedAt time.Time
	Available  int64
	Fits       bool
}

type CapacityGate interface {
	MeasureAndCheck(context.Context, int64) (CapacityObservation, error)
}

type Gateway interface {
	Preview(context.Context, *sharedv1.GatewayRequest) (*routingv1.PreviewRouteResponse, error)
	Execute(context.Context, *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error)
}

type CaseStatus string

const (
	StatusSucceeded   CaseStatus = "succeeded"
	StatusFailed      CaseStatus = "failed"
	StatusRefused     CaseStatus = "refused"
	StatusUnattempted CaseStatus = "unattempted"
)

type CaseReceipt struct {
	SourceEventID   string     `json:"source_event_id"`
	SourceRunID     string     `json:"source_run_id"`
	SourceSequence  int64      `json:"source_sequence"`
	EventTimestamp  string     `json:"event_timestamp,omitempty"`
	RequestID       string     `json:"request_id"`
	TaskRole        string     `json:"task_role"`
	PolicyRole      string     `json:"policy_role"`
	Consumer        string     `json:"consumer"`
	GatewayEventID  string     `json:"gateway_event_id,omitempty"`
	EffectiveModel  string     `json:"effective_model,omitempty"`
	Locality        string     `json:"locality,omitempty"`
	LatencyMillis   int64      `json:"latency_ms,omitempty"`
	RightsResult    string     `json:"rights_result"`
	CapacityClaimID string     `json:"capacity_claim_id,omitempty"`
	RequiredBytes   int64      `json:"capacity_required_bytes,omitempty"`
	GrantedBytes    int64      `json:"capacity_granted_bytes,omitempty"`
	Status          CaseStatus `json:"status"`
	FailureCode     string     `json:"failure_code,omitempty"`
}

type Receipt struct {
	DecisionID          string        `json:"decision_id"`
	DecisionScope       string        `json:"decision_scope"`
	CaseSetSHA256       string        `json:"case_set_sha256"`
	Denominator         int           `json:"denominator"`
	Succeeded           int           `json:"succeeded"`
	Failed              int           `json:"failed"`
	Refused             int           `json:"refused"`
	Unattempted         int           `json:"unattempted"`
	EffectiveModel      string        `json:"effective_model,omitempty"`
	DiskFreeBefore      int64         `json:"disk_free_bytes_before,omitempty"`
	DiskFreeAfter       int64         `json:"disk_free_bytes_after,omitempty"`
	ArtifactBytesBefore int64         `json:"artifact_bytes_before,omitempty"`
	ArtifactBytesAfter  int64         `json:"artifact_bytes_after,omitempty"`
	ArtifactHashBefore  string        `json:"artifact_sha256_before,omitempty"`
	ArtifactHashAfter   string        `json:"artifact_sha256_after,omitempty"`
	ReceiptBytes        int64         `json:"metadata_receipt_bytes,omitempty"`
	ReceiptSHA256       string        `json:"metadata_receipt_sha256,omitempty"`
	Cases               []CaseReceipt `json:"cases"`
}

type Runner struct {
	Approval ApprovalSource
	Selector CaseSelector
	Source   Source
	Privacy  PrivacyVerifier
	Disk     DiskGate
	Capacity CapacityGate
	Gateway  Gateway
	Now      func() time.Time
}

func (r Runner) Execute(ctx context.Context, request Request) (receipt Receipt, returnErr error) {
	defer func() { finalizeReceipt(&receipt) }()
	if r.Approval == nil || r.Source == nil || r.Privacy == nil || r.Disk == nil || r.Capacity == nil || r.Gateway == nil {
		return Receipt{}, fmt.Errorf("%w: required owner path is unavailable", ErrAdmissionRefused)
	}
	if request.DecisionID != AdmissionDecisionID {
		return Receipt{}, fmt.Errorf("%w: exact decision id is required", ErrAdmissionRefused)
	}
	approval, err := r.Approval.ReadDecision(ctx, AdmissionDecisionID)
	if err != nil || approval.DecisionID != AdmissionDecisionID || strings.ToLower(strings.TrimSpace(approval.Answer)) != "approve" || approval.Scope != AdmissionScope || approval.RecordedAt.IsZero() {
		return Receipt{}, fmt.Errorf("%w: exact current approve answer and scope are required", ErrAdmissionRefused)
	}
	if len(request.Cases) == 0 {
		if r.Selector == nil {
			return Receipt{}, fmt.Errorf("%w: Agent Manager case selector is unavailable", ErrAdmissionRefused)
		}
		limit := request.MaxCases
		if limit == 0 {
			limit = MaximumCases
		}
		if limit < 1 || limit > MaximumCases {
			return Receipt{}, fmt.Errorf("%w: maximum case count is outside the approved bound", ErrAdmissionRefused)
		}
		request.Cases, err = r.Selector.Select(ctx, limit)
		if err != nil {
			return Receipt{}, fmt.Errorf("%w: Agent Manager could not select a fixed source set", ErrAdmissionRefused)
		}
		request.CaseSetHash, err = CaseSetDigest(request.Cases)
		if err != nil {
			return Receipt{}, fmt.Errorf("%w: Agent Manager could not freeze the source set", ErrAdmissionRefused)
		}
	}
	if err := validateRefs(request.Cases); err != nil {
		return Receipt{}, fmt.Errorf("%w: invalid frozen case set", ErrAdmissionRefused)
	}
	digest, err := CaseSetDigest(request.Cases)
	if err != nil || !strings.EqualFold(digest, request.CaseSetHash) {
		return Receipt{}, fmt.Errorf("%w: frozen case-set digest mismatch", ErrAdmissionRefused)
	}
	receipt = Receipt{DecisionID: approval.DecisionID, DecisionScope: approval.Scope, CaseSetSHA256: digest, Denominator: len(request.Cases), Cases: make([]CaseReceipt, 0, len(request.Cases))}
	for _, ref := range request.Cases {
		receipt.Cases = append(receipt.Cases, receiptForRef(ref))
	}
	if err := r.Privacy.ProveNoPromptOrOutputPersistence(ctx); err != nil {
		refuseAll(&receipt, "prompt_output_non_persistence_unproven")
		return receipt, fmt.Errorf("%w: prompt/output privacy path is not proven", ErrAdmissionRefused)
	}
	_, err = r.previewAll(ctx, request.Cases)
	if err != nil {
		refuseAll(&receipt, "current_local_policy_or_rights_unavailable")
		return receipt, fmt.Errorf("%w: current local policy or rights not proven", ErrAdmissionRefused)
	}
	for i := range receipt.Cases {
		receipt.Cases[i].RightsResult = "allowed"
	}
	diskBefore, capacity, err := r.preflight(ctx)
	if err != nil {
		refuseAll(&receipt, "fresh_disk_or_capacity_gate_failed")
		return receipt, fmt.Errorf("%w: fresh disk or capacity gate failed", ErrAdmissionRefused)
	}
	_ = capacity // The actual operation claim is acquired by public Gateway execution.
	receipt.DiskFreeBefore = diskBefore.FreeBytes
	receipt.ArtifactBytesBefore = diskBefore.ArtifactBytes
	receipt.ArtifactHashBefore = strings.ToLower(diskBefore.ArtifactHash)

	for i, ref := range request.Cases {
		if _, err := r.previewCase(ctx, ref); err != nil {
			receipt.Cases[i].RightsResult = "unknown"
			markRefusedThenUnattempted(&receipt, i, "current_local_policy_or_rights_changed")
			return receipt, fmt.Errorf("%w: current policy or rights changed", ErrAdmissionRefused)
		}
		if err := r.Privacy.ProveNoPromptOrOutputPersistence(ctx); err != nil {
			markRefusedThenUnattempted(&receipt, i, "prompt_output_non_persistence_unproven")
			return receipt, fmt.Errorf("%w: prompt/output privacy proof expired", ErrAdmissionRefused)
		}
		// Read the source body only after all gates pass and immediately before
		// the one corresponding dispatch. The prompt stays in this stack frame;
		// future case bodies are not preloaded into the process.
		item, err := r.Source.VerifyAndLoad(ctx, ref)
		if err != nil || !sameRef(ref, item.Ref) || item.Source != domain.MessageSourceOperator || item.Timestamp.IsZero() || strings.TrimSpace(item.Prompt) == "" {
			markRefusedThenUnattempted(&receipt, i, "source_identity_or_operator_message_unverified")
			return receipt, fmt.Errorf("%w: source identity did not verify", ErrAdmissionRefused)
		}
		receipt.Cases[i].EventTimestamp = item.Timestamp.UTC().Format(time.RFC3339Nano)
		if i == 0 {
			claimer, ok := r.Approval.(DispatchAdmissionClaimer)
			if !ok || claimer.ClaimDispatch(ctx, approval, digest) != nil {
				markRefusedThenUnattempted(&receipt, i, "one_time_dispatch_admission_already_consumed_or_unavailable")
				return receipt, fmt.Errorf("%w: one-time dispatch admission could not be claimed", ErrAdmissionRefused)
			}
		}
		started := time.Now()
		if r.Now != nil {
			started = r.Now()
		}
		vram := RequiredVRAMBytes
		response, callErr := r.Gateway.Execute(ctx, &routingv1.ExecuteRouteRequest{
			Request: gatewayRequest(item.Ref, vram), InputText: item.Prompt,
		})
		elapsed := time.Since(started)
		if r.Now != nil {
			elapsed = r.Now().Sub(started)
		}
		if response == nil || response.GetEvidence() == nil {
			receipt.Cases[i].Status = StatusFailed
			receipt.Cases[i].FailureCode = "gateway_receipt_missing"
		} else {
			ev := response.GetEvidence()
			receipt.Cases[i].GatewayEventID = ev.GetEventId()
			receipt.Cases[i].EffectiveModel = ev.GetSelectedModel()
			receipt.Cases[i].Locality = ev.GetSelectedLocality()
			receipt.Cases[i].LatencyMillis = elapsed.Milliseconds()
			receipt.Cases[i].CapacityClaimID = ev.GetCapacityClaimId()
			receipt.Cases[i].RequiredBytes = ev.GetCapacityRequiredBytes()
			receipt.Cases[i].GrantedBytes = ev.GetCapacityGrantedBytes()
			if callErr != nil || ev.GetStatus() != "succeeded" || ev.GetRequestId() != item.Ref.RequestID || ev.GetTaskRole() != item.Ref.TaskRole || ev.GetSelectedLocality() != "local" || strings.TrimSpace(ev.GetSelectedModel()) == "" || (receipt.EffectiveModel != "" && ev.GetSelectedModel() != receipt.EffectiveModel) || ev.GetCapacityVerdict() != "fit" || ev.GetCapacityClaimId() == "" || ev.GetCapacityRequiredBytes() < RequiredVRAMBytes || ev.GetCapacityGrantedBytes() < RequiredVRAMBytes {
				receipt.Cases[i].Status = StatusFailed
				receipt.Cases[i].FailureCode = "gateway_execution_or_receipt_contract_failed"
			} else {
				receipt.EffectiveModel = ev.GetSelectedModel()
				receipt.Cases[i].Status = StatusSucceeded
			}
		}
		diskAfter, diskErr := r.Disk.Measure(ctx)
		if diskErr == nil {
			receipt.DiskFreeAfter = diskAfter.FreeBytes
			receipt.ArtifactBytesAfter = diskAfter.ArtifactBytes
			receipt.ArtifactHashAfter = strings.ToLower(diskAfter.ArtifactHash)
		}
		if receipt.Cases[i].Status == StatusFailed || diskErr != nil || !validDisk(diskAfter, now(r)) {
			if receipt.Cases[i].Status != StatusFailed {
				receipt.Cases[i].Status = StatusFailed
				receipt.Cases[i].FailureCode = "post_dispatch_disk_gate_failed"
			}
			markUnattempted(&receipt, i+1)
			countOutcomes(&receipt)
			return receipt, errors.New("model-eval stopped after a failed case or postcondition")
		}
	}
	countOutcomes(&receipt)
	return receipt, nil
}

func (r Runner) previewAll(ctx context.Context, refs []SourceRef) (string, error) {
	provider := ""
	for _, ref := range refs {
		selected, err := r.previewCase(ctx, ref)
		if err != nil || (provider != "" && provider != selected) {
			return "", errors.New("local route preview failed or changed")
		}
		provider = selected
	}
	return provider, nil
}

func (r Runner) previewCase(ctx context.Context, ref SourceRef) (string, error) {
	request := gatewayRequest(ref, RequiredVRAMBytes)
	preview, err := r.Gateway.Preview(ctx, request)
	if err != nil || preview == nil || !preview.GetValid() || strings.TrimSpace(preview.GetSelectedProvider()) == "" {
		return "", errors.New("local route preview unavailable")
	}
	for _, candidate := range preview.GetCandidates() {
		if candidate.GetSelected() {
			if !strings.EqualFold(candidate.GetLocality(), "local") {
				return "", errors.New("remote route is forbidden")
			}
			if candidate.GetRole() != "chat.default" {
				return "", errors.New("selected route role is not chat.default")
			}
			return candidate.GetProvider(), nil
		}
	}
	return "", errors.New("no selected local route")
}

func (r Runner) preflight(ctx context.Context) (DiskObservation, CapacityObservation, error) {
	measuredAt := now(r)
	disk, err := r.Disk.Measure(ctx)
	if err != nil || !validDisk(disk, measuredAt) {
		return DiskObservation{}, CapacityObservation{}, errors.New("disk or artifact bound failed")
	}
	capacity, err := r.Capacity.MeasureAndCheck(ctx, RequiredVRAMBytes)
	if err != nil || !capacity.Fits || capacity.Available < RequiredVRAMBytes || capacity.ObservedAt.IsZero() || measuredAt.Sub(capacity.ObservedAt) > 30*time.Second || capacity.ObservedAt.After(measuredAt.Add(10*time.Second)) {
		return DiskObservation{}, CapacityObservation{}, errors.New("full host/broker capacity did not fit")
	}
	return disk, capacity, nil
}

func gatewayRequest(ref SourceRef, required int64) *sharedv1.GatewayRequest {
	return &sharedv1.GatewayRequest{
		Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role: "chat.default", TaskRole: ref.TaskRole,
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "agent-manager.operator_source_message",
		Operation:         "model-eval", Scenario: "agent-manager",
		RequestId: ref.RequestID, RequiredVramBytes: &required, MaxOutputTokens: 128,
	}
}

func CaseSetDigest(refs []SourceRef) (string, error) {
	encoded, err := json.Marshal(refs)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateRefs(refs []SourceRef) error {
	if len(refs) == 0 || len(refs) > MaximumCases {
		return errors.New("case count outside bounded range")
	}
	events, requests := map[string]struct{}{}, map[string]struct{}{}
	for _, ref := range refs {
		if _, err := uuid.Parse(ref.EventID); err != nil {
			return errors.New("source event ID is invalid")
		}
		if _, err := time.Parse(time.RFC3339Nano, ref.Timestamp); err != nil || strings.TrimSpace(ref.Timestamp) == "" {
			return errors.New("source event timestamp is invalid")
		}
		if _, err := uuid.Parse(ref.RunID); err != nil || ref.Sequence < 0 || strings.TrimSpace(ref.TaskRole) == "" || strings.TrimSpace(ref.RequestID) == "" || ref.PolicyRole != "chat.default" || ref.Consumer != "agent-manager" {
			return errors.New("case identity or caller metadata is incomplete")
		}
		key := ref.RunID + "/" + ref.EventID + "/" + fmt.Sprint(ref.Sequence)
		if _, ok := events[key]; ok {
			return errors.New("duplicate source event identity")
		}
		if _, ok := requests[ref.RequestID]; ok {
			return errors.New("duplicate request identity")
		}
		events[key], requests[ref.RequestID] = struct{}{}, struct{}{}
	}
	return nil
}

func verifyRequestIdentity(ref SourceRef) bool {
	requestID, err := searcheval.AgentManagerReplayRequestID(ref.RunID, ref.EventID, ref.Sequence, ref.TaskRole, ref.Consumer)
	return err == nil && requestID == ref.RequestID
}

func validDisk(disk DiskObservation, at time.Time) bool {
	artifactValid := disk.ArtifactBytes >= 0 && disk.ArtifactBytes <= MaximumArtifactSize && validHash(disk.ArtifactHash)
	return !disk.ObservedAt.IsZero() && at.Sub(disk.ObservedAt) <= 2*time.Minute && !disk.ObservedAt.After(at.Add(30*time.Second)) && disk.FreeBytes >= MinimumDiskBytes && artifactValid
}

func finalizeReceipt(receipt *Receipt) {
	if receipt == nil || len(receipt.Cases) == 0 {
		return
	}
	copyReceipt := *receipt
	copyReceipt.ReceiptBytes = 0
	copyReceipt.ReceiptSHA256 = ""
	data, err := json.Marshal(copyReceipt)
	if err != nil || int64(len(data)) > MaximumArtifactSize {
		return
	}
	receipt.ReceiptSHA256 = hashBytes(data)
	// ReceiptBytes measures the complete serialized receipt, including its
	// digest and byte-count fields. Iterate to the fixed point because the
	// decimal digit count itself contributes to the encoded length.
	for i := 0; i < 4; i++ {
		encoded, marshalErr := json.Marshal(receipt)
		if marshalErr != nil || int64(len(encoded)) > MaximumArtifactSize {
			receipt.ReceiptBytes = 0
			receipt.ReceiptSHA256 = ""
			return
		}
		if receipt.ReceiptBytes == int64(len(encoded)) {
			return
		}
		receipt.ReceiptBytes = int64(len(encoded))
	}
}

func hashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func validHash(value string) bool {
	if len(strings.TrimSpace(value)) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func now(r Runner) time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

func sameRef(a, b SourceRef) bool {
	return a.EventID == b.EventID && a.RunID == b.RunID && a.Sequence == b.Sequence && a.Timestamp == b.Timestamp && a.TaskRole == b.TaskRole && a.RequestID == b.RequestID && a.PolicyRole == b.PolicyRole && a.Consumer == b.Consumer
}

func receiptForRef(ref SourceRef) CaseReceipt {
	return CaseReceipt{SourceEventID: ref.EventID, SourceRunID: ref.RunID, SourceSequence: ref.Sequence, EventTimestamp: ref.Timestamp, RequestID: ref.RequestID, TaskRole: ref.TaskRole, PolicyRole: ref.PolicyRole, Consumer: ref.Consumer, RightsResult: "unknown"}
}

func refuseAll(receipt *Receipt, code string) {
	for i := range receipt.Cases {
		receipt.Cases[i].Status = StatusRefused
		receipt.Cases[i].FailureCode = code
	}
	countOutcomes(receipt)
}

func markRefusedThenUnattempted(receipt *Receipt, index int, code string) {
	receipt.Cases[index].Status = StatusRefused
	receipt.Cases[index].FailureCode = code
	markUnattempted(receipt, index+1)
	countOutcomes(receipt)
}

func markUnattempted(receipt *Receipt, start int) {
	for i := start; i < len(receipt.Cases); i++ {
		receipt.Cases[i].Status = StatusUnattempted
	}
}

func countOutcomes(receipt *Receipt) {
	receipt.Succeeded, receipt.Failed, receipt.Refused, receipt.Unattempted = 0, 0, 0, 0
	for _, item := range receipt.Cases {
		switch item.Status {
		case StatusSucceeded:
			receipt.Succeeded++
		case StatusFailed:
			receipt.Failed++
		case StatusRefused:
			receipt.Refused++
		case StatusUnattempted:
			receipt.Unattempted++
		}
	}
}

// SQLiteEventSource is the Agent Manager-owned event reader. Event payloads
// remain inside Agent Manager memory; only verified metadata leaves this owner.
type SQLiteEventSource struct {
	Runs interface {
		GetRun(context.Context, uuid.UUID) (*domain.Run, error)
		ListRuns(context.Context, orchestration.RunListOptions) ([]*domain.Run, error)
	}
	Events interface {
		GetRunEvents(context.Context, uuid.UUID, event.GetOptions) ([]*domain.RunEvent, error)
		ListRunEventMetadata(context.Context, uuid.UUID, event.GetOptions) ([]event.Metadata, bool, error)
	}
}

func (s SQLiteEventSource) Select(ctx context.Context, limit int) ([]SourceRef, error) {
	if s.Runs == nil || s.Events == nil || limit < 1 || limit > MaximumCases {
		return nil, errors.New("Agent Manager metadata source is unavailable")
	}
	completed := domain.RunStatusComplete
	runs, err := s.Runs.ListRuns(ctx, orchestration.RunListOptions{
		ListOptions: orchestration.ListOptions{Limit: 500}, Status: &completed, Origin: "agent_manager", TopLevelOnly: true,
	})
	if err != nil {
		return nil, errors.New("Agent Manager run inventory unavailable")
	}
	type candidate struct {
		ref SourceRef
		at  time.Time
	}
	var candidates []candidate
	for _, run := range runs {
		if run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.RoleRef) == "" {
			continue
		}
		var after int64 = -1
		for page := 0; page < 8; page++ {
			rows, more, listErr := s.Events.ListRunEventMetadata(ctx, run.ID, event.GetOptions{
				AfterSequence: after, Limit: 500, EventTypes: []domain.RunEventType{domain.EventTypeMessage},
			})
			if listErr != nil {
				return nil, errors.New("Agent Manager source metadata unavailable")
			}
			for _, row := range rows {
				if !strings.EqualFold(row.Role, "user") || row.Source != domain.MessageSourceOperator {
					continue
				}
				taskRole, consumer := strings.TrimSpace(run.ResolvedConfig.RoleRef), "agent-manager"
				requestID, requestErr := searcheval.AgentManagerReplayRequestID(run.ID.String(), row.EventID.String(), row.Sequence, taskRole, consumer)
				if requestErr != nil {
					continue
				}
				candidates = append(candidates, candidate{ref: SourceRef{
					EventID: row.EventID.String(), RunID: run.ID.String(), Sequence: row.Sequence,
					Timestamp: row.Timestamp.UTC().Format(time.RFC3339Nano), TaskRole: taskRole, RequestID: requestID,
					PolicyRole: "chat.default", Consumer: consumer,
				}, at: row.Timestamp})
			}
			if !more || len(rows) == 0 {
				break
			}
			after = rows[len(rows)-1].Sequence
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].at.Equal(candidates[j].at) {
			return candidates[i].ref.EventID < candidates[j].ref.EventID
		}
		return candidates[i].at.After(candidates[j].at)
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].at.Equal(candidates[j].at) {
			return candidates[i].ref.EventID < candidates[j].ref.EventID
		}
		return candidates[i].at.Before(candidates[j].at)
	})
	refs := make([]SourceRef, 0, len(candidates))
	for _, item := range candidates {
		refs = append(refs, item.ref)
	}
	return refs, nil
}

func (s SQLiteEventSource) VerifyAndLoad(ctx context.Context, ref SourceRef) (VerifiedCase, error) {
	if s.Runs == nil || s.Events == nil {
		return VerifiedCase{}, errors.New("Agent Manager source is unavailable")
	}
	if !strings.EqualFold(ref.PolicyRole, "chat.default") || ref.Consumer != "agent-manager" || !verifyRequestIdentity(ref) {
		return VerifiedCase{}, errors.New("caller identity does not match the frozen case")
	}
	runID, _ := uuid.Parse(ref.RunID)
	eventID, _ := uuid.Parse(ref.EventID)
	run, err := s.Runs.GetRun(ctx, runID)
	if err != nil || run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.RoleRef) != ref.TaskRole {
		return VerifiedCase{}, errors.New("run role no longer matches frozen caller metadata")
	}
	events, err := s.Events.GetRunEvents(ctx, runID, event.GetOptions{AfterSequence: ref.Sequence - 1, Limit: 1, EventTypes: []domain.RunEventType{domain.EventTypeMessage}})
	if err != nil || len(events) != 1 || events[0].ID != eventID || events[0].RunID != runID || events[0].Sequence != ref.Sequence || events[0].EventType != domain.EventTypeMessage {
		return VerifiedCase{}, errors.New("message event identity changed")
	}
	message, ok := events[0].Data.(*domain.MessageEventData)
	if !ok || !strings.EqualFold(message.Role, "user") || domain.MessageSource(events[0]) != domain.MessageSourceOperator {
		return VerifiedCase{}, errors.New("source event is not an operator user message")
	}
	if events[0].Timestamp.UTC().Format(time.RFC3339Nano) != strings.TrimSpace(ref.Timestamp) {
		return VerifiedCase{}, errors.New("message timestamp changed")
	}
	return VerifiedCase{Ref: ref, Timestamp: events[0].Timestamp, Source: domain.MessageSource(events[0]), Prompt: message.Content}, nil
}
