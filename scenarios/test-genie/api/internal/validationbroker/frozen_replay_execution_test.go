package validationbroker

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/durationpb"
)

type frozenReplayTestResolver struct {
	identity *validationv1.SourceIdentity
	bundle   FrozenEvaluatorBundle
	verified int
}

func (r *frozenReplayTestResolver) Resolve(context.Context, *validationv1.ValidationIntent) (*validationv1.SourceIdentity, error) {
	return r.identity, nil
}
func (r *frozenReplayTestResolver) ResolveReplaySeal(context.Context, *validationv1.WorkspaceSandboxSealReference) (string, error) {
	return "/private/owner/path", nil
}
func (r *frozenReplayTestResolver) VerifyReplaySeal(context.Context, *validationv1.WorkspaceSandboxSealReference) error {
	r.verified++
	return nil
}
func (r *frozenReplayTestResolver) ResolveFrozenEvaluatorBundle(context.Context, *validationv1.FrozenReplayExecution) (FrozenEvaluatorBundle, error) {
	return r.bundle, nil
}

type frozenReplayTestRunner struct {
	result  FrozenReplayTestResult
	results []FrozenReplayTestResult
	starts  int
}

func (r *frozenReplayTestRunner) Start(context.Context, string, *validationv1.FrozenReplayExecution, FrozenEvaluatorBundle, time.Duration) (FrozenReplayTestResult, error) {
	if len(r.results) > 0 {
		result := r.results[r.starts]
		r.starts++
		return result, nil
	}
	return r.result, nil
}
func (r *frozenReplayTestRunner) Wait(context.Context, FrozenReplayTestResult, time.Duration) (FrozenReplayTestResult, error) {
	return r.result, nil
}

func TestFrozenReplayProducerSeparatesAssertionFailureFromInfrastructureFailure(t *testing.T) {
	for _, tc := range []struct {
		name       string
		outcome    FrozenReplayTestOutcome
		wantState  validationv1.ReceiptState
		wantReason validationv1.ValidationReasonCode
	}{
		{name: "assertion", outcome: FrozenReplayAssertionFailed, wantState: validationv1.ReceiptState_RECEIPT_STATE_FAILED, wantReason: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED},
		{name: "infrastructure", outcome: FrozenReplayInfrastructureFailed, wantState: validationv1.ReceiptState_RECEIPT_STATE_FAILED, wantReason: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE},
		{name: "pass", outcome: FrozenReplayPassed, wantState: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, wantReason: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			intent, resolver := frozenReplayTestIntent(t)
			repo := NewRepository(testsqllite(t))
			admission, err := repo.Admit(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			result := FrozenReplayTestResult{Success: true, OperationID: "sandbox-op", CaseID: "case-1", Role: intent.GetFrozenReplayExecution().GetRole(), SealDigest: intent.GetFrozenReplayExecution().GetSealDigest(), EvaluatorBundleID: resolver.bundle.EvaluatorBundleID, EvaluatorBundleDigest: resolver.bundle.ContentDigest, EntryPoint: resolver.bundle.EntryPoint, Outcome: tc.outcome, ResultDigest: sha256Hex([]byte("result")), SealIntactBefore: true, SealIntactAfter: true}
			runner := &frozenReplayTestRunner{result: result}
			producer := NewRunProducer(nil, resolver).WithFrozenReplayTestRunner(runner)
			if err := producer.ExecuteValidation(ctx, admission.Receipt, intent, repo.Transition); err != nil {
				t.Fatal(err)
			}
			stored, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
			if err != nil {
				t.Fatal(err)
			}
			if stored.GetState() != tc.wantState || stored.GetReasonCode() != tc.wantReason || stored.GetFrozenReplayResult().GetOutcome() == validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_UNSPECIFIED {
				t.Fatalf("receipt state/reason/result = %s/%s/%s", stored.GetState(), stored.GetReasonCode(), stored.GetFrozenReplayResult().GetOutcome())
			}
		})
	}
}

func TestFrozenReplayInterruptedAttemptReattachesThenStartsStableRetry(t *testing.T) {
	ctx := context.Background()
	intent, resolver := frozenReplayTestIntent(t)
	intent.DeadlinePolicy.MaximumAttempts = 2
	repo := NewRepository(testsqllite(t))
	admission, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	base := FrozenReplayTestResult{Success: true, CaseID: "case-1", Role: intent.GetFrozenReplayExecution().GetRole(), SealDigest: intent.GetFrozenReplayExecution().GetSealDigest(), EvaluatorBundleID: resolver.bundle.EvaluatorBundleID, EvaluatorBundleDigest: resolver.bundle.ContentDigest, EntryPoint: resolver.bundle.EntryPoint, ResultDigest: sha256Hex([]byte("result")), SealIntactBefore: true, SealIntactAfter: true}
	interrupted := base
	interrupted.OperationID = "sandbox-op-interrupted"
	interrupted.Outcome = FrozenReplayInterrupted
	interrupted.SealIntactBefore = false
	interrupted.SealIntactAfter = false
	passed := base
	passed.OperationID = "sandbox-op-recovered"
	passed.Outcome = FrozenReplayPassed
	runner := &frozenReplayTestRunner{results: []FrozenReplayTestResult{interrupted, passed}}
	producer := NewRunProducer(nil, resolver).WithFrozenReplayTestRunner(runner)
	if err := producer.ExecuteValidation(ctx, admission.Receipt, intent, repo.Transition); err != errEvidencePending {
		t.Fatalf("first interrupted attempt error = %v, want pending recovery", err)
	}
	interim, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if len(interim.GetFrozenReplayAttempts()) != 1 || interim.GetFrozenReplayAttempts()[0].GetOutcome() != validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_INTERRUPTED || interim.GetFrozenReplayAttempts()[0].GetSealIntactBefore() {
		t.Fatalf("interruption attempt was not retained: %#v", interim.GetFrozenReplayAttempts())
	}
	if err := producer.ExecuteValidation(ctx, interim, intent, repo.Transition); err != nil {
		t.Fatal(err)
	}
	final, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if final.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || final.GetFrozenReplayResult().GetRunId() != "sandbox-op-recovered" || len(final.GetFrozenReplayAttempts()) != 2 || final.GetFrozenReplayAttempts()[0].GetOutcome() != validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_INTERRUPTED {
		t.Fatalf("recovered receipt = state %s result %#v attempts %#v", final.GetState(), final.GetFrozenReplayResult(), final.GetFrozenReplayAttempts())
	}
}

func frozenReplayTestIntent(t *testing.T) (*validationv1.ValidationIntent, *frozenReplayTestResolver) {
	t.Helper()
	seal := hex.EncodeToString(make([]byte, 32))
	evaluatorBytes := []byte("module.exports = () => true")
	evaluatorDigest := sha256Hex(evaluatorBytes)
	files := []FrozenEvaluatorFile{{RelativePath: "acceptance.cjs", Content: evaluatorBytes, SHA256: evaluatorDigest}}
	manifest, err := json.Marshal([]frozenEvaluatorDigestTuple{{RelativePath: files[0].RelativePath, SHA256: evaluatorDigest, Size: int64(len(evaluatorBytes))}})
	if err != nil {
		t.Fatal(err)
	}
	contentDigest := sha256Hex(manifest)
	bundle := FrozenEvaluatorBundle{EvaluatorBundleID: "case-1@1:bundle", CaseID: "case-1", EntryPoint: "acceptance.cjs", EvaluatorDigest: evaluatorDigest, ContentDigest: contentDigest, Files: files}
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "sha256:" + seal, Roots: []*validationv1.ContentRootIdentity{{Name: "source", Identity: "sha256:" + seal}, {Name: "frozen-evaluator", Identity: "sha256:" + contentDigest, Files: []*validationv1.ContentFileIdentity{{Path: "acceptance.cjs", Digest: evaluatorDigest, Size: int64(len(evaluatorBytes))}}}}}
	resolver := &frozenReplayTestResolver{identity: identity, bundle: bundle}
	intent := &validationv1.ValidationIntent{
		SchemaVersion: 1, IntentId: uuid.NewString(), IdempotencyKey: uuid.NewString(), CallerScenario: "f9-test", Purpose: validationv1.ValidationPurpose_VALIDATION_PURPOSE_PHASE, RequiredStrength: validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED,
		Targets:          []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT, Id: "case-1"}},
		ExpectedIdentity: identity, EvidencePolicy: &validationv1.EvidencePolicy{}, ReusePolicy: &validationv1.ReusePolicy{Mode: validationv1.ReuseMode_REUSE_MODE_NEVER}, ConcurrencyPolicy: &validationv1.ConcurrencyPolicy{Mode: validationv1.ConcurrencyMode_CONCURRENCY_MODE_EXCLUSIVE}, DeadlinePolicy: &validationv1.DeadlinePolicy{ExecutionBudget: durationpb.New(time.Minute), MaximumAttempts: 1},
		ContentInputs:         []*validationv1.ContentInputRoot{{Name: "source", WorkspaceSandboxSeal: &validationv1.WorkspaceSandboxSealReference{CaseId: "case-1", Role: validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_SOURCE, SealDigest: seal}, Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}}, {Name: "frozen-evaluator", Dependency: true, FrozenEvaluatorBundle: &validationv1.FrozenEvaluatorBundleReference{EvaluatorBundleId: bundle.EvaluatorBundleID, EvaluatorDigest: evaluatorDigest, EntryPoint: bundle.EntryPoint}}},
		FrozenReplayExecution: &validationv1.FrozenReplayExecution{CaseId: "case-1", Role: validationv1.WorkspaceSandboxSealRole_WORKSPACE_SANDBOX_SEAL_ROLE_SOURCE, SealDigest: seal, EvaluatorBundleId: bundle.EvaluatorBundleID, EvaluatorDigest: evaluatorDigest, EvaluatorEntryPoint: bundle.EntryPoint},
	}
	return intent, resolver
}
