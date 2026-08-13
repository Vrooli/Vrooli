// Package versions is the CLI's version-history surface. Mirrors the
// API's Connect-RPC VersionsService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package versions

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "versions"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"VersionsService.ListVersions":                 h.list,
		"VersionsService.GetVersion":                   h.show,
		"VersionsService.DiffVersions":                 h.diff,
		"VersionLifecycleService.ListRetireCandidates": h.retireCandidates,
		"VersionLifecycleService.ListVersionLedger":    h.progression,
		"VersionLifecycleService.DeprecateVersion":     func(ctx cliapp.RunContext) error { return h.transition(ctx, "deprecate") },
		"VersionLifecycleService.ArchiveVersion":       func(ctx cliapp.RunContext) error { return h.transition(ctx, "archive") },
		"VersionLifecycleService.RetireVersion":        func(ctx cliapp.RunContext) error { return h.transition(ctx, "retire") },
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("versions: load from manifest: %w", err)
	}
	return group, nil
}
