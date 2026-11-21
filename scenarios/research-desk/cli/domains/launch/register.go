// Package launch is the CLI's launch side: packets through packet.py,
// launches, the two messages for a launch by hand, and controls. Commands come from
// cli/manifest.json and dispatch at the top level (flat group).
package launch

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	launchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch/launch_v1connect"
)

// GroupName is the manifest group this package owns.
const GroupName = "launch"

// Register builds the launch commands from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.CommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: launchconnect.NewLaunchServiceClient(httpClient, baseURL)}
	service := launchconnect.LaunchServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		service + "GetLaunchOptions":  cliapp.ProtoList(h.optionsCall, optionsReport),
		service + "Preflight":         cliapp.ProtoList(h.preflightCall, preflightReport),
		service + "Build":             cliapp.ProtoList(h.buildCall, buildReport),
		service + "Launch":            cliapp.ProtoList(h.launchCall, launchReport),
		service + "GetLaunchMessages": cliapp.ProtoList(h.messagesCall, messagesReport),
		service + "GetLaunch":         cliapp.ProtoList(h.statusCall, statusReport),
		service + "Control":           cliapp.ProtoList(h.controlCall, controlReport),
		service + "TestLaunch":        cliapp.ProtoList(h.testCall, testReport),
		service + "GetLaunchTest":     cliapp.ProtoList(h.testStatusCall, testReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.CommandGroup{}, fmt.Errorf("launch: load from manifest: %w", err)
	}
	return cliapp.CommandGroup{Title: "Build, launch and control", Commands: group.Subcommands}, nil
}
