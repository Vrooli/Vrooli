// Package activity is the CLI's read-only view of the research home: now,
// commissions, commission, topics and changes. Commands come from
// cli/manifest.json and dispatch at the top level (flat group).
package activity

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity/activity_v1connect"
)

// GroupName is the manifest group this package owns.
const GroupName = "activity"

// Register builds the activity commands from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.CommandGroup, error) {
	h := newHandlers(core)
	service := activityconnect.ActivityServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		service + "GetOverview":     cliapp.ProtoList(h.nowCall, h.nowReport),
		service + "ListCommissions": cliapp.ProtoList(h.commissionsCall, h.commissionsReport),
		service + "GetCommission":   cliapp.ProtoList(h.commissionCall, h.commissionReport),
		service + "ListTopics":      cliapp.ProtoList(h.topicsCall, h.topicsReport),
		service + "GetChanges":      cliapp.ProtoList(h.changesCall, h.changesReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.CommandGroup{}, fmt.Errorf("activity: load from manifest: %w", err)
	}
	return cliapp.CommandGroup{Title: "What is moving", Commands: group.Subcommands}, nil
}
