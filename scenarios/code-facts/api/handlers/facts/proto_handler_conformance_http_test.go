package facts

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	internalfacts "code-facts/internal/facts"
)

type protoHandlerConformanceServiceFunc func(context.Context, string) (*internalfacts.ProtoHandlerConformanceResponse, error)

func (f protoHandlerConformanceServiceFunc) ProtoHandlerConformance(ctx context.Context, scenario string) (*internalfacts.ProtoHandlerConformanceResponse, error) {
	return f(ctx, scenario)
}

func TestProtoHandlerConformanceHTTPRequiresExactScenarioAndReturnsSourceContract(t *testing.T) {
	called := ""
	service := protoHandlerConformanceServiceFunc(func(_ context.Context, scenario string) (*internalfacts.ProtoHandlerConformanceResponse, error) {
		called = scenario
		return &internalfacts.ProtoHandlerConformanceResponse{Scenario: scenario, Scope: "scenario:" + scenario + "/api/handlers", Services: []internalfacts.ProtoHandlerService{{Service: "WidgetService", Status: "MATCH"}}, Degradations: []string{}}, nil
	})
	request := httptest.NewRequest("POST", "/api/v1/proto-handler-conformance", strings.NewReader(`{"scenario":"demo"}`))
	response := httptest.NewRecorder()
	ProtoHandlerConformanceHTTPHandler(service).ServeHTTP(response, request)
	if response.Code != 200 || called != "demo" {
		t.Fatalf("status=%d scenario=%q body=%s", response.Code, called, response.Body.String())
	}
	var got internalfacts.ProtoHandlerConformanceResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scenario != "demo" || len(got.Services) != 1 || got.Services[0].Status != "MATCH" {
		t.Fatalf("response = %+v", got)
	}
}

func TestProtoHandlerConformanceHTTPRejectsAmbiguousOrUnscopedBodies(t *testing.T) {
	handler := ProtoHandlerConformanceHTTPHandler(protoHandlerConformanceServiceFunc(func(context.Context, string) (*internalfacts.ProtoHandlerConformanceResponse, error) {
		t.Fatal("service must not be called for invalid scope")
		return nil, nil
	}))
	for _, body := range []string{`{}`, `{"scenario":"../outside"}`, `{"scenario":"demo","extra":true}`, `{"scenario":"demo"}{}`} {
		request := httptest.NewRequest("POST", "/api/v1/proto-handler-conformance", strings.NewReader(body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 400 {
			t.Errorf("body %s status=%d, want 400", body, response.Code)
		}
	}
}
