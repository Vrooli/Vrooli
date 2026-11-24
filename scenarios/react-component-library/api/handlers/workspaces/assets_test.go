package workspaces_test

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"
	"react-component-library/handlers/workspaces"
	domain "react-component-library/internal/workspaces"
)

func pngBytes(width, height uint32) []byte {
	data := append([]byte{}, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}...)
	data = append(data, 0, 0, 0, 13)
	data = append(data, "IHDR"...)
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[:4], width)
	binary.BigEndian.PutUint32(dimensions[4:], height)
	return append(data, dimensions[:]...)
}

func setupAssetServer(t *testing.T) (*httptest.Server, workspaceconnect.WorkspaceServiceClient, string) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), workspaces.Schema())
	require.NoError(t, err)
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Asset host"})
	require.NoError(t, err)
	router := mux.NewRouter()
	workspaces.ModuleFromRepository(repo, log.New(io.Discard, "", 0)).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL), created.GetId()
}

func TestWorkspaceAssetConnectMetadataAndBinaryBytes(t *testing.T) {
	server, client, workspaceID := setupAssetServer(t)
	data := pngBytes(640, 480)
	created, err := client.CreateWorkspaceAsset(context.Background(), connect.NewRequest(&workspacev1.CreateWorkspaceAssetRequest{WorkspaceId: workspaceID, Name: "hero.png", Mime: "image/png", Data: data, Fidelity: "captured"}))
	require.NoError(t, err)
	asset := created.Msg.GetAsset()
	require.NotNil(t, asset)
	require.Equal(t, "hero.png", asset.Name)
	require.Equal(t, int32(640), asset.Width)
	require.Equal(t, int32(480), asset.Height)
	listed, err := client.ListWorkspaceAssets(context.Background(), connect.NewRequest(&workspacev1.ListWorkspaceAssetsRequest{WorkspaceId: workspaceID}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Assets, 1)
	require.Equal(t, asset.Id, listed.Msg.Assets[0].Id)
	all, err := client.ListAllWorkspaceAssets(context.Background(), connect.NewRequest(&workspacev1.ListAllWorkspaceAssetsRequest{}))
	require.NoError(t, err)
	require.Len(t, all.Msg.Assets, 1)
	response, err := server.Client().Get(server.URL + asset.Url)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "image/png", response.Header.Get("Content-Type"))
	require.Equal(t, asset.Id, response.Header.Get("ETag"))
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, data, body)
	deleted, err := client.DeleteWorkspaceAsset(context.Background(), connect.NewRequest(&workspacev1.DeleteWorkspaceAssetRequest{WorkspaceId: workspaceID, AssetId: asset.Id}))
	require.NoError(t, err)
	require.True(t, deleted.Msg.Deleted)
}
