package integrations

import (
	integrationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/integrations/integrations_v1connect"

	"portal/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "integrations_status", Path: integrationsconnect.IntegrationsServiceStatusProcedure, Method: "POST", Summary: "Get integration status", Description: "Returns portal-side readiness, rolling stats, active behavior mode, override, and reason.", Category: "integrations"},
	{ID: "integrations_override_update", Path: integrationsconnect.IntegrationsServiceUpdateOverrideProcedure, Method: "POST", Summary: "Update behavior override", Description: "Sets the behavior-mode override to auto, force-off, or force-passive.", Category: "integrations"},
}
