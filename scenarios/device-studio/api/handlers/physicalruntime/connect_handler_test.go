package physicalruntime

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"device-studio/internal/devices"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	domain "device-studio/internal/physicalruntime"
	"device-studio/internal/scenes"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
)

type starterServiceStub struct{ state domain.StarterState }

func (s starterServiceStub) InitializeReviewedStarter(context.Context) (domain.StarterState, error) {
	return s.state, nil
}

func TestStarterResponsePlacementPinsReturnedCandidateRefs(t *testing.T) {
	state := domain.StarterState{
		Phone:  devices.DeviceRevision{DeviceID: "phone", Revision: 2, SHA256: "device-sha"},
		Scene:  scenes.Revision{SceneID: "scene", Revision: 3, SHA256: "scene-sha"},
		Motion: motions.Revision{MotionID: "motion", Revision: 4, SHA256: "motion-sha"},
		PhoneStudyPlacement: domain.ReviewedPhoneStudyPlacement{
			ProfileID: domain.ReviewedPhoneStudyPlacementProfileID, ProfileVersion: domain.ReviewedPhoneStudyPlacementProfileVersion,
			DisplayID: "main_display", DeviceSupportID: "device-root-support", SceneSupportID: "table-top-flat", SelectedCameraID: "phone-study-camera",
			SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
			SupportPivotM:     physical.Vec3{Y: .0071073174476624},
			DeviceRootRest:    physical.TRS{Rotation: physical.QuatXYZW{X: -.6885449545442038, Y: -.08161248882759843, Z: -.07844978823902271, W: .7163036213536392}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}},
		},
	}
	handler := NewConnectHandler(starterServiceStub{state: state})
	response, err := handler.InitializeReviewedStarter(context.Background(), connect.NewRequest(&physicalv1.InitializeReviewedStarterRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	placement := response.Msg.GetPhoneStudyPlacement()
	if placement == nil || placement.GetProfileId() != domain.ReviewedPhoneStudyPlacementProfileID || placement.GetProfileVersion() != 1 {
		t.Fatalf("starter placement profile missing or changed: %+v", placement)
	}
	for _, pair := range []struct {
		name      string
		placement *physicalv1.ComponentRevisionRef
		response  *physicalv1.ComponentRevisionRef
	}{
		{name: "device", placement: placement.GetDevice(), response: response.Msg.GetPhoneDevice()},
		{name: "scene", placement: placement.GetScene(), response: response.Msg.GetPhoneStudyScene()},
		{name: "motion", placement: placement.GetMotion(), response: response.Msg.GetPhoneStillMotion()},
	} {
		if pair.placement == nil || pair.response == nil || pair.placement.GetId() != pair.response.GetId() || pair.placement.GetRevision() != pair.response.GetRevision() || pair.placement.GetSha256() != pair.response.GetSha256() {
			t.Errorf("placement %s ref differs from the returned candidate ref: placement=%+v response=%+v", pair.name, pair.placement, pair.response)
		}
	}
	if placement.GetSupportPivotM().GetY() != .0071073174476624 || placement.GetDeviceRootRest().GetRotationXyzw().GetY() != -.08161248882759843 {
		t.Fatalf("starter placement transform changed: %+v", placement)
	}
}
