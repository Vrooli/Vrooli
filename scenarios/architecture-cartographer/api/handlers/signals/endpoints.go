package signals

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"
)

// Endpoints describes the signals domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "signals.score",
		Path:        signals_v1connect.SignalsServiceScoreChunkProcedure,
		Method:      "POST",
		Summary:     "Score a chunk",
		Description: "Runs every registered Signal on the chunk and aggregates them into a Verdict (auto_place / suggest / conflict).",
		Category:    "signals",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart signals score"},
	},
	{
		ID:          "signals.explain",
		Path:        signals_v1connect.SignalsServiceExplainVerdictProcedure,
		Method:      "POST",
		Summary:     "Explain a verdict",
		Description: "Returns the same Verdict shape as ScoreChunk with full per-signal evidence so operators can inspect the placement decision.",
		Category:    "signals",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart signals explain"},
	},
	{
		ID:          "signals.list",
		Path:        signals_v1connect.SignalsServiceListSignalsProcedure,
		Method:      "POST",
		Summary:     "List registered signals",
		Description: "Returns the set of Signal plug-ins, their default weights, and per-scenario manifest-overlaid weights when scenario is supplied.",
		Category:    "signals",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart signals list"},
	},
	{
		ID:          "signals.boundaries",
		Path:        signals_v1connect.SignalsServiceBoundaryHealthProcedure,
		Method:      "POST",
		Summary:     "Domain boundary-health scores",
		Description: "Computes per-domain coupling metrics (efferent/afferent, instability, fan-out, stable-kernel) and a graded boundary-health score with advisory smells.",
		Category:    "signals",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart signals boundaries"},
	},
}
