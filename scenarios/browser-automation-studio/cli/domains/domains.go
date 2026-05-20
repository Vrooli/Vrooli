// Package domains assembles the BAS CLI command surface from per-domain
// register packages. Hand-coded local-only domains (status, playbooks)
// stay as flat CommandGroups; every manifest-bound domain returns a
// SubcommandGroup built from cli/manifest.json via cliapp.LoadFromManifest.
//
// Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc) wired every manifest group
// listed below; legacy hand-coded REST-driven packages were deleted in the
// same pass.
package domains

import (
	"browser-automation-studio/cli/ai"
	"browser-automation-studio/cli/capture"
	"browser-automation-studio/cli/entitlement"
	"browser-automation-studio/cli/executions"
	"browser-automation-studio/cli/exports"
	"browser-automation-studio/cli/internal/appctx"
	"browser-automation-studio/cli/observability"
	"browser-automation-studio/cli/playbooks"
	"browser-automation-studio/cli/project_files"
	"browser-automation-studio/cli/projects"
	"browser-automation-studio/cli/recordings"
	"browser-automation-studio/cli/replay_config"
	"browser-automation-studio/cli/scenarios"
	"browser-automation-studio/cli/schedules"
	"browser-automation-studio/cli/schema"
	"browser-automation-studio/cli/session_profiles"
	"browser-automation-studio/cli/status"
	"browser-automation-studio/cli/tools"
	"browser-automation-studio/cli/uxmetrics"
	"browser-automation-studio/cli/workflows"
	"fmt"
	"os"

	visionnavigation "browser-automation-studio/cli/vision_navigation"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups returns the flat (legacy/local-only) command groups.
// Manifest-bound subcommand groups are surfaced via SubcommandGroups
// below.
func CommandGroups(ctx *appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		status.Commands(ctx),
		playbooks.Commands(ctx),
		capture.Commands(ctx),
		recordings.Commands(ctx),
	}
}

// SubcommandGroups builds every manifest-bound domain subcommand group by
// loading cli/manifest.json and dispatching each command via the proto
// service descriptor declared in its binding.
//
// Registration failures are non-fatal at startup: an unparseable manifest
// or missing proto service descriptor for one group must not stop the CLI
// from offering the other groups. The error is written to stderr; the
// failing group is omitted from help and dispatch.
func SubcommandGroups(ctx *appctx.Context, manifest []byte) []cliapp.SubcommandGroup {
	type registerFn func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error)
	regs := []struct {
		name string
		fn   registerFn
	}{
		{"scenarios", scenarios.Register},
		{"tools", tools.Register},
		{"schema", schema.Register},
		{"entitlement", entitlement.Register},
		{"projects", projects.Register},
		{"workflows", workflows.Register},
		{"project-files", project_files.Register},
		{"executions", executions.Register},
		{"uxmetrics", uxmetrics.Register},
		{"replay-config", replay_config.Register},
		{"schedules", schedules.Register},
		{"ai", ai.Register},
		{"vision-navigation", visionnavigation.Register},
		{"session-profiles", session_profiles.Register},
		{"observability", observability.Register},
		{"exports", exports.Register},
	}

	out := make([]cliapp.SubcommandGroup, 0, len(regs))
	for _, r := range regs {
		grp, err := r.fn(ctx.Core, manifest)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: subcommand group %q not registered: %v\n", r.name, err)
			continue
		}
		out = append(out, grp)
	}
	return out
}
