package eval

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/scheduletest"
)

func TestBuildComparisonIdentityBindsFixedReviewedDenominatorAndHeldoutSplit(t *testing.T) {
	suite := comparisonSuite(
		&evalv1.EvalCase{CaseId: "positive-a", Query: "a", Tags: []string{"heldout"}, ExpectIds: []string{"doc-a"}},
		&evalv1.EvalCase{CaseId: "generated", Query: "g", Tags: []string{"generated"}, ExpectIds: []string{"doc-g"}},
		&evalv1.EvalCase{CaseId: "generated-reviewed", Query: "gr", Tags: []string{"generated", "heldout"}, ExpectIds: []string{"doc-gr"}},
		&evalv1.EvalCase{CaseId: "candidate", Query: "c", Status: "candidate", ExpectIds: []string{"doc-c"}},
	)
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	validation := completeValidation(suite, now)
	caller := CallerIdentity{ID: "caller-app/search", Version: "caller-build-7"}
	first := BuildComparisonIdentity(suite, &evalv1.ConfigSnapshot{
		ProviderRevision: "provider-rev-99", ModelId: "model-id", ArtifactSha256: strings.Repeat("a", 64),
		LicenseId: "license-x", RightsDecisionId: "rights-17", RuntimeId: "runtime-x",
		RuntimeSha256: strings.Repeat("b", 64), RecipeSha256: strings.Repeat("c", 64),
	}, caller, validation, now)
	second := BuildComparisonIdentity(suite, &evalv1.ConfigSnapshot{
		ProviderRevision: "provider-rev-99", ModelId: "another-model", ArtifactSha256: strings.Repeat("e", 64),
		LicenseId: "license-y", RightsDecisionId: "rights-18", RuntimeId: "runtime-y",
		RuntimeSha256: strings.Repeat("f", 64), RecipeSha256: strings.Repeat("1", 64),
	}, caller, validation, now)
	require.Equal(t, []string{"generated-reviewed", "positive-a"}, first.GetDenominatorCaseIds())
	require.Empty(t, first.GetHeldoutCaseIds(), "unreceipted owner tags cannot qualify a reranker held-out split")
	require.NotContains(t, first.GetDenominatorCaseIds(), "generated", "unreviewed generated labels cannot enter the evaluation denominator")
	require.NotContains(t, first.GetDenominatorCaseIds(), "candidate")
	require.Equal(t, "provider", first.GetCorpusProviderId())
	require.Equal(t, caller.ID, first.GetCallerId())
	require.Equal(t, caller.Version, first.GetCallerVersion())
	require.NotEqual(t, first.GetCorpusProviderId(), first.GetCallerId())
	require.NotEqual(t, "provider-rev-99", first.GetCallerVersion(), "provider revision is not caller version")
	require.Equal(t, first.GetCorpusSha256(), second.GetCorpusSha256())
	require.Equal(t, first.GetProviderValidationSha256(), second.GetProviderValidationSha256())
	require.NoError(t, ValidatePromotionIdentity(first))
	require.NoError(t, ValidateComparableRuns(first, second), "arm-specific model/runtime identities may differ under the same fixed comparison contract")

	changedSuite := comparisonSuite(&evalv1.EvalCase{CaseId: "positive-a", Query: "changed", ExpectIds: []string{"doc-a"}})
	changed := BuildComparisonIdentity(changedSuite, completeSnapshot(), caller, completeValidation(changedSuite, now), now)
	require.ErrorContains(t, ValidateComparableRuns(first, changed), "mismatch")

	differentCaller := BuildComparisonIdentity(suite, &evalv1.ConfigSnapshot{
		ModelId: "third-model", ArtifactSha256: strings.Repeat("9", 64), LicenseId: "license-z", RightsDecisionId: "rights-z",
		RuntimeId: "runtime-z", RuntimeSha256: strings.Repeat("8", 64), RecipeSha256: strings.Repeat("7", 64),
	}, CallerIdentity{ID: "different-app", Version: "caller-build-7"}, validation, now)
	require.ErrorContains(t, ValidateComparableRuns(first, differentCaller), "mismatch")
}

func TestBuildComparisonIdentityLeavesCallerUnknownWithoutAuthoritativeEvidence(t *testing.T) {
	suite := comparisonSuite(&evalv1.EvalCase{CaseId: "case-1", Query: "q", ExpectIds: []string{"doc"}})
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	identity := BuildComparisonIdentity(suite, completeSnapshot(), CallerIdentity{}, completeValidation(suite, now), now)
	require.Empty(t, identity.GetCallerId())
	require.Empty(t, identity.GetCallerVersion())
	require.ErrorContains(t, ValidatePromotionIdentity(identity), "caller_id is unbound")

	identity.CallerId = "caller-app"
	require.ErrorContains(t, ValidatePromotionIdentity(identity), "caller_version is unbound")
}

func TestBuildComparisonIdentityRequiresFreshExactCompleteValidationReceipt(t *testing.T) {
	suite := comparisonSuite(
		&evalv1.EvalCase{CaseId: "positive-a", Query: "a", ExpectIds: []string{"doc-a"}},
		&evalv1.EvalCase{CaseId: "positive-b", Query: "b", ExpectIds: []string{"doc-b"}},
	)
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	caller := CallerIdentity{ID: "caller-app", Version: "caller-build"}
	snapshot := completeSnapshot()
	valid := completeValidation(suite, now)
	for _, test := range []struct {
		name   string
		mutate func(*CorpusValidationRecord)
	}{
		{name: "missing", mutate: func(record *CorpusValidationRecord) { *record = CorpusValidationRecord{} }},
		{name: "stale", mutate: func(record *CorpusValidationRecord) {
			record.CreatedAt = now.Add(-providerValidationFreshnessWindow - time.Second)
		}},
		{name: "suite digest mismatch", mutate: func(record *CorpusValidationRecord) { record.Result.SuiteSha256 = strings.Repeat("0", 64) }},
		{name: "denominator mismatch", mutate: func(record *CorpusValidationRecord) { record.Result.DenominatorCaseIds = []string{"positive-a"} }},
		{name: "incomplete outcomes", mutate: func(record *CorpusValidationRecord) {
			record.Result.Cases = record.Result.Cases[:1]
			record.Result.Rollup.Live = 1
			record.Result.Rollup.Hard = 0
		}},
		{name: "stale outcome", mutate: func(record *CorpusValidationRecord) {
			record.Result.Cases[0].Referential = evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_STALE
			record.Result.Rollup.Stale = 1
			record.Result.Rollup.Live = 1
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			record := cloneValidation(valid)
			test.mutate(record)
			identity := BuildComparisonIdentity(suite, snapshot, caller, record, now)
			require.Empty(t, identity.GetProviderValidationSha256())
			require.Empty(t, identity.GetProviderValidationAt())
			require.ErrorContains(t, ValidatePromotionIdentity(identity), "provider_validation_sha256 is unbound")
		})
	}
}

func TestBuildComparisonIdentityDoesNotTrustProviderStatusValidationFields(t *testing.T) {
	suite := comparisonSuite(&evalv1.EvalCase{CaseId: "case-1", Query: "q", ExpectIds: []string{"doc"}})
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	identity := BuildComparisonIdentity(suite, &evalv1.ConfigSnapshot{ModelId: "model"}, CallerIdentity{ID: "caller", Version: "v1"}, nil, now)
	require.Empty(t, identity.GetProviderValidationSha256())
	require.Empty(t, identity.GetProviderValidationAt())
}

func TestUpdateComparisonArmIdentityPreservesCallerAndValidationEvidence(t *testing.T) {
	identity := &evalv1.EvaluationIdentity{CallerId: "caller-app", CallerVersion: "caller-build", ProviderValidationSha256: strings.Repeat("f", 64), ProviderValidationAt: "2026-11-11T08:00:00Z"}
	updated := UpdateComparisonArmIdentity(identity, &evalv1.ConfigSnapshot{ModelId: "model-b", RecipeSha256: strings.Repeat("a", 64)})
	require.Equal(t, identity.GetCallerId(), updated.GetCallerId())
	require.Equal(t, identity.GetCallerVersion(), updated.GetCallerVersion())
	require.Equal(t, identity.GetProviderValidationSha256(), updated.GetProviderValidationSha256())
	require.NotSame(t, identity, updated)
}

func TestRunnerBindsOnlyResolvedCallerAndPersistedValidationReceipt(t *testing.T) {
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	suite := comparisonSuite(&evalv1.EvalCase{CaseId: "case-1", Query: "q", ExpectIds: []string{"doc"}})
	receipt := completeValidation(suite, now)
	client := identityTestClient{snapshot: &evalv1.ConfigSnapshot{
		ProviderRevision: "provider-revision", ModelId: "model", ArtifactSha256: strings.Repeat("c", 64),
		LicenseId: "license", RightsDecisionId: "rights", RuntimeId: "runtime", RuntimeSha256: strings.Repeat("d", 64),
		RecipeSha256: strings.Repeat("e", 64),
	}}
	clk := scheduletest.New(now)
	runner := NewRunner(identityTestResolver{}, client, clk, func() string { return "run" }, IdentityEvidenceReaders{
		Caller:     identityTestCaller{identity: CallerIdentity{ID: "trusted-caller", Version: "trusted-build"}},
		Validation: identityTestValidation{record: receipt},
	})
	run, err := runner.Run(context.Background(), suite, "tag", 1)
	require.NoError(t, err)
	identity := run.GetComparisonIdentity()
	require.Equal(t, "provider", identity.GetCorpusProviderId())
	require.Equal(t, "trusted-caller", identity.GetCallerId())
	require.Equal(t, "trusted-build", identity.GetCallerVersion())
	require.NotEqual(t, identity.GetCorpusProviderId(), identity.GetCallerId())
	require.NotEqual(t, "provider-revision", identity.GetCallerVersion())
	require.Equal(t, CorpusValidationReceiptDigest(receipt.Result), identity.GetProviderValidationSha256())
	require.Equal(t, now.Format(time.RFC3339Nano), identity.GetProviderValidationAt())
	require.NoError(t, ValidatePromotionIdentity(identity))

	withoutCaller := NewRunner(identityTestResolver{}, client, clk, func() string { return "run-2" }, IdentityEvidenceReaders{Validation: identityTestValidation{record: receipt}})
	withoutCallerRun, err := withoutCaller.Run(context.Background(), suite, "tag", 1)
	require.NoError(t, err)
	require.Empty(t, withoutCallerRun.GetComparisonIdentity().GetCallerId())
	require.Empty(t, withoutCallerRun.GetComparisonIdentity().GetCallerVersion())
}

type identityTestResolver struct{}

func (identityTestResolver) Get(_ context.Context, providerID string) (*registryv1.ProviderDescriptor, error) {
	return &registryv1.ProviderDescriptor{ProviderId: providerID}, nil
}

type identityTestClient struct{ snapshot *evalv1.ConfigSnapshot }

func (identityTestClient) Search(_ context.Context, _ *registryv1.ProviderDescriptor, _ string, _ int32, _ SearchCallOptions) ([]*routingv1.SearchHit, error) {
	return []*routingv1.SearchHit{{Id: "doc", Score: 0.9}}, nil
}

func (c identityTestClient) Snapshot(context.Context, *registryv1.ProviderDescriptor) *evalv1.ConfigSnapshot {
	return c.snapshot
}

type identityTestCaller struct{ identity CallerIdentity }

func (c identityTestCaller) ResolveCallerIdentity(context.Context, *evalv1.EvalSuite) (CallerIdentity, error) {
	return c.identity, nil
}

type identityTestValidation struct{ record *CorpusValidationRecord }

func (v identityTestValidation) LatestCorpusValidation(context.Context, string) (*CorpusValidationRecord, error) {
	return v.record, nil
}

func completeValidation(suite *evalv1.EvalSuite, createdAt time.Time) *CorpusValidationRecord {
	ids := reviewedPositiveCaseIDs(suite)
	var candidates int32
	for _, c := range suite.GetCases() {
		if positiveProtoCase(c) && c.GetStatus() == "candidate" {
			candidates++
		}
	}
	rows := make([]*evalv1.CorpusValidationCase, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, &evalv1.CorpusValidationCase{CaseId: id, Referential: evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE})
	}
	return &CorpusValidationRecord{CreatedAt: createdAt, Result: &evalv1.ValidateCorpusResponse{
		SuiteId: suite.GetSuiteId(), ProviderId: suite.GetProviderId(), SuiteSha256: corpusDigest(suite),
		DenominatorCaseIds: ids, DenominatorSha256: denominatorDigest(ids), Cases: rows,
		Rollup: &evalv1.CorpusValidationRollup{Positives: int32(len(ids)), Candidate: candidates, Live: int32(len(ids))},
	}}
}

func completeSnapshot() *evalv1.ConfigSnapshot {
	return &evalv1.ConfigSnapshot{ProviderRevision: "provider-revision", ModelId: "model", ArtifactSha256: strings.Repeat("c", 64), LicenseId: "license", RightsDecisionId: "rights", RuntimeId: "runtime", RuntimeSha256: strings.Repeat("d", 64), RecipeSha256: strings.Repeat("e", 64)}
}

func cloneValidation(record *CorpusValidationRecord) *CorpusValidationRecord {
	return &CorpusValidationRecord{CreatedAt: record.CreatedAt, Result: proto.Clone(record.Result).(*evalv1.ValidateCorpusResponse)}
}

func comparisonSuite(cases ...*evalv1.EvalCase) *evalv1.EvalSuite {
	return &evalv1.EvalSuite{SuiteId: "suite", ProviderId: "provider", Cases: cases}
}

func TestValidatePromotionIdentityRejectsMalformedFields(t *testing.T) {
	identity := completeIdentity()
	require.NoError(t, ValidatePromotionIdentity(identity))
	identity.LicenseId = ""
	require.ErrorContains(t, ValidatePromotionIdentity(identity), "license_id is unbound")
	identity.LicenseId = "license"
	identity.ArtifactSha256 = "mutable-name-only"
	require.ErrorContains(t, ValidatePromotionIdentity(identity), "artifact_sha256 must be a SHA-256 digest")
}

func completeIdentity() *evalv1.EvaluationIdentity {
	return &evalv1.EvaluationIdentity{CorpusProviderId: "provider", CorpusId: "suite", CorpusSha256: strings.Repeat("a", 64), SplitVersion: "split-v1", DenominatorCaseIds: []string{"case-1"}, DenominatorSha256: "ba225b9895eafb5ed01ea5320527c9d986b70b8fc754ee8e7693cb81b760d156", CallerId: "caller", CallerVersion: "caller-version", ModelId: "model", ArtifactSha256: strings.Repeat("c", 64), LicenseId: "license", RightsDecisionId: "rights", RuntimeId: "runtime", RuntimeSha256: strings.Repeat("d", 64), RecipeSha256: strings.Repeat("e", 64), EvaluationVersion: "eval-v1", ProviderValidationSha256: strings.Repeat("f", 64), ProviderValidationAt: "2026-11-11T08:00:00Z"}
}
