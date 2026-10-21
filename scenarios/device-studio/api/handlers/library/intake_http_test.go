package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	domain "device-studio/internal/library"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
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

func handlerMaterialGLB() []byte {
	chunk := []byte(`{"asset":{"version":"2.0"},"materials":[{"name":"shell","pbrMetallicRoughness":{"roughnessFactor":0.8,"metallicFactor":0}}]}`)
	for len(chunk)%4 != 0 {
		chunk = append(chunk, ' ')
	}
	bin := []byte{9, 8, 7, 6}
	out := make([]byte, 12+8+len(chunk)+8+len(bin))
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(chunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], chunk)
	position := 20 + len(chunk)
	binary.LittleEndian.PutUint32(out[position:position+4], uint32(len(bin)))
	binary.LittleEndian.PutUint32(out[position+4:position+8], 0x004e4942)
	copy(out[position+8:], bin)
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

func TestIntakeMaterialHTTPInventoryAndCorrectionBindExactParentRoute(t *testing.T) {
	service := handlerIntakeService(t)
	original := handlerMaterialGLB()
	staged, err := service.BeginImport(context.Background(), domain.IntakeClaims{CandidateID: "http-material-parent", DisplayName: "Phone", SourceLabel: "local file"}, "phone.glb", int64(len(original)), bytes.NewReader(original))
	require.NoError(t, err)
	parent, err := staged.Commit(context.Background())
	require.NoError(t, err)
	router := mux.NewRouter()
	router.Handle("/api/v1/library/intake-candidates/{candidate_id}/revisions/{revision}/materials", newIntakeMaterialsHandler(service, nil, nil, nil)).Methods(http.MethodGet)
	router.Handle("/api/v1/library/intake-candidates/{candidate_id}/revisions/{revision}/material-corrections", newIntakeMaterialCorrectionHandler(service, nil, nil, nil)).Methods(http.MethodPost)
	path := "/api/v1/library/intake-candidates/" + parent.ID + "/revisions/1"
	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, path+"/materials?revision_sha256="+parent.RevisionSHA256, nil))
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	require.Contains(t, get.Body.String(), `"name":"shell"`)
	require.Contains(t, get.Body.String(), parent.OriginalSHA256)
	postBody := `{"parent_candidate_id":"http-material-parent","parent_revision":1,"parent_revision_sha256":"` + parent.RevisionSHA256 + `","child_candidate_id":"http-material-child","material_index":0,"material_name":"shell","roughness_factor":0.25}`
	post := httptest.NewRecorder()
	router.ServeHTTP(post, httptest.NewRequest(http.MethodPost, path+"/material-corrections", strings.NewReader(postBody)))
	require.Equal(t, http.StatusOK, post.Code, post.Body.String())
	require.Contains(t, post.Body.String(), `"import_method":"local_material_correction"`)
	require.Contains(t, post.Body.String(), `"roughness_factor":0.25`)
	require.Contains(t, post.Body.String(), `"review_status":"unreviewed"`)
	var created struct {
		Candidate domain.IntakeCandidate `json:"candidate"`
	}
	require.NoError(t, json.Unmarshal(post.Body.Bytes(), &created))
	require.NotNil(t, created.Candidate.Correction)
	childInventory := httptest.NewRecorder()
	childPath := "/api/v1/library/intake-candidates/" + created.Candidate.ID + "/revisions/1/materials?revision_sha256=" + created.Candidate.RevisionSHA256
	router.ServeHTTP(childInventory, httptest.NewRequest(http.MethodGet, childPath, nil))
	require.Equal(t, http.StatusOK, childInventory.Code, childInventory.Body.String())
	require.Contains(t, childInventory.Body.String(), `"material_correction_lineage"`)
	wrongRoute := strings.Replace(path, "http-material-parent", "other-parent", 1) + "/material-corrections"
	rejected := httptest.NewRecorder()
	router.ServeHTTP(rejected, httptest.NewRequest(http.MethodPost, wrongRoute, strings.NewReader(postBody)))
	require.Equal(t, http.StatusBadRequest, rejected.Code)
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

func TestIntakeConnectListAndGetPreserveCreationAndCorrectionLineage(t *testing.T) {
	service := handlerIntakeService(t)
	ctx := context.Background()

	parentBytes := handlerMaterialGLB()
	parentStage, err := service.BeginImport(ctx, domain.IntakeClaims{CandidateID: "lineage-parent", DisplayName: "Source watch", SourceLabel: "local fixture", LicenseClaim: "CC0-1.0"}, "watch.glb", int64(len(parentBytes)), bytes.NewReader(parentBytes))
	require.NoError(t, err)
	parent, err := parentStage.Commit(ctx)
	require.NoError(t, err)
	child, err := service.CorrectMaterial(ctx, domain.MaterialCorrectionRequest{
		ParentCandidateID: parent.ID, ParentRevision: parent.Revision, ParentRevisionSHA256: parent.RevisionSHA256,
		ChildCandidateID: "lineage-corrected", MaterialIndex: 0, MaterialName: "shell", RoughnessFactor: float64Pointer(0.25),
	})
	require.NoError(t, err)
	require.NotNil(t, child.Correction)

	recipeDigest := sha256.Sum256([]byte("recipe"))
	referenceDigest := sha256.Sum256([]byte("reference"))
	outcomeDigest := sha256.Sum256([]byte("outcome"))
	creation, err := service.PersistGenerated(ctx, domain.IntakeClaims{CandidateID: "lineage-generated", DisplayName: "Generated fixture", SourceLabel: "local AI", LicenseClaim: "unknown"}, "geometry-job", domain.CreationLineage{
		JobID: "geometry-job", RecipeSHA256: hex.EncodeToString(recipeDigest[:]), ReferenceSHA256: []string{hex.EncodeToString(referenceDigest[:])},
		Provider: "ollama", Model: "qwen-local", ProviderCalls: 1, InputTokens: 11, OutputTokens: 22,
		InputTokensProvenance:  inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED,
		OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE,
		CostProvenance:         inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
		AppliedSettings:        &sharedv1.AppliedSettings{TemperatureSupport: sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN, MaxOutputTokensEffective: 4096, MaxOutputTokensSource: sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST},
		InferenceOutcomeSHA256: hex.EncodeToString(outcomeDigest[:]),
	}, handlerIntakeGLB())
	require.NoError(t, err)
	require.NotNil(t, creation.Creation)

	path, handler := api.NewLibraryCatalogServiceHandler(NewConnectHandlerWithIntake(nil, service))
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	server := httptest.NewServer(router)
	defer server.Close()
	client := api.NewLibraryCatalogServiceClient(server.Client(), server.URL)
	page, err := client.ListIntakeCandidates(ctx, connect.NewRequest(&v1.ListIntakeCandidatesRequest{PageSize: 10}))
	require.NoError(t, err)
	byID := make(map[string]*v1.IntakeCandidateRevision, len(page.Msg.Candidates))
	for _, candidate := range page.Msg.Candidates {
		byID[candidate.Id] = candidate
	}
	corrected := byID[child.ID]
	require.NotNil(t, corrected)
	require.Equal(t, child.Correction.ParentCandidateID, corrected.MaterialCorrectionLineage.ParentCandidateId)
	require.Equal(t, child.Correction.ParentSHA256, corrected.MaterialCorrectionLineage.ParentSha256)
	require.Equal(t, child.Correction.BeforeGLBSHA256, corrected.MaterialCorrectionLineage.BeforeGlbSha256)
	require.Equal(t, child.Correction.AfterGLBSHA256, corrected.MaterialCorrectionLineage.AfterGlbSha256)
	require.True(t, corrected.MaterialCorrectionLineage.GeometryUnchanged)
	require.InDelta(t, 0.8, corrected.MaterialCorrectionLineage.Before.AsMap()["roughness_factor"], 1e-9)
	require.InDelta(t, 0.25, corrected.MaterialCorrectionLineage.After.AsMap()["roughness_factor"], 1e-9)
	require.Equal(t, child.ReviewStatus, corrected.ReviewStatus)

	generated := byID[creation.ID]
	require.NotNil(t, generated)
	require.Equal(t, creation.Creation.JobID, generated.CreationLineage.JobId)
	require.Equal(t, creation.Creation.ReferenceSHA256, generated.CreationLineage.ReferenceSha256)
	require.Equal(t, creation.Creation.InferenceOutcomeSHA256, generated.CreationLineage.InferenceOutcomeSha256)
	require.Equal(t, creation.Creation.InputTokensProvenance, generated.CreationLineage.InputTokensProvenance)
	require.Equal(t, creation.Creation.OutputTokensProvenance, generated.CreationLineage.OutputTokensProvenance)
	require.Equal(t, creation.Creation.CostProvenance, generated.CreationLineage.CostProvenance)
	require.Equal(t, int32(4096), generated.CreationLineage.AppliedSettings.MaxOutputTokensEffective)

	for _, id := range []string{child.ID, creation.ID} {
		response, getErr := client.GetIntakeCandidate(ctx, connect.NewRequest(&v1.GetIntakeCandidateRequest{CandidateId: id, Revision: 1}))
		require.NoError(t, getErr)
		require.NotNil(t, response.Msg.Candidate)
		require.Equal(t, byID[id].RevisionSha256, response.Msg.Candidate.RevisionSha256)
	}
}

func float64Pointer(value float64) *float64 { return &value }
