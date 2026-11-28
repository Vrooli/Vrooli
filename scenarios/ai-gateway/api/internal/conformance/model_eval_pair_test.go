package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

func TestCompareCodeEvalPairPreservesFixedDenominatorAndQualifiesCandidate(t *testing.T) {
	input := validCodeEvalPairInput()
	result, err := CompareCodeEvalPair(input)
	if err != nil {
		t.Fatalf("CompareCodeEvalPair() error = %v", err)
	}
	if result.Decision != "eligible_for_role_canary" || !result.Qualified || !result.PromotionThresholdMet {
		t.Fatalf("comparison did not reach pre-registered canary frontier: %+v", result)
	}
	if result.Denominator != CodeEvalMinimumCases || result.CandidatePassed != CodeEvalMinimumCases || result.PairedWins != CodeEvalMinimumNetPairedWins || result.PairedLosses != 0 {
		t.Fatalf("unexpected paired counts: %+v", result)
	}
	if len(result.Cases) != CodeEvalMinimumCases {
		t.Fatalf("failed or successful cases disappeared from denominator: got %d", len(result.Cases))
	}
}

func TestCompareCodeEvalPairRetainsIncumbentWhenQualityTies(t *testing.T) {
	input := validCodeEvalPairInput()
	for index := range input.BaselineOutcomes {
		if input.BaselineOutcomes[index].Status == "failed" {
			input.CandidateOutcomes[index].Status = "failed"
			input.CandidateOutcomes[index].Passed = false
		}
	}
	input.BaselineOutcomes[0].Status = "failed"
	input.BaselineOutcomes[0].Passed = false
	input.CandidateOutcomes[0].Status = "failed"
	input.CandidateOutcomes[0].Passed = false
	result, err := CompareCodeEvalPair(input)
	if err != nil {
		t.Fatalf("CompareCodeEvalPair() error = %v", err)
	}
	if !result.Qualified || result.CandidatePassed != CodeEvalMinimumPassingCases || result.Decision != "retain_incumbent" {
		t.Fatalf("quality tie did not retain incumbent with full denominator: %+v", result)
	}
}

func TestCompareCodeEvalPairScalesPassThresholdWithDenominator(t *testing.T) {
	input := validCodeEvalPairInput()
	for index := 0; index < 8; index++ {
		id := "extra-case-" + string(rune('a'+index))
		input.Cases = append(input.Cases, CodeEvalCase{CaseID: id, Strata: []string{"other-caller"}})
		input.BaselineOutcomes = append(input.BaselineOutcomes, CodeEvalOutcome{CaseID: id, Status: "failed"})
		input.CandidateOutcomes = append(input.CandidateOutcomes, CodeEvalOutcome{CaseID: id, Status: "failed"})
	}
	ids := make([]string, 0, len(input.Cases))
	for _, item := range input.Cases {
		ids = append(ids, item.CaseID)
	}
	input.DenominatorSHA256 = searcheval.CaseIDsDigest(ids)
	result, err := CompareCodeEvalPair(input)
	if err != nil {
		t.Fatalf("CompareCodeEvalPair() error = %v", err)
	}
	if result.Denominator != 28 || result.CandidatePassed != 20 || result.Qualified || result.Decision != "not_qualified" {
		t.Fatalf("pass threshold did not scale with fixed denominator: %+v", result)
	}
}

func TestCompareCodeEvalPairRefusesMismatchedDenominator(t *testing.T) {
	input := validCodeEvalPairInput()
	input.DenominatorSHA256 = hashForTest("wrong-denominator")
	if _, err := CompareCodeEvalPair(input); err == nil {
		t.Fatal("expected mismatch between denominator digest and case IDs to refuse")
	}
}

func TestCompareCodeEvalPairRetainsFailuresAndDisqualifiesHardGate(t *testing.T) {
	input := validCodeEvalPairInput()
	input.CandidateOutcomes[0].Status = "refused"
	input.CandidateOutcomes[0].Passed = false
	input.CandidateOutcomes[0].HardGateFailureCode = "remote_route_observed"
	result, err := CompareCodeEvalPair(input)
	if err != nil {
		t.Fatalf("CompareCodeEvalPair() error = %v", err)
	}
	if result.Denominator != CodeEvalMinimumCases || result.CandidatePassed != CodeEvalMinimumCases-1 {
		t.Fatalf("refused case was not retained as a denominator failure: %+v", result)
	}
	if result.Qualified || result.PromotionThresholdMet || result.Decision != "disqualify_hard_gate_failure" {
		t.Fatalf("hard-gate failure did not disqualify arm: %+v", result)
	}
}

func TestCompareCodeEvalPairRequiresCommonContextAndToolContract(t *testing.T) {
	input := validCodeEvalPairInput()
	input.Candidate.ContextSHA256 = hashForTest("different-context")
	if _, err := CompareCodeEvalPair(input); err == nil {
		t.Fatal("expected paired arms with different context to refuse")
	}
}

func validCodeEvalPairInput() CodeEvalPairInput {
	strata := []string{
		"image-tools-transcript",
		"repo-grounded-citation", "repo-grounded-citation", "repo-grounded-citation",
		"single-file-change", "single-file-change", "single-file-change",
		"multi-file-change", "multi-file-change", "multi-file-change",
		"tool-recovery", "tool-recovery",
		"review-positive", "review-positive",
		"review-clean-control", "review-clean-control",
		"remediation-validation", "remediation-validation",
		"other-caller", "other-caller",
	}
	cases := make([]CodeEvalCase, 0, len(strata))
	baseline := make([]CodeEvalOutcome, 0, len(strata))
	candidate := make([]CodeEvalOutcome, 0, len(strata))
	ids := make([]string, 0, len(strata))
	for index, stratum := range strata {
		id := "case-" + string(rune('a'+index))
		ids = append(ids, id)
		cases = append(cases, CodeEvalCase{CaseID: id, Strata: []string{stratum}})
		baselinePass := stratum != "single-file-change"
		baseline = append(baseline, outcomeForTest(id, baselinePass))
		candidate = append(candidate, outcomeForTest(id, true))
	}
	return CodeEvalPairInput{
		SchemaVersion: CodeEvalPairSchemaVersion, Role: CodeEvalRole,
		CorpusSHA256: hashForTest("corpus"), DenominatorSHA256: searcheval.CaseIDsDigest(ids),
		RepositoryBaseSHA: hashForTest("repository-base"), PairOrderSeed: "order", PairOrderSeedSHA256: hashForTest("order"),
		CorpusReviewReceiptSHA256:     hashForTest("corpus-review"),
		CorpusValidationReceiptSHA256: hashForTest("corpus-validation"),
		HostAdmissionReceiptSHA256:    hashForTest("host-admission"), RightsDecisionID: "rights-decision-1",
		MeasurementReceiptSHA256:  hashForTest("measurements"),
		RollbackPlanReceiptSHA256: hashForTest("rollback-plan"),
		Baseline: CodeEvalArmIdentity{
			Name: "incumbent", ModelID: "gemma4:12b", ArtifactSHA256: hashForTest("baseline-artifact"),
			RuntimeID: "ollama", RuntimeSHA256: hashForTest("runtime"), RecipeSHA256: hashForTest("recipe"),
			ContextSHA256: hashForTest("context"), ToolContractHash: hashForTest("tools"),
		},
		Candidate: CodeEvalArmIdentity{
			Name: "candidate", ModelID: "qwen3.5:9b", ArtifactSHA256: hashForTest("candidate-artifact"),
			RuntimeID: "ollama", RuntimeSHA256: hashForTest("runtime"), RecipeSHA256: hashForTest("recipe"),
			ContextSHA256: hashForTest("context"), ToolContractHash: hashForTest("tools"),
		},
		Cases: cases, BaselineOutcomes: baseline, CandidateOutcomes: candidate,
	}
}

func outcomeForTest(caseID string, passed bool) CodeEvalOutcome {
	status := "failed"
	if passed {
		status = "succeeded"
	}
	return CodeEvalOutcome{
		CaseID: caseID, Status: status, Passed: passed, LatencyMillis: 250, LatencyKnown: true,
		InterventionCountKnown: true, ValidationReceiptHash: hashForTest("validation-" + caseID),
	}
}

func hashForTest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
