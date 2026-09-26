package eligibility

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/eligibility/eligibility_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{{ID: "eligibility_evaluate", Path: connect.EligibilityServiceEvaluateProcedure, Method: "POST", Summary: "Evaluate recipe eligibility", Category: "eligibility"}}
