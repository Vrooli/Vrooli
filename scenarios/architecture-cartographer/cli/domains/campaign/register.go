// Package campaign is the CLI's campaign-domain command surface — the
// agent-facing scenario-improvement tracker. It mirrors the API's
// Connect-RPC CampaignService: ingest a test-genie audit, work the
// profile-ranked worklist, and reconcile re-audits by stable id.
//
// Like every domain package it follows the graph-domain shape: a
// Register(core, manifest) returning a cliapp.SubcommandGroup built from
// cli/manifest.json via cliapp.LoadFromManifest, plus one handler per
// Connect-RPC subcommand in handlers.go.
package campaign

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "campaign"

// Register builds the campaign subcommand group from the embedded
// manifest and wires every CampaignService Connect-RPC binding to a
// handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CampaignService.CreateCampaign":    h.create,
		"CampaignService.ListCampaigns":     h.list,
		"CampaignService.GetCampaignStatus": h.status,
		"CampaignService.NextCampaignStep":  h.next,
		"CampaignService.ResolveItem":       h.resolve,
		"CampaignService.ApplyItem":         h.apply,
		"CampaignService.ReauditCampaign":   h.reaudit,
		"CampaignService.CloseCampaign":     h.close,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("campaign: load from manifest: %w", err)
	}
	return group, nil
}
