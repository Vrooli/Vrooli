// Package migration is the CLI's migration-domain command surface — the
// agent-facing migration tracker. It mirrors the API's Connect-RPC
// MigrationService: ingest a test-genie audit, work the prioritized
// findings, and reconcile re-audits by stable id.
//
// Like every domain package it follows the graph-domain shape: a
// Register(core, manifest) returning a cliapp.SubcommandGroup built from
// cli/manifest.json via cliapp.LoadFromManifest, plus one handler per
// Connect-RPC subcommand in handlers.go.
package migration

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "migration"

// Register builds the migration subcommand group from the embedded
// manifest and wires every MigrationService Connect-RPC binding to a
// handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MigrationService.CreateMigration":    h.create,
		"MigrationService.ListMigrations":     h.list,
		"MigrationService.GetMigrationStatus": h.status,
		"MigrationService.NextMigrationStep":  h.next,
		"MigrationService.ResolveFinding":     h.resolve,
		"MigrationService.ApplyFinding":       h.apply,
		"MigrationService.ReauditMigration":   h.reaudit,
		"MigrationService.CloseMigration":     h.close,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("migration: load from manifest: %w", err)
	}
	return group, nil
}
