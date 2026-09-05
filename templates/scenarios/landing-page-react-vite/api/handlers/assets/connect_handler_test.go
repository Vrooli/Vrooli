package assets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	assetsH "landing-page-react-vite-api/handlers/assets"
	internalassets "landing-page-react-vite-api/internal/assets"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{40, 120, 200, 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestAssetsUploadServeAndCRUD(t *testing.T) {
	t.Setenv("UPLOAD_DIR", t.TempDir())

	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, internalassets.Schema)
	_, err := db.Exec(`DELETE FROM assets`)
	require.NoError(t, err)

	svc := internalassets.NewService(db)
	router := mux.NewRouter()
	assetsH.Module(svc, nil).Mount(router)

	// Multipart upload of a logo PNG -> 201 + derivatives generated.
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	require.NoError(t, mw.WriteField("category", "logo"))
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="logo.png"`)
	partHeader.Set("Content-Type", "image/png")
	fw, err := mw.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = fw.Write(pngBytes(t, 400, 400))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var uploaded struct {
		ID          json.Number       `json:"id"`
		StoragePath string            `json:"storage_path"`
		Derivatives map[string]string `json:"derivatives"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &uploaded))
	assetID, err := uploaded.ID.Int64()
	require.NoError(t, err)
	require.NotZero(t, assetID)
	require.Contains(t, uploaded.Derivatives, "logo_256")

	// Static serve of the uploaded file -> 200.
	serveRec := httptest.NewRecorder()
	router.ServeHTTP(serveRec, httptest.NewRequest(http.MethodGet, "/api/v1/uploads/"+uploaded.StoragePath, nil))
	require.Equal(t, http.StatusOK, serveRec.Code)

	// Connect CRUD.
	h := assetsH.NewConnectHandler(assetsH.Deps{Service: svc})
	ctx := context.Background()

	list, err := h.ListAssets(ctx, connect.NewRequest(&landingv1.ListAssetsRequest{Category: "logo"}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Assets, 1)

	got, err := h.GetAsset(ctx, connect.NewRequest(&landingv1.GetAssetRequest{Id: assetID}))
	require.NoError(t, err)
	require.Equal(t, "logo", got.Msg.Asset.Category)

	del, err := h.DeleteAsset(ctx, connect.NewRequest(&landingv1.DeleteAssetRequest{Id: assetID}))
	require.NoError(t, err)
	require.True(t, del.Msg.Deleted)

	_, err = h.GetAsset(ctx, connect.NewRequest(&landingv1.GetAssetRequest{Id: assetID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
