package eval

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/proto"
)

func TestQualifyRerankCorpusRequiresOwnerReceiptReviewedDenominatorAndFixedCandidateBytes(t *testing.T) {
	suite := ownerApprovedRerankSuite()
	qualification := testQualifyRerankCorpus(suite)
	require.True(t, qualification.GetQualified(), "%+v", qualification)
	require.Equal(t, int32(12), qualification.GetReviewedPositive())
	require.Equal(t, int32(3), qualification.GetHeldoutPositive())
	require.Equal(t, ownerCorpusDigest(suite), suite.GetRerankCorpusOwnerReceipt().GetCorpusSha256())
	require.Equal(t, RerankCandidateSnapshotsDigest(suite), qualification.GetCandidateSnapshotsSha256())
	require.Equal(t, RerankCorpusOwnerReceiptDigest(suite.GetRerankCorpusOwnerReceipt()), qualification.GetOwnerReceiptSha256())
}

func TestQualifyRerankCorpusRefusesMissingOrChangedOwnerEvidence(t *testing.T) {
	t.Run("missing receipt", func(t *testing.T) {
		q := testQualifyRerankCorpus(rerankCorpusCases())
		require.False(t, q.GetQualified())
		require.Equal(t, "owner_receipt_missing", q.GetRefusalReason())
	})
	t.Run("owner source verifier unavailable", func(t *testing.T) {
		q := QualifyRerankCorpus(context.Background(), ownerApprovedRerankSuite(), nil)
		require.False(t, q.GetQualified())
		require.Equal(t, "owner_receipt_verifier_unavailable", q.GetRefusalReason())
	})
	t.Run("receipt from another owner", func(t *testing.T) {
		suite := ownerApprovedRerankSuite()
		suite.RerankCorpusOwnerReceipt.OwnerProviderId = "different-provider"
		q := testQualifyRerankCorpus(suite)
		require.False(t, q.GetQualified())
		require.Equal(t, "owner_provider_mismatch", q.GetRefusalReason())
	})
	t.Run("snapshot mutation after receipt", func(t *testing.T) {
		suite := ownerApprovedRerankSuite()
		suite.Cases[0].PreRerankCandidates[0].Snippet = "changed after owner review"
		q := testQualifyRerankCorpus(suite)
		require.False(t, q.GetQualified())
		require.True(t, strings.Contains(q.GetRefusalReason(), "digest_mismatch"), q.GetRefusalReason())
	})
	t.Run("short corpus", func(t *testing.T) {
		suite := rerankCorpusCases()
		suite.Cases = suite.Cases[:10]
		sealOwnerReceipt(suite)
		q := testQualifyRerankCorpus(suite)
		require.False(t, q.GetQualified())
		require.Equal(t, "too_few_reviewed_positives", q.GetRefusalReason())
	})
	t.Run("too few owner heldout cases", func(t *testing.T) {
		suite := rerankCorpusCases()
		for i := 0; i < 3; i++ {
			suite.Cases[i].Tags = nil
		}
		sealOwnerReceipt(suite)
		q := testQualifyRerankCorpus(suite)
		require.False(t, q.GetQualified())
		require.Equal(t, "too_few_owner_heldout_positives", q.GetRefusalReason())
	})
	t.Run("candidate not pre-rerank", func(t *testing.T) {
		suite := rerankCorpusCases()
		suite.Cases[0].PreRerankCandidates[0].RerankScore = 0.2
		sealOwnerReceipt(suite)
		q := testQualifyRerankCorpus(suite)
		require.False(t, q.GetQualified())
		require.Equal(t, "candidate_snapshot_invalid:case-00:candidate 0 is not pre-rerank (rerank score is set)", q.GetRefusalReason())
	})
	t.Run("candidate limit", func(t *testing.T) {
		suite := rerankCorpusCases()
		for i := 0; i <= maxRerankSnapshotCandidates; i++ {
			suite.Cases[0].PreRerankCandidates = append(suite.Cases[0].PreRerankCandidates, &routingv1.SearchHit{Id: fmt.Sprintf("doc-extra-%d", i), Score: 0.1})
		}
		sealOwnerReceipt(suite)
		q := testQualifyRerankCorpus(suite)
		require.False(t, q.GetQualified())
		require.Contains(t, q.GetRefusalReason(), "candidate count")
	})
}

func TestBuildRerankerComparisonIdentityRefusesUnqualifiedCorpusAndBindsSnapshots(t *testing.T) {
	now := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	caller := CallerIdentity{ID: "search-hub", Version: "build-sha"}
	plain := rerankCorpusCases()
	_, err := BuildRerankerComparisonIdentity(plain, completeSnapshot(), caller, nil, now, VerifiedRerankCorpusOwnerReceipt{})
	require.ErrorContains(t, err, "owner_receipt_unverified")

	suite := ownerApprovedRerankSuite()
	verifiedReceipt, err := VerifyRerankCorpusOwnerReceipt(context.Background(), suite, testOwnerReceiptVerifier())
	require.NoError(t, err)
	validation := completeValidation(suite, now)
	first, err := BuildRerankerComparisonIdentity(suite, completeSnapshot(), caller, validation, now, verifiedReceipt)
	require.NoError(t, err)
	require.True(t, first.GetRerankerComparison())
	require.Equal(t, RerankCandidateSnapshotsDigest(suite), first.GetCandidateSnapshotsSha256())
	require.Equal(t, RerankCorpusOwnerReceiptDigest(suite.GetRerankCorpusOwnerReceipt()), first.GetCandidateOwnerReceiptSha256())
	require.Equal(t, []string{"case-00", "case-01", "case-02"}, first.GetHeldoutCaseIds())
	require.NoError(t, ValidatePromotionIdentity(first))

	second := protoCloneIdentity(first)
	second.ModelId = "different-model"
	second.ArtifactSha256 = strings.Repeat("9", 64)
	require.NoError(t, ValidateComparableRuns(first, second))
	second.CandidateSnapshotsSha256 = strings.Repeat("0", 64)
	require.ErrorContains(t, ValidateComparableRuns(first, second), "candidate snapshots or owner receipt differs")
}

func ownerApprovedRerankSuite() *evalv1.EvalSuite {
	suite := rerankCorpusCases()
	sealOwnerReceipt(suite)
	return suite
}

func testQualifyRerankCorpus(suite *evalv1.EvalSuite) *evalv1.RerankCorpusQualification {
	return QualifyRerankCorpus(context.Background(), suite, testOwnerReceiptVerifier())
}

func testOwnerReceiptVerifier() RerankCorpusOwnerReceiptVerifier {
	return RerankCorpusOwnerReceiptVerifierFunc(func(_ context.Context, suite *evalv1.EvalSuite, receiptSHA256 string) error {
		if suite.GetRerankCorpusOwnerReceipt() == nil || RerankCorpusOwnerReceiptDigest(suite.GetRerankCorpusOwnerReceipt()) != receiptSHA256 {
			return fmt.Errorf("owner receipt not present in trusted test source")
		}
		return nil
	})
}

func rerankCorpusCases() *evalv1.EvalSuite {
	suite := &evalv1.EvalSuite{SuiteId: "provider.docs.primary", ProviderId: "provider.docs"}
	for i := 0; i < minimumRerankReviewedPositives; i++ {
		id := fmt.Sprintf("case-%02d", i)
		tags := []string{"strong"}
		if i < minimumRerankHeldoutPositives {
			tags = append(tags, "heldout")
		}
		suite.Cases = append(suite.Cases, &evalv1.EvalCase{
			CaseId: id, Query: "reviewed query " + id, Status: "reviewed", Tags: tags,
			ExpectIds:           []string{"doc-" + id},
			PreRerankCandidates: []*routingv1.SearchHit{{ProviderId: "provider.docs", Id: "doc-" + id, Title: "Title " + id, Snippet: "Text " + id, Score: 0.8}},
		})
	}
	return suite
}

func sealOwnerReceipt(suite *evalv1.EvalSuite) {
	receipt := &evalv1.RerankCorpusOwnerReceipt{
		OwnerProviderId: suite.GetProviderId(), ApprovalRef: "provider-owned-review/retrieval-w11",
		SourceRevision: "source-revision-123", ApprovedBy: "provider-owner",
		ApprovedAt:               "2026-11-11T08:00:00Z",
		CorpusSha256:             ownerCorpusDigest(suite),
		DenominatorSha256:        denominatorDigest(reviewedPositiveCaseIDs(suite)),
		CandidateSnapshotsSha256: RerankCandidateSnapshotsDigest(suite),
	}
	receipt.ReceiptSha256 = RerankCorpusOwnerReceiptDigest(receipt)
	suite.RerankCorpusOwnerReceipt = receipt
}

func protoCloneIdentity(identity *evalv1.EvaluationIdentity) *evalv1.EvaluationIdentity {
	return proto.Clone(identity).(*evalv1.EvaluationIdentity)
}
