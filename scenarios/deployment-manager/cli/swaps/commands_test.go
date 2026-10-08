package swaps

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	swapsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/swaps/swapsv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeSwapsClient struct {
	method  string
	request *structpb.Value
}

func (f *fakeSwapsClient) response(method string, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	f.method, f.request = method, request.Msg
	value, err := structpb.NewValue([]interface{}{map[string]interface{}{"from": "postgres", "to": "sqlite"}})
	if method != "List" {
		value, err = structpb.NewValue(map[string]interface{}{"from": "postgres", "to": "sqlite"})
	}
	return connect.NewResponse(value), err
}

func (f *fakeSwapsClient) List(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return f.response("List", request)
}
func (f *fakeSwapsClient) Analyze(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return f.response("Analyze", request)
}
func (f *fakeSwapsClient) Cascade(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return f.response("Cascade", request)
}
func (f *fakeSwapsClient) Apply(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return f.response("Apply", request)
}
func (f *fakeSwapsClient) ApplyToProfile(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	return f.response("ApplyToProfile", request)
}

var _ swapsconnect.SwapsServiceClient = (*fakeSwapsClient)(nil)

func TestTypedSwapCommandsUseGeneratedClient(t *testing.T) {
	fake := &fakeSwapsClient{}
	cmd := NewWithConnectClient(nil, fake)
	if err := cmd.Run([]string{"list", "demo", "--format", "json"}); err != nil {
		t.Fatalf("typed swap list failed: %v", err)
	}
	request := fake.request.AsInterface().(map[string]interface{})
	if fake.method != "List" || request["scenario"] != "demo" {
		t.Fatalf("unexpected typed swap request: method=%s request=%#v", fake.method, request)
	}
}

func TestRunRequiresSubcommand(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{}); err == nil || !strings.Contains(err.Error(), "swaps subcommand is required") {
		t.Fatalf("expected subcommand error, got %v", err)
	}
}

func TestApplyCallsAPI(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/profiles/demo/swaps" && r.Method == http.MethodPost {
			called = true
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Run([]string{"apply", "demo", "from", "to"}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !called {
		t.Fatalf("expected swap apply request to be sent")
	}
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
