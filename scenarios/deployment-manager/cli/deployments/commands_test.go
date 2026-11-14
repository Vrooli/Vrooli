package deployments

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	deploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/deployments/deploymentsv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeDeploymentsClient struct {
	deployRequest *structpb.Value
	statusRequest *structpb.Value
}

func (f *fakeDeploymentsClient) Deploy(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	f.deployRequest = request.Msg
	return connect.NewResponse(mustDeploymentValue(map[string]interface{}{"status": "started"})), nil
}

func (f *fakeDeploymentsClient) DeployDesktop(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	f.deployRequest = request.Msg
	return connect.NewResponse(mustDeploymentValue(map[string]interface{}{"status": "started"})), nil
}

func (f *fakeDeploymentsClient) Status(_ context.Context, request *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error) {
	f.statusRequest = request.Msg
	return connect.NewResponse(mustDeploymentValue(map[string]interface{}{"status": "running"})), nil
}

func mustDeploymentValue(value map[string]interface{}) *structpb.Value {
	result, err := structpb.NewValue(value)
	if err != nil {
		panic(err)
	}
	return result
}

var _ deploymentsconnect.DeploymentsServiceClient = (*fakeDeploymentsClient)(nil)

func TestTypedDeploymentCommandsUseGeneratedClient(t *testing.T) {
	fake := &fakeDeploymentsClient{}
	cmd := NewWithConnectClient(nil, fake)
	if err := cmd.Deploy([]string{"profile-1", "--dry-run", "--async", "--format", "json"}); err != nil {
		t.Fatalf("typed deploy failed: %v", err)
	}
	if got := fake.deployRequest.AsInterface().(map[string]interface{}); got["profile_id"] != "profile-1" || got["dry_run"] != true || got["async"] != true {
		t.Fatalf("unexpected typed deploy request: %#v", got)
	}
	if err := cmd.Deployment([]string{"status", "deployment-1", "--format", "json"}); err != nil {
		t.Fatalf("typed status failed: %v", err)
	}
	if got := fake.statusRequest.AsInterface().(map[string]interface{}); got["deployment_id"] != "deployment-1" {
		t.Fatalf("unexpected typed status request: %#v", got)
	}
}

func TestDeployValidateOnlyShortCircuits(t *testing.T) {
	var validateCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/validate") {
			validateCalled = true
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Deploy([]string{"demo", "--validate-only"}); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
	if !validateCalled {
		t.Fatalf("expected validate endpoint to be called")
	}
}

func TestLogsAcceptsFilters(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Logs([]string{"demo", "--level", "error", "--search", "fail"}); err != nil {
		t.Fatalf("logs failed: %v", err)
	}
	if !strings.Contains(query, "level=error") || !strings.Contains(query, "search=fail") {
		t.Fatalf("expected query params, got %s", query)
	}
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
