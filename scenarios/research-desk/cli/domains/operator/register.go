// Package operator is the CLI's write side: answers, direction, steering,
// verdicts and the courier. Commands come from cli/manifest.json and dispatch
// at the top level (flat group).
package operator

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	operatorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator/operator_v1connect"
)

// GroupName is the manifest group this package owns.
const GroupName = "operator"

// Register builds the operator commands from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.CommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: operatorconnect.NewOperatorServiceClient(httpClient, baseURL)}
	service := operatorconnect.OperatorServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		service + "ListNeedsYou":   cliapp.ProtoList(h.needsYouCall, needsYouReport),
		service + "Answer":         cliapp.ProtoList(h.answerCall, answerReport),
		service + "AddEntry":       cliapp.ProtoList(h.sayCall, entryReport),
		service + "EnableSteering": cliapp.ProtoList(h.steeringCall, steeringReport),
		service + "ListReviews":    cliapp.ProtoList(h.reviewsCall, reviewsReport),
		service + "AddVerdict":     cliapp.ProtoList(h.verdictCall, verdictReport),
		service + "SetCourier":     cliapp.ProtoList(h.courierCall, courierReport),
		service + "TestCourier":    cliapp.ProtoList(h.courierTestCall, courierTestReport),
		service + "MarkSeen":       cliapp.ProtoList(h.markSeenCall, markSeenReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.CommandGroup{}, fmt.Errorf("operator: load from manifest: %w", err)
	}
	return cliapp.CommandGroup{Title: "Steer, answer and review", Commands: group.Subcommands}, nil
}
