package monitor

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	monitorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/monitor/monitor_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "monitor_status_get", Path: monitorconnect.MonitorServiceGetMonitorStatusProcedure, Method: "POST", Summary: "Get monitor status", Description: "Returns recurring deep-validation monitor status.", Category: "monitor"},
}
