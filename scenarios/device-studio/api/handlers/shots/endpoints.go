package shots

import (
	"device-studio/internal/module"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "shots_create", Path: shotsconnect.ShotsServiceCreateShotProcedure, Method: "POST", Summary: "Create shot revision", Description: "Creates an immutable first shot revision in a project.", Category: "shots"},
	{ID: "shots_save", Path: shotsconnect.ShotsServiceSaveShotProcedure, Method: "POST", Summary: "Save shot revision", Description: "Appends an immutable revision when expected_revision matches; conflicts return ShotConflict detail.", Category: "shots"},
	{ID: "shots_list", Path: shotsconnect.ShotsServiceListShotsProcedure, Method: "POST", Summary: "List shots", Description: "Returns a bounded keyset page of project shots.", Category: "shots"},
	{ID: "shots_get", Path: shotsconnect.ShotsServiceGetShotProcedure, Method: "POST", Summary: "Get shot", Description: "Returns a shot and its immutable head revision.", Category: "shots"},
	{ID: "shots_revision_get", Path: shotsconnect.ShotsServiceGetShotRevisionProcedure, Method: "POST", Summary: "Get shot revision", Description: "Returns one immutable historical shot revision.", Category: "shots"},
	{ID: "shots_revision_list", Path: shotsconnect.ShotsServiceListShotRevisionsProcedure, Method: "POST", Summary: "List shot revisions", Description: "Returns a bounded page of immutable shot revisions.", Category: "shots"},
	{ID: "shots_compile", Path: shotsconnect.ShotsServiceCompileShotProcedure, Method: "POST", Summary: "Compile shot", Description: "Resolves source media metadata, semantic digests, and render eligibility for an immutable revision.", Category: "shots"},
	{ID: "shots_validate", Path: shotsconnect.ShotsServiceValidateShotProcedure, Method: "POST", Summary: "Validate shot draft", Description: "Validates typed shot fields without persisting a revision.", Category: "shots"},
}
