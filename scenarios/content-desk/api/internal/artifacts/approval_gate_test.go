package artifacts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/provenance"
)

func TestApprovalGateNamesEveryBlockingReason(t *testing.T) {
	verdict := EvaluateApproval(context.Background(), ApprovalInput{DraftStatus: DraftDrafted, UnverifiedClaimIDs: []string{"claim-1"}, PostTypeActive: false, ReviewPassed: false})
	require.False(t, verdict.Allowed)
	require.Equal(t, []string{"draft is not ready for approval", "claim claim-1 is not verified", "post type is inactive", "review has blocking verdicts"}, verdict.Blockers)
}

func TestApprovalGateAllowsReviewedOperatorWithPassingGates(t *testing.T) {
	verdict := EvaluateApproval(context.Background(), ApprovalInput{DraftStatus: DraftReviewed, PostTypeActive: true, ReviewPassed: true})
	require.True(t, verdict.Allowed)
	require.Empty(t, verdict.Blockers)
}

func TestApprovalGateRejectsVerifiedAgent(t *testing.T) {
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	verdict := EvaluateApproval(ctx, ApprovalInput{DraftStatus: DraftReviewed, PostTypeActive: true, ReviewPassed: true})
	require.False(t, verdict.Allowed)
	require.Equal(t, []string{"agents cannot approve drafts"}, verdict.Blockers)
}
