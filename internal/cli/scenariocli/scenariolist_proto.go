package scenariocli

import (
	"github.com/vrooli/vrooli/internal/discovery"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ScenarioListResponse maps the CLI's internal list types onto the
// vrooli.cli.v1 wire contract — the single producer-side translation that every
// consumer (EM, …) decodes. A proto field rename breaks this mapping at compile
// time, which is the drift guard.
func ScenarioListResponse(items []ListItemOutput, runningCount int, failures []discovery.Failure) *cliv1.ScenarioListResponse {
	resp := &cliv1.ScenarioListResponse{
		Success: true,
		Summary: &cliv1.ScenarioListSummary{
			TotalScenarios: int32(len(items)),
			Running:        int32(runningCount),
			Available:      int32(len(items) - runningCount),
		},
	}
	for _, item := range items {
		scenario := &cliv1.Scenario{
			Name:        item.Name,
			Description: item.Description,
			Version:     item.Version,
			Status:      item.Status,
			Tags:        item.Tags,
			Path:        item.Path,
		}
		for _, port := range item.Ports {
			scenario.Ports = append(scenario.Ports, &cliv1.ScenarioPort{
				Key:            port.Key,
				Step:           port.Step,
				Port:           int32(port.Port),
				ListenerStatus: port.ListenerStatus,
			})
		}
		resp.Scenarios = append(resp.Scenarios, scenario)
	}
	for _, failure := range failures {
		resp.DiscoveryFailures = append(resp.DiscoveryFailures, &cliv1.DiscoveryFailure{
			Kind:  failure.Kind,
			Name:  failure.Name,
			Path:  failure.Path,
			Stage: failure.Stage,
			Error: failure.Error,
		})
	}
	return resp
}
