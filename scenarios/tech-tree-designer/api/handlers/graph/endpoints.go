package graph

import (
	"tech-tree-designer/internal/module"

	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph/graph_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "graph-describe",
		Path:        graphconnect.GraphServiceDescribeTechTreeProcedure,
		Method:      "POST",
		Summary:     "Describe the scenario interface graph",
		Description: "Returns live scenario nodes and proto-import edges from the configured graph source.",
		Category:    "graph",
		Request:     &module.Schema{Type: "DescribeTechTreeRequest"},
		Response:    &module.Schema{Type: "DescribeTechTreeResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer graph describe"},
	},
	{
		ID:          "graph-neighborhood",
		Path:        graphconnect.GraphServiceGetNeighborhoodProcedure,
		Method:      "POST",
		Summary:     "Get a graph neighborhood",
		Description: "Returns the nodes and edges within a local directed or incoming radius around one scenario.",
		Category:    "graph",
		Request:     &module.Schema{Type: "GetNeighborhoodRequest"},
		Response:    &module.Schema{Type: "DescribeTechTreeResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer graph neighbors", Args: []string{"<scenario>"}},
	},
	{
		ID:          "graph-path",
		Path:        graphconnect.GraphServiceFindPathProcedure,
		Method:      "POST",
		Summary:     "Find a graph path",
		Description: "Returns the shortest directed path between two scenarios when an interface-dependency path exists.",
		Category:    "graph",
		Request:     &module.Schema{Type: "FindPathRequest"},
		Response:    &module.Schema{Type: "DescribeTechTreeResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer graph path", Args: []string{"<from>", "<to>"}},
	},
	{
		ID:          "graph-ancestors",
		Path:        graphconnect.GraphServiceListAncestorsProcedure,
		Method:      "POST",
		Summary:     "List graph ancestors",
		Description: "Returns the dependency ancestors reachable from a scenario through proto-import edges.",
		Category:    "graph",
		Request:     &module.Schema{Type: "ListAncestorsRequest"},
		Response:    &module.Schema{Type: "DescribeTechTreeResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer graph ancestors", Args: []string{"<scenario>"}},
	},
	{
		ID:          "graph-export",
		Path:        graphconnect.GraphServiceExportTechTreeProcedure,
		Method:      "POST",
		Summary:     "Export the graph",
		Description: "Exports the scenario interface graph as DOT, JSON, or plain text.",
		Category:    "graph",
		Request:     &module.Schema{Type: "ExportTechTreeRequest"},
		Response:    &module.Schema{Type: "ExportTechTreeResponse"},
		CLIMapping:  &module.CLIMapping{Command: "tech-tree-designer graph export"},
	},
}
