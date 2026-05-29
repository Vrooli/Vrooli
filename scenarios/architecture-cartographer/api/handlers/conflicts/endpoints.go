package conflicts

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
)

// Endpoints describes the conflicts domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "conflicts.detect",
		Path:        conflicts_v1connect.ConflictsServiceDetectConflictsProcedure,
		Method:      "POST",
		Summary:     "Detect conflicts",
		Description: "Runs every registered Detector against a scenario's graph snapshot + manifest, persists the resulting Conflict envelopes, and emits one analytics event per detection.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts detect"},
	},
	{
		ID:          "conflicts.list",
		Path:        conflicts_v1connect.ConflictsServiceListConflictsProcedure,
		Method:      "POST",
		Summary:     "List persisted conflicts",
		Description: "Cursor-paginated list of stored conflicts, filterable by scenario and type.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts list"},
	},
	{
		ID:          "conflicts.get",
		Path:        conflicts_v1connect.ConflictsServiceGetConflictProcedure,
		Method:      "POST",
		Summary:     "Get one conflict",
		Description: "Returns the Conflict envelope for the supplied id.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts show"},
	},
	{
		ID:          "conflicts.validate",
		Path:        conflicts_v1connect.ConflictsServiceValidateConflictsProcedure,
		Method:      "POST",
		Summary:     "Validate cartographer-clean closure",
		Description: "Returns outstanding conflicts and a clean=true gate when zero error-severity rows remain. Used by the dogfood test.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts validate"},
	},
	{
		ID:          "conflicts.list-detectors",
		Path:        conflicts_v1connect.ConflictsServiceListDetectorsProcedure,
		Method:      "POST",
		Summary:     "List registered detectors",
		Description: "Returns the set of Detector plug-ins compiled into this binary.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts detectors"},
	},
	{
		ID:          "conflicts.list-resolvers",
		Path:        conflicts_v1connect.ConflictsServiceListResolversProcedure,
		Method:      "POST",
		Summary:     "List registered resolvers",
		Description: "Returns the set of Resolver plug-ins compiled into this binary.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts resolvers"},
	},
}
