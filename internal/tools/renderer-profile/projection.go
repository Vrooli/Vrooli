// Package rendererprofile projects low-level tool observations onto the
// generated CLI profile contracts shared by host and offline bundle callers.
package rendererprofile

import (
	"strings"
	"time"

	blenderrenderer "github.com/vrooli/vrooli/internal/tools/blender-renderer"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// Blender returns the canonical generated profile projection for an owner
// observation. Keep this shared so desktop and host CLI JSON cannot drift.
func Blender(observation blenderrenderer.Observation) *cliv1.BlenderToolProfile {
	response := &cliv1.BlenderToolProfile{
		SchemaVersion: 1, ProfileId: blenderrenderer.ProfileID,
		State: blenderState(observation.State), ArtifactIdentity: observation.ArtifactIdentity,
		EnvironmentIdentity: observation.EnvironmentID, Os: observation.OS, Architecture: observation.Architecture,
		ArchiveSha256: observation.ArchiveSHA256, ArchiveSizeBytes: uint64(observation.ArchiveSize),
		ExpectedTreeSha256: observation.ExpectedTreeSHA256, LicenseId: observation.LicenseID,
		LicenseSha256: observation.LicenseSHA256, LicenseDetails: observation.LicenseDetails,
		RuntimePrerequisites: append([]string(nil), observation.Prerequisites...),
		QualificationScope:   "managed-tree-and-fixed-version-probe-only; OS prerequisites, render workflow, and redistribution remain unqualified",
		Detail:               observation.Detail, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if observation.State == blenderrenderer.StatePresent {
		response.Blender = &cliv1.RendererToolBinary{
			Path: observation.BinaryPath, ArchiveMemberPath: blenderrenderer.BinaryMember,
			Sha256: observation.BinarySHA256, SizeBytes: uint64(observation.BinarySize), Version: observation.Version,
		}
	}
	return response
}

func blenderState(state blenderrenderer.State) cliv1.RendererToolProfileState {
	switch strings.TrimSpace(string(state)) {
	case string(blenderrenderer.StateUnsupported):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM
	case string(blenderrenderer.StateMissing):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING
	case string(blenderrenderer.StateUnmanaged):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNMANAGED
	case string(blenderrenderer.StatePartial):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PARTIAL
	case string(blenderrenderer.StateTampered):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
	case string(blenderrenderer.StateProbeFailed):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED
	case string(blenderrenderer.StatePresent):
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT
	default:
		return cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNSPECIFIED
	}
}
