package overview

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/dependencies/dependenciesv1connect"
	fitnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/fitness/fitnessv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeDependenciesClient struct{ request *structpb.Value }

func (f *fakeDependenciesClient) Analyze(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	f.request = request.Msg
	value, err := structpb.NewValue(map[string]interface{}{"scenario": "demo"})
	return connect.NewResponse(value), err
}

type fakeFitnessClient struct{ request *structpb.Value }

func (f *fakeFitnessClient) Score(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	f.request = request.Msg
	value, err := structpb.NewValue(map[string]interface{}{"score": float64(42)})
	return connect.NewResponse(value), err
}

var _ dependenciesconnect.DependenciesServiceClient = (*fakeDependenciesClient)(nil)
var _ fitnessconnect.FitnessServiceClient = (*fakeFitnessClient)(nil)

func TestTypedOverviewCommandsUseGeneratedClients(t *testing.T) {
	dependencies := &fakeDependenciesClient{}
	fitness := &fakeFitnessClient{}
	cmd := NewWithConnectClients(nil, dependencies, fitness)
	if err := cmd.Analyze([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("typed dependency analysis failed: %v", err)
	}
	if got := dependencies.request.AsInterface().(map[string]interface{})["scenario"]; got != "demo" {
		t.Fatalf("unexpected dependency request: %#v", dependencies.request.AsInterface())
	}
	if err := cmd.Fitness([]string{"demo", "--tier", "2", "--format", "json"}); err != nil {
		t.Fatalf("typed fitness failed: %v", err)
	}
	request := fitness.request.AsInterface().(map[string]interface{})
	if request["scenario"] != "demo" {
		t.Fatalf("unexpected fitness request: %#v", request)
	}
}

func TestAnalyzeRequiresScenario(t *testing.T) {
	cmd := New(dummyAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	if err := cmd.Analyze([]string{}); err == nil || !strings.Contains(err.Error(), "scenario is required") {
		t.Fatalf("expected scenario required error, got %v", err)
	}
}

func TestFitnessPostsPayload(t *testing.T) {
	var gotPath string
	var method string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		method = r.Method
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))

	if err := cmd.Fitness([]string{"demo", "--tier", "3"}); err != nil {
		t.Fatalf("fitness failed: %v", err)
	}
	if gotPath != "/api/v1/fitness/score" || method != http.MethodPost {
		t.Fatalf("unexpected call: %s %s", method, gotPath)
	}
	if !strings.Contains(body, `"tiers":[3]`) {
		t.Fatalf("expected tier payload, got %s", body)
	}
}

func dummyAPIClient(t *testing.T, handler http.Handler) *cliutil.APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return testAPIClient(server.URL)
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
			BaseOptions: cliutil.APIBaseOptions{DefaultBase: base},
		}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
