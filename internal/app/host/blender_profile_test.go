package hostapp

import (
	"testing"

	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	rendererprofile "github.com/vrooli/vrooli/internal/tools/renderer-profile"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestBlenderProfileProtoReusesRendererStateAndBinary(t *testing.T) {
	for _, tc := range []struct {
		state blenderrenderer.State
		want  cliv1.RendererToolProfileState
	}{
		{blenderrenderer.StateUnsupported, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM},
		{blenderrenderer.StateMissing, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING},
		{blenderrenderer.StateUnmanaged, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNMANAGED},
		{blenderrenderer.StatePartial, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PARTIAL},
		{blenderrenderer.StateTampered, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED},
		{blenderrenderer.StateProbeFailed, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED},
		{blenderrenderer.StatePresent, cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT},
	} {
		observation := blenderrenderer.Observation{
			State: tc.state, ArtifactIdentity: "artifact", EnvironmentID: "linux-amd64-unverified",
			OS: "linux", Architecture: "amd64", ArchiveSHA256: blenderrenderer.ArchiveSHA256,
			ArchiveSize: blenderrenderer.ArchiveSize, ExpectedTreeSHA256: blenderrenderer.TreeSHA256,
			BinaryPath:   "/managed/opt/blender-renderer/" + blenderrenderer.BinaryMember,
			BinarySHA256: blenderrenderer.BinarySHA256, BinarySize: blenderrenderer.BinarySize,
			Version: "Blender 5.2.2", LicenseID: blenderrenderer.LicenseID,
			LicenseSHA256: blenderrenderer.LicenseSHA256, LicenseDetails: "component notices",
		}
		response := rendererprofile.Blender(observation)
		if response.State != tc.want || response.ExpectedTreeSha256 != blenderrenderer.TreeSHA256 {
			t.Fatalf("state %q mapped to %q, expected %q; tree=%q", tc.state, response.State, tc.want, response.ExpectedTreeSha256)
		}
		if tc.state == blenderrenderer.StatePresent {
			if response.Blender == nil || response.Blender.ArchiveMemberPath != blenderrenderer.BinaryMember || response.Blender.Version != observation.Version {
				t.Fatalf("verified binary not projected from existing binary message: %+v", response.Blender)
			}
		} else if response.Blender != nil {
			t.Fatalf("unverified binary leaked into %q response: %+v", tc.state, response.Blender)
		}
		if response.GetQualificationScope() == "" || response.GetArtifactIdentity() == "" {
			t.Fatalf("profile omitted qualification/content identity: %+v", response)
		}
	}
}
