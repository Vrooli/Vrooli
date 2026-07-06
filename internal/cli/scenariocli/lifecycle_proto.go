package scenariocli

import (
	"io"

	"github.com/vrooli/vrooli/internal/lifecycle"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// scenarioLifecycleItem maps a LifecycleItemOutput onto its proto message.
func scenarioLifecycleItem(item LifecycleItemOutput) *cliv1.ScenarioLifecycleItem {
	msg := &cliv1.ScenarioLifecycleItem{
		Name:               item.Name,
		Status:             item.Status,
		Health:             item.Health,
		Ports:              copyInt32Map(item.Ports),
		FailedDependencies: item.FailedDependencies,
		FailedResources:    item.FailedResources,
		Verdict:            item.Verdict,
		Operation:          ScenarioStartOperationMessage(item.Operation),
	}
	for _, ep := range item.Endpoints {
		msg.Endpoints = append(msg.Endpoints, &cliv1.ScenarioEndpoint{
			Name:        ep.Name,
			Key:         ep.Key,
			Description: ep.Description,
			Port:        int32(ep.Port),
			Url:         ep.URL,
		})
	}
	return msg
}

// -----------------------------------------------------------------------------
// lifecycle items (`scenario start`/`stop`/`restart` summary)
// -----------------------------------------------------------------------------

// ScenarioLifecycleResponse maps the lifecycle items payload onto its wire
// contract (cliout.WriteSuccessJSON under the "scenarios" key).
func ScenarioLifecycleResponse(items []LifecycleItemOutput) *cliv1.ScenarioLifecycleResponse {
	resp := &cliv1.ScenarioLifecycleResponse{Success: true}
	for _, item := range items {
		resp.Scenarios = append(resp.Scenarios, scenarioLifecycleItem(item))
	}
	return resp
}

func writeScenarioLifecycleJSON(w io.Writer, items []LifecycleItemOutput) error {
	return marshalScenarioStatus(w, ScenarioLifecycleResponse(items))
}

// -----------------------------------------------------------------------------
// lifecycle batch report (`scenario stop-all` / multi-scenario start/stop)
// -----------------------------------------------------------------------------

// ScenarioBatchResponse maps the lifecycle batch payload onto its wire contract
// (cliout.WriteSuccessJSON under the "data" key). Under the typed contract the
// started/stopped slices are always emitted ([] when empty), an additive
// normalization of the legacy "present only when non-empty" output.
func ScenarioBatchResponse(resp BatchResponse) *cliv1.ScenarioBatchResponse {
	data := &cliv1.ScenarioBatchData{
		Stopped: resp.Stopped,
	}
	for _, item := range resp.Started {
		data.Started = append(data.Started, scenarioLifecycleItem(item))
	}
	for _, f := range resp.Failed {
		data.Failed = append(data.Failed, &cliv1.ScenarioBatchFailure{
			Name:  f.Name,
			Error: f.Error,
		})
	}
	return &cliv1.ScenarioBatchResponse{Success: true, Data: data}
}

func writeScenarioBatchJSON(w io.Writer, resp BatchResponse) error {
	return marshalScenarioStatus(w, ScenarioBatchResponse(resp))
}

// -----------------------------------------------------------------------------
// `scenario setup`
// -----------------------------------------------------------------------------

// ScenarioSetupResponse maps a lifecycle.PhaseResult onto the setup wire
// contract (cliout.WriteSuccessFields envelope).
func ScenarioSetupResponse(result lifecycle.PhaseResult) *cliv1.ScenarioSetupResponse {
	return &cliv1.ScenarioSetupResponse{
		Success: true,
		Phase:   "setup",
		Status:  string(result.Status),
		Defined: result.Defined,
		Steps: &cliv1.ScenarioSetupSteps{
			Executed: int32(result.ExecutedSteps),
			Skipped:  int32(result.SkippedSteps),
		},
	}
}

func writeScenarioSetupJSON(w io.Writer, result lifecycle.PhaseResult) error {
	return marshalScenarioStatus(w, ScenarioSetupResponse(result))
}
