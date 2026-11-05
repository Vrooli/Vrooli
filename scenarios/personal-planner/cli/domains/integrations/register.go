package integrations

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "integrations"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"IntegrationsService.ListConnections":         cliapp.ProtoList(h.listCall, h.listReport),
		"IntegrationsService.CreateFixtureConnection": cliapp.ProtoMutation(h.createFixtureCall, h.createReport),
		"IntegrationsService.SyncConnection":          cliapp.ProtoMutation(h.syncCall, h.syncReport),
		"IntegrationsService.DisconnectConnection":    cliapp.ProtoMutation(h.disconnectCall, h.disconnectReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("integrations: load from manifest: %w", err)
	}
	return group, nil
}
