// Package metrics is the CLI's typed measures command surface. It mirrors the
// API's MeasuresService Connect-RPC methods declared in cli/manifest.json.
package metrics

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "metrics"

// Register builds the metrics subcommand group from the embedded manifest and
// wires MeasuresService bindings to primitive-backed handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"MeasuresService.FederatedLatency":        cliapp.ProtoList(h.federatedLatencyCall, h.federatedLatencyReport),
		"MeasuresService.DegradedQueryRate":       cliapp.ProtoList(h.degradedQueryRateCall, h.degradedQueryRateReport),
		"MeasuresService.ProviderDegradationRate": cliapp.ProtoList(h.providerDegradationRateCall, h.providerDegradationRateReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("metrics: load from manifest: %w", err)
	}
	return group, nil
}
