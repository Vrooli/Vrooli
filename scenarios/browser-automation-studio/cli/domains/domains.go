package domains

import (
	"browser-automation-studio/cli/ai"
	"browser-automation-studio/cli/executions"
	"browser-automation-studio/cli/internal/appctx"
	"browser-automation-studio/cli/playbooks"
	"browser-automation-studio/cli/recordings"
	"browser-automation-studio/cli/schema"
	"browser-automation-studio/cli/sessions"
	"browser-automation-studio/cli/status"
	"browser-automation-studio/cli/workflows"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(ctx *appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		status.Commands(ctx),
		playbooks.Commands(ctx),
		workflows.Commands(ctx),
		executions.Commands(ctx),
		recordings.Commands(ctx),
		sessions.Commands(ctx),
		ai.Commands(ctx),
		schema.Commands(ctx),
	}
}
