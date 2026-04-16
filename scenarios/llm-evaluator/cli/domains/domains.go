package domains

import "github.com/vrooli/cli-core/cliapp"

// CommandGroups aggregates flat command groups from domain packages.
//
// The LLM Evaluator API currently exposes only a root /health endpoint, which
// is already surfaced by cli-core's built-in `status` command — no custom
// flat command groups are needed yet. As new endpoints land, register their
// domains here (see packages/cli-core docs for `CommandGroup`).
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Prefer domain packages as the default growth path:
//
//	cli/domains/evaluations/register.go
//	cli/domains/benchmarks/register.go
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
//   - use cliapp.RenderOperationalReport / RenderListReport /
//     RenderMutationReport for default human output contracts
//   - use cliapp.PrintReportJSON(...) when a --json mode should mirror the
//     same structured report
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}
