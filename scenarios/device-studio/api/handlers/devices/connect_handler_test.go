package devices

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/devices"
	"device-studio/internal/physical"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
)

func TestDeviceCandidateConnectRevisionContract(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(domain.Schema)))
	repo := domain.NewSQLiteDeviceRevisionRepository(db)
	service, err := domain.NewDeviceRevisionService(repo, nil)
	require.NoError(t, err)
	router := mux.NewRouter()
	ModuleWithService(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := devicesconnect.NewDevicesServiceClient(server.Client(), server.URL)
	candidate := validDeviceCandidate()
	primitiveIndex, materialIndex := uint32(1), uint32(0)
	candidate.DisplayBindings[0].ScreenPrimitiveIndex = &primitiveIndex
	candidate.DisplayBindings[0].ScreenMaterialIndex = &materialIndex
	message := &devicesv1.DeviceCandidate{}
	require.NoError(t, revisionapi.Encode(candidate, message))
	decoded, err := revisionapi.Decode[domain.DeviceCandidate](message)
	require.NoError(t, err)
	_, _, _, err = domain.CanonicalizeDeviceCandidate(decoded)
	require.NoError(t, err)
	saved, err := client.SaveDeviceCandidate(ctx, connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: message}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), saved.Msg.Revision.Revision)
	require.Equal(t, "candidate", saved.Msg.Revision.State)
	got, err := client.GetDeviceRevision(ctx, connect.NewRequest(&devicesv1.GetDeviceRevisionRequest{DeviceId: candidate.Definition.ID, Revision: 1}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, got.Msg.Revision.Sha256)
	gotBinding := got.Msg.Revision.Candidate.DisplayBindings[0]
	require.NotNil(t, gotBinding.ScreenPrimitiveIndex)
	require.NotNil(t, gotBinding.ScreenMaterialIndex, "optional zero-valued material index must retain presence")
	require.Equal(t, primitiveIndex, gotBinding.GetScreenPrimitiveIndex())
	require.Equal(t, materialIndex, gotBinding.GetScreenMaterialIndex())
	head, err := client.GetDeviceHead(ctx, connect.NewRequest(&devicesv1.GetDeviceHeadRequest{DeviceId: candidate.Definition.ID}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, head.Msg.Revision.Sha256)
	page, err := client.ListDeviceRevisions(ctx, connect.NewRequest(&devicesv1.ListDeviceRevisionsRequest{DeviceId: candidate.Definition.ID, PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Revisions, 1)
	changed := candidate
	changed.Definition.Description = "immutable head two"
	changedMessage := &devicesv1.DeviceCandidate{}
	require.NoError(t, revisionapi.Encode(changed, changedMessage))
	second, err := client.SaveDeviceCandidate(ctx, connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: changedMessage, ExpectedHead: 1}))
	require.NoError(t, err)
	other := validDeviceCandidate()
	other.Definition.ID = "z-device"
	otherMessage := &devicesv1.DeviceCandidate{}
	require.NoError(t, revisionapi.Encode(other, otherMessage))
	_, err = client.SaveDeviceCandidate(ctx, connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: otherMessage}))
	require.NoError(t, err)
	firstHeads, err := client.ListDeviceCandidates(ctx, connect.NewRequest(&devicesv1.ListDeviceCandidatesRequest{PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, firstHeads.Msg.Candidates, 1)
	require.Equal(t, uint32(2), firstHeads.Msg.Candidates[0].Revision, "catalog returns only current head per candidate")
	require.Equal(t, second.Msg.Revision.Sha256, firstHeads.Msg.Candidates[0].Sha256)
	require.Equal(t, "api-device", firstHeads.Msg.NextAfterDeviceId)
	nextHeads, err := client.ListDeviceCandidates(ctx, connect.NewRequest(&devicesv1.ListDeviceCandidatesRequest{AfterDeviceId: firstHeads.Msg.NextAfterDeviceId, PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, nextHeads.Msg.Candidates, 1)
	require.Equal(t, "z-device", nextHeads.Msg.Candidates[0].DeviceId)
	_, err = client.SaveDeviceCandidate(ctx, connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: message, ExpectedHead: 4}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeAborted, connectErr.Code())
	_, err = client.SaveDeviceCandidate(ctx, connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: &devicesv1.DeviceCandidate{Version: "unknown"}}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	invalidGraph := validDeviceCandidate()
	invalidGraph.Parts = append(invalidGraph.Parts, invalidGraph.Parts[0])
	invalidMessage := &devicesv1.DeviceCandidate{}
	require.NoError(t, revisionapi.Encode(invalidGraph, invalidMessage))
	_, err = client.SaveDeviceCandidate(ctx, connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: invalidMessage}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

func TestSaveDeviceCandidateNilServiceIsInternal(t *testing.T) {
	_, err := NewConnectHandler(nil).SaveDeviceCandidate(context.Background(), connect.NewRequest(&devicesv1.SaveDeviceCandidateRequest{Candidate: &devicesv1.DeviceCandidate{}}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
}

type qualificationReviewerStub struct {
	request      domain.QualificationReviewRequest
	result       domain.QualificationReviewResult
	packageBytes []byte
	imported     []byte
	head         uint32
}

func (s *qualificationReviewerStub) ReviewCandidate(_ context.Context, request domain.QualificationReviewRequest) (domain.QualificationReviewResult, error) {
	s.request = request
	return s.result, nil
}

func (s *qualificationReviewerStub) ExportReviewPackage(_ context.Context, projectID, deviceID string, revision uint32) ([]byte, error) {
	s.request = domain.QualificationReviewRequest{ProjectID: projectID, DeviceID: deviceID, CandidateRevision: revision}
	return s.packageBytes, nil
}

func (s *qualificationReviewerStub) ImportReviewPackage(_ context.Context, data []byte, expectedHead uint32) (domain.QualificationReviewResult, error) {
	s.imported, s.head = append([]byte(nil), data...), expectedHead
	return s.result, nil
}

func TestQualificationPackageConnectRoutesPreserveBytesAndExactSelectors(t *testing.T) {
	stub := &qualificationReviewerStub{packageBytes: []byte(`{"version":1}`), result: domain.QualificationReviewResult{Revision: domain.DeviceRevision{DeviceID: "device", Revision: 4, State: domain.DeviceRevisionQualified}}}
	handler := NewConnectHandler(&cursorOnlyDeviceService{}, stub)
	exported, err := handler.ExportQualificationPackage(context.Background(), connect.NewRequest(&devicesv1.ExportQualificationPackageRequest{ProjectId: "project", DeviceId: "device", QualifiedRevision: 3}))
	require.NoError(t, err)
	require.Equal(t, stub.packageBytes, exported.Msg.PackageJson)
	require.Equal(t, domain.QualificationReviewRequest{ProjectID: "project", DeviceID: "device", CandidateRevision: 3}, stub.request)
	imported, err := handler.ImportQualificationPackage(context.Background(), connect.NewRequest(&devicesv1.ImportQualificationPackageRequest{PackageJson: stub.packageBytes, ExpectedHead: 2}))
	require.NoError(t, err)
	require.Equal(t, stub.packageBytes, stub.imported)
	require.Equal(t, uint32(2), stub.head)
	require.Equal(t, uint32(4), imported.Msg.Revision.Revision)
}

func TestReviewDeviceCandidateConnectPreservesExactBindingsAndPresetResult(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(domain.Schema)))
	service, err := domain.NewDeviceRevisionService(domain.NewSQLiteDeviceRevisionRepository(db), nil)
	require.NoError(t, err)
	digest := strings.Repeat("a", 64)
	review := domain.QualificationReview{Scope: "physical_geometry", Result: "passed", Reviewer: "operator", Evidence: []domain.Evidence{{Source: "job-1", Kind: "human_review"}}, CandidateRevision: 1, CandidateSHA256: digest,
		SourceBundleID: "bundle", SourceBundleRevision: 2, SourceManifestSHA256: digest, SourceReviewSHA256: digest, SourceArtifactID: "device", SourceArtifactSHA256: digest, SourceAllowedUses: []string{"render"}, DimensionStates: []string{"closed"}, DisplayIDs: []string{"front"}, AllowedMotionIDs: []string{"static"}, SupportIDs: []string{"none"},
		PresetID: "preset", PresetRevision: 3, PresetSHA256: digest, SceneID: "scene", SceneRevision: 4, SceneSHA256: digest, MotionID: "motion", MotionRevision: 5, MotionSHA256: digest, NativeRenderJobID: "render-job", NativeOutputSHA256: digest, NativeOutputKind: "physical_still_png", RightsClaimsAcknowledged: true, ReviewedAt: "2026-10-21T00:00:00Z"}
	wantRequest := domain.QualificationReviewRequest{ProjectID: "project", DeviceID: "device", CandidateRevision: 1, CandidateSHA256: digest, ExpectedHead: 1, Review: review}
	reviewer := &qualificationReviewerStub{request: domain.QualificationReviewRequest{}, result: domain.QualificationReviewResult{Revision: domain.DeviceRevision{DeviceID: "device", Revision: 2, SHA256: "b" + strings.Repeat("a", 63), State: domain.DeviceRevisionQualified}, PresetID: "preset", PresetRevision: 4, PresetSHA256: "c" + strings.Repeat("a", 63)}}
	router := mux.NewRouter()
	ModuleWithService(service, reviewer).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := devicesconnect.NewDevicesServiceClient(server.Client(), server.URL)
	message := &devicesv1.QualificationReview{}
	require.NoError(t, revisionapi.Encode(review, message))
	response, err := client.ReviewDeviceCandidate(ctx, connect.NewRequest(&devicesv1.ReviewDeviceCandidateRequest{ProjectId: wantRequest.ProjectID, DeviceId: wantRequest.DeviceID, CandidateRevision: wantRequest.CandidateRevision, CandidateSha256: wantRequest.CandidateSHA256, ExpectedHead: wantRequest.ExpectedHead, Review: message}))
	require.NoError(t, err)
	require.Equal(t, wantRequest, reviewer.request)
	require.Equal(t, "qualified", response.Msg.Revision.State)
	require.Equal(t, "preset", response.Msg.PresetId)
	require.Equal(t, uint32(4), response.Msg.PresetRevision)
	require.Equal(t, "c"+strings.Repeat("a", 63), response.Msg.PresetSha256)
}

func TestListDeviceCandidatesRejectsMalformedCursor(t *testing.T) {
	handler := NewConnectHandler(nil)
	_, err := handler.ListDeviceCandidates(context.Background(), connect.NewRequest(&devicesv1.ListDeviceCandidatesRequest{AfterDeviceId: "bad\nID"}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code(), "availability is checked before cursor validation")

	handler = NewConnectHandler(&cursorOnlyDeviceService{})
	_, err = handler.ListDeviceCandidates(context.Background(), connect.NewRequest(&devicesv1.ListDeviceCandidatesRequest{AfterDeviceId: "bad\nID"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	_, err = handler.ListDeviceCandidates(context.Background(), connect.NewRequest(&devicesv1.ListDeviceCandidatesRequest{AfterDeviceId: strings.Repeat("a", 129)}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

type cursorOnlyDeviceService struct{ Service }

func (*cursorOnlyDeviceService) ListHeads(context.Context, string, int) ([]domain.DeviceRevision, error) {
	panic("invalid cursor must be rejected before service access")
}

func validDeviceCandidate() domain.DeviceCandidate {
	unknown := func() domain.Measurement {
		return domain.Measurement{Unit: "mm", Method: "unknown", Uncertainty: domain.Uncertainty{Status: "unknown"}}
	}
	return domain.DeviceCandidate{
		Version:    domain.DeviceCandidateVersion,
		Definition: domain.Definition{SchemaVersion: "device-definition/v1", ID: "api-device", Group: domain.GroupPhone, Kind: "fixture", IdentityStatus: domain.IdentityGeneric, Model: "Fixture", Qualification: "unqualified", Dimensions: map[string]domain.Dimensions{"default": {Width: unknown(), Height: unknown(), Depth: unknown()}}, Displays: []domain.DisplaySurface{{ID: "front", Topology: "planar", PhysicalBoundary: domain.Region{Status: "unknown"}, ActiveArea: domain.Region{Status: "unknown"}, AppSafeArea: domain.Region{Status: "unknown"}}}, Joints: []domain.Joint{}, SupportedPoses: []domain.Pose{}, SupportedMotion: []domain.Capability{}, Supports: []domain.SupportCapability{}, ModelCandidates: []domain.ModelCandidate{}, Provenance: []domain.Evidence{}},
		Geometry:   physical.ArtifactRef{BundleID: "bundle", BundleRevision: 1, ManifestSHA256: digest('a'), ReviewSHA256: digest('b'), ArtifactID: "geometry", Role: "geometry_glb", SizeBytes: 64, SHA256: digest('c')}, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical: physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		Parts:             []domain.SemanticPart{{ID: "body", Role: "assembly", NodeIndex: 0, NodeName: "root", LocalFrame: physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}}}, DisplayBindings: []domain.DisplayPartBinding{{DisplayID: "front", PartID: "body", LocalFrame: physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}}}, SupportFrames: []domain.DeviceSupportFrame{},
	}
}
func digest(ch byte) string { return strings.Repeat(string(ch), 64) }
