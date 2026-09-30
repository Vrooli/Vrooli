package routing

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "routing"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings, err := cliapp.ProtoBindings(core, "vrooli.ai_gateway.v1.routing.RoutingService", cliapp.ProtoBindingOptions{
		Render: map[string]cliapp.Renderer{
			"RoutingService.PreviewRoute": renderPreview,
			"RoutingService.ExecuteRoute": renderExecute,
			"RoutingService.SubmitMedia":  renderMediaSubmit,
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("routing: build proto bindings: %w", err)
	}
	bindings["RoutingService.GetMediaExecution"] = h.mediaShow
	bindings["RoutingService.WaitMediaExecution"] = h.mediaWait
	bindings["RoutingService.CancelMediaExecution"] = h.mediaCancel
	bindings["RoutingService.RetryMediaExecution"] = h.mediaRetry
	bindings["RoutingService.ListRouteEvidence"] = h.evidenceList
	bindings["RoutingService.GetRouteEvidence"] = h.evidenceShow
	bindings["RoutingService.ListProviderHealth"] = h.health
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("routing: load from manifest: %w", err)
	}
	return group, nil
}
