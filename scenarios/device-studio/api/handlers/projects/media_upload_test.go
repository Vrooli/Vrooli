package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	domain "device-studio/internal/projects"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mediaProjectStub struct {
	exists bool
	calls  int
}

func (s *mediaProjectStub) Create(context.Context, domain.CreateInput) (domain.Project, error) {
	panic("unused")
}
func (s *mediaProjectStub) Get(context.Context, string) (domain.Project, error) {
	s.calls++
	if !s.exists {
		return domain.Project{}, domain.ErrProjectNotFound{ID: "missing"}
	}
	return domain.Project{ID: "p"}, nil
}
func (s *mediaProjectStub) List(context.Context, int32, string) ([]domain.Project, string, error) {
	panic("unused")
}
func (s *mediaProjectStub) Rename(context.Context, domain.RenameInput) (domain.Project, error) {
	panic("unused")
}

type mediaServiceStub struct {
	imported bool
	opened   bool
}

func (s *mediaServiceStub) Import(_ context.Context, p, n string, size int64, r io.Reader) (domain.SourceMedia, error) {
	s.imported = true
	data, _ := io.ReadAll(r)
	return domain.SourceMedia{ID: "m1", ProjectID: p, OriginalFilename: n, SizeBytes: int64(len(data)), MediaType: "image/png", Width: 2, Height: 1, ImportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func (s *mediaServiceStub) Get(context.Context, string, string) (domain.SourceMedia, error) {
	return domain.SourceMedia{}, nil
}
func (s *mediaServiceStub) List(context.Context, string, int32, string) ([]domain.SourceMedia, string, error) {
	return nil, "", nil
}
func (s *mediaServiceStub) Open(context.Context, string, string) (domain.SourceMedia, io.ReadCloser, error) {
	s.opened = true
	return domain.SourceMedia{SizeBytes: 3, MediaType: "image/png"}, io.NopCloser(bytes.NewReader([]byte("png"))), nil
}

func multipartRequest(t *testing.T, filename string, contents []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(contents)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}
func TestMediaUploadReturnsTypedResponseAndChecksProjectBeforeReading(t *testing.T) {
	projects := &mediaProjectStub{exists: true}
	media := &mediaServiceStub{}
	router := mux.NewRouter()
	router.Handle("/api/v1/projects/{project_id}/media", newMediaUploadHandler(projects, media, log.New(io.Discard, "", 0))).Methods("POST")
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, multipartRequest(t, "mask.png", []byte("not decoded by seam")))
	require.Equal(t, http.StatusCreated, rw.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	source := got["source_media"].(map[string]any)
	require.Equal(t, "m1", source["id"])
	require.True(t, media.imported)
	projects.exists = false
	media.imported = false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p/media", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	rw = httptest.NewRecorder()
	router.ServeHTTP(rw, req)
	require.Equal(t, http.StatusNotFound, rw.Code)
	require.False(t, media.imported)
	require.Equal(t, 2, projects.calls)
}

func TestMediaUploadWithoutConfiguredStorageReturnsUnavailable(t *testing.T) {
	projects := &mediaProjectStub{exists: true}
	router := mux.NewRouter()
	router.Handle("/api/v1/projects/{project_id}/media", newMediaUploadHandler(projects, nil, log.New(io.Discard, "", 0))).Methods("POST")
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, multipartRequest(t, "mask.png", []byte("not decoded by seam")))
	require.Equal(t, http.StatusServiceUnavailable, rw.Code)
	require.Equal(t, 0, projects.calls, "missing media service should fail before doing project work")
}

func TestMediaContentUsesVerifiedMimeAndNosniffHeaders(t *testing.T) {
	media := &mediaServiceStub{}
	router := mux.NewRouter()
	router.Handle("/api/v1/projects/{project_id}/media/{media_id}/content", newMediaContentHandler(media, log.New(io.Discard, "", 0))).Methods("GET")
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p/media/m1/content", nil)
	router.ServeHTTP(rw, req)
	require.Equal(t, http.StatusOK, rw.Code)
	require.Equal(t, "image/png", rw.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", rw.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "png", rw.Body.String())
	require.True(t, media.opened)
}
func TestMediaUploadMapsProjectDatabaseFailureToRedacted500(t *testing.T) {
	projects := &mediaProjectErrorStub{}
	router := mux.NewRouter()
	router.Handle("/api/v1/projects/{project_id}/media", newMediaUploadHandler(projects, &mediaServiceStub{}, log.New(io.Discard, "", 0)))
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, multipartRequest(t, "mask.png", []byte("x")))
	require.Equal(t, http.StatusInternalServerError, rw.Code)
	require.NotContains(t, rw.Body.String(), "database secret")
}

func TestMediaUploadRejectsQuotedPrintableWithoutDecodingBytes(t *testing.T) {
	projects := &mediaProjectStub{exists: true}
	media := &mediaServiceStub{}
	router := mux.NewRouter()
	router.Handle("/api/v1/projects/{project_id}/media", newMediaUploadHandler(projects, media, log.New(io.Discard, "", 0)))
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": "x.png"}))
	header.Set("Content-Type", "image/png")
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := mw.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("=FF"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)
	require.Equal(t, http.StatusBadRequest, rw.Code)
	require.False(t, media.imported)
}

type mediaProjectErrorStub struct{ mediaProjectStub }

func (*mediaProjectErrorStub) Get(context.Context, string) (domain.Project, error) {
	return domain.Project{}, errors.New("database secret")
}
