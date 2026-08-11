package notes_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"scenario-to-ios/handlers/notes"
	"scenario-to-ios/internal/clock"
	"scenario-to-ios/internal/testutil/assertx"
	"scenario-to-ios/internal/testutil/db"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-ios/v1/notes"

	localdb "scenario-to-ios/internal/database"
	internalnotes "scenario-to-ios/internal/notes"
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
	router, store, _ := newAttachmentsRouter(t)
	body, contentType := multipartBody(t, "file", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/missing/attachments", body)
	req.Header.Set("Content-Type", contentType)
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	require.Equal(t, http.StatusNotFound, rw.Code, "body=%s", rw.Body.String())
	require.Contains(t, rw.Body.String(), "not_found")
	require.NotEmpty(t, store.DeletedKeys())
}

func TestAttachmentsHandlerBlobStoreFailureDoesNotCreateMetadata(t *testing.T) {
	service := &recordingAttachmentsService{}
	store := &failingPutStore{err: errors.New("disk full")}
	router := newAttachmentsRouterWithDeps(service, store)
	body, contentType := multipartBody(t, "file", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/note-1/attachments", body)
	req.Header.Set("Content-Type", contentType)
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code, "body=%s", rw.Body.String())
	require.False(t, service.called)
}

func TestAttachmentsHandlerMetadataFailureDeletesStoredBlob(t *testing.T) {
	service := &recordingAttachmentsService{err: errors.New("metadata unavailable")}
	store := newTrackingBlobStore()
	router := newAttachmentsRouterWithDeps(service, store)
	body, contentType := multipartBody(t, "file", "hello")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes/note-1/attachments", body)
	req.Header.Set("Content-Type", contentType)
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code, "body=%s", rw.Body.String())
	require.True(t, service.called)
	require.Len(t, store.DeletedKeys(), 1)
}

func newAttachmentsRouter(t *testing.T) (*mux.Router, *trackingBlobStore, string) {
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

	store := newTrackingBlobStore()
	handler := notes.NewAttachmentsHandler(notes.AttachmentsDeps{
		Service: internalnotes.NewAttachmentsService(repo, attachmentRepo),
		Store:   store,
		Logger:  log.New(io.Discard, "", 0),
	})
	router := mux.NewRouter()
	router.PathPrefix("/api/v1/notes").Handler(handler)
	return router, store, created.ID
}

func newAttachmentsRouterWithDeps(service internalnotes.AttachmentsService, store blobstore.BlobStore) *mux.Router {
	handler := notes.NewAttachmentsHandler(notes.AttachmentsDeps{
		Service: service,
		Store:   store,
		Logger:  log.New(io.Discard, "", 0),
	})
	router := mux.NewRouter()
	router.PathPrefix("/api/v1/notes").Handler(handler)
	return router
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

type recordingAttachmentsService struct {
	called bool
	err    error
}

func (s *recordingAttachmentsService) Create(_ context.Context, in internalnotes.CreateAttachmentInput) (internalnotes.Attachment, error) {
	s.called = true
	if s.err != nil {
		return internalnotes.Attachment{}, s.err
	}
	return internalnotes.Attachment{
		NoteID:    in.NoteID,
		Key:       in.Key,
		MIMEType:  in.MIMEType,
		SizeBytes: in.SizeBytes,
	}, nil
}

type failingPutStore struct {
	err error
}

func (s *failingPutStore) Put(context.Context, string, io.Reader, string) error {
	return s.err
}

func (s *failingPutStore) Get(context.Context, string) (io.ReadCloser, string, error) {
	return nil, "", errors.New("not found")
}

func (s *failingPutStore) Delete(context.Context, string) error {
	return nil
}

type trackingBlobStore struct {
	inner   blobstore.BlobStore
	deleted []string
}

func newTrackingBlobStore() *trackingBlobStore {
	return &trackingBlobStore{inner: blobstore.NewMemoryBlobStore()}
}

func (s *trackingBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	return s.inner.Put(ctx, key, r, mime)
}

func (s *trackingBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	return s.inner.Get(ctx, key)
}

func (s *trackingBlobStore) Delete(ctx context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return s.inner.Delete(ctx, key)
}

func (s *trackingBlobStore) DeletedKeys() []string {
	return append([]string(nil), s.deleted...)
}
