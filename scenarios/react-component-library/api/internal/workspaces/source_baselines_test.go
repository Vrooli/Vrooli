package workspaces

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func TestSourceBaselineRequiresExplicitAcceptanceAndComparesAgainstReceipt(t *testing.T) {
	ctx := context.Background()
	repo, workspaceID := newAssetRepo(t)
	workspace, err := repo.Get(ctx, workspaceID)
	require.NoError(t, err)
	state := workspace.GetPages()[0].GetDisplays()[0].GetStates()[0]
	data := pngHeader(12, 8)
	asset, err := repo.CreateWorkspaceAsset(ctx, AssetInput{WorkspaceID: workspaceID, Name: "source.png", Mime: "image/png", Data: data})
	require.NoError(t, err)
	digest := sha256.Sum256(data)
	candidate, err := repo.CreateSourceBaseline(ctx, workspaceID, state.GetId(), asset.ID, hex.EncodeToString(digest[:]), "https://example.test/design", "2026-11-22T12:00:00Z")
	require.NoError(t, err)
	require.Empty(t, candidate.AcceptedAt)
	listed, err := repo.ListSourceBaselines(ctx, workspaceID, state.GetId())
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Empty(t, listed[0].AcceptedBy)
	_, err = repo.AcceptSourceBaseline(ctx, workspaceID, state.GetId(), candidate.ID, "", "2026-11-22T12:01:00Z")
	require.ErrorIs(t, err, ErrInvalid)
	accepted, err := repo.AcceptSourceBaseline(ctx, workspaceID, state.GetId(), candidate.ID, "designer@example.test", "2026-11-22T12:02:00Z")
	require.NoError(t, err)
	require.Equal(t, "designer@example.test", accepted.AcceptedBy)
	require.NotEmpty(t, accepted.AcceptedAt)
	accepted, err = repo.GetAcceptedSourceBaseline(ctx, workspaceID, state.GetId())
	require.NoError(t, err)
	require.Equal(t, candidate.ID, accepted.ID)
	err = repo.DeleteWorkspaceAsset(ctx, workspaceID, asset.ID)
	require.ErrorIs(t, err, ErrSourceBaselineAsset, "accepted source bytes must remain available for drift history")

	_, err = repo.AttachCaptureEvidence(ctx, workspaceID, state.GetId(), workspace.GetRevision(), "attempt-1", "token-1", "capture-1", `{"schemaVersion":"rcl-workspace-state-capture/v1"}`, "2026-11-22T12:03:00Z")
	require.NoError(t, err)
	receiptsList, err := repo.ListCaptureReceipts(ctx, workspaceID, state.GetId())
	require.NoError(t, err)
	require.Len(t, receiptsList, 1)
	comparison, err := repo.CreateSourceVisualComparison(ctx, SourceVisualComparison{WorkspaceID: workspaceID, StateID: state.GetId(), SourceBaselineID: candidate.ID, SourceAssetID: asset.ID, CurrentReceiptID: receiptsList[0].ID, ResultJSON: `{"verdict":"changed"}`, HeatmapRef: "diff/11111111-1111-4111-8111-111111111111.png", CreatedAt: "2026-11-22T12:04:00Z"})
	require.NoError(t, err)
	comparisons, err := repo.ListSourceVisualComparisons(ctx, workspaceID, state.GetId())
	require.NoError(t, err)
	require.Len(t, comparisons, 1)
	require.Equal(t, comparison.ID, comparisons[0].ID)
	require.Equal(t, candidate.ID, comparisons[0].SourceBaselineID)
	require.Equal(t, asset.ID, comparisons[0].SourceAssetID)
	require.NoError(t, Migrate(ctx, repo.db), "re-applying schema keeps source baseline records and comparisons")
	acceptedAfterMigration, err := repo.GetAcceptedSourceBaseline(ctx, workspaceID, state.GetId())
	require.NoError(t, err)
	require.Equal(t, candidate.ID, acceptedAfterMigration.ID)
	require.NoError(t, repo.Delete(ctx, workspaceID), "workspace deletion cascades source anchor records and assets together")
}

func TestSourceBaselineRejectsWrongWorkspaceAndDigest(t *testing.T) {
	ctx := context.Background()
	repo, workspaceID := newAssetRepo(t)
	workspace, err := repo.Get(ctx, workspaceID)
	require.NoError(t, err)
	stateID := workspace.GetPages()[0].GetDisplays()[0].GetStates()[0].GetId()
	asset, err := repo.CreateWorkspaceAsset(ctx, AssetInput{WorkspaceID: workspaceID, Name: "source.png", Mime: "image/png", Data: pngHeader(4, 4)})
	require.NoError(t, err)
	_, err = repo.CreateSourceBaseline(ctx, workspaceID, stateID, asset.ID, "00", "", "now")
	require.ErrorIs(t, err, ErrInvalid)
	_, err = repo.CreateSourceBaseline(ctx, workspaceID, "state_missing", asset.ID, "0000000000000000000000000000000000000000000000000000000000000000", "", "now")
	require.Error(t, err)
	other, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Other workspace"})
	require.NoError(t, err)
	_, err = repo.CreateSourceBaseline(ctx, other.GetId(), stateID, asset.ID, "0000000000000000000000000000000000000000000000000000000000000000", "", "now")
	require.ErrorIs(t, err, ErrNotFound)
}
