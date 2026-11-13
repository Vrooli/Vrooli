package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"google.golang.org/protobuf/proto"
)

const (
	evaluationVersion                 = "search-hub-eval/fixed-denominator/v1"
	evaluationSplitVersion            = "owner-tagged-heldout-excludes-generated/v2"
	providerValidationFreshnessWindow = 30 * 24 * time.Hour
)

// CallerIdentity is resolved at the trusted caller boundary. The evaluator
// never fills it from the corpus provider descriptor or the provider's status
// revision. A missing resolver or unresolved identity remains empty.
type CallerIdentity struct {
	ID      string
	Version string
}

// CallerIdentityResolver is an owner-controlled seam for authoritative
// caller evidence. Request-supplied or provider-reported labels are not an
// implementation of this contract unless their authority is verified first.
type CallerIdentityResolver interface {
	ResolveCallerIdentity(ctx context.Context, suite *evalv1.EvalSuite) (CallerIdentity, error)
}

// CallerIdentityResolverFunc adapts an owner-controlled function to the
// resolver seam. Production wiring uses it only for evaluation paths that
// exercise Search Hub's own query behavior.
type CallerIdentityResolverFunc func(context.Context, *evalv1.EvalSuite) (CallerIdentity, error)

func (f CallerIdentityResolverFunc) ResolveCallerIdentity(ctx context.Context, suite *evalv1.EvalSuite) (CallerIdentity, error) {
	return f(ctx, suite)
}

// hasSearchHubWorkloadCase reports whether a federated suite contains a real
// query that the runner will send through Search Hub. Empty suites and cases
// excluded from evaluation cannot establish workload identity.
func hasSearchHubWorkloadCase(suite *evalv1.EvalSuite) bool {
	if suite == nil {
		return false
	}
	for _, c := range suite.GetCases() {
		if isExcludedEvaluationCase(c) || strings.TrimSpace(c.GetQuery()) == "" {
			continue
		}
		return true
	}
	return false
}

// IdentityEvidenceReaders contains trusted caller and durable provider
// validation evidence used when an EvalRun is created.
type IdentityEvidenceReaders struct {
	Caller       CallerIdentityResolver
	Validation   CorpusValidationReader
	Denominators FixedDenominatorReader
}

// BuildComparisonIdentity binds a run to the provider-authored corpus, its
// owner-tagged held-out membership, independently resolved caller evidence,
// model/runtime identity and a fresh persisted validation receipt. Missing
// owner evidence is preserved as empty and prevents promotion.
func BuildComparisonIdentity(suite *evalv1.EvalSuite, snapshot *evalv1.ConfigSnapshot, caller CallerIdentity, validation *CorpusValidationRecord, now time.Time) *evalv1.EvaluationIdentity {
	if suite == nil {
		return &evalv1.EvaluationIdentity{EvaluationVersion: evaluationVersion, SplitVersion: evaluationSplitVersion}
	}
	corpusDigest := corpusDigest(suite)
	ids := reviewedPositiveCaseIDs(suite)
	identity := &evalv1.EvaluationIdentity{
		CorpusProviderId:   suite.GetProviderId(),
		CorpusId:           suite.GetSuiteId(),
		CorpusSha256:       corpusDigest,
		SplitVersion:       evaluationSplitVersion,
		DenominatorCaseIds: append([]string(nil), ids...),
		DenominatorSha256:  denominatorDigest(ids),
		CallerId:           strings.TrimSpace(caller.ID),
		CallerVersion:      strings.TrimSpace(caller.Version),
		EvaluationVersion:  evaluationVersion,
	}
	if snapshot != nil {
		identity.ModelId = snapshot.GetModelId()
		identity.ArtifactSha256 = snapshot.GetArtifactSha256()
		identity.LicenseId = snapshot.GetLicenseId()
		identity.RightsDecisionId = snapshot.GetRightsDecisionId()
		identity.RuntimeId = snapshot.GetRuntimeId()
		identity.RuntimeSha256 = snapshot.GetRuntimeSha256()
		identity.RecipeSha256 = recipeDigest(snapshot)
	}
	if validationReceiptMatchesSuite(suite, ids, validation, now) {
		identity.ProviderValidationSha256 = CorpusValidationReceiptDigest(validation.Result)
		identity.ProviderValidationAt = validation.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return identity
}

// BindFixedDenominatorIdentity replaces the live-suites' positive-ID digest
// with the exact reviewed snapshot digest. Provider validation remains tied to
// the current suite bytes by BuildComparisonIdentity; a stale validation
// receipt is never carried forward after suite edits.
func BindFixedDenominatorIdentity(identity *evalv1.EvaluationIdentity, snapshot *evalv1.FixedDenominatorSnapshot) *evalv1.EvaluationIdentity {
	if identity == nil || snapshot == nil {
		return identity
	}
	updated := proto.Clone(identity).(*evalv1.EvaluationIdentity)
	updated.DenominatorCaseIds = append([]string(nil), snapshot.GetCaseIds()...)
	updated.DenominatorSha256 = snapshot.GetDenominatorSha256()
	return updated
}

// BuildRerankerComparisonIdentity is the stricter identity path for an arm
// that reranks a frozen, owner-approved candidate set. Ordinary provider evals
// must not call this: their live retrieval output is not the saved snapshot.
func BuildRerankerComparisonIdentity(suite *evalv1.EvalSuite, snapshot *evalv1.ConfigSnapshot, caller CallerIdentity, validation *CorpusValidationRecord, now time.Time, verifiedReceipt VerifiedRerankCorpusOwnerReceipt) (*evalv1.EvaluationIdentity, error) {
	if !verifiedReceipt.verified || verifiedReceipt.receiptSHA256 == "" {
		return nil, fmt.Errorf("reranker comparison refused: owner_receipt_unverified")
	}
	verifier := RerankCorpusOwnerReceiptVerifierFunc(func(_ context.Context, _ *evalv1.EvalSuite, receiptSHA256 string) error {
		if receiptSHA256 != verifiedReceipt.receiptSHA256 {
			return fmt.Errorf("verified owner receipt digest changed")
		}
		return nil
	})
	qualification := QualifyRerankCorpus(context.Background(), suite, verifier)
	if !qualification.GetQualified() {
		return nil, fmt.Errorf("reranker comparison refused: %s", qualification.GetRefusalReason())
	}
	identity := BuildComparisonIdentity(suite, snapshot, caller, validation, now)
	identity.RerankerComparison = true
	identity.CandidateSnapshotsSha256 = qualification.GetCandidateSnapshotsSha256()
	identity.CandidateOwnerReceiptSha256 = qualification.GetOwnerReceiptSha256()
	identity.HeldoutCaseIds = RerankHeldoutIDs(suite)
	return identity, nil
}

// reviewedPositiveCaseIDs is the fixed acceptance denominator shared by
// grading, corpus validation and comparison identity. Unreviewed generated
// cases are excluded; an explicit owner-authored heldout tag is the review
// record that admits a generated case. Composed suites retain only cases with
// an explicit owning provider.
func reviewedPositiveCaseIDs(suite *evalv1.EvalSuite) []string {
	if suite == nil {
		return nil
	}
	composed := false
	for _, c := range suite.GetCases() {
		if c != nil && c.GetExpectedProviderId() != "" {
			composed = true
			break
		}
	}
	ids := make([]string, 0, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if isExcludedEvaluationCase(c) || !positiveProtoCase(c) || (composed && c.GetExpectedProviderId() == "") {
			continue
		}
		ids = append(ids, c.GetCaseId())
	}
	sort.Strings(ids)
	return ids
}

// isExcludedEvaluationCase keeps candidates and machine-generated cases out of
// every evaluation denominator until an owner explicitly reviews generated
// labels by adding TagHeldout. The heldout tag is the durable review/provenance
// fact carried in the owner-controlled corpus bytes.
func isExcludedEvaluationCase(c *evalv1.EvalCase) bool {
	if c == nil || c.GetStatus() == "candidate" {
		return true
	}
	return hasEvaluationTag(c.GetTags(), aisearch.TagGenerated) && !hasEvaluationTag(c.GetTags(), aisearch.TagHeldout)
}

func validationReceiptMatchesSuite(suite *evalv1.EvalSuite, denominator []string, record *CorpusValidationRecord, now time.Time) bool {
	if suite == nil || record == nil || record.Result == nil || record.Result.GetRollup() == nil || record.CreatedAt.IsZero() || now.IsZero() {
		return false
	}
	age := now.UTC().Sub(record.CreatedAt.UTC())
	if age < 0 || age > providerValidationFreshnessWindow {
		return false
	}
	result := record.Result
	if result.GetSuiteId() != suite.GetSuiteId() || result.GetProviderId() != suite.GetProviderId() ||
		result.GetSuiteSha256() != corpusDigest(suite) ||
		result.GetDenominatorSha256() != denominatorDigest(denominator) ||
		!equalStrings(result.GetDenominatorCaseIds(), denominator) {
		return false
	}
	rollup := result.GetRollup()
	candidateCount := 0
	for _, c := range suite.GetCases() {
		if positiveProtoCase(c) && c.GetStatus() == "candidate" {
			candidateCount++
		}
	}
	if int(rollup.GetPositives()) != len(denominator) || int(rollup.GetCandidate()) != candidateCount ||
		rollup.GetStale() != 0 || rollup.GetInconclusive() != 0 || rollup.GetProviderErrors() != 0 {
		return false
	}
	seen := make(map[string]struct{}, len(result.GetCases()))
	var live, hard int32
	for _, row := range result.GetCases() {
		if row == nil || !containsIdentityCase(denominator, row.GetCaseId()) {
			return false
		}
		if _, duplicate := seen[row.GetCaseId()]; duplicate {
			return false
		}
		seen[row.GetCaseId()] = struct{}{}
		outcome := row.GetReferential()
		switch outcome {
		case evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE:
			live++
		case evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_HARD:
			hard++
		default:
			return false
		}
	}
	return len(seen) == len(denominator) && live == rollup.GetLive() && hard == rollup.GetHard()
}

// CorpusValidationReceiptDigest returns the stable digest of the complete
// persisted validation result. Read surfaces can expose this digest without
// exposing provider-owned queries or error text.
func CorpusValidationReceiptDigest(result *evalv1.ValidateCorpusResponse) string {
	if result == nil {
		return ""
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(result)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// UpdateComparisonArmIdentity applies one sweep arm's captured config to the
// same identity that the runner built from authoritative caller/validation
// evidence. It never synthesizes or replaces caller or receipt fields.
func UpdateComparisonArmIdentity(identity *evalv1.EvaluationIdentity, snapshot *evalv1.ConfigSnapshot) *evalv1.EvaluationIdentity {
	if identity == nil {
		return nil
	}
	updated := proto.Clone(identity).(*evalv1.EvaluationIdentity)
	if snapshot == nil {
		return updated
	}
	updated.ModelId = snapshot.GetModelId()
	updated.ArtifactSha256 = snapshot.GetArtifactSha256()
	updated.LicenseId = snapshot.GetLicenseId()
	updated.RightsDecisionId = snapshot.GetRightsDecisionId()
	updated.RuntimeId = snapshot.GetRuntimeId()
	updated.RuntimeSha256 = snapshot.GetRuntimeSha256()
	updated.RecipeSha256 = recipeDigest(snapshot)
	return updated
}

// ValidateComparableRuns verifies that two immutable runs use the exact same
// caller, corpus, split, evaluator and denominator and both have complete
// owner-reported arm identities. It intentionally refuses display names or
// missing IDs as substitutes for artifact, license, runtime or recipe identity.
func ValidateComparableRuns(a, b *evalv1.EvaluationIdentity) error {
	if err := ValidatePromotionIdentity(a); err != nil {
		return fmt.Errorf("first run identity: %w", err)
	}
	if err := ValidatePromotionIdentity(b); err != nil {
		return fmt.Errorf("second run identity: %w", err)
	}
	if a.GetCorpusProviderId() != b.GetCorpusProviderId() ||
		a.GetCorpusId() != b.GetCorpusId() ||
		a.GetCorpusSha256() != b.GetCorpusSha256() ||
		a.GetSplitVersion() != b.GetSplitVersion() ||
		a.GetCallerId() != b.GetCallerId() ||
		a.GetCallerVersion() != b.GetCallerVersion() ||
		a.GetEvaluationVersion() != b.GetEvaluationVersion() ||
		a.GetProviderValidationSha256() != b.GetProviderValidationSha256() ||
		a.GetProviderValidationAt() != b.GetProviderValidationAt() ||
		a.GetDenominatorSha256() != b.GetDenominatorSha256() ||
		!equalStrings(a.GetDenominatorCaseIds(), b.GetDenominatorCaseIds()) ||
		!equalStrings(a.GetHeldoutCaseIds(), b.GetHeldoutCaseIds()) ||
		a.GetRerankerComparison() != b.GetRerankerComparison() {
		return fmt.Errorf("comparison contract mismatch: caller, corpus, validation, split, evaluation version, or fixed denominator differs")
	}
	if a.GetRerankerComparison() && (a.GetCandidateSnapshotsSha256() != b.GetCandidateSnapshotsSha256() || a.GetCandidateOwnerReceiptSha256() != b.GetCandidateOwnerReceiptSha256()) {
		return fmt.Errorf("comparison contract mismatch: reranker candidate snapshots or owner receipt differs")
	}
	return nil
}

func ValidatePromotionIdentity(identity *evalv1.EvaluationIdentity) error {
	if identity == nil {
		return fmt.Errorf("comparison identity is required")
	}
	for _, field := range []struct{ name, value string }{
		{"corpus_provider_id", identity.GetCorpusProviderId()},
		{"corpus_id", identity.GetCorpusId()},
		{"corpus_sha256", identity.GetCorpusSha256()},
		{"split_version", identity.GetSplitVersion()},
		{"denominator_sha256", identity.GetDenominatorSha256()},
		{"caller_id", identity.GetCallerId()},
		{"caller_version", identity.GetCallerVersion()},
		{"model_id", identity.GetModelId()},
		{"artifact_sha256", identity.GetArtifactSha256()},
		{"license_id", identity.GetLicenseId()},
		{"rights_decision_id", identity.GetRightsDecisionId()},
		{"runtime_id", identity.GetRuntimeId()},
		{"runtime_sha256", identity.GetRuntimeSha256()},
		{"recipe_sha256", identity.GetRecipeSha256()},
		{"evaluation_version", identity.GetEvaluationVersion()},
		{"provider_validation_sha256", identity.GetProviderValidationSha256()},
		{"provider_validation_at", identity.GetProviderValidationAt()},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is unbound", field.name)
		}
	}
	for _, field := range []struct{ name, value string }{
		{"corpus_sha256", identity.GetCorpusSha256()},
		{"denominator_sha256", identity.GetDenominatorSha256()},
		{"artifact_sha256", identity.GetArtifactSha256()},
		{"runtime_sha256", identity.GetRuntimeSha256()},
		{"recipe_sha256", identity.GetRecipeSha256()},
		{"provider_validation_sha256", identity.GetProviderValidationSha256()},
	} {
		decoded, err := hex.DecodeString(field.value)
		if err != nil || len(decoded) != sha256.Size {
			return fmt.Errorf("%s must be a SHA-256 digest", field.name)
		}
	}
	if identity.GetRerankerComparison() {
		for _, field := range []struct{ name, value string }{
			{"candidate_snapshots_sha256", identity.GetCandidateSnapshotsSha256()},
			{"candidate_owner_receipt_sha256", identity.GetCandidateOwnerReceiptSha256()},
		} {
			if strings.TrimSpace(field.value) == "" {
				return fmt.Errorf("%s is unbound", field.name)
			}
			if !isSHA256(field.value) {
				return fmt.Errorf("%s must be a SHA-256 digest", field.name)
			}
		}
	} else if identity.GetCandidateSnapshotsSha256() != "" || identity.GetCandidateOwnerReceiptSha256() != "" {
		return fmt.Errorf("candidate snapshot identity is only valid for a reranker comparison")
	}
	if len(identity.GetDenominatorCaseIds()) == 0 {
		return fmt.Errorf("fixed denominator is empty")
	}
	if !sort.StringsAreSorted(identity.GetDenominatorCaseIds()) || !uniqueCaseIDs(identity.GetDenominatorCaseIds()) {
		return fmt.Errorf("fixed denominator case IDs must be sorted, non-empty, and unique")
	}
	if denominatorDigest(identity.GetDenominatorCaseIds()) != identity.GetDenominatorSha256() {
		return fmt.Errorf("denominator_sha256 does not match its case IDs")
	}
	if !sort.StringsAreSorted(identity.GetHeldoutCaseIds()) || !uniqueCaseIDs(identity.GetHeldoutCaseIds()) {
		return fmt.Errorf("held-out case IDs must be sorted, non-empty, and unique")
	}
	for _, id := range identity.GetHeldoutCaseIds() {
		if !containsIdentityCase(identity.GetDenominatorCaseIds(), id) {
			return fmt.Errorf("held-out case %q is outside the fixed denominator", id)
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, identity.GetProviderValidationAt()); err != nil {
		return fmt.Errorf("provider_validation_at must be RFC3339Nano")
	}
	return nil
}

func corpusDigest(suite *evalv1.EvalSuite) string {
	canonical := proto.Clone(suite).(*evalv1.EvalSuite)
	canonical.Name = ""
	canonical.Description = ""
	canonical.CreatedAt = ""
	canonical.UpdatedAt = ""
	bytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(bytes)
	return hex.EncodeToString(digest[:])
}

func denominatorDigest(ids []string) string {
	canonical := append([]string(nil), ids...)
	sort.Strings(canonical)
	digest := sha256.Sum256([]byte(strings.Join(canonical, "\n")))
	return hex.EncodeToString(digest[:])
}

func recipeDigest(snapshot *evalv1.ConfigSnapshot) string {
	if snapshot == nil || !isSHA256(snapshot.GetRecipeSha256()) {
		return ""
	}
	canonical := proto.Clone(snapshot).(*evalv1.ConfigSnapshot)
	base := canonical.GetRecipeSha256()
	canonical.RecipeSha256 = ""
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(append(append([]byte(base), '\n'), encoded...))
	return hex.EncodeToString(digest[:])
}

func isSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func uniqueCaseIDs(ids []string) bool {
	for index, id := range ids {
		if id == "" || (index > 0 && ids[index-1] == id) {
			return false
		}
	}
	return true
}

func containsIdentityCase(ids []string, want string) bool {
	index := sort.SearchStrings(ids, want)
	return index < len(ids) && ids[index] == want
}

func hasEvaluationTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}
