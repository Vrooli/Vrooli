package summarize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	SummaryEvaluationRole        = "summarize.default"
	SummaryEvaluatorID           = "audio-tools.internal.summarize.fixed-denominator.v1"
	SummaryEvaluationCases       = 20
	SummaryEvaluationCalibration = 12
	SummaryEvaluationHeldOut     = 8
)

var sha256Hex = regexp.MustCompile(`^[a-f0-9]{64}$`)

// SummaryEvaluationCase is one independently reviewed real caller input.
// SourceGroup must identify its originating conversation/document so the
// dataset validator can prevent calibration/held-out leakage.
type SummaryEvaluationCase struct {
	ID                   string   `json:"id"`
	SourceGroup          string   `json:"source_group"`
	Split                string   `json:"split"`
	Level                string   `json:"level"`
	CandidateContextTier int      `json:"candidate_context_tier"`
	Input                string   `json:"input"`
	RequiredFacts        []string `json:"required_facts"`
	ReviewerOracleID     string   `json:"reviewer_oracle_id"`
}

// SummaryEvaluationDataset is fixed before either arm is executed. It stores
// reviewed truth with the caller inputs; arm outputs cannot add, remove, or
// reclassify denominator rows.
type SummaryEvaluationDataset struct {
	ID                             string                  `json:"id"`
	Version                        string                  `json:"version"`
	ArtifactPath                   string                  `json:"artifact_path"`
	CallerID                       string                  `json:"caller_id"`
	CallerRevision                 string                  `json:"caller_revision"`
	HeldoutMembershipOwner         string                  `json:"heldout_membership_owner"`
	HeldoutMembershipReceiptID     string                  `json:"heldout_membership_receipt_id"`
	HeldoutMembershipReceiptPath   string                  `json:"heldout_membership_receipt_path"`
	HeldoutMembershipReceiptSHA256 string                  `json:"heldout_membership_receipt_sha256"`
	HeldoutMembershipSHA256        string                  `json:"heldout_membership_sha256"`
	Cases                          []SummaryEvaluationCase `json:"cases"`
}

// SummaryEvaluationIdentity binds both arms to exact owner receipts. Values
// are supplied by the resource and lifecycle owners; this evaluator never
// infers artifact, rights, serving, or policy identity from a model label.
type SummaryEvaluationIdentity struct {
	EvaluatorID              string `json:"evaluator_id"`
	EvaluatorSourceSHA256    string `json:"evaluator_source_sha256"`
	CallerID                 string `json:"caller_id"`
	CallerRevisionSHA256     string `json:"caller_revision_sha256"`
	Role                     string `json:"role"`
	ResolvedModel            string `json:"resolved_model"`
	ArtifactDigest           string `json:"artifact_digest"`
	RightsReceiptSHA256      string `json:"rights_receipt_sha256"`
	RightsReceiptPath        string `json:"rights_receipt_path"`
	RightsReceiptFileSHA256  string `json:"rights_receipt_file_sha256"`
	RightsDecisionSHA256     string `json:"rights_decision_sha256"`
	RightsEvaluatedAt        string `json:"rights_evaluated_at"`
	ServingReceiptSHA256     string `json:"serving_receipt_sha256"`
	RuntimeBuildID           string `json:"runtime_build_id"`
	PolicySHA256             string `json:"policy_sha256"`
	RuntimeConfigSHA256      string `json:"runtime_config_sha256"`
	ResourceReceiptPath      string `json:"resource_receipt_path"`
	ResourceReceiptSHA256    string `json:"resource_receipt_sha256"`
	BaselineEffectiveContext int    `json:"baseline_effective_context"`
}

type SummaryEvaluationArm struct {
	ID          string `json:"id"`
	ContextTier int    `json:"context_tier"`
}

type SummaryEvaluationResult struct {
	CaseID                  string `json:"case_id"`
	SourceGroup             string `json:"source_group"`
	Split                   string `json:"split"`
	ReviewerOracleID        string `json:"reviewer_oracle_id"`
	ArmID                   string `json:"arm_id"`
	EffectiveContextTier    int    `json:"effective_context_tier"`
	ResolvedModel           string `json:"resolved_model,omitempty"`
	Output                  string `json:"output,omitempty"`
	ElapsedMs               int64  `json:"elapsed_ms"`
	OutputTokens            int    `json:"output_tokens"`
	ProviderTotalDurationNS int64  `json:"provider_total_duration_ns"`
	ModelLoadDurationNS     int64  `json:"model_load_duration_ns"`
	PromptEvalCount         int    `json:"prompt_eval_count"`
	PromptEvalDurationNS    int64  `json:"prompt_eval_duration_ns"`
	DecodeDurationNS        int64  `json:"decode_duration_ns"`
	DoneReason              string `json:"done_reason,omitempty"`
	Error                   string `json:"error,omitempty"`
	IdentityError           string `json:"identity_error,omitempty"`
	LengthPass              bool   `json:"length_pass"`
	RequiredFactsHit        int    `json:"required_facts_hit"`
	RequiredFacts           int    `json:"required_facts"`
}

type SummaryEvaluationReport struct {
	DatasetID                      string                    `json:"dataset_id"`
	DatasetVersion                 string                    `json:"dataset_version"`
	DatasetArtifactPath            string                    `json:"dataset_artifact_path"`
	CallerID                       string                    `json:"caller_id"`
	CallerRevision                 string                    `json:"caller_revision"`
	DatasetSHA256                  string                    `json:"dataset_sha256"`
	DenominatorSHA256              string                    `json:"denominator_sha256"`
	HeldoutMembershipReceiptID     string                    `json:"heldout_membership_receipt_id"`
	HeldoutMembershipOwner         string                    `json:"heldout_membership_owner"`
	HeldoutMembershipReceiptPath   string                    `json:"heldout_membership_receipt_path"`
	HeldoutMembershipReceiptSHA256 string                    `json:"heldout_membership_receipt_sha256"`
	HeldoutMembershipSHA256        string                    `json:"heldout_membership_sha256"`
	Identity                       SummaryEvaluationIdentity `json:"identity"`
	ReviewedReportSHA256           string                    `json:"reviewed_report_sha256,omitempty"`
	HeldoutReviewReceiptID         string                    `json:"heldout_review_receipt_id,omitempty"`
	HeldoutReviewReceiptPath       string                    `json:"heldout_review_receipt_path,omitempty"`
	HeldoutReviewReceiptSHA256     string                    `json:"heldout_review_receipt_sha256,omitempty"`
	Arms                           []SummaryEvaluationArm    `json:"arms"`
	Results                        []SummaryEvaluationResult `json:"results"`
	BaselineCalls                  int                       `json:"baseline_calls"`
	CandidateCalls                 int                       `json:"candidate_calls"`
	BaselineP50Ms                  int64                     `json:"baseline_p50_ms"`
	BaselineP95Ms                  int64                     `json:"baseline_p95_ms"`
	CandidateP50Ms                 int64                     `json:"candidate_p50_ms"`
	CandidateP95Ms                 int64                     `json:"candidate_p95_ms"`
	LatencyImproved                bool                      `json:"latency_improved"`
	FailureCount                   int                       `json:"failure_count"`
	RetryCount                     int                       `json:"retry_count"`
	P50Ms                          int64                     `json:"p50_ms"`
	P95Ms                          int64                     `json:"p95_ms"`
	QualityEligible                bool                      `json:"quality_eligible"`
	BlindReviewRequired            bool                      `json:"blind_review_required"`
}

// Validate enforces the predeclared 12/8 split and caller provenance before
// any model output can be observed. It intentionally rejects partial corpora.
func (d SummaryEvaluationDataset) Validate() error {
	if strings.TrimSpace(d.ID) == "" || strings.TrimSpace(d.Version) == "" || strings.TrimSpace(d.CallerID) == "" || !sha256Hex.MatchString(d.CallerRevision) || !validRelativeReceiptPath(d.ArtifactPath) {
		return errors.New("dataset id, version, caller id, revision SHA-256, and contained artifact path are required")
	}
	if strings.TrimSpace(d.HeldoutMembershipOwner) == "" || strings.TrimSpace(d.HeldoutMembershipReceiptID) == "" || !validRelativeReceiptPath(d.HeldoutMembershipReceiptPath) || !sha256Hex.MatchString(d.HeldoutMembershipReceiptSHA256) || !sha256Hex.MatchString(d.HeldoutMembershipSHA256) {
		return errors.New("owner-approved heldout membership receipt and exact membership SHA-256 are required")
	}
	if len(d.Cases) != SummaryEvaluationCases {
		return fmt.Errorf("dataset has %d cases, want fixed denominator %d", len(d.Cases), SummaryEvaluationCases)
	}
	ids := make(map[string]struct{}, len(d.Cases))
	groupSplits := make(map[string]string, len(d.Cases))
	counts := map[string]int{"calibration": 0, "heldout": 0}
	for i, c := range d.Cases {
		if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.SourceGroup) == "" || strings.TrimSpace(c.Input) == "" || strings.TrimSpace(c.ReviewerOracleID) == "" {
			return fmt.Errorf("cases[%d] requires id, source group, input, and reviewer oracle id", i)
		}
		if _, exists := ids[c.ID]; exists {
			return fmt.Errorf("duplicate case id %q", c.ID)
		}
		ids[c.ID] = struct{}{}
		if _, ok := counts[c.Split]; !ok {
			return fmt.Errorf("cases[%d] has invalid split %q", i, c.Split)
		}
		counts[c.Split]++
		if previous, ok := groupSplits[c.SourceGroup]; ok && previous != c.Split {
			return fmt.Errorf("source group %q appears in both splits", c.SourceGroup)
		}
		groupSplits[c.SourceGroup] = c.Split
		if c.Level != "light" && c.Level != "moderate" && c.Level != "heavy" {
			return fmt.Errorf("cases[%d] has invalid summary level %q", i, c.Level)
		}
		if c.CandidateContextTier <= 0 {
			return fmt.Errorf("cases[%d] requires a pretrial candidate context tier", i)
		}
		if len(c.RequiredFacts) == 0 {
			return fmt.Errorf("cases[%d] requires independently reviewed facts", i)
		}
		for j, fact := range c.RequiredFacts {
			if strings.TrimSpace(fact) == "" {
				return fmt.Errorf("cases[%d].required_facts[%d] is empty", i, j)
			}
		}
	}
	if counts["calibration"] != SummaryEvaluationCalibration || counts["heldout"] != SummaryEvaluationHeldOut {
		return fmt.Errorf("dataset split is calibration=%d heldout=%d, want %d/%d", counts["calibration"], counts["heldout"], SummaryEvaluationCalibration, SummaryEvaluationHeldOut)
	}
	if d.HeldoutMembershipSHA256 != heldoutMembershipDigest(d.Cases) {
		return errors.New("heldout membership SHA-256 does not match the frozen heldout case IDs")
	}
	return nil
}

func heldoutMembershipDigest(cases []SummaryEvaluationCase) string {
	members := make([]string, 0, SummaryEvaluationHeldOut)
	for _, c := range cases {
		if c.Split == "heldout" {
			members = append(members, c.ID+"\x00"+c.SourceGroup+"\x00"+c.ReviewerOracleID)
		}
	}
	sort.Strings(members)
	digest := sha256.Sum256([]byte(strings.Join(members, "\n")))
	return hex.EncodeToString(digest[:])
}

func (d SummaryEvaluationDataset) Digest() (datasetSHA256, denominatorSHA256 string, err error) {
	if err := d.Validate(); err != nil {
		return "", "", err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return "", "", fmt.Errorf("marshal evaluation dataset: %w", err)
	}
	datasetDigest := sha256.Sum256(data)
	ids := make([]string, 0, len(d.Cases))
	for _, c := range d.Cases {
		ids = append(ids, c.ID+"\x00"+c.Split+"\x00"+c.ReviewerOracleID)
	}
	sort.Strings(ids)
	denomDigest := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	return hex.EncodeToString(datasetDigest[:]), hex.EncodeToString(denomDigest[:]), nil
}

func (id SummaryEvaluationIdentity) validate() error {
	if id.EvaluatorID != SummaryEvaluatorID || !sha256Hex.MatchString(id.EvaluatorSourceSHA256) {
		return errors.New("exact evaluator id and source SHA-256 are required")
	}
	if strings.TrimSpace(id.CallerID) == "" || !sha256Hex.MatchString(id.CallerRevisionSHA256) {
		return errors.New("exact caller id and caller revision SHA-256 are required")
	}
	if id.Role != SummaryEvaluationRole {
		return fmt.Errorf("evaluation role must be %q", SummaryEvaluationRole)
	}
	if strings.TrimSpace(id.ResolvedModel) == "" || !sha256Hex.MatchString(id.ArtifactDigest) {
		return errors.New("exact resolved model and artifact SHA-256 are required")
	}
	for name, value := range map[string]string{
		"rights receipt SHA-256":   id.RightsReceiptSHA256,
		"rights receipt path":      id.RightsReceiptPath,
		"rights receipt file SHA":  id.RightsReceiptFileSHA256,
		"rights decision SHA":      id.RightsDecisionSHA256,
		"rights evaluation time":   id.RightsEvaluatedAt,
		"serving receipt SHA-256":  id.ServingReceiptSHA256,
		"runtime build id":         id.RuntimeBuildID,
		"policy SHA-256":           id.PolicySHA256,
		"runtime config SHA-256":   id.RuntimeConfigSHA256,
		"resource receipt SHA-256": id.ResourceReceiptSHA256,
	} {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if !sha256Hex.MatchString(id.RightsReceiptSHA256) || !sha256Hex.MatchString(id.RightsReceiptFileSHA256) || !sha256Hex.MatchString(id.RightsDecisionSHA256) || !sha256Hex.MatchString(id.ServingReceiptSHA256) || !sha256Hex.MatchString(id.PolicySHA256) || !sha256Hex.MatchString(id.RuntimeConfigSHA256) || !sha256Hex.MatchString(id.ResourceReceiptSHA256) {
		return errors.New("receipt and policy identities must be SHA-256 values")
	}
	if _, err := time.Parse(time.RFC3339Nano, id.RightsEvaluatedAt); err != nil {
		return errors.New("rights decision evaluation timestamp must be RFC3339Nano")
	}
	if id.BaselineEffectiveContext <= 0 {
		return errors.New("baseline effective context must be positive")
	}
	if !validRelativeReceiptPath(id.ResourceReceiptPath) {
		return errors.New("resource receipt path must be policy-relative and contained")
	}
	if !validRelativeReceiptPath(id.RightsReceiptPath) {
		return errors.New("rights decision receipt path must be relative and contained")
	}
	return nil
}

// AttachHeldoutReview binds a completed blinded review receipt to an otherwise
// eligible report. The report is not promotable until this owner-issued review
// identity has been attached and the resulting report retained.
type HeldoutReviewVerifier interface {
	VerifyHeldoutReview(ctx context.Context, reportSHA256, receiptID, receiptPath, receiptSHA256 string) error
}

func (r *SummaryEvaluationReport) AttachHeldoutReview(ctx context.Context, verifier HeldoutReviewVerifier, receiptID, receiptPath, receiptSHA256 string) error {
	if r == nil || !r.QualityEligible || len(r.Results) != 2*SummaryEvaluationCases {
		return errors.New("heldout review requires a complete quality-eligible fixed-denominator report")
	}
	if verifier == nil || strings.TrimSpace(receiptID) == "" || !validRelativeReceiptPath(receiptPath) || !sha256Hex.MatchString(receiptSHA256) {
		return errors.New("heldout review receipt id and SHA-256 are required")
	}
	preReviewDigest, err := r.Digest()
	if err != nil {
		return err
	}
	if err := verifier.VerifyHeldoutReview(ctx, preReviewDigest, receiptID, receiptPath, receiptSHA256); err != nil {
		return fmt.Errorf("verify owner heldout review receipt: %w", err)
	}
	r.ReviewedReportSHA256 = preReviewDigest
	r.HeldoutReviewReceiptID = receiptID
	r.HeldoutReviewReceiptPath = receiptPath
	r.HeldoutReviewReceiptSHA256 = receiptSHA256
	r.BlindReviewRequired = false
	return nil
}

func validRelativeReceiptPath(value string) bool {
	return strings.TrimSpace(value) != "" && !strings.Contains(value, "\\") && !path.IsAbs(value) && path.Clean(value) == value && value != ".." && !strings.HasPrefix(value, "../")
}

// HeldoutMembershipVerifier is implemented by the evaluation-set owner. It
// resolves the immutable approval and verifies its exact case membership.
type HeldoutMembershipVerifier interface {
	VerifyHeldoutMembership(ctx context.Context, owner, receiptID, receiptPath, receiptSHA256, datasetSHA256, membershipSHA256 string) error
}

// RightsDecisionReceiptVerifier is implemented by the resource/rights owner.
// It must resolve and verify the exact persisted receipt against current
// authoritative artifact, policy and trusted-context inputs.
type RightsDecisionReceiptVerifier interface {
	VerifyRightsDecisionReceipt(ctx context.Context, role, model, receiptPath, receiptFileSHA256, receiptSHA256, decisionSHA256, evaluatedAt string) error
}

// Digest returns a stable SHA-256 over the complete report, including outputs,
// failures, the fixed denominator, heldout membership receipt and exact runtime
// identity. Persistent runtime policy may cite this digest; callers must retain
// the report itself as the attributable measurement artifact.
func (r SummaryEvaluationReport) Digest() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("marshal summary evaluation report: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return "", fmt.Errorf("canonicalize summary evaluation report: %w", err)
	}
	canonical, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("canonicalize summary evaluation report: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// RunSummaryEvaluation performs exactly two calls for every frozen case,
// paired baseline then candidate, through the maintained Audio Tools
// SummarizationService and resource-ollama gateway. It has no retry path.
func RunSummaryEvaluation(ctx context.Context, service *SummarizationService, dataset SummaryEvaluationDataset, identity SummaryEvaluationIdentity, candidateContext int, membershipVerifier HeldoutMembershipVerifier, rightsVerifier RightsDecisionReceiptVerifier) (SummaryEvaluationReport, error) {
	if service == nil {
		return SummaryEvaluationReport{}, errors.New("summarization service is required")
	}
	if err := dataset.Validate(); err != nil {
		return SummaryEvaluationReport{}, err
	}
	dataDigest, denominatorDigest, err := dataset.Digest()
	if err != nil {
		return SummaryEvaluationReport{}, err
	}
	if membershipVerifier == nil {
		return SummaryEvaluationReport{}, errors.New("owner heldout-membership verifier is required")
	}
	if err := membershipVerifier.VerifyHeldoutMembership(ctx, dataset.HeldoutMembershipOwner, dataset.HeldoutMembershipReceiptID, dataset.HeldoutMembershipReceiptPath, dataset.HeldoutMembershipReceiptSHA256, dataDigest, dataset.HeldoutMembershipSHA256); err != nil {
		return SummaryEvaluationReport{}, fmt.Errorf("verify owner heldout membership receipt: %w", err)
	}
	if rightsVerifier == nil {
		return SummaryEvaluationReport{}, errors.New("owner rights-decision receipt verifier is required")
	}
	if err := identity.validate(); err != nil {
		return SummaryEvaluationReport{}, err
	}
	if err := rightsVerifier.VerifyRightsDecisionReceipt(ctx, identity.Role, identity.ResolvedModel, identity.RightsReceiptPath, identity.RightsReceiptFileSHA256, identity.RightsReceiptSHA256, identity.RightsDecisionSHA256, identity.RightsEvaluatedAt); err != nil {
		return SummaryEvaluationReport{}, fmt.Errorf("verify owner rights-decision receipt: %w", err)
	}
	if dataset.CallerID != identity.CallerID || dataset.CallerRevision != identity.CallerRevisionSHA256 {
		return SummaryEvaluationReport{}, errors.New("dataset caller identity does not match the owner-verified caller identity")
	}
	if candidateContext <= 0 || candidateContext >= identity.BaselineEffectiveContext {
		return SummaryEvaluationReport{}, errors.New("candidate context must be positive and smaller than baseline context")
	}
	for i, c := range dataset.Cases {
		if c.CandidateContextTier != candidateContext && c.CandidateContextTier != identity.BaselineEffectiveContext {
			return SummaryEvaluationReport{}, fmt.Errorf("cases[%d] candidate context %d is outside the frozen compact/fallback tiers %d/%d", i, c.CandidateContextTier, candidateContext, identity.BaselineEffectiveContext)
		}
	}
	report := SummaryEvaluationReport{
		DatasetID: dataset.ID, DatasetVersion: dataset.Version, DatasetArtifactPath: dataset.ArtifactPath,
		CallerID: dataset.CallerID, CallerRevision: dataset.CallerRevision,
		DatasetSHA256: dataDigest, DenominatorSHA256: denominatorDigest,
		HeldoutMembershipReceiptID:     dataset.HeldoutMembershipReceiptID,
		HeldoutMembershipOwner:         dataset.HeldoutMembershipOwner,
		HeldoutMembershipReceiptPath:   dataset.HeldoutMembershipReceiptPath,
		HeldoutMembershipReceiptSHA256: dataset.HeldoutMembershipReceiptSHA256,
		HeldoutMembershipSHA256:        dataset.HeldoutMembershipSHA256,
		Identity:                       identity,
		Arms: []SummaryEvaluationArm{
			// Freeze the baseline's current daemon context as an explicit
			// setting. This prevents later persistent role-policy changes from
			// silently changing the comparison's control arm.
			{ID: "baseline", ContextTier: identity.BaselineEffectiveContext},
			{ID: "candidate", ContextTier: candidateContext},
		},
		Results:       make([]SummaryEvaluationResult, 0, 2*len(dataset.Cases)),
		BaselineCalls: len(dataset.Cases), CandidateCalls: len(dataset.Cases),
		BlindReviewRequired: true,
	}
	latencies := make([]int64, 0, 2*len(dataset.Cases))
	armLatencies := map[string][]int64{"baseline": {}, "candidate": {}}
	for _, c := range dataset.Cases {
		for _, arm := range report.Arms {
			if err := ctx.Err(); err != nil {
				return report, fmt.Errorf("evaluation cancelled with %d/%d calls recorded: %w", len(report.Results), 2*len(dataset.Cases), err)
			}
			contextTier := arm.ContextTier
			if arm.ID == "candidate" {
				contextTier = c.CandidateContextTier
			}
			result, err := service.Summarize(ctx, SummarizeRequest{
				EventID: fmt.Sprintf("w12r-%s-%s-%s", dataset.ID, c.ID, arm.ID),
				Path:    "evaluation", Text: c.Input, Level: c.Level,
				RuntimeContextTier:     contextTier,
				RightsDecisionSHA256:   identity.RightsDecisionSHA256,
				RightsEvaluatedAt:      identity.RightsEvaluatedAt,
				RightsReceiptSHA256:    identity.RightsReceiptSHA256,
				PolicyFileSHA256:       identity.PolicySHA256,
				ExpectedModel:          identity.ResolvedModel,
				ExpectedArtifactDigest: identity.ArtifactDigest,
			})
			row := scoreSummaryResult(c, arm.ID, result, err)
			row.EffectiveContextTier = contextTier
			if result.Config.Model != identity.Role {
				row.IdentityError = fmt.Sprintf("caller role mismatch: got %q, want %q", result.Config.Model, identity.Role)
			} else if err == nil && (row.ResolvedModel == "" || row.ResolvedModel != identity.ResolvedModel) {
				row.IdentityError = fmt.Sprintf("resolved model mismatch: got %q, want %q", row.ResolvedModel, identity.ResolvedModel)
				row.Output = ""
			}
			if row.Error != "" || row.IdentityError != "" {
				report.FailureCount++
			}
			latencies = append(latencies, row.ElapsedMs)
			armLatencies[arm.ID] = append(armLatencies[arm.ID], row.ElapsedMs)
			report.Results = append(report.Results, row)
		}
	}
	report.RetryCount = 0
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	report.P50Ms = percentileInt64(latencies, 0.50)
	report.P95Ms = percentileInt64(latencies, 0.95)
	for armID := range armLatencies {
		sort.Slice(armLatencies[armID], func(i, j int) bool { return armLatencies[armID][i] < armLatencies[armID][j] })
	}
	report.BaselineP50Ms = percentileInt64(armLatencies["baseline"], 0.50)
	report.BaselineP95Ms = percentileInt64(armLatencies["baseline"], 0.95)
	report.CandidateP50Ms = percentileInt64(armLatencies["candidate"], 0.50)
	report.CandidateP95Ms = percentileInt64(armLatencies["candidate"], 0.95)
	report.LatencyImproved = report.CandidateP95Ms < report.BaselineP95Ms
	report.QualityEligible = len(report.Results) == 2*SummaryEvaluationCases && report.FailureCount == 0
	return report, nil
}

func scoreSummaryResult(c SummaryEvaluationCase, arm string, result SummarizeResult, err error) SummaryEvaluationResult {
	row := SummaryEvaluationResult{
		CaseID: c.ID, SourceGroup: c.SourceGroup, Split: c.Split, ReviewerOracleID: c.ReviewerOracleID,
		ArmID: arm, ResolvedModel: result.Model,
		Output: result.Summary, ElapsedMs: result.ElapsedMs, OutputTokens: result.EvalCount,
		ProviderTotalDurationNS: result.TotalDurationNS, ModelLoadDurationNS: result.LoadDurationNS,
		PromptEvalCount: result.PromptEvalCount, PromptEvalDurationNS: result.PromptEvalDurationNS,
		DecodeDurationNS: result.EvalDurationNS,
		DoneReason:       result.DoneReason, RequiredFacts: len(c.RequiredFacts),
	}
	if err != nil {
		row.Error = err.Error()
		return row
	}
	wordCount := len(strings.Fields(result.Summary))
	sourceWords := len(strings.Fields(c.Input))
	switch c.Level {
	case "light":
		row.LengthPass = wordCount*100 <= sourceWords*55
	case "moderate":
		row.LengthPass = wordCount*100 <= sourceWords*35
	case "heavy":
		row.LengthPass = wordCount <= 40 && sentenceCount(result.Summary) <= 2
	}
	normalizedOutput := strings.ToLower(strings.Join(strings.Fields(result.Summary), " "))
	for _, fact := range c.RequiredFacts {
		if strings.Contains(normalizedOutput, strings.ToLower(strings.Join(strings.Fields(fact), " "))) {
			row.RequiredFactsHit++
		}
	}
	if !row.LengthPass {
		row.Error = "summary exceeds caller length budget"
	} else if row.RequiredFactsHit != row.RequiredFacts {
		row.Error = "summary omits one or more independently reviewed required facts"
	}
	return row
}

func sentenceCount(value string) int {
	count := 0
	for _, r := range value {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	return count
}

func percentileInt64(sorted []int64, q float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * q)
	return sorted[index]
}
