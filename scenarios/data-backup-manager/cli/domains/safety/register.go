// Package safety is the CLI's safety-domain command surface. Mirrors the API's
// Connect-RPC SafetyService — the Baseline Modes data substrate the platform
// recovery floor shells out to for pre-promote scenario snapshots.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, governance, RPC bindings); this package only wires bindings to
// handlers in handlers.go.
package safety

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "safety"

// Register builds the safety subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SafetyService.EnsureSafetyDestination": h.ensureDestination,
		"SafetyService.BackupScenarioNow":       h.backupNow,
		"SafetyService.RegisterScenarioTargets": h.registerTargets,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("safety: load from manifest: %w", err)
	}
	return group, nil
}
