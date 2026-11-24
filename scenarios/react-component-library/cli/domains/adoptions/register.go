// Package adoptions is the CLI's adoption-registry surface. Its public
// commands use AdoptionsService; intake workflows remain internal consumers
// of AdoptionIntakeService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package adoptions

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "adoptions"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"AdoptionsService.ListAdoptions":          cliapp.ProtoList(h.listCall, h.listReport),
		"AdoptionsService.ListEffectiveAdoptions": cliapp.ProtoList(h.listEffectiveCall, h.listEffectiveReport),
		"AdoptionsService.PreflightAdoption":      cliapp.ProtoList(h.preflightCall, h.preflightReport),
		"obligations":                             {Run: h.obligations},
		"AdoptionsService.SyncScenarioTokens":     cliapp.ProtoCustom(h.syncTokensCall, h.syncTokensHuman),
		"AdoptionsService.PruneScenarioTokens":    cliapp.ProtoCustom(h.pruneTokensCall, h.pruneTokensHuman),
		"AdoptionsService.LinkAdoption":           cliapp.ProtoMutation(h.linkCall, h.linkReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("adoptions: load from manifest: %w", err)
	}
	return group, nil
}
