package harness

import (
	"vrooli-memory/internal/module"

	harnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness/harness_v1connect"
)

var Endpoints = []module.EndpointDescriptor{{ID: "harness_import", Path: harnessconnect.HarnessServiceRunImportProcedure, Method: "POST", Summary: "Start harness memory import", Category: "harness"}, {ID: "harness_import_status", Path: harnessconnect.HarnessServiceGetImportStatusProcedure, Method: "POST", Summary: "Read durable import status", Category: "harness"}, {ID: "harness_project", Path: harnessconnect.HarnessServiceRefreshProjectionProcedure, Method: "POST", Summary: "Refresh memory projection", Category: "harness"}, {ID: "harness_capture", Path: harnessconnect.HarnessServiceCaptureWriteProcedure, Method: "POST", Summary: "Capture native memory write", Category: "harness"}, {ID: "harness_maintenance_status", Path: harnessconnect.HarnessServiceGetMaintenanceStatusProcedure, Method: "POST", Summary: "Read last maintenance run", Category: "harness"}}
