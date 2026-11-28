package facts

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	internalfacts "code-facts/internal/facts"
)

type flowSearchStub struct {
	called   bool
	response *internalfacts.FlowSearchResponse
}

func (s *flowSearchStub) SearchFeatureFlow(_ context.Context, query, scenario, path string, limit int) (*internalfacts.FlowSearchResponse, error) {
	s.called = true
	if query != "SearchComposition" || scenario != "code-facts" || path != "api/internal/facts" || limit != 3 {
		panic("unexpected flow request")
	}
	if s.response != nil {
		return s.response, nil
	}
	return &internalfacts.FlowSearchResponse{Query: query, Scenario: scenario, Results: []internalfacts.FlowSearchHit{{
		ID: "flow:composition", Attestation: "DERIVED/PARTIAL", GraphHash: "graph-sha", Steps: []internalfacts.FlowStep{
			{From: "SearchComposition", To: "loadProvider", Call: "loadProvider", Path: "api/internal/facts/search.go", ToPath: "api/internal/registry/registry.go", Line: 42, GraphHash: "graph-sha", ToGraphHash: "registry-sha", Resolution: "source_symbol_resolved"},
			{From: "loadProvider", Call: "lookupDynamicOwner", Path: "api/internal/registry/registry.go", Line: 81, GraphHash: "registry-sha", Resolution: "callee_symbol_unmatched"},
		},
	}}}, nil
}

func TestSearchFlowHTTPHandlerReturnsBoundedSourceFlow(t *testing.T) {
	service := &flowSearchStub{}
	req := httptest.NewRequest("POST", "/api/v1/flows/search", strings.NewReader(`{"query":"SearchComposition","scenario":"code-facts","path":"api/internal/facts","limit":3}`))
	res := httptest.NewRecorder()
	SearchFlowHTTPHandler(service)(res, req)
	var got internalfacts.FlowSearchResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || len(got.Results) != 1 || got.Results[0].Attestation != "DERIVED/PARTIAL" || !service.called {
		t.Fatalf("status=%d called=%t body=%s", res.Code, service.called, res.Body.String())
	}
	if len(got.Results[0].Steps) != 2 || got.Results[0].Steps[0].Path == "" || got.Results[0].Steps[0].Line != 42 || got.Results[0].Steps[0].ToPath != "api/internal/registry/registry.go" || got.Results[0].Steps[0].GraphHash != "graph-sha" || got.Results[0].Steps[0].ToGraphHash != "registry-sha" || got.Results[0].Steps[1].Resolution != "callee_symbol_unmatched" || got.Results[0].Steps[1].GraphHash != "registry-sha" {
		t.Fatalf("handler did not preserve bounded per-edge provenance: %+v", got.Results[0].Steps)
	}
}

func TestSearchFlowHTTPHandlerPreservesNoMatchDegradation(t *testing.T) {
	service := &flowSearchStub{response: &internalfacts.FlowSearchResponse{Query: "SearchComposition", Scenario: "code-facts", Results: []internalfacts.FlowSearchHit{}, Degradations: []internalfacts.FlowDegradation{{Code: "no_matching_flow_entry", Message: "No source function matched this query."}}}}
	req := httptest.NewRequest("POST", "/api/v1/flows/search", strings.NewReader(`{"query":"SearchComposition","scenario":"code-facts","path":"api/internal/facts","limit":3}`))
	res := httptest.NewRecorder()
	SearchFlowHTTPHandler(service)(res, req)
	var got internalfacts.FlowSearchResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || len(got.Results) != 0 || len(got.Degradations) != 1 || got.Degradations[0].Code != "no_matching_flow_entry" {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestSearchFlowHTTPHandlerRejectsTraversalBeforeServiceCall(t *testing.T) {
	service := &flowSearchStub{}
	req := httptest.NewRequest("POST", "/api/v1/flows/search", strings.NewReader(`{"query":"SearchComposition","scenario":"code-facts","path":"../../cmd/vrooli","limit":3}`))
	res := httptest.NewRecorder()
	SearchFlowHTTPHandler(service)(res, req)
	if res.Code != 400 || service.called {
		t.Fatalf("status=%d called=%t body=%s", res.Code, service.called, res.Body.String())
	}
}
