package findings_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"web-search/internal/findings"
)

func TestCorrectionPreservesOriginalFinding(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	original, err := repo.Add(ctx, findings.NewFinding{Claim: "original claim"}, "agent")
	require.NoError(t, err)
	correction, err := repo.RecordCorrection(ctx, findings.CorrectionInput{
		FindingID: original.ID, Identity: "review-1", OriginalClaimHash: findings.ClaimHash(original.Claim),
		Disposition: "contradicted", Reason: "caller checked the result", EvidenceRefs: []string{"receipt:r1"},
		InvestigationIDs: []string{"run:r1"}, MethodRevisions: []string{"method:v1"},
	}, "operator")
	require.NoError(t, err)
	require.Equal(t, original.ID, correction.FindingID)
	require.Equal(t, findings.ClaimHash(original.Claim), correction.OriginalClaimHash)
	require.Equal(t, "original claim", mustGetFinding(t, repo, original.ID).Claim)
	require.Equal(t, findings.StatusDisputed, mustGetFinding(t, repo, original.ID).Status)
	var auditCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM finding_audit WHERE finding_id = ? AND mutation_type = ?`, original.ID, findings.MutationCorrection).Scan(&auditCount))
	require.Equal(t, 1, auditCount)
}

func TestCorrectionReplayDeduplicatesAndConflictsAreRejected(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	original, err := repo.Add(ctx, findings.NewFinding{Claim: "claim"}, "agent")
	require.NoError(t, err)
	in := findings.CorrectionInput{FindingID: original.ID, Identity: "review-1", OriginalClaimHash: findings.ClaimHash(original.Claim), Disposition: "supported"}
	first, err := repo.RecordCorrection(ctx, in, "operator")
	require.NoError(t, err)
	second, err := repo.RecordCorrection(ctx, in, "operator")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	in.Disposition = "contradicted"
	_, err = repo.RecordCorrection(ctx, in, "operator")
	var invalid findings.ErrInvalidFinding
	require.ErrorAs(t, err, &invalid)
}

func mustGetFinding(t *testing.T, repo findings.Repository, id string) findings.Finding {
	t.Helper()
	f, err := repo.Get(context.Background(), id)
	require.NoError(t, err)
	return f
}
