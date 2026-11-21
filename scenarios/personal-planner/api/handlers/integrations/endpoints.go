package integrations

import (
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/integrations/integrations_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "integrations_list_connections", Path: v.IntegrationsServiceListConnectionsProcedure, Method: "POST", Summary: "List calendar connections", Description: "Lists provider-neutral read-only calendar connection state without returning secrets.", Category: "integrations"},
	{ID: "integrations_create_fixture", Path: v.IntegrationsServiceCreateFixtureConnectionProcedure, Method: "POST", Summary: "Create fixture calendar", Description: "Creates a clearly synthetic read-only calendar connection for contract and UI verification.", Category: "integrations"},
	{ID: "integrations_sync", Path: v.IntegrationsServiceSyncConnectionProcedure, Method: "POST", Summary: "Sync calendar connection", Description: "Refreshes a connection through its configured adapter with revision protection.", Category: "integrations"},
	{ID: "integrations_disconnect", Path: v.IntegrationsServiceDisconnectConnectionProcedure, Method: "POST", Summary: "Disconnect calendar", Description: "Disconnects a read-only provider connection without claiming external deletion.", Category: "integrations"},
}
