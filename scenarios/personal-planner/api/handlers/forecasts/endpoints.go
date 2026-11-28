package forecasts

import (
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/forecasts/forecasts_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "forecasts_get", Path: c.ForecastsServiceGetForecastProcedure, Method: "POST", Summary: "Get forecast", Description: "Returns the current deterministic central and cautious completion outlook.", Category: "forecasts"},
	{ID: "forecasts_history", Path: c.ForecastsServiceListForecastSnapshotsProcedure, Method: "POST", Summary: "List forecast history", Description: "Returns recent durable forecast snapshots and changes.", Category: "forecasts"},
}
