package routing

import (
	"strings"
	"testing"

	"search-hub/internal/module"
)

func endpointByID(t *testing.T, id string) module.EndpointDescriptor {
	t.Helper()
	for _, endpoint := range Endpoints {
		if endpoint.ID == id {
			return endpoint
		}
	}
	t.Fatalf("endpoint %q is missing", id)
	return module.EndpointDescriptor{}
}

func TestQueryEndpointDescriptorMatchesCurrentRoutingContract(t *testing.T) {
	endpoint := endpointByID(t, "routing_query")
	if strings.Contains(endpoint.Description, "model-free lexical strategy") {
		t.Fatal("query discovery still hardcodes the lexical strategy")
	}
	for _, phrase := range []string{"active provider-selection strategy", "verified current caller context", "lexical fallback", "No-route and partial responses may contain no provider groups"} {
		if !strings.Contains(endpoint.Description, phrase) {
			t.Errorf("query description missing %q", phrase)
		}
	}
	if endpoint.Response == nil {
		t.Fatal("query response schema is missing")
	}
	for _, field := range []string{
		"groups", "ranked", "corpora_searched", "routing_explanation", "degraded", "partial",
		"pending_providers", "reranked", "reranker_leg", "routing_degrade_reason", "ordered_by",
		"selector_leg", "routing_trace", "degradations", "query_parts", "correction_candidates", "latency_ms",
	} {
		if _, ok := endpoint.Response.Properties[field]; !ok {
			t.Errorf("query response schema omits proto field %q", field)
		}
	}
	if groups := endpoint.Response.Properties["groups"]; !strings.Contains(groups, "may be empty") || strings.Contains(groups, "always populated") {
		t.Errorf("query group contract is inaccurate: %q", groups)
	}
}

func TestStatusEndpointDescriptorExposesActiveStrategyConsumerContract(t *testing.T) {
	endpoint := endpointByID(t, "routing_status")
	if !strings.Contains(endpoint.Description, "active provider-selection strategy") {
		t.Fatal("status discovery omits the active strategy used by CLI and UI consumers")
	}
	if endpoint.Response == nil {
		t.Fatal("status response schema is missing")
	}
	for _, field := range []string{
		"providers", "audit_providers", "active_strategy", "strategies", "classifier_available",
		"reranker_available", "reranker_leg", "incubating", "circuit_open_share", "circuit_open_quorum", "federation_degraded",
	} {
		if _, ok := endpoint.Response.Properties[field]; !ok {
			t.Errorf("status response schema omits proto field %q", field)
		}
	}
	if classifier := endpoint.Response.Properties["classifier_available"]; !strings.Contains(classifier, "retained compatibility") || !strings.Contains(classifier, "false") {
		t.Errorf("status classifier field is not documented as compatibility-only: %q", classifier)
	}
}
