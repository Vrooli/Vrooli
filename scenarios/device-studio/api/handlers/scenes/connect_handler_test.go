package scenes

import (
	"context"
	"math"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	"device-studio/internal/physical"
	domain "device-studio/internal/scenes"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	scenesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
	scenesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
)

func TestListSceneCandidatesRejectsMalformedCursor(t *testing.T) {
	_, err := NewConnectHandler(nil).ListSceneCandidates(context.Background(), connect.NewRequest(&scenesv1.ListSceneCandidatesRequest{AfterSceneId: "bad\nID"}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
	handler := NewConnectHandler(&cursorOnlySceneService{})
	_, err = handler.ListSceneCandidates(context.Background(), connect.NewRequest(&scenesv1.ListSceneCandidatesRequest{AfterSceneId: "bad\nID"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	_, err = handler.ListSceneCandidates(context.Background(), connect.NewRequest(&scenesv1.ListSceneCandidatesRequest{AfterSceneId: strings.Repeat("a", 129)}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

type cursorOnlySceneService struct{ Service }

func (*cursorOnlySceneService) ListHeads(context.Context, string, int) ([]domain.Revision, error) {
	panic("invalid cursor must be rejected before service access")
}

func TestSceneCandidateConnectRevisionContract(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(domain.Schema)))
	repo, err := domain.NewSQLiteRevisionRepository(db)
	require.NoError(t, err)
	service, err := domain.NewRevisionService(repo, nil)
	require.NoError(t, err)
	router := mux.NewRouter()
	ModuleWithService(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := scenesconnect.NewScenesServiceClient(server.Client(), server.URL)
	candidate := validSceneCandidate()
	message := &scenesv1.SceneCandidate{}
	require.NoError(t, revisionapi.Encode(candidate, message))
	decoded, err := revisionapi.Decode[domain.SceneCandidate](message)
	require.NoError(t, err)
	_, _, _, err = domain.CanonicalizeCandidate(decoded)
	require.NoError(t, err)
	saved, err := client.SaveSceneCandidate(ctx, connect.NewRequest(&scenesv1.SaveSceneCandidateRequest{Candidate: message}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), saved.Msg.Revision.Revision)
	require.Equal(t, "candidate", saved.Msg.Revision.State)
	got, err := client.GetSceneRevision(ctx, connect.NewRequest(&scenesv1.GetSceneRevisionRequest{SceneId: candidate.SceneID, Revision: 1}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, got.Msg.Revision.Sha256)
	head, err := client.GetSceneHead(ctx, connect.NewRequest(&scenesv1.GetSceneHeadRequest{SceneId: candidate.SceneID}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, head.Msg.Revision.Sha256)
	page, err := client.ListSceneRevisions(ctx, connect.NewRequest(&scenesv1.ListSceneRevisionsRequest{SceneId: candidate.SceneID, PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Revisions, 1)
	heads, err := client.ListSceneCandidates(ctx, connect.NewRequest(&scenesv1.ListSceneCandidatesRequest{PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, heads.Msg.Candidates, 1)
	require.Equal(t, candidate.SceneID, heads.Msg.Candidates[0].SceneId)
	require.Equal(t, "candidate", heads.Msg.Candidates[0].State)
	_, err = client.SaveSceneCandidate(ctx, connect.NewRequest(&scenesv1.SaveSceneCandidateRequest{Candidate: message, ExpectedHead: 5}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeAborted, connectErr.Code())
	invalid := &scenesv1.SceneCandidate{Version: "unknown", SceneId: candidate.SceneID}
	_, err = client.SaveSceneCandidate(ctx, connect.NewRequest(&scenesv1.SaveSceneCandidateRequest{Candidate: invalid}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

func TestSaveSceneCandidateNilServiceIsInternal(t *testing.T) {
	_, err := NewConnectHandler(nil).SaveSceneCandidate(context.Background(), connect.NewRequest(&scenesv1.SaveSceneCandidateRequest{Candidate: &scenesv1.SceneCandidate{}}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
}

func TestSceneProtoFixedArraysRejectLossyLengths(t *testing.T) {
	for _, values := range [][]float64{nil, {}, {0, 0, 0}, {0, 0, 0, 0, 0}} {
		_, err := revisionapi.Decode[domain.SceneCandidate](&scenesv1.SceneCandidate{
			Version:   domain.CandidateVersion,
			Materials: []*scenesv1.SceneMaterial{{BaseColorLinearRgba: values}},
		})
		require.Error(t, err)
	}
	for _, values := range [][]float64{{1, 0, 0, 0, 0}, {1, 0, 0, 0, 0, 1, 0}} {
		_, err := revisionapi.Decode[domain.SceneCandidate](&scenesv1.SceneCandidate{
			Version:   domain.CandidateVersion,
			Materials: []*scenesv1.SceneMaterial{{Inputs: []*scenesv1.SceneMaterialInput{{AffineUv: values}}}},
		})
		require.Error(t, err)
	}
	decoded, err := revisionapi.Decode[domain.SceneCandidate](&scenesv1.SceneCandidate{
		Version:   domain.CandidateVersion,
		Materials: []*scenesv1.SceneMaterial{{BaseColorLinearRgba: []float64{0, 0, 0, 1}}},
	})
	require.NoError(t, err)
	require.Equal(t, [4]float64{0, 0, 0, 1}, decoded.Materials[0].BaseColorLinearRGBA)

	size := int64(1<<53 + 127)
	decoded, err = revisionapi.Decode[domain.SceneCandidate](&scenesv1.SceneCandidate{
		Version:   domain.CandidateVersion,
		Artifacts: []*scenesv1.SceneArtifactBinding{{Ref: &physicalv1.ArtifactRef{SizeBytes: size}}},
	})
	require.NoError(t, err)
	require.Equal(t, size, decoded.Artifacts[0].Ref.SizeBytes)
	require.Greater(t, decoded.Artifacts[0].Ref.SizeBytes, int64(math.MaxInt32))
}

func validSceneCandidate() domain.SceneCandidate {
	identity := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	return domain.SceneCandidate{Version: domain.CandidateVersion, SceneID: "api-scene", CoordinateConvention: physical.CoordinateConvention,
		Artifacts: []domain.SceneArtifactBinding{{ID: "geometry", Ref: physical.ArtifactRef{BundleID: "bundle", BundleRevision: 1, ManifestSHA256: strings.Repeat("a", 64), ReviewSHA256: strings.Repeat("b", 64), ArtifactID: "scene", Role: "geometry_glb", SizeBytes: 64, SHA256: strings.Repeat("c", 64)}, SourceUnit: "m", SourceToCanonical: &identity}},
		Parts:     []domain.ScenePart{{ID: "root", Role: "assembly", ArtifactBindingID: "geometry", NodeIndex: 0, NodeName: "root", LocalFrame: identity}}, Environment: domain.SceneEnvironment{Model: "CONSTANT_V1"}}
}
