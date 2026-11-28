package cost

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cost/cost_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "cost_list_price_observations", Path: connect.CostServiceListPriceObservationsProcedure, Method: "POST", Summary: "List preserved price observations", Category: "cost"},
	{ID: "cost_create_price_observation", Path: connect.CostServiceCreatePriceObservationProcedure, Method: "POST", Summary: "Record a price observation", Category: "cost"},
}
