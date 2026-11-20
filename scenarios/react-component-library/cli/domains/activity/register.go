// Package activity is the CLI's append-only mutation-log surface. Mirrors the
// API's Connect-RPC ActivityService. Command surface loads from cli/manifest.json
// via cliapp.LoadFromManifestPrimitives.
package activity

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "activity"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ActivityService.ListActivity":       cliapp.ProtoList(h.eventsCall, h.eventsReport),
		"ActivityService.GetActivitySummary": cliapp.ProtoList(h.summaryCall, h.summaryReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("activity: load from manifest: %w", err)
	}
	return group, nil
}
