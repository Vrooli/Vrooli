package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/devices"
	"device-sync-hub/internal/testutil/db"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	localdb "device-sync-hub/internal/database"
)

func newREST(t *testing.T, cfg internaltransfer.Config) (*uploadHandler, *downloadHandler, internaltransfer.Service) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaltransfer.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	store := blobstore.NewMemoryBlobStore()
	cfg.Repo = internaltransfer.NewSQLiteRepository(d, clk)
	cfg.Blobs = store
	cfg.Clock = clk
	svc := internaltransfer.NewService(cfg)
	up := newUploadHandler(UploadDeps{Service: svc, Store: store, DB: d})
	down := newDownloadHandler(DownloadDeps{Service: svc, Store: store})
	return up, down, svc
}

func deviceReq(t *testing.T, method, target string, body io.Reader, contentType string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx := deviceauth.WithDevice(req.Context(), devices.Device{ID: "dev-a", OwnerID: "owner-1", TrustState: devices.TrustTrusted})
	return req.WithContext(ctx)
}

func multipartFile(t *testing.T, field, filename, contentType string, data []byte, extra map[string]string) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range extra {
		require.NoError(t, w.WriteField(k, v))
	}
	hdr := make(map[string][]string)
	if contentType != "" {
		hdr["Content-Type"] = []string{contentType}
	}
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + field + `"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	})
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func TestUploadRequiresDeviceToken(t *testing.T) {
	up, _, _ := newREST(t, internaltransfer.Config{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfer/items", nil) // no device ctx
	rr := httptest.NewRecorder()
	up.handleUpload(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestUploadDownloadRoundTrip(t *testing.T) {
	up, down, svc := newREST(t, internaltransfer.Config{})
	payload := []byte("the quick brown fox")
	body, ct := multipartFile(t, "file", "fox.txt", "text/plain", payload, map[string]string{"retention": "pinned"})

	req := deviceReq(t, http.MethodPost, "/api/v1/transfer/items", body, ct)
	rr := httptest.NewRecorder()
	up.handleUpload(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	// Find the created item to learn its id.
	list, err := svc.List(context.Background(), "owner-1", "dev-a", internaltransfer.ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	item := list[0]
	assert.Equal(t, internaltransfer.KindFile, item.Kind)
	assert.Equal(t, "fox.txt", item.Name)
	assert.Equal(t, int64(len(payload)), item.SizeBytes)

	// Download streams the original bytes with a filename disposition.
	dreq := deviceReq(t, http.MethodGet, "/api/v1/transfer/items/"+item.ID+"/content", nil, "")
	dreq = mux.SetURLVars(dreq, map[string]string{"id": item.ID})
	drr := httptest.NewRecorder()
	down.handleDownload(drr, dreq)
	require.Equal(t, http.StatusOK, drr.Code)
	assert.Equal(t, payload, drr.Body.Bytes())
	assert.Contains(t, drr.Header().Get("Content-Disposition"), `filename="fox.txt"`)
}

func TestUploadGeneratesThumbnailForImage(t *testing.T) {
	up, down, svc := newREST(t, internaltransfer.Config{})

	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, img))

	body, ct := multipartFile(t, "file", "pic.png", "image/png", pngBuf.Bytes(), nil)
	req := deviceReq(t, http.MethodPost, "/api/v1/transfer/items", body, ct)
	rr := httptest.NewRecorder()
	up.handleUpload(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	list, err := svc.List(context.Background(), "owner-1", "dev-a", internaltransfer.ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].HasThumbnail(), "image upload generates a thumbnail")

	// ?thumb=1 streams the JPEG thumbnail.
	dreq := deviceReq(t, http.MethodGet, "/api/v1/transfer/items/"+list[0].ID+"/content?thumb=1", nil, "")
	dreq = mux.SetURLVars(dreq, map[string]string{"id": list[0].ID})
	drr := httptest.NewRecorder()
	down.handleDownload(drr, dreq)
	require.Equal(t, http.StatusOK, drr.Code)
	assert.Equal(t, "image/jpeg", drr.Header().Get("Content-Type"))
	_, _, err = image.Decode(bytes.NewReader(drr.Body.Bytes()))
	require.NoError(t, err, "thumbnail is a decodable image")
}

func TestUploadRejectsOverQuota(t *testing.T) {
	up, _, _ := newREST(t, internaltransfer.Config{OwnerQuotaBytes: 4})
	body, ct := multipartFile(t, "file", "big.bin", "application/octet-stream", []byte("way more than four bytes"), nil)
	req := deviceReq(t, http.MethodPost, "/api/v1/transfer/items", body, ct)
	rr := httptest.NewRecorder()
	up.handleUpload(rr, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

func TestResumableUploadAssemblesChunksIntoOneItem(t *testing.T) {
	up, down, svc := newREST(t, internaltransfer.Config{})
	create := deviceReq(t, http.MethodPost, "/api/v1/transfer/uploads", strings.NewReader(`{"name":"archive.zip","mime":"application/zip","size_bytes":6,"retention":"held"}`), "application/json")
	created := httptest.NewRecorder()
	up.handleCreateSession(created, create)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	var session uploadSession
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &session))
	for _, chunk := range []string{"abc", "def"} {
		req := deviceReq(t, http.MethodPut, "/api/v1/transfer/uploads/"+session.ID+"/chunks/0", strings.NewReader(chunk), "application/octet-stream")
		req = mux.SetURLVars(req, map[string]string{"id": session.ID, "index": "0"})
		rr := httptest.NewRecorder()
		up.handleChunk(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code, "short chunk must not be accepted")
	}
	// A one-chunk session has exactly six bytes; upload the correct chunk.
	req := deviceReq(t, http.MethodPut, "/api/v1/transfer/uploads/"+session.ID+"/chunks/0", strings.NewReader("abcdef"), "application/octet-stream")
	req = mux.SetURLVars(req, map[string]string{"id": session.ID, "index": "0"})
	rr := httptest.NewRecorder()
	up.handleChunk(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	complete := deviceReq(t, http.MethodPost, "/api/v1/transfer/uploads/"+session.ID+"/complete", nil, "")
	complete = mux.SetURLVars(complete, map[string]string{"id": session.ID})
	done := httptest.NewRecorder()
	up.handleCompleteSession(done, complete)
	require.Equal(t, http.StatusCreated, done.Code, done.Body.String())
	items, err := svc.List(context.Background(), "owner-1", "dev-a", internaltransfer.ListFilter{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	dreq := deviceReq(t, http.MethodGet, "/api/v1/transfer/items/"+items[0].ID+"/content", nil, "")
	dreq = mux.SetURLVars(dreq, map[string]string{"id": items[0].ID})
	got := httptest.NewRecorder()
	down.handleDownload(got, dreq)
	require.Equal(t, http.StatusOK, got.Code)
	assert.Equal(t, []byte("abcdef"), got.Body.Bytes())
}
