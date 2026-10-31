package scenes

import (
	"context"
	"math"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	devicedomain "device-studio/internal/devices"
	"device-studio/internal/library"
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
	candidate.Photograph = &domain.PhotographicBackground{
		ContractVersion: "hybrid-device-scene/v1", Route: "PHOTOGRAPH_ROUTE_EMPTY_SCENE",
		SourceBundleSHA256: strings.Repeat("a", 64), SourceRasterSHA256: strings.Repeat("b", 64),
		GeometryArchiveSHA256: strings.Repeat("c", 64), ScaleStatus: "estimated_monocular_scale_not_measured",
		EstimatedScaleMeters: 1.25, GeometryModel: "moge-2-vits-normal", CheckpointSHA256: strings.Repeat("d", 64),
		Postprocess:        "MoGe-2 infer focal/shift recovery and pinhole reprojection",
		RenderMode:         "SOURCE_CAMERA_PLATE_V1",
		SupportPlaneNormal: [3]float64{0, -1, 0}, SupportPlaneOffsetM: .4, SupportPlaneInliers: 1200,
		SupportPlaneResidual95: .003, CameraEnvelopeTranslationM: .02, CameraEnvelopeRotationDegrees: 1,
		OverscanReviewed: true, SourceReviewSHA256: strings.Repeat("e", 64), SourceReviewer: "photo-reviewer",
		SourceReviewRationale: "Reviewed empty plate and support region", SourceReviewEvidence: []string{"evidence:sha256:" + strings.Repeat("f", 64)},
		SourceUseClaimSHA256: strings.Repeat("9", 64), LocalRenderAuthorized: true, SourceUseRationale: "Local photographic compositing is permitted.",
	}
	message := &scenesv1.SceneCandidate{}
	require.NoError(t, revisionapi.Encode(candidate, message))
	require.NotNil(t, message.PhotographicBackground)
	decoded, err := revisionapi.Decode[domain.SceneCandidate](message)
	require.NoError(t, err)
	require.Equal(t, candidate.Photograph, decoded.Photograph)
	_, _, _, err = domain.CanonicalizeCandidate(decoded)
	require.NoError(t, err)
	saved, err := client.SaveSceneCandidate(ctx, connect.NewRequest(&scenesv1.SaveSceneCandidateRequest{Candidate: message}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), saved.Msg.Revision.Revision)
	require.Equal(t, "candidate", saved.Msg.Revision.State)
	got, err := client.GetSceneRevision(ctx, connect.NewRequest(&scenesv1.GetSceneRevisionRequest{SceneId: candidate.SceneID, Revision: 1}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, got.Msg.Revision.Sha256)
	storedCandidate, err := revisionapi.Decode[domain.SceneCandidate](got.Msg.Revision.Candidate)
	require.NoError(t, err)
	require.Equal(t, candidate.Photograph, storedCandidate.Photograph, "photo source provenance must survive SceneCandidate save/get")
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

type preparedPhotoReadStub struct{ called bool }

func (s *preparedPhotoReadStub) PreparedPhotograph(context.Context, string) ([]byte, error) {
	s.called = true
	return nil, nil
}

type preparedPhotoPublisherStub struct{}

func (preparedPhotoPublisherStub) Publish(context.Context, library.PublishInput) (library.BundleRevision, error) {
	return library.BundleRevision{}, nil
}

func TestImportPreparedPhotographRequiresExactExpectedBundleHashBeforeOwnerRead(t *testing.T) {
	source := &preparedPhotoReadStub{}
	handler := NewConnectHandlerWithPreparedPhotographs(&cursorOnlySceneService{}, source, preparedPhotoPublisherStub{})
	_, err := handler.ImportPreparedPhotograph(context.Background(), connect.NewRequest(&scenesv1.ImportPreparedPhotographRequest{
		PreparationId: strings.Repeat("a", 64), ExpectedBundleSha256: strings.Repeat("b", 64), LicenseId: "license", Attribution: "source", SourceLabel: "plate",
	}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	require.False(t, source.called, "invalid expected SHA must be rejected before contacting Image Tools")
}

type poseFitSceneService struct {
	Service
	revision domain.Revision
}

type poseFitDeviceResolver struct {
	DeviceRevisionResolver
	revision devicedomain.DeviceRevision
}

func (s poseFitDeviceResolver) GetRevision(context.Context, string, uint32) (devicedomain.DeviceRevision, error) {
	return s.revision, nil
}

func (s poseFitSceneService) GetRevision(context.Context, string, uint32) (domain.Revision, error) {
	return s.revision, nil
}

func TestFitKnownCameraPoseResolvesImmutableCameraAndKeepsHingeSeparate(t *testing.T) {
	identity := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	cameraPose := identity
	cameraPose.Translation.Z = 4
	focalMM, sensorMM := 50.0, 36.0
	candidate := validSceneCandidate()
	candidate.CameraFrames = []domain.SceneCameraFrame{{ID: "known", Frame: cameraPose, Projection: domain.ProjectionPerspective, FocalLengthMM: &focalMM, SensorWidthMM: &sensorMM, SensorFit: domain.SensorFitHorizontal}}
	rev := domain.Revision{SceneID: candidate.SceneID, Revision: 2, Candidate: candidate}
	points := []physical.Vec3{{X: -.4, Y: -.25, Z: -.1}, {X: .4, Y: -.25, Z: -.1}, {X: .4, Y: .25, Z: -.1}, {X: -.4, Y: .25, Z: -.1}, {X: -.4, Y: -.25, Z: .1}, {X: .4, Y: -.25, Z: .1}, {X: .4, Y: .25, Z: .1}, {X: -.4, Y: .25, Z: .1}}
	landmarks := make([]*scenesv1.DeviceFitLandmark, 0, len(points))
	fx := 1280 * focalMM / sensorMM
	for i, p := range points {
		wx, wy, wz := p.X+.12, p.Y-.08, p.Z+.15
		depth := 4 - wz
		landmarks = append(landmarks, &scenesv1.DeviceFitLandmark{Id: string(rune('a' + i)), DeviceLocalM: &physicalv1.Vec3{X: p.X, Y: p.Y, Z: p.Z}, PixelX: 640 + fx*wx/depth, PixelY: 360 - fx*wy/depth})
	}
	request := &scenesv1.FitKnownCameraPoseRequest{SceneId: candidate.SceneID, SceneRevision: 2, CameraFrameId: "known", GeometryAssetId: "laptop-neutral-rig.glb", GeometrySha256: strings.Repeat("a", 64), DeviceId: "laptop", DeviceRevision: 1, DeviceRevisionSha256: strings.Repeat("d", 64), RasterWidth: 1280, RasterHeight: 720, Landmarks: landmarks, InitialPoseWorld: &physicalv1.TRS{TranslationM: &physicalv1.Vec3{X: .08, Y: -.04, Z: .1}, RotationXyzw: &physicalv1.QuatXYZW{W: 1}, Scale: &physicalv1.Vec3{X: 1, Y: 1, Z: 1}}, HingeJointId: "lid-hinge", HingeAngleRadians: .55, HingeRule: "held-fixed-before-rigid-fit", MaxRmsPixels: .1, MaxPointPixels: .2}
	angleMin, angleMax := -1.0, 1.0
	deviceRev := devicedomain.DeviceRevision{DeviceID: "laptop", Revision: 1, SHA256: strings.Repeat("d", 64), Candidate: devicedomain.DeviceCandidate{Geometry: physical.ArtifactRef{ArtifactID: "laptop-neutral-rig.glb", SHA256: strings.Repeat("a", 64)}, Definition: devicedomain.Definition{Joints: []devicedomain.Joint{{ID: "lid-hinge", LowerRadians: &angleMin, UpperRadians: &angleMax}}}}}
	response, err := NewConnectHandler(poseFitSceneService{revision: rev}, poseFitDeviceResolver{revision: deviceRev}).FitKnownCameraPose(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Less(t, response.Msg.RmsPixels, 0.01)
	require.Equal(t, "lid-hinge", response.Msg.HingeJointId)
	require.Equal(t, .55, response.Msg.HingeAngleRadians)
	require.Equal(t, uint32(2), response.Msg.SceneRevision)
	require.Equal(t, "known", response.Msg.CameraFrameId)
	require.Equal(t, "laptop-neutral-rig.glb", response.Msg.GeometryAssetId)
	require.Equal(t, "laptop", response.Msg.DeviceId)
	require.Equal(t, uint32(1), response.Msg.DeviceRevision)
	require.Equal(t, strings.Repeat("d", 64), response.Msg.DeviceRevisionSha256)
	require.Len(t, response.Msg.Residuals, len(points))
	require.Contains(t, response.Msg.OverlaySvg, "#ef476f")
	_, connectHandler := scenesconnect.NewScenesServiceHandler(NewConnectHandler(poseFitSceneService{revision: rev}, poseFitDeviceResolver{revision: deviceRev}))
	rpcRouter := mux.NewRouter()
	rpcRouter.PathPrefix("/").Handler(connectHandler)
	rpcServer := httptest.NewServer(rpcRouter)
	defer rpcServer.Close()
	rpcClient := scenesconnect.NewScenesServiceClient(rpcServer.Client(), rpcServer.URL)
	rpcResponse, err := rpcClient.FitKnownCameraPose(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Less(t, rpcResponse.Msg.RmsPixels, .01)
	require.Equal(t, uint32(1280), rpcResponse.Msg.RasterWidth)
	require.Equal(t, "known", rpcResponse.Msg.CameraFrame.Id)
	bad := *request
	bad.GeometrySha256 = strings.Repeat("b", 64)
	_, err = NewConnectHandler(poseFitSceneService{revision: rev}, poseFitDeviceResolver{revision: deviceRev}).FitKnownCameraPose(context.Background(), connect.NewRequest(&bad))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	bad = *request
	bad.HingeAngleRadians = 2
	_, err = NewConnectHandler(poseFitSceneService{revision: rev}, poseFitDeviceResolver{revision: deviceRev}).FitKnownCameraPose(context.Background(), connect.NewRequest(&bad))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
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
