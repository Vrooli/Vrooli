package migration

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/migration/migration_v1connect"
)

// Endpoints describes the migration domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "migration.create",
		Path:        migration_v1connect.MigrationServiceCreateMigrationProcedure,
		Method:      "POST",
		Summary:     "Open a migration and ingest findings",
		Description: "Opens a tracked migration for a scenario and ingests the initial ArchitectureFinding set (all start detected).",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration create"},
	},
	{
		ID:          "migration.list",
		Path:        migration_v1connect.MigrationServiceListMigrationsProcedure,
		Method:      "POST",
		Summary:     "List a scenario's migrations",
		Description: "Returns the migration headers for a scenario (newest first), or every migration when scenario is empty.",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration list"},
	},
	{
		ID:          "migration.status",
		Path:        migration_v1connect.MigrationServiceGetMigrationStatusProcedure,
		Method:      "POST",
		Summary:     "Get migration status",
		Description: "Returns the migration plus every tracked finding and rollup counts.",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration status"},
	},
	{
		ID:          "migration.next",
		Path:        migration_v1connect.MigrationServiceNextMigrationStepProcedure,
		Method:      "POST",
		Summary:     "Get the next worklist chunk",
		Description: "Returns the prioritized, dependency-aware worklist of open findings (regressions first, then cycles, then severity).",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration next"},
	},
	{
		ID:          "migration.resolve",
		Path:        migration_v1connect.MigrationServiceResolveFindingProcedure,
		Method:      "POST",
		Summary:     "Mark a finding resolved",
		Description: "Records that the agent fixed a finding by hand, with an operator note.",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration resolve"},
	},
	{
		ID:          "migration.apply",
		Path:        migration_v1connect.MigrationServiceApplyFindingProcedure,
		Method:      "POST",
		Summary:     "Apply a finding fix (status-only)",
		Description: "Records a hand-fix as a status-only transition. Auto-execution of file-op fixes stays deferred to the apply-execution plan.",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration apply"},
	},
	{
		ID:          "migration.reaudit",
		Path:        migration_v1connect.MigrationServiceReauditMigrationProcedure,
		Method:      "POST",
		Summary:     "Reconcile a re-audit",
		Description: "Reconciles a fresh findings set against the tracked set by stable id: absent→validated, present→stay, (re)appeared→regression.",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration reaudit"},
	},
	{
		ID:          "migration.close",
		Path:        migration_v1connect.MigrationServiceCloseMigrationProcedure,
		Method:      "POST",
		Summary:     "Close a migration",
		Description: "Marks the migration closed.",
		Category:    "migration",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart migration close"},
	},
}
