package library

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/library"
	"device-studio/internal/routedblobstore"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestListBundleRevisionsRejectsMalformedCursor(t *testing.T) {
	_, err := NewConnectHandler(nil).ListBundleRevisions(context.Background(), connect.NewRequest(&v1.ListBundleRevisionsRequest{AfterBundleId: "bad\nID"}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
	handler := NewConnectHandler(&cursorOnlyLibraryService{})
	_, err = handler.ListBundleRevisions(context.Background(), connect.NewRequest(&v1.ListBundleRevisionsRequest{AfterBundleId: "bad\nID"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	_, err = handler.ListBundleRevisions(context.Background(), connect.NewRequest(&v1.ListBundleRevisionsRequest{AfterBundleId: strings.Repeat("a", 129)}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

type cursorOnlyLibraryService struct{ Service }

func (*cursorOnlyLibraryService) ListRevisions(context.Context, string, uint32, int) ([]domain.BundleRevision, error) {
	panic("invalid cursor must be rejected before service access")
}

type catalogStub struct{ rows []domain.BundleRevision }

func (s catalogStub) GetRevision(_ context.Context, id string, revision uint32) (domain.BundleRevision, error) {
	for _, row := range s.rows {
		if row.Manifest.BundleID == id && row.Manifest.Revision == revision {
			return row, nil
		}
	}
	return domain.BundleRevision{}, domain.ErrRevisionNotFound
}
func (s catalogStub) ListRevisions(_ context.Context, after string, rev uint32, limit int) ([]domain.BundleRevision, error) {
	var rows []domain.BundleRevision
	for _, row := range s.rows {
		id, revision := row.Manifest.BundleID, row.Manifest.Revision
		if id > after || (id == after && revision > rev) {
			rows = append(rows, row)
		}
		if len(rows) == limit {
			break
		}
	}
	return rows, nil
}

func TestCatalogConnectExposesReviewedMetadataWithoutArtifactStorageDetails(t *testing.T) {
	now := time.Date(2026, 10, 20, 18, 0, 0, 0, time.UTC)
	row := domain.BundleRevision{
		Manifest:       domain.BundleManifest{Version: domain.ManifestVersion, BundleID: "phone-asset", Revision: 3, Artifacts: []domain.Artifact{{ID: "neutral-glb", Role: domain.RoleGeometryGLB, MIME: "model/gltf-binary", SizeBytes: 128, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LicenseID: "AGPL-3.0-only", Attribution: "Local authored asset", Provenance: domain.ArtifactProvenance{Kind: domain.ProvenanceAuthored, SourceLabel: "local source"}, Rights: []domain.Use{domain.UseRender}}}},
		ManifestSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Review:         domain.ReviewBinding{ID: "review-1", ReviewerID: "reviewer", ReviewedAt: now, Decision: "approved", Rationale: "Local render is permitted for bounded evaluation.", ManifestSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", AllowedUses: []domain.Use{domain.UseRender}},
		ReviewSHA256:   "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", PublishedAt: now,
	}
	projected := &v1.BundleRevisionMetadata{}
	require.NoError(t, revisionapi.Encode(row, projected))
	router := mux.NewRouter()
	path, handler := api.NewLibraryCatalogServiceHandler(NewConnectHandler(catalogStub{rows: []domain.BundleRevision{row}}))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	server := httptest.NewServer(router)
	defer server.Close()
	client := api.NewLibraryCatalogServiceClient(server.Client(), server.URL)
	page, err := client.ListBundleRevisions(context.Background(), connect.NewRequest(&v1.ListBundleRevisionsRequest{PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Revisions, 1)
	metadata := page.Msg.Revisions[0]
	require.Equal(t, "phone-asset", metadata.Manifest.BundleId)
	require.Equal(t, uint32(3), metadata.Manifest.Revision)
	require.Equal(t, row.ManifestSHA256, metadata.ManifestSha256)
	require.Equal(t, row.ReviewSHA256, metadata.ReviewSha256)
	require.Equal(t, "approved", metadata.Review.Decision)
	require.Equal(t, "Local render is permitted for bounded evaluation.", metadata.Review.Rationale)
	require.Equal(t, []string{"render"}, metadata.Review.AllowedUses)
	encoded, err := protojson.Marshal(metadata)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "blob_key")
	require.NotContains(t, string(encoded), "storage_key")
	got, err := client.GetBundleRevision(context.Background(), connect.NewRequest(&v1.GetBundleRevisionRequest{BundleId: "phone-asset", Revision: 3}))
	require.NoError(t, err)
	require.Equal(t, metadata.ManifestSha256, got.Msg.Revision.ManifestSha256)
	require.Equal(t, metadata.Review.Rationale, got.Msg.Revision.Review.Rationale)
}

func TestCatalogPageBoundAndMissingRevisionErrors(t *testing.T) {
	router := mux.NewRouter()
	path, handler := api.NewLibraryCatalogServiceHandler(NewConnectHandler(catalogStub{}))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	server := httptest.NewServer(router)
	defer server.Close()
	client := api.NewLibraryCatalogServiceClient(server.Client(), server.URL)
	_, err := client.ListBundleRevisions(context.Background(), connect.NewRequest(&v1.ListBundleRevisionsRequest{PageSize: 101}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	_, err = client.GetBundleRevision(context.Background(), connect.NewRequest(&v1.GetBundleRevisionRequest{BundleId: "missing", Revision: 1}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
}

func TestGetIntakeCandidateProjectsExactCreationCorrectionLineage(t *testing.T) {
	correction := &domain.CreationCorrectionSource{
		Mode:             "visual_refinement",
		Feedback:         "Keep the authored envelope and refine the bezel.",
		JobID:            "source-job",
		InputFingerprint: "input-fingerprint",
		BriefSHA256:      "brief-digest",
		RecipeSHA256:     "recipe-digest",
	}
	intake := newHandlerIntakeService(t)
	ref := sha256.Sum256([]byte("reference"))
	recipe := sha256.Sum256([]byte("recipe"))
	outcome := sha256.Sum256([]byte("outcome"))
	candidate, err := intake.PersistGenerated(database.WithTestMode(context.Background()), domain.IntakeClaims{
		CandidateID: "creation-candidate", DisplayName: "Generated candidate", SourceLabel: "local test generation",
		LicenseClaim: "unknown", Attribution: "test fixture",
	}, "source-job", domain.CreationLineage{
		JobID: "source-job", RecipeSHA256: hex.EncodeToString(recipe[:]), InferenceOutcomeSHA256: hex.EncodeToString(outcome[:]),
		ReferenceSHA256: []string{hex.EncodeToString(ref[:])}, Provider: "test-provider", Model: "test-model", ProviderCalls: 1,
		AppliedSettings:        &sharedv1.AppliedSettings{TemperatureSupport: sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN, MaxOutputTokensEffective: 4096, MaxOutputTokensSource: sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST},
		InputTokensProvenance:  inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
		OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
		CostProvenance:         inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN,
		CorrectionSource:       correction,
	}, minimalIntakeGLB())
	require.NoError(t, err)
	response, err := NewConnectHandlerWithIntake(nil, intake).GetIntakeCandidate(
		context.Background(),
		connect.NewRequest(&v1.GetIntakeCandidateRequest{CandidateId: candidate.ID, Revision: candidate.Revision}),
	)
	require.NoError(t, err)
	require.NotNil(t, response.Msg.Candidate.CreationLineage)
	got := response.Msg.Candidate.CreationLineage.CorrectionSource
	require.NotNil(t, got)
	require.Equal(t, correction.Mode, got.Mode)
	require.Equal(t, correction.Feedback, got.Feedback)
	require.Equal(t, correction.JobID, got.JobId)
	require.Equal(t, correction.InputFingerprint, got.InputFingerprint)
	require.Equal(t, correction.BriefSHA256, got.BriefSha256)
	require.Equal(t, correction.RecipeSHA256, got.RecipeSha256)
}

func newHandlerIntakeService(t *testing.T) *domain.IntakeService {
	t.Helper()
	db := databasetest.NewSQLite(t)
	ctx := database.WithTestMode(context.Background())
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(domain.Schema)))
	dataRoot := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: t.TempDir()})
	require.NoError(t, roots.InstallTestRoots(storage.Paths{DataDir: dataRoot}, "library-connect-handler-test", time.Minute))
	store, err := routedblobstore.New(roots, "library")
	require.NoError(t, err)
	service, err := domain.NewIntakeService(db, store, func(context.Context, []byte) (domain.IntakeInspection, error) {
		return domain.IntakeInspection{Profile: "handler-test-v1", ProfileVersion: 1, Supported: true, TriangleCount: 1}, nil
	}, func() time.Time { return time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC) }, func() string { return "stage-handler-test" })
	require.NoError(t, err)
	return service
}

func minimalIntakeGLB() []byte {
	jsonChunk := []byte(`{"asset":{"version":"2.0"}}`)
	for len(jsonChunk)%4 != 0 {
		jsonChunk = append(jsonChunk, ' ')
	}
	data := make([]byte, 20+len(jsonChunk))
	binary.LittleEndian.PutUint32(data[0:4], 0x46546c67)
	binary.LittleEndian.PutUint32(data[4:8], 2)
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[12:16], uint32(len(jsonChunk)))
	binary.LittleEndian.PutUint32(data[16:20], 0x4e4f534a)
	copy(data[20:], jsonChunk)
	return data
}
