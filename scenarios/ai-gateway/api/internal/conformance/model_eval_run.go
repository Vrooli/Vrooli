package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	modelEvalAdmissionID    = "LM2-INFERENCE-ADMISSION-20261116"
	modelEvalAdmissionScope = "one local-only sequential batch of up to 20 fixed Agent Manager caller cases using the already-installed chat.default model; metadata-only receipts; no acquisition, remote inference, promotion, or deletion"
	modelEvalMaxCases       = 20
	modelEvalMinDiskBytes   = int64(120 * 1024 * 1024 * 1024)
	modelEvalMaxArtifact    = int64(10 * 1024 * 1024)
	modelEvalMinClaimBytes  = int64(13_541_310_464)
)

var errModelEvalAdmissionRefused = errors.New("model-eval inference admission refused")

type ModelEvalApproval struct {
	DecisionID string
	Answer     string
	Scope      string
	RecordedAt time.Time
}

// ModelEvalApprovalSource reads the current explicit operator answer. It must
// not infer approval from a historical note, an empty response, or a generic
// operator decision.
type ModelEvalApprovalSource interface {
	ReadModelEvalApproval(context.Context, string) (ModelEvalApproval, error)
}

// ModelEvalCase is held only in memory while a batch runs. Prompt is excluded
// from every receipt type and must never be logged or persisted by its source.
type ModelEvalCase struct {
	EventID        string
	RunID          string
	Sequence       int64
	EventTimestamp time.Time
	RequestID      string
	PolicyRole     string
	TaskRole       string
	Consumer       string
	Prompt         string
}

type ModelEvalSource interface {
	LoadFrozenCases(context.Context, int) ([]ModelEvalCase, error)
	VerifyCaseIdentity(context.Context, ModelEvalCase) error
}

type ModelEvalPrivacyVerifier interface {
	ProvePromptOutputNonPersistence(context.Context) error
}

type ModelEvalCurrentPolicy interface {
	ResolveChatDefault(context.Context) (model string, locality string, allowRemote bool, err error)
	CheckCurrentRights(context.Context, string, string) error
}

type ModelEvalDiskMeasurement struct {
	FreshAt        time.Time
	FreeBytes      int64
	ArtifactBytes  int64
	ArtifactSHA256 string
}

type ModelEvalDiskGate interface {
	MeasureModelEvalDisk(context.Context) (ModelEvalDiskMeasurement, error)
}

type ModelEvalCapacityReceipt struct {
	ClaimID       string
	RequiredBytes int64
	GrantedBytes  int64
	FreeBytes     int64
	ObservedAt    time.Time
}

type ModelEvalCapacityGate interface {
	AcquireModelEvalClaim(context.Context, string, int64) (ModelEvalCapacityReceipt, func(context.Context) error, error)
}

type ModelEvalDispatchReceipt struct {
	EffectiveModel string
	Locality       string
	Latency        time.Duration
	Capacity       ModelEvalCapacityReceipt
	RightsAllowed  bool
}

// ModelEvalDispatcher must keep request and response bodies ephemeral and
// return only metadata. Implementations must not log bodies or include them in
// provider errors.
type ModelEvalDispatcher interface {
	DispatchModelEval(context.Context, string, string, string) (ModelEvalDispatchReceipt, error)
}

type ModelEvalCaseStatus string

const (
	ModelEvalCaseSucceeded   ModelEvalCaseStatus = "succeeded"
	ModelEvalCaseFailed      ModelEvalCaseStatus = "failed"
	ModelEvalCaseRefused     ModelEvalCaseStatus = "refused"
	ModelEvalCaseUnattempted ModelEvalCaseStatus = "unattempted"
)

type ModelEvalCaseReceipt struct {
	SourceEventID  string              `json:"source_event_id"`
	SourceRunID    string              `json:"source_run_id"`
	Sequence       int64               `json:"sequence"`
	RequestID      string              `json:"request_id"`
	PolicyRole     string              `json:"policy_role"`
	TaskRole       string              `json:"task_role"`
	Consumer       string              `json:"consumer"`
	EffectiveModel string              `json:"effective_model,omitempty"`
	Locality       string              `json:"locality,omitempty"`
	Status         ModelEvalCaseStatus `json:"status"`
	FailureCode    string              `json:"failure_code,omitempty"`
	LatencyMillis  int64               `json:"latency_ms,omitempty"`
	RightsAllowed  bool                `json:"rights_allowed"`
	CapacityClaim  string              `json:"capacity_claim_id,omitempty"`
	RequiredBytes  int64               `json:"required_bytes,omitempty"`
	GrantedBytes   int64               `json:"granted_bytes,omitempty"`
}

type ModelEvalBatchReceipt struct {
	DecisionID          string                 `json:"decision_id"`
	DecisionScope       string                 `json:"decision_scope"`
	CaseSetSHA256       string                 `json:"case_set_sha256"`
	Denominator         int                    `json:"denominator"`
	Succeeded           int                    `json:"succeeded"`
	Failed              int                    `json:"failed"`
	Refused             int                    `json:"refused"`
	Unattempted         int                    `json:"unattempted"`
	EffectiveModel      string                 `json:"effective_model"`
	DiskFreeBytesBefore int64                  `json:"disk_free_bytes_before"`
	DiskFreeBytesAfter  int64                  `json:"disk_free_bytes_after,omitempty"`
	ArtifactBytes       int64                  `json:"artifact_bytes"`
	ArtifactSHA256      string                 `json:"artifact_sha256"`
	ArtifactBytesAfter  int64                  `json:"artifact_bytes_after,omitempty"`
	ArtifactSHA256After string                 `json:"artifact_sha256_after,omitempty"`
	Cases               []ModelEvalCaseReceipt `json:"cases"`
}

type ModelEvalBatchRunner struct {
	Approval ModelEvalApprovalSource
	Source   ModelEvalSource
	Privacy  ModelEvalPrivacyVerifier
	Policy   ModelEvalCurrentPolicy
	Disk     ModelEvalDiskGate
	Capacity ModelEvalCapacityGate
	Dispatch ModelEvalDispatcher
	Now      func() time.Time
}

func (r ModelEvalBatchRunner) Run(ctx context.Context) (ModelEvalBatchReceipt, error) {
	if r.Approval == nil || r.Source == nil || r.Privacy == nil || r.Policy == nil || r.Disk == nil || r.Capacity == nil || r.Dispatch == nil {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: required owner gate is unavailable", errModelEvalAdmissionRefused)
	}
	approval, err := r.Approval.ReadModelEvalApproval(ctx, modelEvalAdmissionID)
	if err != nil {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: explicit decision unavailable", errModelEvalAdmissionRefused)
	}
	if approval.DecisionID != modelEvalAdmissionID || strings.ToLower(strings.TrimSpace(approval.Answer)) != "approve" || approval.Scope != modelEvalAdmissionScope || approval.RecordedAt.IsZero() {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: exact approve answer and scope required", errModelEvalAdmissionRefused)
	}
	if err := r.Privacy.ProvePromptOutputNonPersistence(ctx); err != nil {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: prompt/output non-persistence is not proven", errModelEvalAdmissionRefused)
	}
	model, locality, allowRemote, err := r.Policy.ResolveChatDefault(ctx)
	if err != nil || strings.TrimSpace(model) == "" || !strings.EqualFold(locality, "local") || allowRemote {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: current chat.default is not proven local-only", errModelEvalAdmissionRefused)
	}
	cases, err := r.Source.LoadFrozenCases(ctx, modelEvalMaxCases)
	if err != nil || len(cases) == 0 || len(cases) > modelEvalMaxCases {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: fixed caller case set is unavailable or outside 1..%d", errModelEvalAdmissionRefused, modelEvalMaxCases)
	}
	if err := validateModelEvalCases(cases); err != nil {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: %v", errModelEvalAdmissionRefused, err)
	}
	caseSetDigest, err := modelEvalCaseSetDigest(cases)
	if err != nil {
		return ModelEvalBatchReceipt{}, fmt.Errorf("%w: cannot freeze case-set identity", errModelEvalAdmissionRefused)
	}
	for _, item := range cases {
		if err := r.Source.VerifyCaseIdentity(ctx, item); err != nil {
			return ModelEvalBatchReceipt{}, fmt.Errorf("%w: source identity verification failed", errModelEvalAdmissionRefused)
		}
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	disk, err := r.Disk.MeasureModelEvalDisk(ctx)
	if err != nil || disk.FreshAt.IsZero() || now().Sub(disk.FreshAt) > 2*time.Minute || disk.FreshAt.After(now().Add(30*time.Second)) || disk.FreeBytes < modelEvalMinDiskBytes || disk.ArtifactBytes < 0 || disk.ArtifactBytes > modelEvalMaxArtifact || !validSHA256(disk.ArtifactSHA256) {
		return refusedModelEvalBatch(approval, caseSetDigest, model, cases, "disk_or_artifact_gate"), fmt.Errorf("%w: fresh disk floor or artifact bound is not proven", errModelEvalAdmissionRefused)
	}

	receipt := ModelEvalBatchReceipt{
		DecisionID: approval.DecisionID, DecisionScope: approval.Scope, CaseSetSHA256: caseSetDigest,
		Denominator: len(cases), EffectiveModel: model, DiskFreeBytesBefore: disk.FreeBytes,
		ArtifactBytes: disk.ArtifactBytes, ArtifactSHA256: strings.ToLower(disk.ArtifactSHA256),
		Cases: make([]ModelEvalCaseReceipt, 0, len(cases)),
	}
	for i, item := range cases {
		caseReceipt := modelEvalCaseReceipt(item)
		if err := r.Source.VerifyCaseIdentity(ctx, item); err != nil {
			caseReceipt.Status = ModelEvalCaseRefused
			caseReceipt.FailureCode = "source_identity_changed_or_unavailable"
			receipt.Cases = append(receipt.Cases, caseReceipt)
			markModelEvalUnattempted(&receipt, cases[i+1:])
			recountModelEvalReceipt(&receipt)
			return receipt, fmt.Errorf("%w: frozen source event identity no longer verifies", errModelEvalAdmissionRefused)
		}
		currentModel, currentLocality, currentRemote, policyErr := r.Policy.ResolveChatDefault(ctx)
		if policyErr != nil || currentModel != model || !strings.EqualFold(currentLocality, "local") || currentRemote {
			caseReceipt.Status = ModelEvalCaseRefused
			caseReceipt.FailureCode = "chat_default_policy_changed_or_unknown"
			receipt.Cases = append(receipt.Cases, caseReceipt)
			markModelEvalUnattempted(&receipt, cases[i+1:])
			recountModelEvalReceipt(&receipt)
			return receipt, fmt.Errorf("%w: current chat.default changed during batch", errModelEvalAdmissionRefused)
		}
		if err := r.Policy.CheckCurrentRights(ctx, model, item.PolicyRole); err != nil {
			caseReceipt.Status = ModelEvalCaseRefused
			caseReceipt.FailureCode = "current_rights_denied_or_unknown"
			receipt.Cases = append(receipt.Cases, caseReceipt)
			markModelEvalUnattempted(&receipt, cases[i+1:])
			recountModelEvalReceipt(&receipt)
			return receipt, fmt.Errorf("%w: current rights gate refused case", errModelEvalAdmissionRefused)
		}
		claim, release, err := r.Capacity.AcquireModelEvalClaim(ctx, "ai-gateway:model-eval:"+item.EventID, modelEvalMinClaimBytes)
		if err != nil || release == nil || strings.TrimSpace(claim.ClaimID) == "" || claim.RequiredBytes < modelEvalMinClaimBytes || claim.GrantedBytes < modelEvalMinClaimBytes || claim.FreeBytes < modelEvalMinClaimBytes || claim.ObservedAt.IsZero() || now().Sub(claim.ObservedAt) > 30*time.Second || claim.ObservedAt.After(now().Add(10*time.Second)) {
			if release != nil {
				_ = release(ctx)
			}
			caseReceipt.Status = ModelEvalCaseRefused
			caseReceipt.FailureCode = "live_capacity_claim_unavailable"
			receipt.Cases = append(receipt.Cases, caseReceipt)
			markModelEvalUnattempted(&receipt, cases[i+1:])
			recountModelEvalReceipt(&receipt)
			return receipt, fmt.Errorf("%w: fresh exact capacity claim unavailable", errModelEvalAdmissionRefused)
		}
		result, dispatchErr := r.Dispatch.DispatchModelEval(ctx, model, item.PolicyRole, item.Prompt)
		releaseErr := release(ctx)
		diskAfter, diskAfterErr := r.Disk.MeasureModelEvalDisk(ctx)
		postDiskInvalid := diskAfterErr != nil || diskAfter.FreshAt.IsZero() || now().Sub(diskAfter.FreshAt) > 2*time.Minute || diskAfter.FreshAt.After(now().Add(30*time.Second)) || diskAfter.FreeBytes < modelEvalMinDiskBytes || diskAfter.ArtifactBytes < 0 || diskAfter.ArtifactBytes > modelEvalMaxArtifact || !validSHA256(diskAfter.ArtifactSHA256)
		if diskAfterErr == nil {
			receipt.DiskFreeBytesAfter = diskAfter.FreeBytes
			receipt.ArtifactBytesAfter = diskAfter.ArtifactBytes
			receipt.ArtifactSHA256After = strings.ToLower(diskAfter.ArtifactSHA256)
		}
		caseReceipt.EffectiveModel = result.EffectiveModel
		caseReceipt.Locality = result.Locality
		caseReceipt.LatencyMillis = result.Latency.Milliseconds()
		caseReceipt.RightsAllowed = result.RightsAllowed
		caseReceipt.CapacityClaim = claim.ClaimID
		caseReceipt.RequiredBytes = claim.RequiredBytes
		caseReceipt.GrantedBytes = claim.GrantedBytes
		if dispatchErr != nil || releaseErr != nil || postDiskInvalid || result.EffectiveModel != model || !strings.EqualFold(result.Locality, "local") || !result.RightsAllowed || result.Capacity.ClaimID != claim.ClaimID || result.Capacity.RequiredBytes < modelEvalMinClaimBytes || result.Capacity.GrantedBytes < modelEvalMinClaimBytes {
			caseReceipt.Status = ModelEvalCaseFailed
			caseReceipt.FailureCode = "dispatch_or_postcondition_failed"
			receipt.Cases = append(receipt.Cases, caseReceipt)
			markModelEvalUnattempted(&receipt, cases[i+1:])
			recountModelEvalReceipt(&receipt)
			return receipt, errors.New("model-eval stopped after a failed dispatch or owner receipt")
		}
		caseReceipt.Status = ModelEvalCaseSucceeded
		receipt.Cases = append(receipt.Cases, caseReceipt)
	}
	recountModelEvalReceipt(&receipt)
	return receipt, nil
}

func validateModelEvalCases(cases []ModelEvalCase) error {
	seen := make(map[string]struct{}, len(cases))
	seenRequests := make(map[string]struct{}, len(cases))
	for _, item := range cases {
		if strings.TrimSpace(item.EventID) == "" || strings.TrimSpace(item.RunID) == "" || item.Sequence < 0 || item.EventTimestamp.IsZero() || strings.TrimSpace(item.RequestID) == "" || strings.TrimSpace(item.PolicyRole) == "" || strings.TrimSpace(item.TaskRole) == "" || strings.TrimSpace(item.Consumer) == "" || strings.TrimSpace(item.Prompt) == "" {
			return errors.New("case is missing exact source identity, caller metadata, or ephemeral input")
		}
		key := item.RunID + "/" + item.EventID + "/" + fmt.Sprint(item.Sequence)
		if _, exists := seen[key]; exists {
			return errors.New("case set contains a duplicate source event identity")
		}
		seen[key] = struct{}{}
		requestID := strings.TrimSpace(item.RequestID)
		if _, exists := seenRequests[requestID]; exists {
			return errors.New("case set contains a duplicate Gateway request identity")
		}
		seenRequests[requestID] = struct{}{}
	}
	return nil
}

func modelEvalCaseSetDigest(cases []ModelEvalCase) (string, error) {
	type identity struct {
		EventID        string `json:"event_id"`
		RunID          string `json:"run_id"`
		Sequence       int64  `json:"sequence"`
		EventTimestamp string `json:"event_timestamp"`
		RequestID      string `json:"request_id"`
		PolicyRole     string `json:"policy_role"`
		TaskRole       string `json:"task_role"`
		Consumer       string `json:"consumer"`
	}
	items := make([]identity, 0, len(cases))
	for _, item := range cases {
		items = append(items, identity{item.EventID, item.RunID, item.Sequence, item.EventTimestamp.UTC().Format(time.RFC3339Nano), item.RequestID, item.PolicyRole, item.TaskRole, item.Consumer})
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func modelEvalCaseReceipt(item ModelEvalCase) ModelEvalCaseReceipt {
	return ModelEvalCaseReceipt{SourceEventID: item.EventID, SourceRunID: item.RunID, Sequence: item.Sequence,
		RequestID: item.RequestID, PolicyRole: item.PolicyRole, TaskRole: item.TaskRole, Consumer: item.Consumer}
}

func refusedModelEvalBatch(approval ModelEvalApproval, digest, model string, cases []ModelEvalCase, code string) ModelEvalBatchReceipt {
	receipt := ModelEvalBatchReceipt{DecisionID: approval.DecisionID, DecisionScope: approval.Scope, CaseSetSHA256: digest, Denominator: len(cases), EffectiveModel: model, Cases: make([]ModelEvalCaseReceipt, 0, len(cases))}
	for _, item := range cases {
		row := modelEvalCaseReceipt(item)
		row.Status = ModelEvalCaseRefused
		row.FailureCode = code
		receipt.Cases = append(receipt.Cases, row)
	}
	recountModelEvalReceipt(&receipt)
	return receipt
}

func markModelEvalUnattempted(receipt *ModelEvalBatchReceipt, cases []ModelEvalCase) {
	for _, item := range cases {
		row := modelEvalCaseReceipt(item)
		row.Status = ModelEvalCaseUnattempted
		receipt.Cases = append(receipt.Cases, row)
	}
}

func recountModelEvalReceipt(receipt *ModelEvalBatchReceipt) {
	receipt.Succeeded, receipt.Failed, receipt.Refused, receipt.Unattempted = 0, 0, 0, 0
	for _, item := range receipt.Cases {
		switch item.Status {
		case ModelEvalCaseSucceeded:
			receipt.Succeeded++
		case ModelEvalCaseFailed:
			receipt.Failed++
		case ModelEvalCaseRefused:
			receipt.Refused++
		case ModelEvalCaseUnattempted:
			receipt.Unattempted++
		}
	}
}

func validSHA256(value string) bool {
	if len(strings.TrimSpace(value)) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
