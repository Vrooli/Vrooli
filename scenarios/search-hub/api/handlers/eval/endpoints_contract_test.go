package eval

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareStrategiesEndpointDescriptorMatchesRuntimeContract(t *testing.T) {
	descriptor := findEndpointDescriptor(t, "evals_compare_strategies")
	require.Contains(t, descriptor.Request["strategy_names"], "must include the active strategy")
	require.Contains(t, descriptor.Request["apply"], "never mutates")
	for _, field := range []string{"suite_id", "active_strategy", "arms", "applied", "writeback_reason"} {
		require.Contains(t, descriptor.Response, field, "endpoint metadata must describe the complete protobuf response")
	}
	require.Contains(t, descriptor.Response["applied"], "false")
	require.Contains(t, descriptor.Description, "never changes the active strategy")
	require.Contains(t, strings.Join(descriptor.Errors, " "), "active strategy must be included")
}

func TestRunSuiteEndpointDescriptorMatchesRuntimeContract(t *testing.T) {
	descriptor := findEndpointDescriptor(t, "evals_run_suite")
	require.Contains(t, descriptor.Description, "router.routing suite defaults to the federated Query path")
	require.Contains(t, descriptor.Request["tier"], "provider_direct default, or federated")
	require.Contains(t, descriptor.Request["strategy_name"], "rejected for provider_direct")
	require.Contains(t, strings.Join(descriptor.Errors, " "), "tier must be provider_direct or federated")
	require.Contains(t, strings.Join(descriptor.Errors, " "), "active executable strategy")
	require.Contains(t, descriptor.Response, "adequacy", "runtime response includes warn-level corpus adequacy")
	require.Contains(t, descriptor.Response, "run")
}

type endpointDescriptorView struct {
	Description string
	Request     map[string]string
	Response    map[string]string
	Errors      []string
}

func findEndpointDescriptor(t *testing.T, id string) endpointDescriptorView {
	t.Helper()
	for _, endpoint := range Endpoints {
		if endpoint.ID != id {
			continue
		}
		view := endpointDescriptorView{
			Description: endpoint.Description,
			Request:     endpoint.Request.Properties,
			Response:    endpoint.Response.Properties,
			Errors:      make([]string, 0, len(endpoint.Errors)),
		}
		for _, apiError := range endpoint.Errors {
			view.Errors = append(view.Errors, apiError.Description)
		}
		return view
	}
	require.FailNow(t, "endpoint descriptor was not found", "id=%s", id)
	return endpointDescriptorView{}
}
