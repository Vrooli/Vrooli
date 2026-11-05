package physicalruntime

import (
	"device-studio/internal/module"
	physicalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical/physical_v1connect"
)

var Endpoints = []module.EndpointDescriptor{{
	ID: "physical_starter_initialize", Path: physicalconnect.PhysicalStarterServiceInitializeReviewedStarterProcedure,
	Method: "POST", Summary: "Initialize reviewed physical starter candidates",
	Description: "Publishes the fixed locally reviewed starter bytes through the Library and creates immutable unqualified candidate revisions on the active storage route.", Category: "physical",
}}
