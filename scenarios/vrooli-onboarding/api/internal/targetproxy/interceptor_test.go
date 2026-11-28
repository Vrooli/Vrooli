package targetproxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/nodereach"
	hostv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host"
	hostconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	operatorinputsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs"
	operatorinputsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs/operatorinputsv1connect"
	hosthandler "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/host"
	hostdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/host"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeReacher struct {
	request nodereach.ScenarioRequest
	result  []byte
	err     error
}

func (f *fakeReacher) CallScenario(_ context.Context, request nodereach.ScenarioRequest) ([]byte, error) {
	f.request = request
	return f.result, f.err
}

type stubService struct{}

func (stubService) ListOperatorInputs(context.Context, *connect.Request[operatorinputsv1.ListOperatorInputsRequest]) (*connect.Response[operatorinputsv1.ListOperatorInputsResponse], error) {
	return connect.NewResponse(&operatorinputsv1.ListOperatorInputsResponse{Version: 7}), nil
}

func (stubService) ResolveOperatorInputs(context.Context, *connect.Request[operatorinputsv1.ResolveOperatorInputsRequest]) (*connect.Response[operatorinputsv1.ResolveOperatorInputsResponse], error) {
	return connect.NewResponse(&operatorinputsv1.ResolveOperatorInputsResponse{}), nil
}

func TestInterceptorForwardsTheGeneratedProcedureWithoutAPIPath(t *testing.T) {
	remoteResponse, err := proto.Marshal(&operatorinputsv1.ListOperatorInputsResponse{Version: 9})
	if err != nil {
		t.Fatal(err)
	}
	reacher := &fakeReacher{result: remoteResponse}
	_, handler := operatorinputsconnect.NewOperatorInputsServiceHandler(stubService{}, connect.WithInterceptors(Interceptor(reacher)))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := operatorinputsconnect.NewOperatorInputsServiceClient(http.DefaultClient, server.URL)
	if _, err := client.ListOperatorInputs(context.Background(), connect.NewRequest(&operatorinputsv1.ListOperatorInputsRequest{Target: "minimouse"})); err != nil {
		t.Fatal(err)
	}
	if reacher.request.Procedure != operatorinputsconnect.OperatorInputsServiceListOperatorInputsProcedure {
		t.Fatalf("forwarded procedure = %q", reacher.request.Procedure)
	}
	if strings.Contains(reacher.request.Procedure, "/api/") {
		t.Fatalf("forwarded procedure contains REST prefix: %q", reacher.request.Procedure)
	}
}

func TestInterceptorLeavesLocalPathForHandler(t *testing.T) {
	reacher := &fakeReacher{}
	_, handler := operatorinputsconnect.NewOperatorInputsServiceHandler(stubService{}, connect.WithInterceptors(Interceptor(reacher)))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := operatorinputsconnect.NewOperatorInputsServiceClient(http.DefaultClient, server.URL)
	response, err := client.ListOperatorInputs(context.Background(), connect.NewRequest(&operatorinputsv1.ListOperatorInputsRequest{Target: "local"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetVersion() != 7 || reacher.request.Procedure != "" {
		t.Fatalf("local response=%v forwarded=%q", response.Msg, reacher.request.Procedure)
	}
}

func TestInterceptorForwardsExplicitAuthorizationForRemoteTarget(t *testing.T) {
	remoteResponse, err := proto.Marshal(&operatorinputsv1.ListOperatorInputsResponse{Version: 11})
	if err != nil {
		t.Fatal(err)
	}
	reacher := &fakeReacher{result: remoteResponse}
	_, handler := operatorinputsconnect.NewOperatorInputsServiceHandler(stubService{}, connect.WithInterceptors(Interceptor(reacher)))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := operatorinputsconnect.NewOperatorInputsServiceClient(http.DefaultClient, server.URL)
	request := connect.NewRequest(&operatorinputsv1.ListOperatorInputsRequest{Target: "node-1"})
	request.Header().Set("Authorization", "Bearer originating-principal")
	if _, err := client.ListOperatorInputs(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(reacher.request.Authorization); got != "Bearer originating-principal" {
		t.Fatalf("forwarded authorization = %q", got)
	}
}

func TestInterceptorDoesNotTreatPersonalLocalPeerAsRemoteAuthority(t *testing.T) {
	reacher := &fakeReacher{result: mustMarshalOperatorInputResponse(t, 12)}
	_, handler := operatorinputsconnect.NewOperatorInputsServiceHandler(stubService{}, connect.WithInterceptors(Interceptor(reacher)))
	protected := authn.Middleware(authn.Config{Providers: []authn.Provider{
		authn.PersonalLocalProvider{SessionToken: "local-session-token"},
	}})(handler)
	server := httptest.NewServer(protected)
	defer server.Close()
	client := operatorinputsconnect.NewOperatorInputsServiceClient(http.DefaultClient, server.URL)
	request := connect.NewRequest(&operatorinputsv1.ListOperatorInputsRequest{Target: "node-1"})
	request.Header().Set("Authorization", "Bearer local-session-token")
	response, err := client.ListOperatorInputs(context.Background(), request)
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("personal-local remote call error = %v, response = %v", err, response)
	}
	if reacher.request.Procedure != "" {
		t.Fatalf("unauthorized remote call reached Bridge: %q", reacher.request.Procedure)
	}
}

func TestRemoteHostMutationsFailClosedBeforeBridgeDispatch(t *testing.T) {
	mutationCalls := 0
	service := hostdomain.Service{
		PatchSafeguardConfigFn: func(context.Context, string, string, string, any) error {
			mutationCalls++
			return nil
		},
		SetNotificationRecipient: func(context.Context, string, string) error {
			mutationCalls++
			return nil
		},
	}
	readResponse, err := proto.Marshal(&hostv1.ListHostRequirementsResponse{})
	if err != nil {
		t.Fatal(err)
	}
	reacher := &fakeReacher{result: readResponse}
	path, handler := hostconnect.NewHostServiceHandler(hosthandler.NewConnectHandler(service), connect.WithInterceptors(Interceptor(reacher)))
	if path == "" {
		t.Fatal("host handler returned an empty path")
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := hostconnect.NewHostServiceClient(http.DefaultClient, server.URL)
	value, err := structpb.NewValue("enabled")
	if err != nil {
		t.Fatal(err)
	}

	for _, call := range []struct {
		name   string
		invoke func() error
	}{
		{name: "safeguard", invoke: func() error {
			_, err := client.PatchHostSafeguardConfig(context.Background(), connect.NewRequest(&hostv1.PatchHostSafeguardConfigRequest{Target: "node-1", SafeguardName: "firewall", ConfigKey: "mode", Value: value}))
			return err
		}},
		{name: "notification recipient", invoke: func() error {
			_, err := client.SetNotificationRecipient(context.Background(), connect.NewRequest(&hostv1.SetNotificationRecipientRequest{Target: "node-1", Subject: "operator@example.test"}))
			return err
		}},
	} {
		t.Run(call.name, func(t *testing.T) {
			err := call.invoke()
			if connect.CodeOf(err) != connect.CodeFailedPrecondition || !strings.Contains(err.Error(), "local host target") {
				t.Fatalf("remote mutation error = %v", err)
			}
			if mutationCalls != 0 || reacher.request.Procedure != "" {
				t.Fatalf("mutation calls = %d, Bridge procedure = %q", mutationCalls, reacher.request.Procedure)
			}
		})
	}

	if _, err := client.ListHostRequirements(context.Background(), connect.NewRequest(&hostv1.ListHostRequirementsRequest{Target: "node-1"})); err != nil {
		t.Fatalf("remote read was unexpectedly blocked: %v", err)
	}
	if reacher.request.Procedure != hostconnect.HostServiceListHostRequirementsProcedure {
		t.Fatalf("supported remote read procedure = %q", reacher.request.Procedure)
	}
	var forwarded hostv1.ListHostRequirementsRequest
	if err := proto.Unmarshal(reacher.request.Body, &forwarded); err != nil {
		t.Fatal(err)
	}
	if forwarded.GetTarget() != "local" {
		t.Fatalf("remote read owner target = %q, want local", forwarded.GetTarget())
	}
}

func mustMarshalOperatorInputResponse(t *testing.T, version int) []byte {
	t.Helper()
	body, err := proto.Marshal(&operatorinputsv1.ListOperatorInputsResponse{Version: int32(version)})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestMapErrorUsesDistinctConnectCodes(t *testing.T) {
	deadline := mapError("minimouse", context.DeadlineExceeded)
	if connect.CodeOf(deadline) != connect.CodeDeadlineExceeded || !strings.Contains(deadline.Error(), "minimouse") {
		t.Fatalf("deadline = %v", deadline)
	}
	scope := mapError("minimouse", &nodereach.Error{Kind: nodereach.ErrMissingScope, Scope: "vrooli-onboarding:write"})
	if connect.CodeOf(scope) != connect.CodePermissionDenied || !strings.Contains(scope.Error(), "vrooli-onboarding:write") {
		t.Fatalf("scope = %v", scope)
	}
	offline := mapError("minimouse", &nodereach.Error{Kind: nodereach.ErrNodeUnavailable, Err: errors.New("offline")})
	if connect.CodeOf(offline) != connect.CodeUnavailable || !strings.Contains(offline.Error(), "minimouse") {
		t.Fatalf("offline = %v", offline)
	}
}
