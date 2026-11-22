package validationbroker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"test-genie/internal/frozenreplay"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/runmanager"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
)

type frozenReplayReceiptFixture struct {
	ReceiptRepository
	receipt *validationv1.ValidationReceipt
	intent  *validationv1.ValidationIntent
}

func (f frozenReplayReceiptFixture) Get(context.Context, string) (*validationv1.ValidationReceipt, error) {
	return proto.Clone(f.receipt).(*validationv1.ValidationReceipt), nil
}

func (f frozenReplayReceiptFixture) GetIntent(context.Context, string) (*validationv1.ValidationIntent, error) {
	return proto.Clone(f.intent).(*validationv1.ValidationIntent), nil
}

type frozenReplayRunFixture struct{ status runmanager.LiveStatus }

func (f frozenReplayRunFixture) Status(string, string) (runmanager.LiveStatus, error) {
	return f.status, nil
}

func TestFrozenReplayVerifierBindsRunSourceAndTypedOutcome(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
	}{
		{name: "true preimage assertion failure", status: "failed"},
		{name: "candidate or reference pass", status: "passed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const runID, receiptID, target = "run-owner-1", "producer-owner-1", "sealed-workspace-1"
			evaluatorDigest, sourceIdentity := "sha256:"+digestHex("evaluator"), "sha256:"+digestHex("source")
			ownerReceipt, intent, ownerStatus := ownerEvidence(tc.status, runID, receiptID, target, evaluatorDigest, sourceIdentity)
			encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(ownerReceipt)
			if err != nil {
				t.Fatal(err)
			}
			artifact := sha256.Sum256(encoded)
			evidence := frozenreplay.RunEvidence{
				Variant: frozenreplay.Preimage, RunID: runID, ProducerReceipt: receiptID,
				SourceIdentity: sourceIdentity, EvaluatorDigest: evaluatorDigest,
				Status: tc.status, ArtifactDigest: "sha256:" + hex.EncodeToString(artifact[:]),
			}
			verifier := FrozenReplayEvidenceVerifier{Receipts: frozenReplayReceiptFixture{receipt: ownerReceipt, intent: intent}, Runs: frozenReplayRunFixture{status: ownerStatus}}
			if err := verifier.VerifyEvidence(context.Background(), evidence); err != nil {
				t.Fatalf("VerifyEvidence: %v", err)
			}
		})
	}
}

func TestFrozenReplayVerifierRejectsInfrastructureFailureAndSourceDrift(t *testing.T) {
	const runID, receiptID, target = "run-owner-2", "producer-owner-2", "sealed-workspace-2"
	evaluatorDigest, sourceIdentity := "sha256:"+digestHex("evaluator"), "sha256:"+digestHex("source")
	ownerReceipt, intent, status := ownerEvidence("failed", runID, receiptID, target, evaluatorDigest, sourceIdentity)
	encoded, _ := proto.MarshalOptions{Deterministic: true}.Marshal(ownerReceipt)
	artifact := sha256.Sum256(encoded)
	evidence := frozenreplay.RunEvidence{Variant: frozenreplay.Preimage, RunID: runID, ProducerReceipt: receiptID, SourceIdentity: sourceIdentity, EvaluatorDigest: evaluatorDigest, Status: "failed", ArtifactDigest: "sha256:" + hex.EncodeToString(artifact[:])}
	verifier := FrozenReplayEvidenceVerifier{Receipts: frozenReplayReceiptFixture{receipt: ownerReceipt, intent: intent}, Runs: frozenReplayRunFixture{status: status}}

	status.Result.FailureReason = "test runtime did not start"
	if err := verifier.VerifyEvidence(context.Background(), evidence); err == nil {
		t.Fatal("accepted infrastructure failure as expected preimage red")
	}
	status.Result.FailureReason = ""
	status.Result.Phases[0].Classification = "system"
	if err := verifier.VerifyEvidence(context.Background(), evidence); err == nil {
		t.Fatal("accepted system failure as expected preimage red")
	}
	status.Result.Phases[0].Classification = "test_failure"
	status.Result.SourceStable = false
	if err := verifier.VerifyEvidence(context.Background(), evidence); err == nil {
		t.Fatal("accepted a source-unstable run")
	}
}

func ownerEvidence(outcome, runID, receiptID, target, evaluatorDigest, sourceIdentity string) (*validationv1.ValidationReceipt, *validationv1.ValidationIntent, runmanager.LiveStatus) {
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: sourceIdentity, Roots: []*validationv1.ContentRootIdentity{{Name: "frozen-evaluator", Files: []*validationv1.ContentFileIdentity{{Path: "evaluator.test", Digest: evaluatorDigest}}}}}
	intent := &validationv1.ValidationIntent{Targets: []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: target}}}
	runStatus, verdict := "passed", "PASS"
	receipt := &validationv1.ValidationReceipt{ReceiptId: receiptID, State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, AdmittedIdentity: proto.Clone(identity).(*validationv1.SourceIdentity), ObservedIdentity: proto.Clone(identity).(*validationv1.SourceIdentity), Children: []*validationv1.ChildOperation{{ChildId: "scenario:" + target + ":1", Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED, Owner: "test-genie", OperationId: runID}}}
	result := &orchestrator.SuiteExecutionResult{RunID: runID, ScenarioName: target, Success: true, Verdict: verdict, SourceStable: true, PhaseSummary: orchestrator.PhaseSummary{Passed: 1}}
	if outcome == "failed" {
		receipt.State = validationv1.ReceiptState_RECEIPT_STATE_FAILED
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED
		receipt.ObservedIdentity = nil
		receipt.Children[0].State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
		runStatus, verdict = "failed", "FAIL"
		result.Success, result.Verdict = false, verdict
		result.PhaseSummary = orchestrator.PhaseSummary{Failed: 1}
		result.Phases = []phases.ExecutionResult{{Name: "unit", Status: "failed", Classification: "test_failure"}}
	}
	return receipt, intent, runmanager.LiveStatus{RunID: runID, Scenario: target, Status: runStatus, Result: result}
}

func digestHex(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
