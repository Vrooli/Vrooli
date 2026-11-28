package rendererprofile

import (
	"testing"

	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestBlenderProjectionUsesGeneratedStateAndPinnedIdentity(t *testing.T) {
	got := Blender(blenderrenderer.Observation{State: blenderrenderer.StatePresent, ArtifactIdentity: "artifact", EnvironmentID: "linux-amd64", OS: "linux", Architecture: "amd64", Version: blenderrenderer.BuildVersion, BinaryPath: "/bundle/tools/blender/blender-5.2.2-linux-x64/blender", BinarySHA256: blenderrenderer.BinarySHA256, BinarySize: blenderrenderer.BinarySize})
	if got.GetProfileId() != blenderrenderer.ProfileID || got.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT || got.GetBlender() == nil || got.GetBlender().GetSha256() != blenderrenderer.BinarySHA256 {
		t.Fatalf("unexpected Blender profile projection: %v", got)
	}
}
