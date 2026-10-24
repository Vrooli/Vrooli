package profile

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/profile/profile_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{{ID: "profile_get", Path: connect.ProfileServiceGetProfileProcedure, Method: "POST", Summary: "Get profile", Category: "profile"}, {ID: "profile_save_draft", Path: connect.ProfileServiceSaveProfileDraftProcedure, Method: "POST", Summary: "Save onboarding draft", Category: "profile"}, {ID: "profile_apply", Path: connect.ProfileServiceApplyProfileProcedure, Method: "POST", Summary: "Apply profile rules", Category: "profile"}}
