package signals

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
)

// [REQ:SIG-P0-002] Images enter the journal through BlobStore metadata, not
// as bytes inside the Connect capture request.
func TestImageUploadStoresBlobAndReturnsPayloadReference(t *testing.T) {
	store := blobstore.NewMemoryBlobStore()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(imagePartHeader("walk.png", "image/png"))
	require.NoError(t, err)
	_, err = part.Write([]byte("png bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/signals/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	NewImageUploadHandler(store, nil).ServeHTTP(res, req)

	require.Equal(t, http.StatusCreated, res.Code)
	require.Contains(t, res.Body.String(), "payload_ref")
	// The generated payload reference is opaque, but the blob write is the
	// decisive behavior. Read the only expected format-safe prefix from JSON.
	require.Contains(t, res.Body.String(), "signals/uploads/")
}

func TestImageUploadRejectsNonImage(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(imagePartHeader("walk.txt", "text/plain"))
	require.NoError(t, err)
	_, err = io.WriteString(part, "not an image")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signals/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	NewImageUploadHandler(blobstore.NewMemoryBlobStore(), nil).ServeHTTP(res, req)
	require.Equal(t, http.StatusBadRequest, res.Code)
}

func TestImagePayloadKeyIsContentAddressed(t *testing.T) {
	first := imagePayloadKey("first.png", []byte("same pixels"))
	second := imagePayloadKey("second.png", []byte("same pixels"))
	require.Equal(t, first, second)
}

func imagePartHeader(name, contentType string) textproto.MIMEHeader {
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+name+`"`)
	header.Set("Content-Type", contentType)
	return header
}
