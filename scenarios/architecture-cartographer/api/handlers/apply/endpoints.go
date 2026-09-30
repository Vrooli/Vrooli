package apply

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply/apply_v1connect"
)

// Endpoints describes the apply domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "apply.plan",
		Path:        apply_v1connect.ApplyServicePlanApplyProcedure,
		Method:      "POST",
		Summary:     "Plan a per-domain apply",
		Description: "Deterministically derives the operation list from resolved/validated conflicts. v0.1 returns the plan envelope; RunApply is unimplemented.",
		Category:    "apply",
	},
	{
		ID:          "apply.run",
		Path:        apply_v1connect.ApplyServiceRunApplyProcedure,
		Method:      "POST",
		Summary:     "Run an apply plan (v0.1: unimplemented)",
		Description: "Executes the supplied plan. v0.1 surfaces CodeUnimplemented; v0.2 wires the executor.",
		Category:    "apply",
	},
	{
		ID:          "apply.list-history",
		Path:        apply_v1connect.ApplyServiceListApplyHistoryProcedure,
		Method:      "POST",
		Summary:     "List apply runs",
		Description: "Cursor-paginated history of apply runs (empty in v0.1).",
		Category:    "apply",
	},
	{
		ID:          "apply.get-build-baseline",
		Path:        apply_v1connect.ApplyServiceGetBuildBaselineProcedure,
		Method:      "POST",
		Summary:     "Get the current build baseline",
		Description: "Returns the toolchain build-green snapshot. v0.1 returns an empty baseline.",
		Category:    "apply",
	},
	{
		ID:          "apply.write-suppression",
		Path:        apply_v1connect.ApplyServiceWriteSuppressionProcedure,
		Method:      "POST",
		Summary:     "Write an in-repo suppression marker",
		Description: "Inserts a durable `// arch:allow` marker into a source file, sanctioning a finding as intentional. Safe, non-destructive (comment-only) write.",
		Category:    "apply",
	},
}
