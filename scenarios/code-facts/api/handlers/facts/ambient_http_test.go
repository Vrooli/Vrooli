package facts

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	internalfacts "code-facts/internal/facts"
)

type ambientSeamStub struct{ called bool }

func (s *ambientSeamStub) SearchAmbientSeams(_ context.Context, query, scenario string, limit int) (*internalfacts.AmbientSeamResponse, error) {
	s.called = true
	if query != "environment production" || scenario != "code-facts" || limit != 5 {
		panic("unexpected ambient seam request")
	}
	return &internalfacts.AmbientSeamResponse{Scenario: scenario, Scope: "static source only", Results: []internalfacts.AmbientSeam{{ID: "env", Kind: "environment", SourcePath: "api/main.go", Line: 32, SourceSHA256: "abc", SourceLayer: "NON_TEST_SOURCE_CANDIDATE", ArgumentContextStatus: "UNKNOWN_STATIC_CONTEXT_ONLY"}}}, nil
}

func TestSearchAmbientSeamsHTTPReturnsSourceAttribution(t *testing.T) {
	service := &ambientSeamStub{}
	req := httptest.NewRequest("POST", "/api/v1/ambient-seams/search", strings.NewReader(`{"query":"environment production","scenario":"code-facts","limit":5}`))
	res := httptest.NewRecorder()
	SearchAmbientSeamsHTTPHandler(service)(res, req)
	if res.Code != 200 || !service.called || !strings.Contains(res.Body.String(), "NON_TEST_SOURCE_CANDIDATE") || !strings.Contains(res.Body.String(), "abc") {
		t.Fatalf("status=%d called=%t body=%s", res.Code, service.called, res.Body.String())
	}
}

func TestSearchAmbientSeamsHTTPRejectsMalformedScope(t *testing.T) {
	service := &ambientSeamStub{}
	req := httptest.NewRequest("POST", "/api/v1/ambient-seams/search", strings.NewReader(`{"query":"clock","scenario":"../../repo"}`))
	res := httptest.NewRecorder()
	SearchAmbientSeamsHTTPHandler(service)(res, req)
	if res.Code != 400 || service.called {
		t.Fatalf("status=%d called=%t body=%s", res.Code, service.called, res.Body.String())
	}
}
