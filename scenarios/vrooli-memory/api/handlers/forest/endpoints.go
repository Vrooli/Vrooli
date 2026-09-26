package forest

import (
	"vrooli-memory/internal/module"

	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest/forest_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "forest_compact", Path: forestconnect.ForestServiceRunCompactionPassProcedure, Method: "POST", Summary: "Run pressure-driven memory compaction", Category: "forest"},
	{ID: "forest_frontier", Path: forestconnect.ForestServiceGetFrontierProcedure, Method: "POST", Summary: "List forest frontier", Category: "forest"},
	{ID: "forest_node", Path: forestconnect.ForestServiceGetNodeProcedure, Method: "POST", Summary: "Get one forest node", Category: "forest"},
	{ID: "forest_rebuild", Path: forestconnect.ForestServiceRebuildForestProcedure, Method: "POST", Summary: "Rebuild derived forest", Category: "forest"},
}
