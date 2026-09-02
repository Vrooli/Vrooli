package channel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/config"

	"connectrpc.com/connect"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
)

type scenarioResponseCollector struct {
	response chan *channelv1.ScenarioResponse
}

func (c scenarioResponseCollector) ReportScenarioResponse(_ context.Context, request *connect.Request[presencev1.ReportScenarioResponseRequest]) (*connect.Response[presencev1.ReportScenarioResponseResponse], error) {
	c.response <- request.Msg.GetResponse()
	return connect.NewResponse(&presencev1.ReportScenarioResponseResponse{Accepted: true}), nil
}

func TestRunScenarioRequestUsesBoundedNodeLocalHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/demo.Service/Get" || r.Header.Get("Content-Type") != "application/proto" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "request" {
			t.Fatalf("request body = %q", body)
		}
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()

	port := 0
	_, _ = fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)
	collector := scenarioResponseCollector{response: make(chan *channelv1.ScenarioResponse, 1)}
	client := NewClient(config.Config{NodeID: "node-1"}, WithHTTPClient(server.Client()), WithScenarioPortResolver(func(context.Context, string) (int, error) { return port, nil }), WithScenarioResponseReporter(collector))
	client.baseCtx = context.Background()
	client.runScenarioRequest(&channelv1.ScenarioRequest{CorrelationId: "corr-1", Scenario: "demo", Service: "demo.Service", Method: "Get", Request: []byte("request"), MaxResponseBytes: 32})

	select {
	case response := <-collector.response:
		if string(response.GetResponse()) != "response" || response.GetError() != "" || response.GetCorrelationId() != "corr-1" {
			t.Fatalf("response = %#v", response)
		}
	case <-time.After(time.Second):
		t.Fatal("scenario response was not reported")
	}
}

func TestRunScenarioRequestUsesDeclaredHTTPMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()
	port := 0
	_, _ = fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)
	collector := scenarioResponseCollector{response: make(chan *channelv1.ScenarioResponse, 1)}
	client := NewClient(config.Config{NodeID: "node-1"}, WithHTTPClient(server.Client()), WithScenarioPortResolver(func(context.Context, string) (int, error) { return port, nil }), WithScenarioResponseReporter(collector))
	client.baseCtx = context.Background()
	client.runScenarioRequest(&channelv1.ScenarioRequest{CorrelationId: "corr-get", Scenario: "demo", Service: "api", Method: "v2/readiness", HttpMethod: http.MethodGet, MaxResponseBytes: 32})
	select {
	case response := <-collector.response:
		if response.GetError() != "" || string(response.GetResponse()) != "response" {
			t.Fatalf("response = %#v", response)
		}
	case <-time.After(time.Second):
		t.Fatal("scenario response was not reported")
	}
}
