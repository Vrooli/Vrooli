package claims_test

import (
	"context"
	"testing"
	"time"

	"content-desk/internal/claims"
	claimsmocks "content-desk/internal/claims/mocks"
	"content-desk/internal/testutil/db"

	localdb "content-desk/internal/database"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func newLibrary(t *testing.T, runner claims.Runner) claims.Library {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(claims.Schema)))
	return claims.NewLibrary(d, runner)
}

// [REQ:CONTENTD-P0-005]
func TestCheckRequiredClaimRejectsCitationOnlyEvidence(t *testing.T) {
	t.Run("[CONTENTD-P0-005] check-required evidence rejects citation only", func(t *testing.T) {
		library := newLibrary(t, &claimsmocks.FakeRunner{})
		_, err := library.Create(context.Background(), claims.Claim{Statement: "There are 10 users", Kind: claims.KindQuantitative}, claims.Evidence{Kind: claims.EvidenceKindCitation, Reference: "https://example.test"})
		require.ErrorIs(t, err, claims.ErrCheckRequired)
	})
}

// [REQ:CONTENTD-P0-004]
func TestSharedClaimCitationsPreserveAnchorsAndReuse(t *testing.T) {
	t.Run("[CONTENTD-P0-004] claims are reusable with anchored citations", func(t *testing.T) {
		library := newLibrary(t, &claimsmocks.FakeRunner{})
		claim, err := library.Create(context.Background(), claims.Claim{Statement: "Vrooli is open source", Kind: claims.KindCapability}, claims.Evidence{Kind: claims.EvidenceKindCitation, Reference: "README.md"})
		require.NoError(t, err)
		body := "Vrooli is open source."
		require.NoError(t, library.Cite(context.Background(), claims.Citation{DraftID: "draft-a", ClaimID: claim.ID, Start: 0, End: len(body)}, body))
		require.NoError(t, library.Cite(context.Background(), claims.Citation{DraftID: "draft-b", ClaimID: claim.ID, Start: 0, End: len(body)}, body))
		require.ErrorIs(t, library.Cite(context.Background(), claims.Citation{DraftID: "draft-c", ClaimID: claim.ID, Start: 0, End: len(body) + 1}, body), claims.ErrInvalidAnchor)
		drafts, err := library.CitingDrafts(context.Background(), claim.ID)
		require.NoError(t, err)
		require.Equal(t, []string{"draft-a", "draft-b"}, drafts)
		all, err := library.List(context.Background())
		require.NoError(t, err)
		require.Len(t, all, 1)
	})
}

func TestListForDraftReturnsOnlyCitedClaims(t *testing.T) {
	library := newLibrary(t, &claimsmocks.FakeRunner{})
	cited, err := library.Create(context.Background(), claims.Claim{Statement: "Cited", Kind: claims.KindCapability}, claims.Evidence{Kind: claims.EvidenceKindCitation, Reference: "canon"})
	require.NoError(t, err)
	_, err = library.Create(context.Background(), claims.Claim{Statement: "Unrelated", Kind: claims.KindCapability}, claims.Evidence{Kind: claims.EvidenceKindCitation, Reference: "canon"})
	require.NoError(t, err)
	require.NoError(t, library.Cite(context.Background(), claims.Citation{DraftID: "draft-a", ClaimID: cited.ID, Start: 0, End: 5}, "Cited"))
	forDraft, err := library.ListForDraft(context.Background(), "draft-a")
	require.NoError(t, err)
	require.Equal(t, []claims.Claim{cited}, forDraft)
}

func TestVerificationStoresResultAndMovesClaimState(t *testing.T) {
	fake := &claimsmocks.FakeRunner{Result: claims.CheckResult{ActualResult: "42", Matches: true}}
	library := newLibrary(t, fake)
	claim, err := library.Create(context.Background(), claims.Claim{Statement: "There are 42 tests", Kind: claims.KindQuantitative}, claims.Evidence{Kind: claims.EvidenceKindCheck, Command: "ignored", ExpectedResult: "42"})
	require.NoError(t, err)
	verified, err := library.Verify(context.Background(), claim.ID)
	require.NoError(t, err)
	require.Equal(t, claims.StateVerified, verified.VerificationStatus)
	fake.Result = claims.CheckResult{ActualResult: "41", Matches: false}
	stale, err := library.Verify(context.Background(), claim.ID)
	require.NoError(t, err)
	require.Equal(t, claims.StateStale, stale.VerificationStatus)
}

// [REQ:CONTENTD-P1-001]
func TestSweepReverifiesEveryCheckBackedClaim(t *testing.T) {
	t.Run("[CONTENTD-P1-001] sweep reverifies check-backed claims", func(t *testing.T) {
		fake := &claimsmocks.FakeRunner{Result: claims.CheckResult{ActualResult: "changed", Matches: false}}
		library := newLibrary(t, fake)
		for _, statement := range []string{"Metric changed", "Status changed"} {
			_, err := library.Create(context.Background(), claims.Claim{Statement: statement, Kind: claims.KindQuantitative}, claims.Evidence{Kind: claims.EvidenceKindCheck, Command: "ignored", ExpectedResult: "expected"})
			require.NoError(t, err)
		}
		updated, err := library.Sweep(context.Background())
		require.NoError(t, err)
		require.Len(t, updated, 2)
		for _, claim := range updated {
			require.Equal(t, claims.StateStale, claim.VerificationStatus)
		}
	})
}

// [REQ:CONTENTD-P1-003]
func TestNoveltyEvidenceExpiresToAssertedAfterConfiguredAge(t *testing.T) {
	t.Run("[CONTENTD-P1-003] novelty evidence expires", func(t *testing.T) {
		library := newLibrary(t, &claimsmocks.FakeRunner{})
		now := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
		claim, err := library.Create(context.Background(), claims.Claim{Statement: "First of its kind", Kind: claims.KindNovelty, VerificationStatus: claims.StateVerified}, claims.Evidence{Kind: claims.EvidenceKindCitation, Reference: "prior-art-search", ObservedAt: now.Add(-31 * 24 * time.Hour)})
		require.NoError(t, err)
		expired, err := library.ExpireNovelty(context.Background(), now, 30*24*time.Hour)
		require.NoError(t, err)
		require.Equal(t, []claims.Claim{{ID: claim.ID, Statement: claim.Statement, Kind: claims.KindNovelty, VerificationStatus: claims.StateAsserted}}, expired)

		_, err = library.Create(context.Background(), claims.Claim{Statement: "Undated novelty", Kind: claims.KindNovelty}, claims.Evidence{Kind: claims.EvidenceKindCitation, Reference: "prior-art-search"})
		require.Error(t, err)
	})
}
