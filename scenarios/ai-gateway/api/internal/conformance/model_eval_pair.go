package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

const (
	CodeEvalRole                 = "code.local"
	CodeEvalPairSchemaVersion    = "local-code-eval/paired/v1"
	CodeEvalMinimumCases         = 20
	CodeEvalMinimumPassingCases  = 16
	CodeEvalMinimumReviewCases   = 4
	CodeEvalMinimumReviewRate    = 0.80
	CodeEvalMinimumNetPairedWins = 3
)

var codeEvalRequiredStrata = map[string]int{
	"image-tools-transcript": 1,
	"repo-grounded-citation": 3,
	"single-file-change":     3,
	"multi-file-change":      3,
	"tool-recovery":          2,
	"review-positive":        2,
	"review-clean-control":   2,
	"remediation-validation": 2,
	"other-caller":           2,
}

// CodeEvalArmIdentity pins the model and the executable recipe. These hashes
// identify artifacts; they do not grant the right to execute them.
type CodeEvalArmIdentity struct {
	Name             string `json:"name"`
	ModelID          string `json:"model_id"`
	ArtifactSHA256   string `json:"artifact_sha256"`
	RuntimeID        string `json:"runtime_id"`
	RuntimeSHA256    string `json:"runtime_sha256"`
	RecipeSHA256     string `json:"recipe_sha256"`
	ContextSHA256    string `json:"context_sha256"`
	ToolContractHash string `json:"tool_contract_sha256"`
}

// CodeEvalCase is metadata only. Prompt, source and expected patch content stay
// with their restricted owner; the pair engine accepts only case identities.
type CodeEvalCase struct {
	CaseID string   `json:"case_id"`
	Strata []string `json:"strata"`
}

// CodeEvalOutcome is the independently graded result of one arm on one case.
// Refused and unattempted outcomes remain in the fixed denominator as failures.
type CodeEvalOutcome struct {
	CaseID                 string `json:"case_id"`
	Status                 string `json:"status"` // succeeded, failed, refused, unattempted
	Passed                 bool   `json:"passed"`
	HardGateFailureCode    string `json:"hard_gate_failure_code,omitempty"`
	LatencyMillis          int64  `json:"latency_ms,omitempty"`
	LatencyKnown           bool   `json:"latency_known"`
	InterventionCount      int    `json:"intervention_count,omitempty"`
	InterventionCountKnown bool   `json:"intervention_count_known"`
	ValidationReceiptHash  string `json:"validation_receipt_sha256,omitempty"`
}

// CodeEvalPairInput must be assembled from two arms that received the same
// frozen cases and execution context. The engine never dispatches a model.
type CodeEvalPairInput struct {
	SchemaVersion                 string              `json:"schema_version"`
	Role                          string              `json:"role"`
	CorpusSHA256                  string              `json:"corpus_sha256"`
	DenominatorSHA256             string              `json:"denominator_sha256"`
	RepositoryBaseSHA             string              `json:"repository_base_sha256"`
	PairOrderSeed                 string              `json:"pair_order_seed"`
	PairOrderSeedSHA256           string              `json:"pair_order_seed_sha256"`
	CorpusReviewReceiptSHA256     string              `json:"corpus_review_receipt_sha256"`
	CorpusValidationReceiptSHA256 string              `json:"corpus_validation_receipt_sha256"`
	HostAdmissionReceiptSHA256    string              `json:"host_admission_receipt_sha256"`
	RightsDecisionID              string              `json:"rights_decision_id"`
	MeasurementReceiptSHA256      string              `json:"measurement_receipt_sha256"`
	RollbackPlanReceiptSHA256     string              `json:"rollback_plan_receipt_sha256"`
	Baseline                      CodeEvalArmIdentity `json:"baseline"`
	Candidate                     CodeEvalArmIdentity `json:"candidate"`
	Cases                         []CodeEvalCase      `json:"cases"`
	BaselineOutcomes              []CodeEvalOutcome   `json:"baseline_outcomes"`
	CandidateOutcomes             []CodeEvalOutcome   `json:"candidate_outcomes"`
}

type CodeEvalStratumResult struct {
	Stratum         string `json:"stratum"`
	Required        int    `json:"required"`
	Cases           int    `json:"cases"`
	BaselinePassed  int    `json:"baseline_passed"`
	CandidatePassed int    `json:"candidate_passed"`
}

type CodeEvalPairedCaseResult struct {
	CaseID                      string `json:"case_id"`
	BaselinePassed              bool   `json:"baseline_passed"`
	CandidatePassed             bool   `json:"candidate_passed"`
	CandidateWin                bool   `json:"candidate_win"`
	CandidateLoss               bool   `json:"candidate_loss"`
	BaselineStatus              string `json:"baseline_status"`
	CandidateStatus             string `json:"candidate_status"`
	BaselineLatencyMillis       int64  `json:"baseline_latency_ms,omitempty"`
	BaselineLatencyKnown        bool   `json:"baseline_latency_known"`
	CandidateLatencyMillis      int64  `json:"candidate_latency_ms,omitempty"`
	CandidateLatencyKnown       bool   `json:"candidate_latency_known"`
	BaselineInterventions       int    `json:"baseline_interventions,omitempty"`
	BaselineInterventionsKnown  bool   `json:"baseline_interventions_known"`
	CandidateInterventions      int    `json:"candidate_interventions,omitempty"`
	CandidateInterventionsKnown bool   `json:"candidate_interventions_known"`
}

type CodeEvalPairResult struct {
	SchemaVersion                 string                     `json:"schema_version"`
	Role                          string                     `json:"role"`
	CorpusSHA256                  string                     `json:"corpus_sha256"`
	DenominatorSHA256             string                     `json:"denominator_sha256"`
	RepositoryBaseSHA256          string                     `json:"repository_base_sha256"`
	PairOrderSeed                 string                     `json:"pair_order_seed"`
	PairOrderSeedSHA256           string                     `json:"pair_order_seed_sha256"`
	CorpusReviewReceiptSHA256     string                     `json:"corpus_review_receipt_sha256"`
	CorpusValidationReceiptSHA256 string                     `json:"corpus_validation_receipt_sha256"`
	HostAdmissionReceiptSHA256    string                     `json:"host_admission_receipt_sha256"`
	RightsDecisionID              string                     `json:"rights_decision_id"`
	MeasurementReceiptSHA256      string                     `json:"measurement_receipt_sha256"`
	RollbackPlanReceiptSHA256     string                     `json:"rollback_plan_receipt_sha256"`
	Baseline                      CodeEvalArmIdentity        `json:"baseline"`
	Candidate                     CodeEvalArmIdentity        `json:"candidate"`
	Denominator                   int                        `json:"denominator"`
	BaselinePassed                int                        `json:"baseline_passed"`
	CandidatePassed               int                        `json:"candidate_passed"`
	PairedWins                    int                        `json:"paired_wins"`
	PairedLosses                  int                        `json:"paired_losses"`
	ReviewCases                   int                        `json:"review_cases"`
	ReviewPassed                  int                        `json:"review_passed"`
	HardGateFailures              int                        `json:"hard_gate_failures"`
	Qualified                     bool                       `json:"qualified"`
	PromotionThresholdMet         bool                       `json:"promotion_threshold_met"`
	Decision                      string                     `json:"decision"`
	Strata                        []CodeEvalStratumResult    `json:"strata"`
	Cases                         []CodeEvalPairedCaseResult `json:"cases"`
}

var errCodeEvalPairInvalid = errors.New("local code evaluation pair is invalid")

// CompareCodeEvalPair validates and scores metadata receipts. Its caller must
// first validate current corpus/rights/capacity receipts through their owners;
// this function verifies only receipt digests and case-ID consistency. It is
// deliberately pure: it performs no inference, filesystem mutation, candidate
// promotion, or transcript persistence.
func CompareCodeEvalPair(input CodeEvalPairInput) (CodeEvalPairResult, error) {
	if err := validateCodeEvalPair(input); err != nil {
		return CodeEvalPairResult{}, fmt.Errorf("%w: %v", errCodeEvalPairInvalid, err)
	}
	baselineOutcomes := make(map[string]CodeEvalOutcome, len(input.BaselineOutcomes))
	candidateOutcomes := make(map[string]CodeEvalOutcome, len(input.CandidateOutcomes))
	for _, item := range input.BaselineOutcomes {
		baselineOutcomes[item.CaseID] = item
	}
	for _, item := range input.CandidateOutcomes {
		candidateOutcomes[item.CaseID] = item
	}

	result := CodeEvalPairResult{
		SchemaVersion: input.SchemaVersion, Role: input.Role, CorpusSHA256: strings.ToLower(input.CorpusSHA256),
		DenominatorSHA256: strings.ToLower(input.DenominatorSHA256), RepositoryBaseSHA256: strings.ToLower(input.RepositoryBaseSHA),
		PairOrderSeed:                 input.PairOrderSeed,
		PairOrderSeedSHA256:           strings.ToLower(input.PairOrderSeedSHA256),
		CorpusReviewReceiptSHA256:     strings.ToLower(input.CorpusReviewReceiptSHA256),
		CorpusValidationReceiptSHA256: strings.ToLower(input.CorpusValidationReceiptSHA256),
		HostAdmissionReceiptSHA256:    strings.ToLower(input.HostAdmissionReceiptSHA256),
		RightsDecisionID:              input.RightsDecisionID,
		MeasurementReceiptSHA256:      strings.ToLower(input.MeasurementReceiptSHA256),
		RollbackPlanReceiptSHA256:     strings.ToLower(input.RollbackPlanReceiptSHA256),
		Baseline:                      input.Baseline, Candidate: input.Candidate,
		Denominator: len(input.Cases), Cases: make([]CodeEvalPairedCaseResult, 0, len(input.Cases)),
	}
	stratumResults := make(map[string]*CodeEvalStratumResult, len(codeEvalRequiredStrata))
	for key, required := range codeEvalRequiredStrata {
		stratumResults[key] = &CodeEvalStratumResult{Stratum: key, Required: required}
	}
	stratumOrder := codeEvalRequiredStratumNames()

	for _, item := range input.Cases {
		baseline := baselineOutcomes[item.CaseID]
		candidate := candidateOutcomes[item.CaseID]
		row := CodeEvalPairedCaseResult{
			CaseID: item.CaseID, BaselinePassed: outcomePassed(baseline), CandidatePassed: outcomePassed(candidate),
			BaselineStatus: baseline.Status, CandidateStatus: candidate.Status,
			BaselineLatencyMillis: baseline.LatencyMillis, BaselineLatencyKnown: baseline.LatencyKnown,
			CandidateLatencyMillis: candidate.LatencyMillis, CandidateLatencyKnown: candidate.LatencyKnown,
			BaselineInterventions: baseline.InterventionCount, BaselineInterventionsKnown: baseline.InterventionCountKnown,
			CandidateInterventions: candidate.InterventionCount, CandidateInterventionsKnown: candidate.InterventionCountKnown,
		}
		if row.BaselinePassed {
			result.BaselinePassed++
		}
		if row.CandidatePassed {
			result.CandidatePassed++
		}
		if row.CandidatePassed && !row.BaselinePassed {
			row.CandidateWin = true
			result.PairedWins++
		}
		if row.BaselinePassed && !row.CandidatePassed {
			row.CandidateLoss = true
			result.PairedLosses++
		}
		if strings.TrimSpace(baseline.HardGateFailureCode) != "" {
			result.HardGateFailures++
		}
		if strings.TrimSpace(candidate.HardGateFailureCode) != "" {
			result.HardGateFailures++
		}
		if hasCodeEvalStratum(item.Strata, "review-positive") || hasCodeEvalStratum(item.Strata, "review-clean-control") {
			result.ReviewCases++
			if row.CandidatePassed {
				result.ReviewPassed++
			}
		}
		for _, stratum := range item.Strata {
			row := stratumResults[stratum]
			row.Cases++
			if outcomePassed(baseline) {
				row.BaselinePassed++
			}
			if outcomePassed(candidate) {
				row.CandidatePassed++
			}
		}
		result.Cases = append(result.Cases, row)
	}
	for _, key := range stratumOrder {
		result.Strata = append(result.Strata, *stratumResults[key])
	}
	minimumPasses := (result.Denominator*4 + 4) / 5
	if minimumPasses < CodeEvalMinimumPassingCases {
		minimumPasses = CodeEvalMinimumPassingCases
	}
	result.Qualified = result.HardGateFailures == 0 && result.CandidatePassed >= minimumPasses &&
		result.ReviewCases >= CodeEvalMinimumReviewCases && float64(result.ReviewPassed)/float64(result.ReviewCases) >= CodeEvalMinimumReviewRate
	result.PromotionThresholdMet = result.Qualified && result.PairedWins-result.PairedLosses >= CodeEvalMinimumNetPairedWins
	for _, row := range result.Strata {
		if row.CandidatePassed < row.BaselinePassed {
			result.PromotionThresholdMet = false
		}
	}
	switch {
	case result.HardGateFailures > 0:
		result.Decision = "disqualify_hard_gate_failure"
	case !result.Qualified:
		result.Decision = "not_qualified"
	case result.PromotionThresholdMet:
		result.Decision = "eligible_for_role_canary"
	default:
		result.Decision = "retain_incumbent"
	}
	return result, nil
}

func validateCodeEvalPair(input CodeEvalPairInput) error {
	if input.SchemaVersion != CodeEvalPairSchemaVersion || input.Role != CodeEvalRole {
		return errors.New("unsupported schema or role")
	}
	for _, field := range []struct{ label, hash string }{
		{"corpus", input.CorpusSHA256}, {"denominator", input.DenominatorSHA256},
		{"repository base", input.RepositoryBaseSHA}, {"pair order", input.PairOrderSeedSHA256},
		{"corpus review receipt", input.CorpusReviewReceiptSHA256},
		{"corpus validation receipt", input.CorpusValidationReceiptSHA256},
		{"host admission receipt", input.HostAdmissionReceiptSHA256},
		{"measurement receipt", input.MeasurementReceiptSHA256},
		{"rollback plan receipt", input.RollbackPlanReceiptSHA256},
	} {
		if !validSHA256(field.hash) {
			return fmt.Errorf("%s digest is missing or invalid", field.label)
		}
	}
	seedDigest := sha256.Sum256([]byte(input.PairOrderSeed))
	if input.PairOrderSeed == "" || hex.EncodeToString(seedDigest[:]) != strings.ToLower(input.PairOrderSeedSHA256) {
		return errors.New("pair order seed does not match its recorded digest")
	}
	if strings.TrimSpace(input.RightsDecisionID) == "" {
		return errors.New("rights decision identity is missing")
	}
	if err := validateCodeEvalArm(input.Baseline); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := validateCodeEvalArm(input.Candidate); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if input.Baseline.Name == input.Candidate.Name || strings.EqualFold(input.Baseline.ArtifactSHA256, input.Candidate.ArtifactSHA256) {
		return errors.New("baseline and candidate identities are not distinct")
	}
	if input.Baseline.ContextSHA256 != input.Candidate.ContextSHA256 || input.Baseline.ToolContractHash != input.Candidate.ToolContractHash {
		return errors.New("paired arms do not share context and tool contract")
	}
	if len(input.Cases) < CodeEvalMinimumCases {
		return fmt.Errorf("fixed denominator has %d cases; need at least %d", len(input.Cases), CodeEvalMinimumCases)
	}
	cases := make(map[string]CodeEvalCase, len(input.Cases))
	coverage := make(map[string]int, len(codeEvalRequiredStrata))
	for _, item := range input.Cases {
		if strings.TrimSpace(item.CaseID) == "" || len(item.Strata) == 0 {
			return errors.New("case identity or strata are missing")
		}
		if _, exists := cases[item.CaseID]; exists {
			return errors.New("duplicate case identity")
		}
		seenStrata := make(map[string]struct{}, len(item.Strata))
		for _, stratum := range item.Strata {
			if _, ok := codeEvalRequiredStrata[stratum]; !ok {
				return fmt.Errorf("unknown required stratum %q", stratum)
			}
			if _, duplicate := seenStrata[stratum]; duplicate {
				return errors.New("duplicate case stratum")
			}
			seenStrata[stratum] = struct{}{}
			coverage[stratum]++
		}
		cases[item.CaseID] = item
	}
	for _, stratum := range codeEvalRequiredStratumNames() {
		required := codeEvalRequiredStrata[stratum]
		if coverage[stratum] < required {
			return fmt.Errorf("stratum %s has %d cases; need %d", stratum, coverage[stratum], required)
		}
	}
	caseIDs := make([]string, 0, len(cases))
	for id := range cases {
		caseIDs = append(caseIDs, id)
	}
	if searcheval.CaseIDsDigest(caseIDs) != strings.ToLower(input.DenominatorSHA256) {
		return errors.New("denominator digest does not match the fixed case IDs")
	}
	if err := validateCodeEvalOutcomes("baseline", input.BaselineOutcomes, cases); err != nil {
		return err
	}
	return validateCodeEvalOutcomes("candidate", input.CandidateOutcomes, cases)
}

func validateCodeEvalArm(arm CodeEvalArmIdentity) error {
	if strings.TrimSpace(arm.Name) == "" || strings.TrimSpace(arm.ModelID) == "" || strings.TrimSpace(arm.RuntimeID) == "" {
		return errors.New("model/runtime identity is missing")
	}
	for _, field := range []struct{ label, hash string }{
		{"artifact", arm.ArtifactSHA256}, {"runtime", arm.RuntimeSHA256},
		{"recipe", arm.RecipeSHA256}, {"context", arm.ContextSHA256},
		{"tool contract", arm.ToolContractHash},
	} {
		if !validSHA256(field.hash) {
			return fmt.Errorf("%s digest is missing or invalid", field.label)
		}
	}
	return nil
}

func validateCodeEvalOutcomes(arm string, outcomes []CodeEvalOutcome, cases map[string]CodeEvalCase) error {
	if len(outcomes) != len(cases) {
		return fmt.Errorf("%s has %d outcomes for %d fixed cases", arm, len(outcomes), len(cases))
	}
	seen := make(map[string]struct{}, len(outcomes))
	for _, item := range outcomes {
		if _, exists := cases[item.CaseID]; !exists {
			return fmt.Errorf("%s includes a case outside the fixed denominator", arm)
		}
		if _, exists := seen[item.CaseID]; exists {
			return fmt.Errorf("%s has a duplicate case outcome", arm)
		}
		seen[item.CaseID] = struct{}{}
		switch item.Status {
		case "succeeded":
		case "failed", "refused", "unattempted":
			if item.Passed {
				return fmt.Errorf("%s marks non-successful outcome as passed", arm)
			}
		default:
			return fmt.Errorf("%s has unknown outcome status %q", arm, item.Status)
		}
		if item.Passed && item.Status != "succeeded" {
			return fmt.Errorf("%s has a passing outcome without successful execution", arm)
		}
		if item.Passed && !validSHA256(item.ValidationReceiptHash) {
			return fmt.Errorf("%s has a passing outcome without a validation receipt", arm)
		}
		if item.LatencyMillis < 0 || item.InterventionCount < 0 {
			return fmt.Errorf("%s has negative measurement", arm)
		}
		if item.LatencyKnown && item.LatencyMillis == 0 {
			return fmt.Errorf("%s marks a zero latency as measured", arm)
		}
		if item.Status == "succeeded" && !item.LatencyKnown {
			return fmt.Errorf("%s has an executed case with unknown latency", arm)
		}
		if strings.TrimSpace(item.ValidationReceiptHash) != "" && !validSHA256(item.ValidationReceiptHash) {
			return fmt.Errorf("%s has invalid validation receipt digest", arm)
		}
	}
	return nil
}

func codeEvalRequiredStratumNames() []string {
	names := make([]string, 0, len(codeEvalRequiredStrata))
	for name := range codeEvalRequiredStrata {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func outcomePassed(outcome CodeEvalOutcome) bool {
	return outcome.Status == "succeeded" && outcome.Passed
}

func hasCodeEvalStratum(strata []string, wanted string) bool {
	return slices.Contains(strata, wanted)
}
