// Package verify is the CLI's verification command surface, a thin
// wrapper over the Connect-RPC VerificationsService. The `verify show`
// subcommand loads from cli/manifest.json via cliapp.LoadFromManifest;
// `verify run` and `verify check` are hand-written exception commands
// appended outside the manifest because both invoke StartVerification
// with different VerificationMode enum values, and cli-manifest/v1
// binds 1:1 between command and RPC method.
package verify

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "verify"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"VerificationsService.GetVerification": h.show,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("verify: load from manifest: %w", err)
	}
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	flowFlag := cliapp.Flag{Name: "flow", Description: "Restrict to a single flow id"}
	group.Subcommands = append(group.Subcommands,
		cliapp.Command{
			Name:        "run",
			Description: "Regenerate artifacts (model.qnt, runtime, replay helper) for every flow",
			Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag}},
			RunCtx:      h.run,
		},
		cliapp.Command{
			Name:        "check",
			Description: "Verify every flow: lint + freshness + Quint check",
			Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag}},
			RunCtx:      h.check,
		},
	)
	return group, nil
}
