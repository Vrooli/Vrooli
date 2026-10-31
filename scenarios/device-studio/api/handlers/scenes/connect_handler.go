package scenes

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	"device-studio/internal/devicefit"
	devicedomain "device-studio/internal/devices"
	"device-studio/internal/photographic"
	"device-studio/internal/physical"
	domain "device-studio/internal/scenes"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	scenesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
)

type Service interface {
	SaveCandidate(context.Context, domain.SceneCandidate, uint32) (domain.Revision, error)
	GetRevision(context.Context, string, uint32) (domain.Revision, error)
	GetHead(context.Context, string) (domain.Revision, error)
	ListRevisions(context.Context, string, uint32, int) ([]domain.Revision, error)
	ListHeads(context.Context, string, int) ([]domain.Revision, error)
}

type DeviceRevisionResolver interface {
	GetRevision(context.Context, string, uint32) (devicedomain.DeviceRevision, error)
}

type PreparedPhotographSource interface {
	PreparedPhotograph(context.Context, string) ([]byte, error)
}

type Handler struct {
	service         Service
	devices         DeviceRevisionResolver
	preparedSource  PreparedPhotographSource
	preparedLibrary photographic.Publisher
	preparedScenes  photographic.SceneSaver
}

func NewConnectHandler(service Service, devices ...DeviceRevisionResolver) *Handler {
	h := &Handler{service: service}
	if len(devices) > 0 {
		h.devices = devices[0]
	}
	return h
}

func NewConnectHandlerWithPreparedPhotographs(service Service, source PreparedPhotographSource, libraryOwner photographic.Publisher, devices ...DeviceRevisionResolver) *Handler {
	h := NewConnectHandler(service, devices...)
	h.preparedSource, h.preparedLibrary, h.preparedScenes = source, libraryOwner, service
	return h
}

func (h *Handler) ImportPreparedPhotograph(ctx context.Context, req *connect.Request[scenesv1.ImportPreparedPhotographRequest]) (*connect.Response[scenesv1.ImportPreparedPhotographResponse], error) {
	if h.preparedSource == nil || h.preparedLibrary == nil || h.preparedScenes == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("prepared photograph import is unavailable"))
	}
	in := req.Msg
	if !validPhotoSHA(in.PreparationId) || !validPhotoSHA(in.ExpectedBundleSha256) || in.PreparationId != in.ExpectedBundleSha256 || strings.TrimSpace(in.LicenseId) == "" || strings.TrimSpace(in.Attribution) == "" || strings.TrimSpace(in.SourceLabel) == "" || len(in.LicenseId) > 128 || len(in.Attribution) > 1024 || len(in.SourceLabel) > 512 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("preparation id, matching expected bundle SHA-256, and bounded explicit source claims are required"))
	}
	raw, err := h.preparedSource.PreparedPhotograph(ctx, in.PreparationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Image Tools preparation identity or retained bytes could not be verified"))
	}
	result, err := photographic.PublishCandidate(ctx, raw, photographic.PublishClaims{LicenseID: in.LicenseId, Attribution: in.Attribution, SourceLabel: in.SourceLabel, AllowLocalRender: in.AllowLocalRender, SourceUseRationale: in.SourceUseRationale}, h.preparedLibrary, h.preparedScenes)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("prepared photograph did not satisfy the edit-only scene import contract"))
	}
	revision := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(result.Scene, revision); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.ImportPreparedPhotographResponse{Revision: revision, LibraryBundleId: result.Library.Manifest.BundleID, LibraryRevision: result.Library.Manifest.Revision, LibraryManifestSha256: result.Library.ManifestSHA256, ScaleStatus: result.Conversion.ScaleStatus, EstimatedScaleM: result.Conversion.EstimatedScaleMeters}), nil
}

func validPhotoSHA(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func (h *Handler) SaveSceneCandidate(ctx context.Context, req *connect.Request[scenesv1.SaveSceneCandidateRequest]) (*connect.Response[scenesv1.SaveSceneCandidateResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if req.Msg.Candidate == nil {
		return nil, revisionapi.InvalidCandidate()
	}
	candidate, err := revisionapi.Decode[domain.SceneCandidate](req.Msg.Candidate)
	if err != nil {
		return nil, revisionapi.InvalidCandidate()
	}
	revision, err := h.service.SaveCandidate(ctx, candidate, req.Msg.ExpectedHead)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.SaveSceneCandidateResponse{Revision: out}), nil
}

func (h *Handler) GetSceneRevision(ctx context.Context, req *connect.Request[scenesv1.GetSceneRevisionRequest]) (*connect.Response[scenesv1.GetSceneRevisionResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetRevision(ctx, req.Msg.SceneId, req.Msg.Revision)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.GetSceneRevisionResponse{Revision: out}), nil
}

func (h *Handler) GetSceneHead(ctx context.Context, req *connect.Request[scenesv1.GetSceneHeadRequest]) (*connect.Response[scenesv1.GetSceneHeadResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetHead(ctx, req.Msg.SceneId)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.GetSceneHeadResponse{Revision: out}), nil
}

func (h *Handler) ListSceneRevisions(ctx context.Context, req *connect.Request[scenesv1.ListSceneRevisionsRequest]) (*connect.Response[scenesv1.ListSceneRevisionsResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.service.ListRevisions(ctx, req.Msg.SceneId, req.Msg.AfterRevision, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.ListSceneRevisionsResponse{Revisions: make([]*scenesv1.SceneRevision, 0, len(rows))}
	for _, row := range rows {
		item := &scenesv1.SceneRevision{}
		if err := revisionapi.Encode(row, item); err != nil {
			return nil, revisionapi.Internal()
		}
		out.Revisions = append(out.Revisions, item)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterRevision = rows[len(rows)-1].Revision
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ListSceneCandidates(ctx context.Context, req *connect.Request[scenesv1.ListSceneCandidatesRequest]) (*connect.Response[scenesv1.ListSceneCandidatesResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if err := revisionapi.ValidateCursorID(req.Msg.AfterSceneId); err != nil {
		return nil, err
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.service.ListHeads(ctx, req.Msg.AfterSceneId, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.ListSceneCandidatesResponse{Candidates: make([]*scenesv1.SceneRevision, 0, len(rows))}
	for _, row := range rows {
		item := &scenesv1.SceneRevision{}
		if err := revisionapi.Encode(row, item); err != nil {
			return nil, revisionapi.Internal()
		}
		out.Candidates = append(out.Candidates, item)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterSceneId = rows[len(rows)-1].SceneID
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) FitKnownCameraPose(ctx context.Context, req *connect.Request[scenesv1.FitKnownCameraPoseRequest]) (*connect.Response[scenesv1.FitKnownCameraPoseResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	m := req.Msg
	if h.devices == nil {
		return nil, revisionapi.Internal()
	}
	deviceRevision, err := h.devices.GetRevision(ctx, m.DeviceId, m.DeviceRevision)
	if err != nil {
		if errors.Is(err, devicedomain.ErrRevisionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("device revision not found"))
		}
		return nil, revisionapi.Internal()
	}
	if deviceRevision.DeviceID != m.DeviceId || deviceRevision.Revision != m.DeviceRevision {
		return nil, revisionapi.Internal()
	}
	if deviceRevision.SHA256 != m.DeviceRevisionSha256 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("device_revision_sha256 does not identify the selected immutable device revision"))
	}
	if deviceRevision.Candidate.Geometry.ArtifactID != m.GeometryAssetId || deviceRevision.Candidate.Geometry.SHA256 != m.GeometrySha256 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("geometry asset ID/SHA-256 does not match the immutable device revision"))
	}
	hingeFound := false
	for _, joint := range deviceRevision.Candidate.Definition.Joints {
		if joint.ID != m.HingeJointId {
			continue
		}
		hingeFound = true
		if joint.LowerRadians != nil && m.HingeAngleRadians < *joint.LowerRadians || joint.UpperRadians != nil && m.HingeAngleRadians > *joint.UpperRadians {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("hinge angle assumption is outside the selected immutable device joint limits"))
		}
		break
	}
	if !hingeFound {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("hinge_joint_id is not present in the selected immutable device revision"))
	}
	revision, err := h.service.GetRevision(ctx, m.SceneId, m.SceneRevision)
	if err != nil {
		return nil, mapError(err)
	}
	if revision.SceneID != m.SceneId || revision.Revision != m.SceneRevision {
		return nil, revisionapi.Internal()
	}
	var camera *domain.SceneCameraFrame
	for i := range revision.Candidate.CameraFrames {
		if revision.Candidate.CameraFrames[i].ID == m.CameraFrameId {
			camera = &revision.Candidate.CameraFrames[i]
			break
		}
	}
	if camera == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("camera_frame_id is not present in the selected immutable scene revision"))
	}
	if camera.Projection != domain.ProjectionPerspective || camera.SensorFit != domain.SensorFitHorizontal || camera.FocalLengthMM == nil || camera.SensorWidthMM == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pose fit requires a perspective camera with explicit focal length, sensor width, and horizontal sensor fit"))
	}
	if m.InitialPoseWorld == nil || m.InitialPoseWorld.TranslationM == nil || m.InitialPoseWorld.RotationXyzw == nil || m.InitialPoseWorld.Scale == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initial_pose_world requires translation, rotation and scale"))
	}
	fit := devicefit.Request{Camera: devicefit.Camera{WorldFrame: camera.Frame, RasterWidth: m.RasterWidth, RasterHeight: m.RasterHeight, FocalLengthMM: *camera.FocalLengthMM, SensorWidthMM: *camera.SensorWidthMM}, InitialPose: decodeTRS(m.InitialPoseWorld), Hinge: devicefit.HingeAssumption{JointID: m.HingeJointId, AngleRadians: m.HingeAngleRadians, Rule: m.HingeRule}, MaxRMSPixels: m.MaxRmsPixels, MaxPointPixels: m.MaxPointPixels, Landmarks: make([]devicefit.Landmark, 0, len(m.Landmarks))}
	fit.SceneID, fit.SceneRevision, fit.CameraFrameID = m.SceneId, m.SceneRevision, m.CameraFrameId
	fit.DeviceID, fit.DeviceRevision, fit.DeviceRevisionSHA256 = m.DeviceId, m.DeviceRevision, m.DeviceRevisionSha256
	fit.GeometryAssetID, fit.GeometrySHA256 = m.GeometryAssetId, m.GeometrySha256
	fit.GeometryAssetID, fit.GeometrySHA256 = m.GeometryAssetId, m.GeometrySha256
	for _, landmark := range m.Landmarks {
		if landmark == nil || landmark.DeviceLocalM == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("every landmark requires a 3D device-local point"))
		}
		fit.Landmarks = append(fit.Landmarks, devicefit.Landmark{ID: landmark.Id, LocalM: physical.Vec3{X: landmark.DeviceLocalM.X, Y: landmark.DeviceLocalM.Y, Z: landmark.DeviceLocalM.Z}, Pixel: devicefit.Pixel{X: landmark.PixelX, Y: landmark.PixelY}})
	}
	result, err := devicefit.Fit(fit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	cameraMessage := &scenesv1.SceneCameraFrame{}
	if err := revisionapi.Encode(*camera, cameraMessage); err != nil {
		return nil, revisionapi.Internal()
	}
	out := &scenesv1.FitKnownCameraPoseResponse{Profile: result.Version, FittedPoseWorld: encodeTRS(result.FittedPose), InitialPoseWorld: encodeTRS(result.InitialPose), HingeJointId: result.Hinge.JointID, HingeAngleRadians: result.Hinge.AngleRadians, HingeRule: result.Hinge.Rule, RmsPixels: result.RMSPixels, MaxPixels: result.MaxPixels, OverlaySvg: result.OverlaySVG, Iterations: uint32(result.Iterations), SceneId: result.SceneID, SceneRevision: result.SceneRevision, CameraFrameId: result.CameraFrameID, GeometryAssetId: result.GeometryAssetID, GeometrySha256: result.GeometrySHA256, DeviceId: result.DeviceID, DeviceRevision: result.DeviceRevision, DeviceRevisionSha256: result.DeviceRevisionSHA256, CameraFrame: cameraMessage, RasterWidth: m.RasterWidth, RasterHeight: m.RasterHeight, Residuals: make([]*scenesv1.DeviceFitResidual, 0, len(result.Residuals))}
	for _, r := range result.Residuals {
		out.Residuals = append(out.Residuals, &scenesv1.DeviceFitResidual{LandmarkId: r.LandmarkID, ObservedX: r.Observed.X, ObservedY: r.Observed.Y, ProjectedX: r.Projected.X, ProjectedY: r.Projected.Y, Dx: r.DX, Dy: r.DY, Magnitude: r.Magnitude})
	}
	return connect.NewResponse(out), nil
}

func decodeTRS(t *physicalv1.TRS) physical.TRS {
	if t == nil || t.TranslationM == nil || t.RotationXyzw == nil || t.Scale == nil {
		return physical.TRS{}
	}
	return physical.TRS{Translation: physical.Vec3{X: t.TranslationM.X, Y: t.TranslationM.Y, Z: t.TranslationM.Z}, Rotation: physical.QuatXYZW{X: t.RotationXyzw.X, Y: t.RotationXyzw.Y, Z: t.RotationXyzw.Z, W: t.RotationXyzw.W}, Scale: physical.Vec3{X: t.Scale.X, Y: t.Scale.Y, Z: t.Scale.Z}}
}

func encodeTRS(t physical.TRS) *physicalv1.TRS {
	return &physicalv1.TRS{TranslationM: &physicalv1.Vec3{X: t.Translation.X, Y: t.Translation.Y, Z: t.Translation.Z}, RotationXyzw: &physicalv1.QuatXYZW{X: t.Rotation.X, Y: t.Rotation.Y, Z: t.Rotation.Z, W: t.Rotation.W}, Scale: &physicalv1.Vec3{X: t.Scale.X, Y: t.Scale.Y, Z: t.Scale.Z}}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCandidate):
		return revisionapi.InvalidCandidate()
	case errors.Is(err, domain.ErrRevisionNotFound):
		return revisionapi.NotFound()
	case errors.Is(err, domain.ErrRevisionConflict):
		return revisionapi.Conflict()
	default:
		return revisionapi.Internal()
	}
}
