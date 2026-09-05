package telemetry

import (
	"program-runtime/internal/module"

	telemetryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry/telemetry_v1connect"
)

var Endpoints = []module.EndpointDescriptor{{ID: "telemetry_events", Method: "POST", Path: telemetryconnect.TelemetryServiceListEventsProcedure, Summary: "List typed program-runtime events.", Category: "telemetry"}}
