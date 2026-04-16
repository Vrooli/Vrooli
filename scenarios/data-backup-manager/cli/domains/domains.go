package domains

import (
	"data-backup-manager/cli/domains/backup"
	"data-backup-manager/cli/domains/compliance"
	"data-backup-manager/cli/domains/maintenance"
	"data-backup-manager/cli/domains/restore"
	"data-backup-manager/cli/domains/schedule"
	"data-backup-manager/cli/domains/visited"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Data Backup Manager has no
// single-verb domains — every domain has multiple subcommands.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		backup.Register(core),
		restore.Register(core),
		schedule.Register(core),
		compliance.Register(core),
		visited.Register(core),
		maintenance.Register(core),
	}
}
