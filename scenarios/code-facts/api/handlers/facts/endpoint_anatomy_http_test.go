package facts

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internalfacts "code-facts/internal/facts"
)

type endpointAnatomyServiceFunc func(context.Context, string, string, int) (*internalfacts.EndpointAnatomyResponse, error)

func (f endpointAnatomyServiceFunc) SearchEndpointAnatomy(ctx context.Context, query, scenario string, limit int) (*internalfacts.EndpointAnatomyResponse, error) {
	return f(ctx, query, scenario, limit)
}

func TestEndpointAnatomyHTTPHandlerBoundsAndPreservesEvidence(t *testing.T) {
	want := &internalfacts.EndpointAnatomyResponse{
		Query: "route registration", Scenario: "code-facts", Scope: "static source only",
		Results:      []internalfacts.EndpointAnatomy{{ID: "ep", State: "DECLARED_ONLY", Limits: []string{"runtime mounting unknown"}}},
		Degradations: []string{},
	}
	called := false
	handler := SearchEndpointAnatomyHTTPHandler(endpointAnatomyServiceFunc(func(_ context.Context, query, scenario string, limit int) (*internalfacts.EndpointAnatomyResponse, error) {
		called = true
		if query != "route registration" || scenario != "code-facts" || limit != 10 {
			t.Fatalf("service received query=%q scenario=%q limit=%d", query, scenario, limit)
		}
		return want, nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/endpoint-anatomy/search", strings.NewReader(`{"query":" route registration ","scenario":" code-facts ","limit":10}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !called {
		t.Fatalf("status=%d called=%t body=%s", res.Code, called, res.Body.String())
	}
	var got internalfacts.EndpointAnatomyResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scope != want.Scope || len(got.Results) != 1 || got.Results[0].State != "DECLARED_ONLY" || len(got.Results[0].Limits) != 1 {
		t.Fatalf("handler lost evidence limits or provenance: %#v", got)
	}
}

func TestEndpointAnatomyHTTPHandlerRejectsInvalidAndUnavailableRequests(t *testing.T) {
	service := endpointAnatomyServiceFunc(func(context.Context, string, string, int) (*internalfacts.EndpointAnatomyResponse, error) {
		return nil, errors.New("source unavailable")
	})
	tests := []struct {
		name, method, body string
		service            endpointAnatomyService
		status             int
	}{
		{name: "wrong method", method: http.MethodGet, body: `{}`, service: service, status: http.StatusMethodNotAllowed},
		{name: "unknown field", method: http.MethodPost, body: `{"query":"x","scenario":"code-facts","reachability":true}`, service: service, status: http.StatusBadRequest},
		{name: "multiple values", method: http.MethodPost, body: `{"query":"x","scenario":"code-facts"} {}`, service: service, status: http.StatusBadRequest},
		{name: "out of range", method: http.MethodPost, body: `{"query":"x","scenario":"code-facts","limit":11}`, service: service, status: http.StatusBadRequest},
		{name: "missing service", method: http.MethodPost, body: `{"query":"x","scenario":"code-facts"}`, status: http.StatusServiceUnavailable},
		{name: "source degraded", method: http.MethodPost, body: `{"query":"x","scenario":"code-facts"}`, service: service, status: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/endpoint-anatomy/search", strings.NewReader(tt.body))
			res := httptest.NewRecorder()
			SearchEndpointAnatomyHTTPHandler(tt.service).ServeHTTP(res, req)
			if res.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", res.Code, tt.status, res.Body.String())
			}
		})
	}
}
