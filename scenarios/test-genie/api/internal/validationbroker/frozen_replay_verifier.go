package validationbroker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"test-genie/internal/frozenreplay"
	"test-genie/internal/runmanager"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
)

// FrozenReplayEvidenceVerifier binds each F9 variant to a durable Test Genie
// validation receipt and its pinned producer declaration. ArtifactDigest is
// the digest of the immutable protobuf receipt that carries the terminal result.
type FrozenReplayRunOwner interface {
	Status(target, runID string) (runmanager.LiveStatus, error)
}

type FrozenReplayEvidenceVerifier struct {
	Receipts ReceiptRepository
	Runs     FrozenReplayRunOwner
}

func (v FrozenReplayEvidenceVerifier) ArtifactDigest(ctx context.Context, evidence frozenreplay.RunEvidence) (string, error) {
	if v.Receipts == nil {
		return "", errors.New("validation receipt repository is unavailable")
	}
	receipt, err := v.Receipts.Get(ctx, evidence.ProducerReceipt)
	if err != nil {
		return "", fmt.Errorf("load producer receipt: %w", err)
	}
	if receipt.GetReceiptId() != evidence.ProducerReceipt {
		return "", errors.New("producer receipt ID mismatch")
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(receipt)
	if err != nil {
		return "", fmt.Errorf("encode owner receipt: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (v FrozenReplayEvidenceVerifier) VerifyEvidence(ctx context.Context, evidence frozenreplay.RunEvidence) error {
	if v.Receipts == nil {
		return errors.New("validation receipt repository is unavailable")
	}
	receipt, err := v.Receipts.Get(ctx, evidence.ProducerReceipt)
	if err != nil {
		return fmt.Errorf("load producer receipt: %w", err)
	}
	if receipt.GetReceiptId() != evidence.ProducerReceipt {
		return errors.New("producer receipt ID mismatch")
	}
	intent, err := v.Receipts.GetIntent(ctx, evidence.ProducerReceipt)
	if err != nil {
		return fmt.Errorf("load producer intent: %w", err)
	}
	if !identityContainsEvaluator(receipt.GetAdmittedIdentity(), evidence.EvaluatorDigest) {
		return errors.New("run receipt does not pin the supplied frozen evaluator file digest")
	}
	if intent.GetFrozenReplayExecution() != nil {
		return v.verifyFrozenReplayRun(ctx, evidence, receipt, intent)
	}
	if v.Runs == nil {
		return errors.New("durable Test Genie run owner is unavailable")
	}
	if len(intent.GetTargets()) != 1 || intent.GetTargets()[0].GetId() == "" {
		return errors.New("validation receipt must bind exactly one Test Genie run target")
	}
	target := intent.GetTargets()[0].GetId()
	var child *validationv1.ChildOperation
	for _, candidate := range receipt.GetChildren() {
		if candidate.GetOperationId() == evidence.RunID {
			child = candidate
			break
		}
	}
	if child == nil {
		return errors.New("run is absent from the immutable producer receipt")
	}
	if child.GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN || child.GetOwner() != "test-genie" {
		return errors.New("receipt child is not an owner-managed Test Genie run")
	}
	status, err := v.Runs.Status(target, evidence.RunID)
	if err != nil {
		return fmt.Errorf("resolve durable Test Genie run: %w", err)
	}
	if status.RunID != evidence.RunID || status.Scenario != target || status.Result == nil || status.Result.RunID != evidence.RunID ||
		status.Result.ScenarioName != target || !status.Result.SourceStable || len(status.DegradedReasons) != 0 {
		return errors.New("durable Test Genie run record is missing, degraded, or source-unstable")
	}
	var observedStatus string
	identity := receipt.GetObservedIdentity()
	switch receipt.GetState() {
	case validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED:
		if !identityContainsEvaluator(receipt.GetObservedIdentity(), evidence.EvaluatorDigest) || child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED || status.Status != "passed" || !status.Result.Success || status.Result.Verdict != "PASS" {
			return errors.New("passing producer receipt does not contain a successful run")
		}
		observedStatus = "passed"
	case validationv1.ReceiptState_RECEIPT_STATE_FAILED:
		if receipt.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED || child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED || status.Status != "failed" || status.Result.Success || status.Result.Verdict != "FAIL" || status.Result.FailureReason != "" || status.Result.PhaseSummary.Failed == 0 {
			return fmt.Errorf("producer failure is not a completed assertion failure (receipt reason %s)", receipt.GetReasonCode())
		}
		assertionFailure := false
		for _, phase := range status.Result.Phases {
			if phase.Status == "failed" && phase.Classification == "test_failure" {
				assertionFailure = true
				break
			}
		}
		if !assertionFailure {
			return errors.New("failed Test Genie run has no typed test_failure phase; refusing infrastructure failures as expected red")
		}
		observedStatus = "failed"
		identity = receipt.GetAdmittedIdentity()
	default:
		return fmt.Errorf("producer receipt is not a completed test outcome: %s", receipt.GetState())
	}
	if observedStatus != evidence.Status || identity.GetIdentity() != evidence.SourceIdentity || receipt.GetAdmittedIdentity().GetIdentity() != evidence.SourceIdentity || status.Result.SourceStable != true {
		return errors.New("run result or source identity differs from its owner receipt")
	}
	artifactDigest, err := v.ArtifactDigest(ctx, evidence)
	if err != nil {
		return err
	}
	if evidence.ArtifactDigest != artifactDigest {
		return errors.New("artifact digest differs from the immutable owner receipt")
	}
	return nil
}

func (v FrozenReplayEvidenceVerifier) verifyFrozenReplayRun(ctx context.Context, evidence frozenreplay.RunEvidence, receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent) error {
	execution := intent.GetFrozenReplayExecution()
	result := receipt.GetFrozenReplayResult()
	if result == nil {
		return errors.New("frozen replay receipt has no typed Sandbox execution result")
	}
	attempts := receipt.GetFrozenReplayAttempts()
	if len(attempts) == 0 || !proto.Equal(attempts[len(attempts)-1], result) {
		return errors.New("frozen replay final result is not the last retained owner attempt")
	}
	for _, attempt := range attempts[:len(attempts)-1] {
		if attempt.GetOutcome() != validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_INTERRUPTED {
			return errors.New("frozen replay retry history contains a non-interruption attempt")
		}
	}
	role, ok := frozenReplayRole(evidence.Variant)
	if !ok || role != execution.GetRole() || result.GetRole() != role {
		return errors.New("frozen replay role does not match the requested three-way variant")
	}
	sealIdentity := "sha256:" + strings.TrimPrefix(strings.ToLower(execution.GetSealDigest()), "sha256:")
	if result.GetRunId() != evidence.RunID || result.GetSandboxOperationId() != evidence.RunID || result.GetCaseId() != execution.GetCaseId() || result.GetSealDigest() != execution.GetSealDigest() || result.GetSourceIdentity() != sealIdentity || result.GetEvaluatorBundleId() != execution.GetEvaluatorBundleId() || strings.TrimPrefix(strings.ToLower(result.GetEvaluatorDigest()), "sha256:") != strings.TrimPrefix(strings.ToLower(evidence.EvaluatorDigest), "sha256:") || result.GetEvaluatorEntryPoint() != execution.GetEvaluatorEntryPoint() || !result.GetSealIntactBefore() || !result.GetSealIntactAfter() || !resultDigestValid(result.GetResultDigest()) {
		return errors.New("frozen replay result is missing a matching owner identity, valid digest, or intact seal")
	}
	if receipt.GetAdmittedIdentity().GetIdentity() != sealIdentity || receipt.GetObservedIdentity().GetIdentity() != sealIdentity || evidence.SourceIdentity != sealIdentity {
		return errors.New("frozen replay source identity differs from the exact sealed source")
	}
	var child *validationv1.ChildOperation
	for _, candidate := range receipt.GetChildren() {
		if candidate.GetOperationId() == evidence.RunID {
			child = candidate
			break
		}
	}
	if child == nil || child.GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN || child.GetOwner() != "workspace-sandbox" || child.GetOperationId() != result.GetSandboxOperationId() {
		return errors.New("frozen replay operation is absent from its immutable Test Genie receipt")
	}
	passed := result.GetOutcome() == validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_PASSED
	assertionFailed := result.GetOutcome() == validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_ASSERTION_FAILED
	if evidence.Status == "passed" {
		if !passed || receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
			return errors.New("passing frozen replay evidence is not backed by a successful Sandbox assertion result")
		}
	} else if evidence.Status == "failed" {
		if !assertionFailed || receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED || receipt.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED || child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED {
			return errors.New("failed frozen replay evidence is not a typed assertion failure; infrastructure failures cannot count as expected red")
		}
	} else {
		return errors.New("frozen replay evidence has an unspecified status")
	}
	artifactDigest, err := v.ArtifactDigest(ctx, evidence)
	if err != nil {
		return err
	}
	if evidence.ArtifactDigest != artifactDigest {
		return errors.New("artifact digest differs from the immutable owner receipt")
	}
	return nil
}

func frozenReplayRole(variant frozenreplay.Variant) (validationv1.WorkspaceSandboxSealRole, bool) {
	switch variant {
	case frozenreplay.Preimage:
		return validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_SOURCE, true
	case frozenreplay.Candidate:
		return validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_CANDIDATE, true
	case frozenreplay.Reference:
		return validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_REFERENCE, true
	default:
		return validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_UNSPECIFIED, false
	}
}

func identityContainsEvaluator(identity *validationv1.SourceIdentity, evaluatorDigest string) bool {
	if identity == nil {
		return false
	}
	for _, root := range identity.GetRoots() {
		if root.GetName() != "frozen-evaluator" {
			continue
		}
		for _, file := range root.GetFiles() {
			if strings.TrimPrefix(strings.ToLower(file.GetDigest()), "sha256:") == strings.TrimPrefix(strings.ToLower(evaluatorDigest), "sha256:") {
				return true
			}
		}
	}
	return false
}
