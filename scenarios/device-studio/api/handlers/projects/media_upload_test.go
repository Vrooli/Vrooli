package projects

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"device-studio/internal/jobs"
	domain "device-studio/internal/projects"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
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
	commits  int
	aborts   int
	row      domain.SourceMedia
}

func (s *mediaServiceStub) Import(_ context.Context, p, n string, size int64, r io.Reader) (domain.SourceMedia, error) {
	s.imported = true
	data, _ := io.ReadAll(r)
	return domain.SourceMedia{ID: "m1", ProjectID: p, OriginalFilename: n, SizeBytes: int64(len(data)), MediaType: "image/png", Width: 2, Height: 1, ImportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}

type stagedMediaStub struct {
	row   domain.SourceMedia
	owner *mediaServiceStub
}

func (s *stagedMediaStub) Media() domain.SourceMedia { return s.row }
func (s *stagedMediaStub) Commit(context.Context) (domain.SourceMedia, error) {
	s.owner.commits++
	return s.row, nil
}
func (s *stagedMediaStub) Abort(context.Context) error { s.owner.aborts++; return nil }
func (s *mediaServiceStub) BeginImport(_ context.Context, p, n string, size int64, r io.Reader) (domain.StagedSourceMediaImport, error) {
	s.imported = true
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	row := s.row
	if row.ID == "" {
		row = domain.SourceMedia{ID: "m1", ProjectID: p, OriginalFilename: n, SizeBytes: int64(len(data)), MediaType: "image/png", Kind: domain.MediaKindImage, Readiness: domain.ReadinessReady, Width: 2, Height: 1, RasterKnown: true, ImportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	} else {
		row.ProjectID, row.OriginalFilename, row.SizeBytes = p, n, int64(len(data))
	}
	return &stagedMediaStub{owner: s, row: row}, nil
}
func (s *mediaServiceStub) BindInspectionJob(context.Context, string, string, string, string) (domain.SourceMedia, error) {
	return domain.SourceMedia{}, nil
}
func (s *mediaServiceStub) BindInspectionAttempt(context.Context, string, string, string, string, string, string) (domain.SourceMedia, error) {
	return domain.SourceMedia{}, nil
}
func (s *mediaServiceStub) StoreInspectionArtifacts(context.Context, string, string, string, string, string, string, []byte, []byte) (string, string, error) {
	return "", "", nil
}
func (s *mediaServiceStub) OpenInspectionArtifacts(context.Context, string, string, string, string, string, string) ([]byte, []byte, error) {
	return nil, nil, nil
}
func (s *mediaServiceStub) ProjectInspection(context.Context, domain.InspectionProjection) error {
	return nil
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

type uploadInspectionStub struct {
	job   jobs.Job
	calls int
	err   error
}

func (s *uploadInspectionStub) EnsureSourceInspection(context.Context, string, string) (jobs.Job, bool, error) {
	s.calls++
	return s.job, true, s.err
}
func (*uploadInspectionStub) GetSourceInspectionStatus(context.Context, string, string) (jobs.Job, error) {
	return jobs.Job{}, jobs.ErrNotFound
}
func (*uploadInspectionStub) RetrySourceInspection(context.Context, string, string, int64) (jobs.Job, error) {
	return jobs.Job{}, jobs.ErrNotFound
}
func (*uploadInspectionStub) CancelSourceInspection(context.Context, string, string, int64) (jobs.Job, error) {
	return jobs.Job{}, jobs.ErrNotFound
}

type uploadRouteActivator struct{ calls int }

func (s *uploadRouteActivator) ActivateTestRoute(context.Context) error { s.calls++; return nil }

func TestVideoUploadSchedulesInspectionAndPreservesImportedRowOnSubmitFailure(t *testing.T) {
	for _, tc := range []struct {
		name          string
		submitErr     error
		wantJob       bool
		wantIssueCode string
	}{
		{name: "accepted", wantJob: true},
		{name: "submission unavailable", submitErr: errors.New("profile resolver unavailable"), wantIssueCode: "inspection_pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectRepo := &mediaProjectStub{exists: true}
			media := &mediaServiceStub{row: domain.SourceMedia{ID: "video-1", MediaType: "video/mp4", Kind: domain.MediaKindVideo, Readiness: domain.ReadinessPending}}
			inspection := &uploadInspectionStub{job: jobs.Job{ID: "job-1", Version: 1, State: jobs.Queued, Attempts: []jobs.Attempt{{ID: "attempt-1"}}}, err: tc.submitErr}
			activator := &uploadRouteActivator{}
			router := mux.NewRouter()
			router.Handle("/api/v1/projects/{project_id}/media", newMediaUploadHandlerWithInspection(projectRepo, media, inspection, activator, nil, nil, log.New(io.Discard, "", 0))).Methods(http.MethodPost)
			rw := httptest.NewRecorder()
			router.ServeHTTP(rw, multipartRequest(t, "capture.mp4", []byte("video")))
			require.Equal(t, http.StatusCreated, rw.Code, "durable import remains successful when scheduling is unavailable")
			var response projectsv1.UploadSourceMediaResponse
			require.NoError(t, protojson.Unmarshal(rw.Body.Bytes(), &response))
			require.Equal(t, "video-1", response.SourceMedia.Id)
			require.Equal(t, 1, inspection.calls)
			if tc.wantJob {
				require.Equal(t, "job-1", response.SourceMedia.InspectionJobId)
				require.Equal(t, "attempt-1", response.SourceMedia.InspectionAttemptId)
				require.Equal(t, 1, activator.calls)
			} else {
				require.Equal(t, "inspection_pending", response.SourceMedia.InspectionIssueCode)
				require.Equal(t, 0, activator.calls)
			}
			require.Equal(t, 1, media.commits, "import metadata must not be rolled back or aborted after inspection failure")
			require.Zero(t, media.aborts)
		})
	}
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

func TestMediaUploadDoesNotCommitExtraPartOrTruncatedBoundary(t *testing.T) {
	for _, truncate := range []bool{false, true} {
		t.Run(map[bool]string{false: "extra-part", true: "truncated-boundary"}[truncate], func(t *testing.T) {
			projects := &mediaProjectStub{exists: true}
			media := &mediaServiceStub{}
			router := mux.NewRouter()
			router.Handle("/api/v1/projects/{project_id}/media", newMediaUploadHandler(projects, media, log.New(io.Discard, "", 0)))
			var body bytes.Buffer
			mw := multipart.NewWriter(&body)
			part, err := mw.CreateFormFile("file", "x.png")
			require.NoError(t, err)
			_, err = part.Write([]byte("streamed"))
			require.NoError(t, err)
			if !truncate {
				extra, createErr := mw.CreateFormField("extra")
				require.NoError(t, createErr)
				_, err = extra.Write([]byte("no"))
				require.NoError(t, err)
			}
			require.NoError(t, mw.Close())
			raw := body.Bytes()
			if truncate {
				raw = raw[:len(raw)-8]
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p/media", bytes.NewReader(raw))
			req.Header.Set("Content-Type", mw.FormDataContentType())
			rw := httptest.NewRecorder()
			router.ServeHTTP(rw, req)
			require.NotEqual(t, http.StatusCreated, rw.Code)
			require.Zero(t, media.commits)
			if !truncate {
				require.Equal(t, 1, media.aborts)
			} else {
				require.Zero(t, media.aborts)
			}
		})
	}
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

func routedMediaUploadFixture(t *testing.T) (*mux.Router, domain.SourceMediaRepository, blobstore.BlobStore, string, *atomic.Int32) {
	return routedMediaUploadFixtureWithLimit(t, maxMediaMultipartBytes)
}

func routedMediaUploadFixtureWithLimit(t *testing.T, maxBytes int64) (*mux.Router, domain.SourceMediaRepository, blobstore.BlobStore, string, *atomic.Int32) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(domain.Schema)))
	repo := domain.NewSQLiteRepository(db, nil)
	projects := domain.NewService(repo, nil)
	project, err := projects.Create(context.Background(), domain.CreateInput{Name: "Upload fixture"})
	require.NoError(t, err)
	mediaRepo := repo.(domain.SourceMediaRepository)
	store := blobstore.NewMemoryBlobStore()
	var seq atomic.Int32
	media := domain.NewSourceMediaService(domain.SourceMediaDeps{Projects: projects, Repository: mediaRepo, Store: store, NewID: func() string { return fmt.Sprintf("route-%d", seq.Add(1)) }})
	router := mux.NewRouter()
	router.Handle("/api/v1/projects/{project_id}/media", http.HandlerFunc((&mediaUploadHandler{projects: projects, media: media, logger: log.New(io.Discard, "", 0), maxBytes: maxBytes}).upload)).Methods(http.MethodPost)
	router.Handle("/api/v1/projects/{project_id}/media/{media_id}/content", newMediaContentHandler(media, log.New(io.Discard, "", 0))).Methods(http.MethodGet)
	return router, mediaRepo, store, project.ID, &seq
}
func TestMediaUploadStreamsPendingVideoAndLeavesUnknownFactsAbsent(t *testing.T) {
	router, repo, _, projectID, _ := routedMediaUploadFixture(t)
	source := []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "clip.mp4")
	require.NoError(t, err)
	_, err = part.Write(source)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rw := httptest.NewRecorder()
	router.ServeHTTP(rw, req)
	require.Equal(t, http.StatusCreated, rw.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &response))
	row := response["source_media"].(map[string]any)
	require.Equal(t, "SOURCE_MEDIA_KIND_VIDEO", row["kind"])
	require.Equal(t, "MEDIA_READINESS_PENDING", row["readiness"])
	require.NotContains(t, row, "width")
	require.NotContains(t, row, "height")
	sum := sha256.Sum256(source)
	saved, err := repo.GetSourceMediaByDigest(context.Background(), projectID, hex.EncodeToString(sum[:]))
	require.NoError(t, err)
	require.Equal(t, domain.MediaKindVideo, saved.Kind)
	require.False(t, saved.RasterKnown)
}
func TestMediaUploadMalformedEnvelopeLeavesNoRowOrStagingBlob(t *testing.T) {
	for _, truncate := range []bool{false, true} {
		t.Run(map[bool]string{false: "extra-field", true: "truncated-final-boundary"}[truncate], func(t *testing.T) {
			router, repo, store, projectID, _ := routedMediaUploadFixture(t)
			source := []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0}
			var body bytes.Buffer
			mw := multipart.NewWriter(&body)
			part, err := mw.CreateFormFile("file", "clip.mp4")
			require.NoError(t, err)
			_, err = part.Write(source)
			require.NoError(t, err)
			if !truncate {
				extra, e := mw.CreateFormField("extra")
				require.NoError(t, e)
				_, err = extra.Write([]byte("x"))
				require.NoError(t, err)
			}
			require.NoError(t, mw.Close())
			raw := body.Bytes()
			if truncate {
				raw = raw[:len(raw)-8]
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/media", bytes.NewReader(raw))
			req.Header.Set("Content-Type", mw.FormDataContentType())
			rw := httptest.NewRecorder()
			router.ServeHTTP(rw, req)
			require.NotEqual(t, http.StatusCreated, rw.Code)
			sum := sha256.Sum256(source)
			_, err = repo.GetSourceMediaByDigest(context.Background(), projectID, hex.EncodeToString(sum[:]))
			require.Error(t, err)
			if _, _, err = store.Get(context.Background(), "projects/"+projectID+"/media/route-2"); err == nil {
				t.Fatal("malformed envelope left operation-owned staging bytes")
			}
		})
	}
}

func TestMediaUploadCountsMultipartEpilogueBeforeCommit(t *testing.T) {
	source := []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "clip.mp4")
	require.NoError(t, err)
	_, err = part.Write(source)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	validEnvelope := append([]byte(nil), body.Bytes()...)

	t.Run("bounded epilogue is consumed", func(t *testing.T) {
		router, repo, _, projectID, _ := routedMediaUploadFixtureWithLimit(t, int64(len(validEnvelope)+32))
		raw := append(append([]byte(nil), validEnvelope...), []byte("small-epilogue")...)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/media", bytes.NewReader(raw))
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rw := httptest.NewRecorder()
		router.ServeHTTP(rw, req)
		require.Equal(t, http.StatusCreated, rw.Code)
		sum := sha256.Sum256(source)
		_, err := repo.GetSourceMediaByDigest(context.Background(), projectID, hex.EncodeToString(sum[:]))
		require.NoError(t, err)
	})

	t.Run("oversized epilogue is rejected without commit", func(t *testing.T) {
		limit := int64(len(validEnvelope) + 8)
		router, repo, store, projectID, _ := routedMediaUploadFixtureWithLimit(t, limit)
		raw := append(append([]byte(nil), validEnvelope...), bytes.Repeat([]byte("x"), 64)...)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/media", bytes.NewReader(raw))
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rw := httptest.NewRecorder()
		router.ServeHTTP(rw, req)
		require.Equal(t, http.StatusRequestEntityTooLarge, rw.Code)
		sum := sha256.Sum256(source)
		_, err := repo.GetSourceMediaByDigest(context.Background(), projectID, hex.EncodeToString(sum[:]))
		require.Error(t, err)
		if _, _, err = store.Get(context.Background(), "projects/"+projectID+"/media/route-2"); err == nil {
			t.Fatal("oversized multipart epilogue left staging bytes")
		}
	})
}

func TestMediaContentCorruptSameSizeCannotCompleteSuccessfulBody(t *testing.T) {
	for _, size := range []int{16, 64 << 10} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			router, _, store, projectID, _ := routedMediaUploadFixture(t)
			source := make([]byte, size)
			copy(source, []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0})
			var body bytes.Buffer
			mw := multipart.NewWriter(&body)
			part, err := mw.CreateFormFile("file", "clip.mp4")
			require.NoError(t, err)
			_, err = part.Write(source)
			require.NoError(t, err)
			require.NoError(t, mw.Close())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/media", &body)
			req.Header.Set("Content-Type", mw.FormDataContentType())
			upload := httptest.NewRecorder()
			router.ServeHTTP(upload, req)
			require.Equal(t, http.StatusCreated, upload.Code)

			corrupt := append([]byte(nil), source...)
			corrupt[len(corrupt)-1] ^= 0xff
			require.NoError(t, store.Put(context.Background(), "projects/"+projectID+"/media/route-2", bytes.NewReader(corrupt), "video/mp4"))

			server := httptest.NewServer(router)
			defer server.Close()
			response, err := server.Client().Get(server.URL + "/api/v1/projects/" + projectID + "/media/route-1/content")
			if err != nil {
				return // Abort before headers is a failed transfer as well.
			}
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Equal(t, int64(len(source)), response.ContentLength)
			served, readErr := io.ReadAll(response.Body)
			require.Error(t, readErr, "corrupt stream must abort rather than complete as a successful response")
			require.Less(t, len(served), len(source), "unverified final byte must remain withheld")
		})
	}
}
