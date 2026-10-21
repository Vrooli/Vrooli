package library

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	domain "device-studio/internal/library"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func handlerIntakeGLB() []byte {
	chunk := []byte(`{"asset":{"version":"2.0"}}`)
	for len(chunk)%4 != 0 {
		chunk = append(chunk, ' ')
	}
	out := make([]byte, 20+len(chunk))
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(chunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], chunk)
	return out
}

func handlerIntakeService(t *testing.T) *domain.IntakeService {
	t.Helper()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(domain.Schema)))
	service, err := domain.NewIntakeService(db, blobstore.NewMemoryBlobStore(), func(context.Context, []byte) (domain.IntakeInspection, error) {
		return domain.IntakeInspection{Profile: "test-inspector-v1", ProfileVersion: 1, Supported: false, Finding: "fixture_is_not_in_renderable_subset"}, nil
	}, nil, nil)
	require.NoError(t, err)
	return service
}

func TestLocalIntakeHTTPUploadAndContentServeExactUnreviewedOriginal(t *testing.T) {
	service := handlerIntakeService(t)
	claims := &v1.IntakeCandidateClaims{CandidateId: "http-candidate", DisplayName: "Imported phone", SourceLabel: "local upload", LicenseClaim: "CC0-1.0", Attribution: "Declared by operator", DeclaredUses: []string{"render"}}
	metadata, err := protojson.Marshal(claims)
	require.NoError(t, err)
	original := handlerIntakeGLB()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metaPart, err := writer.CreateFormField("metadata")
	require.NoError(t, err)
	_, err = metaPart.Write(metadata)
	require.NoError(t, err)
	filePart, err := writer.CreateFormFile("file", "phone.glb")
	require.NoError(t, err)
	_, err = filePart.Write(original)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	router := mux.NewRouter()
	router.Handle("/api/v1/library/intake-candidates", newIntakeUploadHandler(service, nil, nil, nil)).Methods(http.MethodPost)
	router.Handle("/api/v1/library/intake-candidates/{candidate_id}/revisions/{revision}/content", newIntakeContentHandler(service, nil, nil, nil)).Methods(http.MethodGet)
	upload := httptest.NewRequest(http.MethodPost, "/api/v1/library/intake-candidates", &body)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, upload)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var created v1.GetIntakeCandidateResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &created))
	require.NotNil(t, created.Candidate)
	require.Equal(t, claims.CandidateId, created.Candidate.Id)
	require.Equal(t, "unreviewed", created.Candidate.ReviewStatus)
	require.Equal(t, "not_reviewed", created.Candidate.RenderStatus)
	require.False(t, created.Candidate.Inspection.Supported)
	require.Contains(t, created.Candidate.Inspection.Finding, "fixture_is_not_in_renderable_subset")
	var publicShape map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &publicShape))
	require.NotContains(t, string(response.Body.Bytes()), "blob_key")
	require.NotContains(t, string(response.Body.Bytes()), "storage_key")

	content := httptest.NewRequest(http.MethodGet, "/api/v1/library/intake-candidates/http-candidate/revisions/1/content", nil)
	contentResponse := httptest.NewRecorder()
	router.ServeHTTP(contentResponse, content)
	require.Equal(t, http.StatusOK, contentResponse.Code)
	require.Equal(t, "model/gltf-binary", contentResponse.Header().Get("Content-Type"))
	require.Equal(t, original, contentResponse.Body.Bytes())
	require.Equal(t, "nosniff", contentResponse.Header().Get("X-Content-Type-Options"))
}

func TestIntakeConnectListAndGetExposeTypedMetadataOnly(t *testing.T) {
	service := handlerIntakeService(t)
	original := handlerIntakeGLB()
	staged, err := service.BeginImport(context.Background(), domain.IntakeClaims{CandidateID: "rpc-candidate", DisplayName: "Model", SourceLabel: "local", LicenseClaim: "CC0-1.0", Attribution: "operator"}, "model.glb", int64(len(original)), bytes.NewReader(original))
	require.NoError(t, err)
	_, err = staged.Commit(context.Background())
	require.NoError(t, err)

	path, handler := api.NewLibraryCatalogServiceHandler(NewConnectHandlerWithIntake(nil, service))
	registerRouter := mux.NewRouter()
	// Connect's service handler is mounted on its generated path so the test
	// exercises the same typed client contract as external consumers.
	connectx.RegisterServices(registerRouter, connectx.ServiceMount{Path: path, Handler: handler})
	server := httptest.NewServer(registerRouter)
	defer server.Close()
	client := api.NewLibraryCatalogServiceClient(server.Client(), server.URL)
	page, err := client.ListIntakeCandidates(context.Background(), connect.NewRequest(&v1.ListIntakeCandidatesRequest{PageSize: 10}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Candidates, 1)
	require.Equal(t, "rpc-candidate", page.Msg.Candidates[0].Id)
	got, err := client.GetIntakeCandidate(context.Background(), connect.NewRequest(&v1.GetIntakeCandidateRequest{CandidateId: "rpc-candidate", Revision: 1}))
	require.NoError(t, err)
	require.Equal(t, page.Msg.Candidates[0].RevisionSha256, got.Msg.Candidate.RevisionSha256)
	encoded, err := protojson.Marshal(got.Msg.Candidate)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "blob_key")
	require.NotContains(t, string(encoded), "storage_key")
}
