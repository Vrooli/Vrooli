package roadmap

import (
	"tech-tree-designer/internal/module"

	roadmapconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap/roadmap_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "roadmap-sectors-list",
		Path:        roadmapconnect.RoadmapServiceListSectorsProcedure,
		Method:      "POST",
		Summary:     "List roadmap sectors",
		Description: "Returns metadata sectors used to group live and planned scenario graph nodes.",
		Category:    "roadmap",
		Request:     &module.Schema{Type: "ListSectorsRequest"},
		Response:    &module.Schema{Type: "ListSectorsResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer roadmap sectors"},
	},
	{
		ID:          "roadmap-sectors-upsert",
		Path:        roadmapconnect.RoadmapServiceUpsertSectorProcedure,
		Method:      "POST",
		Summary:     "Create or update a roadmap sector",
		Description: "Stores sector metadata for graph grouping and roadmap views.",
		Category:    "roadmap",
		Request:     &module.Schema{Type: "UpsertSectorRequest"},
		Response:    &module.Schema{Type: "Sector"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer roadmap sector", Args: []string{"<slug>"}},
	},
	{
		ID:          "roadmap-milestones-list",
		Path:        roadmapconnect.RoadmapServiceListMilestonesProcedure,
		Method:      "POST",
		Summary:     "List roadmap milestones",
		Description: "Returns strategic milestones layered over scenario graph nodes.",
		Category:    "roadmap",
		Request:     &module.Schema{Type: "ListMilestonesRequest"},
		Response:    &module.Schema{Type: "ListMilestonesResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer roadmap milestones"},
	},
	{
		ID:          "roadmap-milestones-upsert",
		Path:        roadmapconnect.RoadmapServiceUpsertMilestoneProcedure,
		Method:      "POST",
		Summary:     "Create or update a roadmap milestone",
		Description: "Stores a named milestone and the scenario nodes required to satisfy it.",
		Category:    "roadmap",
		Request:     &module.Schema{Type: "UpsertMilestoneRequest"},
		Response:    &module.Schema{Type: "Milestone"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer roadmap milestone", Args: []string{"<id>"}},
	},
	{
		ID:          "roadmap-progress",
		Path:        roadmapconnect.RoadmapServiceGetProgressProcedure,
		Method:      "POST",
		Summary:     "Get roadmap progress",
		Description: "Rolls up live, planned, beta, and stable graph nodes by sector and tier.",
		Category:    "roadmap",
		Request:     &module.Schema{Type: "GetProgressRequest"},
		Response:    &module.Schema{Type: "ProgressRollup"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer roadmap progress"},
	},
}
