package shots

import (
	domain "device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func shotToProto(value domain.Shot) *shotsv1.Shot {
	return &shotsv1.Shot{Id: value.ID, ProjectId: value.ProjectID, Name: value.Name, CreatedAt: timestamppb.New(value.CreatedAt.UTC()), UpdatedAt: timestamppb.New(value.UpdatedAt.UTC()), HeadRevision: value.HeadRevision}
}

func revisionToProto(value domain.Revision) *shotsv1.ShotRevision {
	return &shotsv1.ShotRevision{
		ShotId: value.ShotID, Revision: value.Revision, CreatedAt: timestamppb.New(value.CreatedAt.UTC()), Name: value.Name,
		Snapshot: value.Snapshot, RecipeSha256: value.RecipeSHA256, SceneSha256: value.SceneSHA256, PresentationSha256: value.PresentationSHA256,
		SceneVersion: value.SceneVersion, PresentationVersion: value.PresentationVersion, CompositorProfile: value.CompositorProfile,
		Plate: value.Plate, ActiveMask: value.ActiveMask, HardwareMask: value.HardwareMask, Screen: value.Screen,
		RenderEligible: value.RenderEligible, Issues: value.Issues,
	}
}

func compiledToProto(projectID string, value domain.Revision) *shotsv1.CompiledShot {
	return &shotsv1.CompiledShot{
		ProjectId: projectID, ShotId: value.ShotID, Revision: value.Revision, RecipeSha256: value.RecipeSHA256,
		SceneSha256: value.SceneSHA256, PresentationSha256: value.PresentationSHA256, SceneVersion: value.SceneVersion, PresentationVersion: value.PresentationVersion,
		CompositorProfile: value.CompositorProfile, Snapshot: value.Snapshot, Plate: value.Plate, ActiveMask: value.ActiveMask,
		HardwareMask: value.HardwareMask, Screen: value.Screen, Issues: value.Issues, RenderEligible: value.RenderEligible,
	}
}
