package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func frozenSummaryDataset() SummaryEvaluationDataset {
	dataset := SummaryEvaluationDataset{
		ID: "w12r-test", Version: "1", CallerID: "audio-tools.tts-summary",
		ArtifactPath:                   "dataset.json",
		CallerRevision:                 strings.Repeat("a", 64),
		HeldoutMembershipOwner:         "test-owner-fixture",
		HeldoutMembershipReceiptID:     "fixture-receipt-001",
		HeldoutMembershipReceiptPath:   "membership.json",
		HeldoutMembershipReceiptSHA256: strings.Repeat("b", 64),
		Cases:                          make([]SummaryEvaluationCase, 0, SummaryEvaluationCases),
	}
	for i := 0; i < SummaryEvaluationCases; i++ {
		split := "calibration"
		if i >= SummaryEvaluationCalibration {
			split = "heldout"
		}
		dataset.Cases = append(dataset.Cases, SummaryEvaluationCase{
			ID: fmt.Sprintf("case-%02d", i), SourceGroup: fmt.Sprintf("source-%02d", i),
			Split: split, Level: "moderate", CandidateContextTier: 4096,
			Input:         strings.Repeat("The report says Q1 revenue rose 5 percent. Details follow. ", 4),
			RequiredFacts: []string{"Q1 revenue rose 5 percent"}, ReviewerOracleID: fmt.Sprintf("oracle-%02d", i),
		})
	}
	dataset.HeldoutMembershipSHA256 = heldoutMembershipDigest(dataset.Cases)
	return dataset
}

func validSummaryIdentity() SummaryEvaluationIdentity {
	return SummaryEvaluationIdentity{
		EvaluatorID: SummaryEvaluatorID, EvaluatorSourceSHA256: strings.Repeat("6", 64),
		CallerID: "audio-tools.tts-summary", CallerRevisionSHA256: strings.Repeat("a", 64),
		Role: SummaryEvaluationRole, ResolvedModel: "qwen3.5:9b",
		ArtifactDigest: strings.Repeat("1", 64), RightsReceiptSHA256: strings.Repeat("2", 64),
		RightsReceiptPath: "rights.json", RightsReceiptFileSHA256: strings.Repeat("2", 64),
		RightsDecisionSHA256: strings.Repeat("9", 64), RightsEvaluatedAt: "2026-11-11T20:00:00Z",
		ServingReceiptSHA256: strings.Repeat("3", 64), RuntimeBuildID: "ollama-build-current",
		PolicySHA256: strings.Repeat("4", 64), RuntimeConfigSHA256: strings.Repeat("5", 64),
		ResourceReceiptPath: "resource.json", ResourceReceiptSHA256: strings.Repeat("8", 64), BaselineEffectiveContext: 8192,
	}
}

type fixtureHeldoutMembershipVerifier struct {
	err error
	got string
}

type fixtureRightsDecisionReceiptVerifier struct {
	err error
	got string
}

func (v *fixtureRightsDecisionReceiptVerifier) VerifyRightsDecisionReceipt(_ context.Context, role, model, path, fileSHA, receiptSHA, decisionSHA, evaluatedAt string) error {
	v.got = strings.Join([]string{role, model, path, fileSHA, receiptSHA, decisionSHA, evaluatedAt}, "|")
	return v.err
}

type fixtureHeldoutReviewVerifier struct {
	err error
	got string
}

func (v *fixtureHeldoutReviewVerifier) VerifyHeldoutReview(_ context.Context, reportSHA256, receiptID, receiptPath, receiptSHA256 string) error {
	v.got = strings.Join([]string{reportSHA256, receiptID, receiptPath, receiptSHA256}, "|")
	return v.err
}

func (v *fixtureHeldoutMembershipVerifier) VerifyHeldoutMembership(_ context.Context, owner, receiptID, receiptPath, receiptSHA256, datasetSHA256, membershipSHA256 string) error {
	v.got = strings.Join([]string{owner, receiptID, receiptPath, receiptSHA256, datasetSHA256, membershipSHA256}, "|")
	return v.err
}

func TestSummaryEvaluationDatasetRequiresFixedIndependentDenominator(t *testing.T) {
	dataset := frozenSummaryDataset()
	if err := dataset.Validate(); err != nil {
		t.Fatalf("valid fixed dataset: %v", err)
	}
	if _, _, err := dataset.Digest(); err != nil {
		t.Fatalf("digest fixed dataset: %v", err)
	}

	dataset.Cases[19].SourceGroup = dataset.Cases[0].SourceGroup
	if err := dataset.Validate(); err == nil || !strings.Contains(err.Error(), "both splits") {
		t.Fatalf("cross-split source group error = %v, want leakage refusal", err)
	}

	dataset = frozenSummaryDataset()
	dataset.Cases[19].ID = "relabelled-calibration-case"
	if err := dataset.Validate(); err == nil || !strings.Contains(err.Error(), "membership SHA-256") {
		t.Fatalf("heldout relabel error = %v, want owner membership receipt mismatch", err)
	}
}

func TestRunSummaryEvaluationUsesActualCallerRoleAndFixedProviderPath(t *testing.T) {
	var observed []string
	summarizer := NewSummarizerWithRunner("resource-ollama-test", func(_ context.Context, args []string, stdin string) ([]byte, error) {
		contextTier := argValue(args, "--num-ctx")
		observed = append(observed, contextTier)
		identity := validSummaryIdentity()
		if argValue(args, "--expected-rights-decision-sha256") != identity.RightsDecisionSHA256 || argValue(args, "--expected-rights-evaluated-at") != identity.RightsEvaluatedAt || argValue(args, "--expected-model") != identity.ResolvedModel || argValue(args, "--expected-artifact-digest") != identity.ArtifactDigest {
			t.Fatalf("gateway request did not carry verified rights/model/artifact identity: args=%v", args)
		}
		if argValue(args, "--role") != SummaryEvaluationRole {
			t.Fatalf("provider role = %q, want %q", argValue(args, "--role"), SummaryEvaluationRole)
		}
		if stdin == "" || !strings.Contains(argValue(args, "--system"), "Rewrite") {
			t.Fatalf("provider request missing frozen moderate prompt/input: args=%v input=%q", args, stdin)
		}
		return json.Marshal(map[string]any{
			"model": "qwen3.5:9b", "response": "Q1 revenue rose 5 percent.",
			"done_reason": "stop", "eval_count": 6,
			"total_duration": 50000000, "load_duration": 10000000,
			"prompt_eval_count": 30, "prompt_eval_duration": 20000000,
			"eval_duration": 15000000,
		})
	})
	config := SummarizeConfig{Enabled: true, Level: "heavy", Model: SummaryEvaluationRole, TimeoutSeconds: 30}
	service := NewSummarizationService(summarizer, func() SummarizeConfig { return config })
	dataset := frozenSummaryDataset()
	dataset.Cases[0].CandidateContextTier = 8192
	verifier := &fixtureHeldoutMembershipVerifier{}
	rightsVerifier := &fixtureRightsDecisionReceiptVerifier{}
	report, err := RunSummaryEvaluation(context.Background(), service, dataset, validSummaryIdentity(), 4096, verifier, rightsVerifier)
	if err != nil {
		t.Fatalf("run integrated summary evaluation: %v", err)
	}
	if len(report.Results) != 2*SummaryEvaluationCases || report.BaselineCalls != 20 || report.CandidateCalls != 20 {
		t.Fatalf("call denominator = results:%d baseline:%d candidate:%d", len(report.Results), report.BaselineCalls, report.CandidateCalls)
	}
	if report.RetryCount != 0 || report.FailureCount != 0 || !report.QualityEligible {
		t.Fatalf("unexpected evaluation disposition: retries=%d failures=%d eligible=%t", report.RetryCount, report.FailureCount, report.QualityEligible)
	}
	if first := report.Results[0]; first.ModelLoadDurationNS != 10000000 || first.PromptEvalCount != 30 || first.PromptEvalDurationNS != 20000000 || first.DecodeDurationNS != 15000000 {
		t.Fatalf("provider duration receipt was not retained: %+v", first)
	}
	if len(observed) != 40 {
		t.Fatalf("provider calls = %d, want exactly 40", len(observed))
	}
	if !strings.Contains(rightsVerifier.got, SummaryEvaluationRole+"|qwen3.5:9b|rights.json|") || !strings.Contains(strings.Join(observed, ","), "4096") {
		t.Fatalf("rights receipt was not bound before evaluator calls: %q", rightsVerifier.got)
	}
	firstDigest, err := report.Digest()
	if err != nil {
		t.Fatalf("digest evaluation report: %v", err)
	}
	secondDigest, err := report.Digest()
	if err != nil || firstDigest != secondDigest || len(firstDigest) != 64 {
		t.Fatalf("report digest stability = %q / %q, err=%v", firstDigest, secondDigest, err)
	}
	if report.HeldoutMembershipReceiptID != dataset.HeldoutMembershipReceiptID || report.HeldoutMembershipReceiptSHA256 != dataset.HeldoutMembershipReceiptSHA256 || report.Identity.RuntimeConfigSHA256 != strings.Repeat("5", 64) {
		t.Fatalf("report lost heldout/runtime identity: %+v", report)
	}
	datasetDigest, _, err := dataset.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if verifier.got != strings.Join([]string{dataset.HeldoutMembershipOwner, dataset.HeldoutMembershipReceiptID, dataset.HeldoutMembershipReceiptPath, dataset.HeldoutMembershipReceiptSHA256, datasetDigest, dataset.HeldoutMembershipSHA256}, "|") {
		t.Fatalf("verifier was not given exact owner receipt/membership: %q", verifier.got)
	}
	beforeReview := firstDigest
	reviewVerifier := &fixtureHeldoutReviewVerifier{}
	if err := report.AttachHeldoutReview(context.Background(), reviewVerifier, "blind-review-001", "review.json", strings.Repeat("7", 64)); err != nil {
		t.Fatalf("attach completed heldout review: %v", err)
	}
	if report.BlindReviewRequired || report.HeldoutReviewReceiptID == "" {
		t.Fatalf("heldout review receipt not retained: %+v", report)
	}
	afterReview, err := report.Digest()
	if err != nil || afterReview == beforeReview {
		t.Fatalf("review receipt did not change bound report digest: before=%q after=%q err=%v", beforeReview, afterReview, err)
	}
	if !strings.HasPrefix(reviewVerifier.got, beforeReview+"|") || report.HeldoutReviewReceiptPath != "review.json" || report.ReviewedReportSHA256 != beforeReview {
		t.Fatalf("heldout review was not bound to the exact pre-review report: verifier=%q reviewed=%q want=%q", reviewVerifier.got, report.ReviewedReportSHA256, beforeReview)
	}
	for i := 0; i < len(observed); i += 2 {
		wantCandidate := "4096"
		if i == 0 {
			wantCandidate = "8192"
		}
		if observed[i] != "8192" || observed[i+1] != wantCandidate {
			t.Fatalf("paired context tiers = %q then %q, want baseline then candidate", observed[i], observed[i+1])
		}
	}
}

func TestRunSummaryEvaluationFailsClosedWithoutExactReceipts(t *testing.T) {
	identity := validSummaryIdentity()
	identity.RightsReceiptSHA256 = ""
	service := NewSummarizationService(testSummarizer(t, "unused", nil), func() SummarizeConfig {
		return SummarizeConfig{Model: SummaryEvaluationRole, Level: "moderate"}
	})
	if _, err := RunSummaryEvaluation(context.Background(), service, frozenSummaryDataset(), identity, 4096, &fixtureHeldoutMembershipVerifier{}, &fixtureRightsDecisionReceiptVerifier{}); err == nil || !strings.Contains(err.Error(), "rights receipt") {
		t.Fatalf("missing rights receipt error = %v, want fail-closed refusal", err)
	}
}

func TestRunSummaryEvaluationRequiresRuntimeConfigIdentityBeforeProviderCall(t *testing.T) {
	calls := 0
	identity := validSummaryIdentity()
	identity.RuntimeConfigSHA256 = ""
	service := NewSummarizationService(NewSummarizerWithRunner("unused", func(context.Context, []string, string) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("provider must not be called")
	}), func() SummarizeConfig {
		return SummarizeConfig{Model: SummaryEvaluationRole, Level: "moderate"}
	})
	if _, err := RunSummaryEvaluation(context.Background(), service, frozenSummaryDataset(), identity, 4096, &fixtureHeldoutMembershipVerifier{}, &fixtureRightsDecisionReceiptVerifier{}); err == nil || !strings.Contains(err.Error(), "runtime config SHA-256") {
		t.Fatalf("missing runtime config identity error = %v, want pre-provider refusal", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want 0 before exact runtime identity", calls)
	}
}

func TestRunSummaryEvaluationRequiresOwnerMembershipVerificationBeforeProviderCall(t *testing.T) {
	calls := 0
	service := NewSummarizationService(NewSummarizerWithRunner("unused", func(context.Context, []string, string) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("provider must not be called")
	}), func() SummarizeConfig {
		return SummarizeConfig{Model: SummaryEvaluationRole, Level: "moderate"}
	})
	verifier := &fixtureHeldoutMembershipVerifier{err: fmt.Errorf("receipt digest is not owner-issued")}
	_, err := RunSummaryEvaluation(context.Background(), service, frozenSummaryDataset(), validSummaryIdentity(), 4096, verifier, &fixtureRightsDecisionReceiptVerifier{})
	if err == nil || !strings.Contains(err.Error(), "verify owner heldout membership receipt") {
		t.Fatalf("unverified owner receipt error = %v, want pre-provider refusal", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want 0 before heldout approval", calls)
	}
}

func TestRunSummaryEvaluationRequiresCurrentRightsReceiptBeforeProviderCall(t *testing.T) {
	calls := 0
	service := NewSummarizationService(NewSummarizerWithRunner("unused", func(context.Context, []string, string) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("provider must not be called")
	}), func() SummarizeConfig {
		return SummarizeConfig{Model: SummaryEvaluationRole, Level: "moderate"}
	})
	rightsVerifier := &fixtureRightsDecisionReceiptVerifier{err: fmt.Errorf("receipt is stale for current trusted context")}
	_, err := RunSummaryEvaluation(context.Background(), service, frozenSummaryDataset(), validSummaryIdentity(), 4096, &fixtureHeldoutMembershipVerifier{}, rightsVerifier)
	if err == nil || !strings.Contains(err.Error(), "verify owner rights-decision receipt") {
		t.Fatalf("rights receipt refusal = %v, want owner-verifier refusal", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want 0 before current rights proof", calls)
	}
}

func TestSummaryEvaluationRequiresOwnerApprovedHeldoutReceipt(t *testing.T) {
	dataset := frozenSummaryDataset()
	dataset.HeldoutMembershipReceiptSHA256 = ""
	if err := dataset.Validate(); err == nil || !strings.Contains(err.Error(), "owner-approved heldout membership receipt") {
		t.Fatalf("missing heldout membership receipt error = %v", err)
	}
}

func TestRunSummaryEvaluationRequiresOwnerBoundCallerIdentity(t *testing.T) {
	identity := validSummaryIdentity()
	identity.CallerID = "unverified.caller"
	service := NewSummarizationService(testSummarizer(t, "unused", nil), func() SummarizeConfig {
		return SummarizeConfig{Model: SummaryEvaluationRole, Level: "moderate"}
	})
	if _, err := RunSummaryEvaluation(context.Background(), service, frozenSummaryDataset(), identity, 4096, &fixtureHeldoutMembershipVerifier{}, &fixtureRightsDecisionReceiptVerifier{}); err == nil || !strings.Contains(err.Error(), "caller identity") {
		t.Fatalf("caller mismatch error = %v, want owner-bound refusal", err)
	}
}

func TestRunSummaryEvaluationRetainsProviderFailuresWithoutRetries(t *testing.T) {
	var calls int
	summarizer := NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("provider timeout %d", calls)
	})
	service := NewSummarizationService(summarizer, func() SummarizeConfig {
		return SummarizeConfig{Enabled: true, Level: "moderate", Model: SummaryEvaluationRole, TimeoutSeconds: 30}
	})
	report, err := RunSummaryEvaluation(context.Background(), service, frozenSummaryDataset(), validSummaryIdentity(), 4096, &fixtureHeldoutMembershipVerifier{}, &fixtureRightsDecisionReceiptVerifier{})
	if err != nil {
		t.Fatalf("run failed-provider comparison: %v", err)
	}
	if calls != 2*SummaryEvaluationCases || report.FailureCount != calls || report.RetryCount != 0 {
		t.Fatalf("call/failure/retry accounting = %d/%d/%d, want 40/40/0", calls, report.FailureCount, report.RetryCount)
	}
	if report.Results[0].Error == "" || !strings.Contains(report.Results[0].Error, "provider timeout") || report.Results[0].IdentityError != "" {
		t.Fatalf("provider failure attribution was lost: %+v", report.Results[0])
	}
}

func TestRunSummaryEvaluationRejectsProviderFallbackIdentityDrift(t *testing.T) {
	call := 0
	summarizer := NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		call++
		model := "qwen3.5:9b"
		if call%2 == 0 {
			model = "qwen3.5:4b"
		}
		return json.Marshal(map[string]any{
			"model": model, "response": "Q1 revenue rose 5 percent.",
			"done_reason": "stop", "eval_count": 6,
		})
	})
	service := NewSummarizationService(summarizer, func() SummarizeConfig {
		return SummarizeConfig{Enabled: true, Level: "moderate", Model: SummaryEvaluationRole, TimeoutSeconds: 30}
	})
	dataset := frozenSummaryDataset()
	dataset.Cases[0].CandidateContextTier = 8192
	report, err := RunSummaryEvaluation(context.Background(), service, dataset, validSummaryIdentity(), 4096, &fixtureHeldoutMembershipVerifier{}, &fixtureRightsDecisionReceiptVerifier{})
	if err != nil {
		t.Fatalf("run identity-drift trial: %v", err)
	}
	if report.FailureCount != SummaryEvaluationCases || report.QualityEligible {
		t.Fatalf("fallback identity disposition: failures=%d eligible=%t, want every candidate call blocked", report.FailureCount, report.QualityEligible)
	}
	for i := 1; i < len(report.Results); i += 2 {
		if report.Results[i].IdentityError != `resolved model mismatch: got "qwen3.5:4b", want "qwen3.5:9b"` || report.Results[i].Output != "" {
			t.Fatalf("candidate fallback result was not rejected: %+v", report.Results[i])
		}
	}
}
