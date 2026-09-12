package brief

import (
	briefconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief/brief_v1connect"
	"portal/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "brief_build", Path: briefconnect.BriefServiceBuildProcedure, Method: "POST", Summary: "Build a gated context brief", Category: "brief"},
	{ID: "brief_get", Path: briefconnect.BriefServiceGetProcedure, Method: "POST", Summary: "Read a stored context brief", Category: "brief"},
	{ID: "brief_list", Path: briefconnect.BriefServiceListProcedure, Method: "POST", Summary: "List stored context briefs", Category: "brief"},
	{ID: "brief_record_use", Path: briefconnect.BriefServiceRecordUseProcedure, Method: "POST", Summary: "Record context brief use", Category: "brief"},
	{ID: "brief_stats", Path: briefconnect.BriefServiceStatsProcedure, Method: "POST", Summary: "Report context brief usage", Category: "brief"},
}
