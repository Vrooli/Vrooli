package workspaces

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func pngHeader(width, height uint32) []byte {
	data := append([]byte{}, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}...)
	data = append(data, 0, 0, 0, 13)
	data = append(data, "IHDR"...)
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[0:4], width)
	binary.BigEndian.PutUint32(dimensions[4:8], height)
	return append(data, dimensions[:]...)
}

func newAssetRepo(t *testing.T) (*SQLiteRepository, string) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(sqlDB)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Assets"})
	require.NoError(t, err)
	return repo, created.Id
}

func TestCreateWorkspaceAssetStoresMetadataAndBytes(t *testing.T) {
	repo, workspaceID := newAssetRepo(t)
	ctx := context.Background()

	asset, err := repo.CreateWorkspaceAsset(ctx, AssetInput{
		WorkspaceID: workspaceID,
		Name:        "  hero.png ",
		Mime:        "IMAGE/PNG; charset=binary",
		Data:        pngHeader(1200, 800),
		Fidelity:    AssetFidelityCaptured,
	})
	require.NoError(t, err)
	require.Equal(t, workspaceID, asset.WorkspaceID)
	require.Equal(t, "hero.png", asset.Name)
	require.Equal(t, "image/png", asset.Mime)
	require.Equal(t, int64(len(pngHeader(1200, 800))), asset.Bytes)
	require.Equal(t, 1200, asset.Width)
	require.Equal(t, 800, asset.Height)
	require.Equal(t, AssetFidelityCaptured, asset.Fidelity)
	require.Equal(t, AssetURL(workspaceID, asset.ID), asset.URL)

	listed, err := repo.ListWorkspaceAssets(ctx, workspaceID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, asset.ID, listed[0].ID)

	got, data, err := repo.GetWorkspaceAsset(ctx, workspaceID, asset.ID)
	require.NoError(t, err)
	require.Equal(t, asset.ID, got.ID)
	require.Equal(t, pngHeader(1200, 800), data)
}

func TestWorkspaceAssetIsScopedToItsWorkspace(t *testing.T) {
	repo, workspaceID := newAssetRepo(t)
	ctx := context.Background()
	other, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Other"})
	require.NoError(t, err)
	asset, err := repo.CreateWorkspaceAsset(ctx, AssetInput{
		WorkspaceID: workspaceID,
		Name:        "hero.png",
		Mime:        "image/png",
		Data:        pngHeader(10, 10),
	})
	require.NoError(t, err)

	_, _, err = repo.GetWorkspaceAsset(ctx, other.Id, asset.ID)
	require.ErrorIs(t, err, ErrNotFound, "an asset is only visible through its workspace")

	// Deleting through the wrong workspace is a no-op, not a cross-delete.
	require.NoError(t, repo.DeleteWorkspaceAsset(ctx, other.Id, asset.ID))
	_, _, err = repo.GetWorkspaceAsset(ctx, workspaceID, asset.ID)
	require.NoError(t, err)
}

func TestWorkspaceAssetRejectsInvalidUploads(t *testing.T) {
	repo, workspaceID := newAssetRepo(t)
	ctx := context.Background()

	cases := map[string]AssetInput{
		"empty name":      {WorkspaceID: workspaceID, Mime: "image/png", Data: pngHeader(1, 1)},
		"non-image mime":  {WorkspaceID: workspaceID, Name: "x", Mime: "application/pdf", Data: pngHeader(1, 1)},
		"empty data":      {WorkspaceID: workspaceID, Name: "x", Mime: "image/png"},
		"over size limit": {WorkspaceID: workspaceID, Name: "x", Mime: "image/png", Data: make([]byte, MaxAssetBytes+1)},
		"bad fidelity":    {WorkspaceID: workspaceID, Name: "x", Mime: "image/png", Data: pngHeader(1, 1), Fidelity: "rendered"},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := repo.CreateWorkspaceAsset(ctx, input)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}

	_, err := repo.CreateWorkspaceAsset(ctx, AssetInput{
		WorkspaceID: "ws_missing", Name: "x", Mime: "image/png", Data: pngHeader(1, 1),
	})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeletingAWorkspaceCascadesItsAssets(t *testing.T) {
	repo, workspaceID := newAssetRepo(t)
	ctx := context.Background()
	_, err := repo.CreateWorkspaceAsset(ctx, AssetInput{
		WorkspaceID: workspaceID, Name: "hero.png", Mime: "image/png", Data: pngHeader(4, 4),
	})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, workspaceID))
	var remaining int
	require.NoError(t, repo.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workspace_assets WHERE workspace_id = ?`, workspaceID).Scan(&remaining))
	require.Zero(t, remaining)
}

func TestCuratedWorkspaceRefusesAssetWrites(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, Schema())
	require.NoError(t, err)
	_, err = Seed(ctx, sqlDB, []byte(`{"schemaVersion":1,"seedVersion":1,"workspaces":[{
		"id":"ws_curated_x","name":"Template","provenance":"WORKSPACE_PROVENANCE_CURATED",
		"revision":1,"createdAt":"2026-11-20T00:00:00.000Z","updatedAt":"2026-11-20T00:00:00.000Z",
		"pages":[{"id":"pg_x","name":"Welcome","specJson":"{}","displays":[{"id":"dp_x",
			"device":"DISPLAY_DEVICE_DESKTOP","states":[{"id":"st_x","name":"Default","argsJson":"{}"}]}]}]
	}]}`))
	require.NoError(t, err)

	repo := NewSQLiteRepository(sqlDB)
	_, err = repo.CreateWorkspaceAsset(ctx, AssetInput{
		WorkspaceID: "ws_curated_x", Name: "x.png", Mime: "image/png", Data: pngHeader(1, 1),
	})
	require.ErrorIs(t, err, ErrCurated)
	require.ErrorIs(t, repo.DeleteWorkspaceAsset(ctx, "ws_curated_x", "asset-x"), ErrCurated)
}
