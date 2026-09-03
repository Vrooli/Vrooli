package meter

import (
	"compute-manager/internal/module"
	meterconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/meter/meter_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "meter_usage", Path: meterconnect.MeterServiceUsageProcedure, Method: "POST", Summary: "List compute usage", Category: "meter"},
	{ID: "meter_reservations", Path: meterconnect.MeterServiceReservationsProcedure, Method: "POST", Summary: "List compute reservations", Category: "meter"},
	{ID: "meter_ceiling", Path: meterconnect.MeterServiceCeilingProcedure, Method: "POST", Summary: "Get compute ceiling", Category: "meter"},
}
