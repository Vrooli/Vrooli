package scenes

import (
	"device-studio/internal/module"
	scenesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "scenes_candidate_save", Path: scenesconnect.ScenesServiceSaveSceneCandidateProcedure, Method: "POST", Summary: "Save scene candidate", Description: "Appends an immutable scene candidate revision at the expected head; external artifacts remain soft references and are not qualified by save.", Category: "scenes"},
	{ID: "scenes_revision_get", Path: scenesconnect.ScenesServiceGetSceneRevisionProcedure, Method: "POST", Summary: "Get scene revision", Description: "Returns one immutable scene candidate revision after canonical digest verification.", Category: "scenes"},
	{ID: "scenes_head_get", Path: scenesconnect.ScenesServiceGetSceneHeadProcedure, Method: "POST", Summary: "Get scene head", Description: "Returns the current immutable scene candidate revision.", Category: "scenes"},
	{ID: "scenes_revision_list", Path: scenesconnect.ScenesServiceListSceneRevisionsProcedure, Method: "POST", Summary: "List scene revisions", Description: "Returns a bounded ascending page of immutable scene revisions.", Category: "scenes"},
	{ID: "scenes_candidate_list", Path: scenesconnect.ScenesServiceListSceneCandidatesProcedure, Method: "POST", Summary: "List scene candidate heads", Description: "Returns one latest immutable candidate revision per scene in stable identifier order.", Category: "scenes"},
	{ID: "scenes_pose_fit", Path: scenesconnect.ScenesServiceFitKnownCameraPoseProcedure, Method: "POST", Summary: "Fit known-camera device pose", Description: "Fits only the rigid device transform from caller-supplied 2D/3D landmarks and an immutable scene camera; a declared hinge assumption remains fixed.", Category: "scenes"},
	{ID: "scenes_import_prepared_photograph", Path: scenesconnect.ScenesServiceImportPreparedPhotographProcedure, Method: "POST", Summary: "Import accepted prepared photograph", Description: "Fetches an exact content-addressed Image Tools child, verifies its expected bundle hash and reviewed source, then publishes an unqualified scene candidate through canonical Library and SceneCandidate owners. Local render use is granted only by an explicit source-use claim and rationale; publication and motion qualification are separate. Monocular scale remains an estimate.", Category: "scenes"},
}
