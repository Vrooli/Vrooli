package intake_test

import (
	"context"
	"database/sql"
	"testing"

	"content-desk/internal/artifacts"
	"content-desk/internal/campaigns"
	"content-desk/internal/intake"
	"content-desk/internal/ledger"
	"content-desk/internal/posttypes"
	"content-desk/internal/projects"
	"content-desk/internal/review"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
)

func promotionSetup(t *testing.T) (*sql.DB, intake.Service, artifacts.Repository) {
	t.Helper()
	db := databasetest.NewSQLite(t)
	ctx := context.Background()
	for _, schema := range []string{intake.Schema(), artifacts.Schema(), campaigns.Schema(), ledger.Schema(), review.Schema(), posttypes.Schema(), projects.Schema()} {
		_, err := db.ExecContext(ctx, schema)
		require.NoError(t, err)
	}
	repo := artifacts.NewSQLiteRepository(db)
	return db, intake.NewServiceWithDrafts(db, repo), repo
}

func TestPromoteExactAdmittedItemVariantsWithReceiptLineageWithoutAuthority(t *testing.T) {
	ctx := context.Background()
	db, service, _ := promotionSetup(t)
	defer db.Close()
	item := good("article-7")
	item.Body = "Original synthetic long article."
	item.Variants = []intake.Variant{
		{ID: "li-short", Name: "LinkedIn short", Channel: "linkedin", Body: "Synthetic LinkedIn body."},
		{ID: "x-thread", Name: "X thread", Channel: "x-twitter", Body: "Synthetic X body."},
	}
	admitted, err := service.Admit(ctx, intake.Request{Source: source(), IdempotencyKey: "synthetic-promote", Items: []intake.Item{item, good("other-item")}})
	require.NoError(t, err)
	require.Len(t, admitted, 2)
	receipt := admitted[0].Receipt.ID
	result, err := service.Promote(ctx, intake.PromoteRequest{
		BatchID: admitted[0].BatchID, ExternalID: "article-7", ExpectedReceiptID: receipt,
		VariantIDs: []string{"x-thread", "li-short"}, Format: "post", PostTypeID: "dev-log",
	})
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "li-short", result[0].Source.VariantID)
	require.Equal(t, "LinkedIn short", result[0].Source.VariantName)
	require.Equal(t, receipt, result[0].Source.ReceiptID)
	require.Equal(t, "fixture-doc-1", result[0].Source.SourceArtifactID)
	require.Equal(t, "rev-2", result[0].Source.SourceRevisionID)
	require.NotEmpty(t, result[0].Source.SourcePayloadHash)
	require.Equal(t, artifacts.DraftRequested, result[0].Draft.Status)
	require.Equal(t, "linkedin", result[0].Draft.Channel)
	require.Equal(t, "Synthetic LinkedIn body.", result[0].Draft.Body)
	require.Equal(t, "x-twitter", result[1].Draft.Channel)
	require.Equal(t, artifacts.DraftRequested, result[1].Draft.Status)
	// Exact retry returns the same draft records and leaves the admitted bytes intact.
	retry, err := service.Promote(ctx, intake.PromoteRequest{BatchID: admitted[0].BatchID, ExternalID: "article-7", ExpectedReceiptID: receipt, VariantIDs: []string{"li-short", "x-thread"}, Format: "post", PostTypeID: "dev-log"})
	require.NoError(t, err)
	require.Equal(t, result, retry)
	stored, err := service.List(ctx, admitted[0].BatchID)
	require.NoError(t, err)
	require.Equal(t, "Original synthetic long article.", stored[0].Item.Body)
	require.Equal(t, receipt, stored[0].Receipt.ID)
	var approvalCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM draft_approvals`).Scan(&approvalCount))
	require.Zero(t, approvalCount)
	var eventCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM draft_events`).Scan(&eventCount))
	require.Zero(t, eventCount)
	var promotedCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM article_intake_promotions`).Scan(&promotedCount))
	require.Equal(t, 2, promotedCount)
}

func TestPromoteRejectsWrongReceiptUnadmittedVariantAndNonAdmissibleItem(t *testing.T) {
	ctx := context.Background()
	db, service, drafts := promotionSetup(t)
	defer db.Close()
	bad := good("rejected")
	bad.RejectedFields = []string{"synthetic-risk"}
	base := good("base")
	base.Variants = []intake.Variant{{ID: "known", Name: "Known synthetic", Channel: "x-twitter", Body: "A synthetic variant."}}
	items, err := service.Admit(ctx, intake.Request{Source: source(), IdempotencyKey: "promotion-refusals", Items: []intake.Item{base, bad}})
	require.NoError(t, err)
	request := intake.PromoteRequest{BatchID: items[0].BatchID, ExternalID: "base", ExpectedReceiptID: "wrong", Format: "post", PostTypeID: "dev-log"}
	_, err = service.Promote(ctx, request)
	require.ErrorContains(t, err, "does not match")
	request.ExpectedReceiptID = items[0].Receipt.ID
	request.VariantIDs = []string{"invented"}
	request.Channel = ""
	_, err = service.Promote(ctx, request)
	require.ErrorContains(t, err, "not named")
	request = intake.PromoteRequest{BatchID: items[1].BatchID, ExternalID: "rejected", ExpectedReceiptID: items[1].Receipt.ID, Channel: "x-twitter", Format: "post", PostTypeID: "dev-log"}
	_, err = service.Promote(ctx, request)
	require.ErrorContains(t, err, "cannot be promoted")
	draftList, err := drafts.List(ctx)
	require.NoError(t, err)
	require.Empty(t, draftList)
}
