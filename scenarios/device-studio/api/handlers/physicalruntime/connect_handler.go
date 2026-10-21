package physicalruntime

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"device-studio/internal/library"
	"device-studio/internal/physical"
	domain "device-studio/internal/physicalruntime"
	"device-studio/internal/shots"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
)

type Service interface {
	InitializeReviewedStarter(context.Context) (domain.StarterState, error)
}

type Handler struct{ service Service }

func NewConnectHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) InitializeReviewedStarter(ctx context.Context, request *connect.Request[physicalv1.InitializeReviewedStarterRequest]) (*connect.Response[physicalv1.InitializeReviewedStarterResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("physical starter service is unavailable"))
	}
	starterID := request.Msg.GetStarterId()
	if starterID != "" {
		service, ok := h.service.(interface {
			InitializePhysicalStarter(context.Context, string) (shots.PhysicalPresetRevision, error)
		})
		if !ok {
			return nil, connect.NewError(connect.CodeUnimplemented, errors.New("candidate starter selection is unavailable"))
		}
		preset, err := service.InitializePhysicalStarter(ctx, starterID)
		if errors.Is(err, domain.ErrUnknownStarterID) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("candidate starter initialization failed"))
		}
		candidateOnly := true
		if statusOwner, ok := h.service.(interface {
			PhysicalStarterCandidateOnly(context.Context, shots.PhysicalPresetRevision) (bool, error)
		}); ok {
			candidateOnly, err = statusOwner.PhysicalStarterCandidateOnly(ctx, preset)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("candidate starter status lookup failed"))
			}
		}
		return connect.NewResponse(&physicalv1.InitializeReviewedStarterResponse{
			SelectedStarterId: starterID,
			CandidateOnly:     candidateOnly,
			InitializedPreset: componentRef(preset.PresetID, preset.Revision, preset.SHA256),
		}), nil
	}
	state, err := h.service.InitializeReviewedStarter(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("reviewed starter initialization failed"))
	}
	response := &physicalv1.InitializeReviewedStarterResponse{
		PhoneDevice:      componentRef(state.Phone.DeviceID, state.Phone.Revision, state.Phone.SHA256),
		PhoneStudyScene:  componentRef(state.Scene.SceneID, state.Scene.Revision, state.Scene.SHA256),
		PhoneStillMotion: componentRef(state.Motion.MotionID, state.Motion.Revision, state.Motion.SHA256),
		CandidateOnly:    state.Phone.State == "candidate" && state.Scene.State == "candidate" && state.Motion.State == "candidate",
		PhoneStudyPreset: &physicalv1.ComponentRevisionRef{Id: state.Preset.PresetID, Revision: state.Preset.Revision, Sha256: state.Preset.SHA256},
		PhoneStudyPlacement: &physicalv1.PhysicalStarterPlacement{
			ProfileId: state.PhoneStudyPlacement.ProfileID, ProfileVersion: state.PhoneStudyPlacement.ProfileVersion,
			DisplayId: state.PhoneStudyPlacement.DisplayID, DeviceSupportId: state.PhoneStudyPlacement.DeviceSupportID,
			SceneSupportId: state.PhoneStudyPlacement.SceneSupportID, SelectedCameraId: state.PhoneStudyPlacement.SelectedCameraID,
			SceneSupportWorld: physicalTRS(state.PhoneStudyPlacement.SceneSupportWorld),
			SupportPivotM:     physicalVec3(state.PhoneStudyPlacement.SupportPivotM),
			DeviceRootRest:    physicalTRS(state.PhoneStudyPlacement.DeviceRootRest),
			Device:            componentRef(state.Phone.DeviceID, state.Phone.Revision, state.Phone.SHA256),
			Scene:             componentRef(state.Scene.SceneID, state.Scene.Revision, state.Scene.SHA256),
			Motion:            componentRef(state.Motion.MotionID, state.Motion.Revision, state.Motion.SHA256),
		},
	}
	response.ReviewedArtifacts = append(response.ReviewedArtifacts, artifactRefs(state.PhoneBundle)...)
	response.ReviewedArtifacts = append(response.ReviewedArtifacts, artifactRefs(state.SceneBundle)...)
	return connect.NewResponse(response), nil
}

func physicalVec3(value physical.Vec3) *physicalv1.Vec3 {
	return &physicalv1.Vec3{X: value.X, Y: value.Y, Z: value.Z}
}

func physicalTRS(value physical.TRS) *physicalv1.TRS {
	return &physicalv1.TRS{
		TranslationM: physicalVec3(value.Translation),
		RotationXyzw: &physicalv1.QuatXYZW{X: value.Rotation.X, Y: value.Rotation.Y, Z: value.Rotation.Z, W: value.Rotation.W},
		Scale:        physicalVec3(value.Scale),
	}
}

func componentRef(id string, revision uint32, sha string) *physicalv1.ComponentRevisionRef {
	return &physicalv1.ComponentRevisionRef{Id: id, Revision: revision, Sha256: sha}
}

func artifactRefs(bundle library.BundleRevision) []*physicalv1.ArtifactRef {
	refs := make([]*physicalv1.ArtifactRef, 0, len(bundle.Manifest.Artifacts))
	for _, artifact := range bundle.Manifest.Artifacts {
		refs = append(refs, &physicalv1.ArtifactRef{
			BundleId: bundle.Manifest.BundleID, BundleRevision: bundle.Manifest.Revision,
			ManifestSha256: bundle.ManifestSHA256, ReviewSha256: bundle.ReviewSHA256,
			ArtifactId: artifact.ID, Role: string(artifact.Role), SizeBytes: artifact.SizeBytes, Sha256: artifact.SHA256,
		})
	}
	return refs
}
