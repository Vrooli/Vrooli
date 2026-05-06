package notes_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	"{{SCENARIO_ID}}/handlers/notes"
	"{{SCENARIO_ID}}/internal/clock"
	localdb "{{SCENARIO_ID}}/internal/database"
	internalnotes "{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/testutil/assertx"
	"{{SCENARIO_ID}}/internal/testutil/db"
)

func TestAttachmentsHandlerUploadSuccess(t *testing.T) {
	router, store, noteID := newAttachmentsRouter(t)
	body, contentType := multipartBody(t, "file", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID+"/attachments", body)
	req.Header.Set("Content-Type", contentType)
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	require.Equal(t, http.StatusCreated, rw.Code, "body=%s", rw.Body.String())
	got := assertx.MustUnmarshalProto[notesv1.UploadAttachmentResponse](t, rw.Body.Bytes())
	require.NotNil(t, got.Attachment)
	require.Equal(t, noteID, got.Attachment.NoteId)
	require.Equal(t, "application/octet-stream", got.Attachment.MimeType)

	rc, _, err := store.Get(context.Background(), got.Attachment.Key)
	require.NoError(t, err)
	defer rc.Close()
	stored, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "hello", string(stored))
}

func TestAttachmentsHandlerMissingFile(t *testing.T) {
	router, _, noteID := newAttachmentsRouter(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/"+noteID+"/attachments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	require.Equal(t, http.StatusBadRequest, rw.Code)
	require.Contains(t, rw.Body.String(), "file field is required")
}

func TestAttachmentsHandlerUnknownNote(t *testing.T) {
	router, _, _ := newAttachmentsRouter(t)
	body, contentType := multipartBody(t, "file", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/missing/attachments", body)
	req.Header.Set("Content-Type", contentType)
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	require.Equal(t, http.StatusNotFound, rw.Code, "body=%s", rw.Body.String())
	require.Contains(t, rw.Body.String(), "not_found")
}

func newAttachmentsRouter(t *testing.T) (*mux.Router, blobstore.BlobStore, string) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internalnotes.Schema),
	))
	repo := internalnotes.NewSQLiteRepository(d, clock.System{})
	attachmentRepo := internalnotes.NewSQLiteAttachmentsRepository(d, clock.System{})
	created, err := repo.Create(context.Background(), internalnotes.Note{Title: "upload target"})
	require.NoError(t, err)

	store := blobstore.NewMemoryBlobStore()
	handler := notes.NewAttachmentsHandler(notes.AttachmentsDeps{
		Service: internalnotes.NewAttachmentsService(repo, attachmentRepo),
		Store:   store,
		Logger:  log.New(io.Discard, "", 0),
	})
	router := mux.NewRouter()
	router.PathPrefix("/api/v1/notes").Handler(handler)
	return router, store, created.ID
}

func multipartBody(t *testing.T, field, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, "example.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}
