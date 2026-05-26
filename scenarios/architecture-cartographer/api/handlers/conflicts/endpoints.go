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
		Description: "Cursor-paginated list of stored conflicts, filterable by scenario, status, and type.",
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
		ID:          "conflicts.assign",
		Path:        conflicts_v1connect.ConflictsServiceAssignConflictProcedure,
		Method:      "POST",
		Summary:     "Assign a conflict to a domain",
		Description: "Transitions the conflict to ASSIGNED with the supplied target domain. Honors X-Dry-Run.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts assign"},
	},
	{
		ID:          "conflicts.resolve",
		Path:        conflicts_v1connect.ConflictsServiceResolveConflictProcedure,
		Method:      "POST",
		Summary:     "Resolve a conflict",
		Description: "Transitions the conflict to RESOLVED (or FORCE_RESOLVED with force=true). Honors X-Dry-Run. ApplyDeferred=true when the chosen fix requires file movement, which is unimplemented in v0.1.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts resolve"},
	},
	{
		ID:          "conflicts.reopen",
		Path:        conflicts_v1connect.ConflictsServiceReopenConflictProcedure,
		Method:      "POST",
		Summary:     "Reopen a conflict",
		Description: "Transitions a resolved/force-resolved/validated conflict back to DETECTED. Honors X-Dry-Run.",
		Category:    "conflicts",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart conflicts reopen"},
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
