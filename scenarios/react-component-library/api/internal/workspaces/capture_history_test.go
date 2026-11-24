package workspaces

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func TestCaptureHistoryAndVisualComparisons(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Capture history"})
	require.NoError(t, err)
	stateID := created.Pages[0].Displays[0].States[0].Id

	first, err := repo.AttachCaptureEvidence(ctx, created.Id, stateID, created.Revision, "attempt-1", "token-1", "capture-1", `{ "result": "first" }`, "2026-11-20T10:00:00Z")
	require.NoError(t, err)
	_, err = repo.AttachCaptureEvidence(ctx, created.Id, stateID, first.Revision, "attempt-2", "token-2", "capture-2", `{ "result": "second" }`, "2026-11-20T11:00:00Z")
	require.NoError(t, err)

	receipts, err := repo.ListCaptureReceipts(ctx, created.Id, stateID)
	require.NoError(t, err)
	require.Len(t, receipts, 2)
	require.Equal(t, "capture-2", receipts[0].CaptureID, "receipts are newest first")
	require.Equal(t, "capture-1", receipts[1].CaptureID)
	gotReceipt, err := repo.GetCaptureReceipt(ctx, created.Id, stateID, receipts[1].ID)
	require.NoError(t, err)
	require.Equal(t, receipts[1], gotReceipt)
	_, err = repo.GetCaptureReceipt(ctx, created.Id, stateID, "cr_missing")
	require.ErrorIs(t, err, sql.ErrNoRows)

	comparison, err := repo.CreateVisualComparison(ctx, created.Id, stateID, receipts[1].ID, receipts[0].ID, `{"matches":false}`, "bas://heatmap/1", "2026-11-20T12:00:00Z")
	require.NoError(t, err)
	require.NotEmpty(t, comparison.ID)
	require.Equal(t, receipts[1].ID, comparison.BaselineReceiptID)
	require.Equal(t, receipts[0].ID, comparison.CurrentReceiptID)
	gotComparison, err := repo.GetVisualComparison(ctx, created.Id, stateID, comparison.ID)
	require.NoError(t, err)
	require.Equal(t, comparison, gotComparison)
	comparisons, err := repo.ListVisualComparisons(ctx, created.Id, stateID)
	require.NoError(t, err)
	require.Equal(t, []VisualComparison{comparison}, comparisons)

	_, err = repo.CreateVisualComparison(ctx, created.Id, stateID, receipts[0].ID, receipts[0].ID, `{}`, "", "now")
	require.ErrorIs(t, err, ErrInvalid, "a comparison needs two distinct receipts")
	_, err = repo.CreateVisualComparison(ctx, created.Id, stateID, receipts[0].ID, receipts[1].ID, `not-json`, "", "now")
	require.ErrorIs(t, err, ErrInvalid, "the result must be valid JSON")
	_, err = repo.CreateVisualComparison(ctx, created.Id, stateID, receipts[0].ID, "cr_missing", `{}`, "", "now")
	require.ErrorIs(t, err, sql.ErrNoRows, "both receipt IDs must belong to this workspace and state")
	_, err = repo.GetVisualComparison(ctx, created.Id, stateID, "vc_missing")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
