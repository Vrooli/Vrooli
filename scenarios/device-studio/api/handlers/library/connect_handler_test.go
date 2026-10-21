package library

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/library"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
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
		Review:         domain.ReviewBinding{ID: "review-1", ReviewerID: "reviewer", ReviewedAt: now, Decision: "approved", ManifestSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", AllowedUses: []domain.Use{domain.UseRender}},
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
	require.Equal(t, []string{"render"}, metadata.Review.AllowedUses)
	encoded, err := protojson.Marshal(metadata)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "blob_key")
	require.NotContains(t, string(encoded), "storage_key")
	got, err := client.GetBundleRevision(context.Background(), connect.NewRequest(&v1.GetBundleRevisionRequest{BundleId: "phone-asset", Revision: 3}))
	require.NoError(t, err)
	require.Equal(t, metadata.ManifestSha256, got.Msg.Revision.ManifestSha256)
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

type correctionIntakeStub struct{ candidate domain.IntakeCandidate }

func (s correctionIntakeStub) Get(context.Context, string, uint32) (domain.IntakeCandidate, error) {
	return s.candidate, nil
}

func (correctionIntakeStub) List(context.Context, string, int) ([]domain.IntakeCandidate, error) {
	return nil, nil
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
	candidate := domain.IntakeCandidate{
		ID:       "creation-candidate",
		Revision: 3,
		Creation: &domain.CreationLineage{CorrectionSource: correction},
	}
	response, err := NewConnectHandlerWithIntake(nil, correctionIntakeStub{candidate: candidate}).GetIntakeCandidate(
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
