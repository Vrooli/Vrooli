package workspaces_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"

	"react-component-library/handlers/workspaces"
	domain "react-component-library/internal/workspaces"
)

func pngBytes(width, height uint32) []byte {
	data := append([]byte{}, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}...)
	data = append(data, 0, 0, 0, 13)
	data = append(data, "IHDR"...)
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[0:4], width)
	binary.BigEndian.PutUint32(dimensions[4:8], height)
	return append(data, dimensions[:]...)
}

func setupAssetServer(t *testing.T) (*httptest.Server, *domain.SQLiteRepository, string) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, workspaces.Schema())
	require.NoError(t, err)
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Asset host"})
	require.NoError(t, err)

	m := workspaces.ModuleFromRepository(repo, log.New(io.Discard, "", 0))
	router := mux.NewRouter()
	m.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, repo, created.GetId()
}

func uploadAsset(t *testing.T, server *httptest.Server, workspaceID, body string) (*http.Response, map[string]any) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost,
		server.URL+"/api/workspaces/"+url.PathEscape(workspaceID)+"/assets",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	payload := map[string]any{}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	return response, payload
}

func TestWorkspaceAssetRoutesRoundTrip(t *testing.T) {
	server, _, workspaceID := setupAssetServer(t)
	encoded := base64.StdEncoding.EncodeToString(pngBytes(640, 480))

	response, payload := uploadAsset(t, server, workspaceID, `{
		"name": "hero.png", "mime": "image/png", "dataBase64": "`+encoded+`", "fidelity": "captured"
	}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	asset, ok := payload["asset"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "hero.png", asset["name"])
	require.EqualValues(t, 640, asset["width"])
	require.EqualValues(t, 480, asset["height"])
	require.Equal(t, "captured", asset["fidelity"])
	assetID, _ := asset["id"].(string)
	require.NotEmpty(t, assetID)

	// The list reports the same asset.
	listResponse, err := server.Client().Get(server.URL + "/api/workspaces/" + url.PathEscape(workspaceID) + "/assets")
	require.NoError(t, err)
	defer listResponse.Body.Close()
	require.Equal(t, http.StatusOK, listResponse.StatusCode)
	var listed struct {
		Assets []domain.Asset `json:"assets"`
	}
	require.NoError(t, json.NewDecoder(listResponse.Body).Decode(&listed))
	require.Len(t, listed.Assets, 1)
	require.Equal(t, assetID, listed.Assets[0].ID)
	require.Equal(t, domain.AssetURL(workspaceID, assetID), listed.Assets[0].URL)

	// Bytes come back with their content type and an ETag that 304s.
	bytesResponse, err := server.Client().Get(server.URL + domain.AssetURL(workspaceID, assetID))
	require.NoError(t, err)
	defer bytesResponse.Body.Close()
	require.Equal(t, http.StatusOK, bytesResponse.StatusCode)
	require.Equal(t, "image/png", bytesResponse.Header.Get("Content-Type"))
	require.Equal(t, assetID, bytesResponse.Header.Get("ETag"))
	body, err := io.ReadAll(bytesResponse.Body)
	require.NoError(t, err)
	require.Equal(t, pngBytes(640, 480), body)

	request, err := http.NewRequest(http.MethodGet, server.URL+domain.AssetURL(workspaceID, assetID), nil)
	require.NoError(t, err)
	request.Header.Set("If-None-Match", `W/"`+assetID+`"`)
	notModified, err := server.Client().Do(request)
	require.NoError(t, err)
	defer notModified.Body.Close()
	require.Equal(t, http.StatusNotModified, notModified.StatusCode)

	// Delete is scoped and idempotent for the owning workspace.
	deleteRequest, err := http.NewRequest(http.MethodDelete,
		server.URL+domain.AssetURL(workspaceID, assetID), nil)
	require.NoError(t, err)
	deleted, err := server.Client().Do(deleteRequest)
	require.NoError(t, err)
	defer deleted.Body.Close()
	require.Equal(t, http.StatusNoContent, deleted.StatusCode)

	missing, err := server.Client().Get(server.URL + domain.AssetURL(workspaceID, assetID))
	require.NoError(t, err)
	defer missing.Body.Close()
	require.Equal(t, http.StatusNotFound, missing.StatusCode)
}

func TestWorkspaceAssetRoutesRejectInvalidUploadsInTheErrorEnvelope(t *testing.T) {
	server, _, workspaceID := setupAssetServer(t)
	encoded := base64.StdEncoding.EncodeToString([]byte("not an image"))

	cases := map[string]struct {
		body   string
		status int
	}{
		"bad mime":      {body: `{"name":"x","mime":"application/pdf","dataBase64":"` + encoded + `"}`, status: http.StatusBadRequest},
		"missing name":  {body: `{"mime":"image/png","dataBase64":"` + encoded + `"}`, status: http.StatusBadRequest},
		"empty data":    {body: `{"name":"x","mime":"image/png","dataBase64":""}`, status: http.StatusBadRequest},
		"bad fidelity":  {body: `{"name":"x","mime":"image/png","dataBase64":"` + encoded + `","fidelity":"rendered"}`, status: http.StatusBadRequest},
		"not json body": {body: `nope`, status: http.StatusBadRequest},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			response, payload := uploadAsset(t, server, workspaceID, tc.body)
			require.Equal(t, tc.status, response.StatusCode)
			envelope, ok := payload["error"].(map[string]any)
			require.True(t, ok, "failures use the canonical envelope")
			require.Equal(t, "invalid_argument", envelope["code"])
			require.NotEmpty(t, envelope["message"])
		})
	}

	response, payload := uploadAsset(t, server, "ws_missing",
		`{"name":"x","mime":"image/png","dataBase64":"`+encoded+`"}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	envelope, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "not_found", envelope["code"])
}
