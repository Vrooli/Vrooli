package capabilities

import (
	"client-rehearsal/internal/module"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/shared/shared_v1connect"
)

var Endpoints = []module.EndpointDescriptor{{ID: "capabilities_describe", Path: connect.CapabilitiesServiceDescribeCapabilitiesProcedure, Method: "POST", Summary: "Describe optional scenario capabilities", Description: "Returns declared optional dependencies and their current recovery guidance as a structured value.", Category: "capabilities"}}
