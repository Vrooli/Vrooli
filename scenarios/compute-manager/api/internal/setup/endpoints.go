package setup

import (
	"compute-manager/internal/module"
	setupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/setup/setup_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "setup_list_providers", Path: setupconnect.SetupServiceListProvidersProcedure, Method: "POST", Summary: "List provider setup catalogue", Category: "setup"},
	{ID: "setup_begin_step", Path: setupconnect.SetupServiceBeginSetupStepProcedure, Method: "POST", Summary: "Ask the operator to confirm a provider setup step", Category: "setup"},
	{ID: "setup_readiness", Path: setupconnect.SetupServiceGetReadinessProcedure, Method: "POST", Summary: "Read provider setup readiness and evidence", Category: "setup"},
	{ID: "setup_verify", Path: setupconnect.SetupServiceVerifyProviderProcedure, Method: "POST", Summary: "Run read-only provider verification probes", Category: "setup"},
}
