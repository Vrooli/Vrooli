package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"
)

type onboardingFixture struct {
	onboardconnect.UnimplementedOnboardServiceHandler
	request *onboardv1.StartOnboardingRequest
}

func (f *onboardingFixture) StartOnboarding(_ context.Context, req *connect.Request[onboardv1.StartOnboardingRequest]) (*connect.Response[onboardv1.StartOnboardingResponse], error) {
	copy := *req.Msg
	f.request = &copy
	return connect.NewResponse(&onboardv1.StartOnboardingResponse{OpId: "op-1"}), nil
}

func TestStartOnboardingWithPasswordForwardsTransientCredentialAndClearsCallerBuffer(t *testing.T) {
	fixture := &onboardingFixture{}
	mux := http.NewServeMux()
	path, handler := onboardconnect.NewOnboardServiceHandler(fixture)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := &BridgeClient{BaseURL: server.URL, Client: server.Client()}
	password := []byte("synthetic-one-time-password")
	op, err := client.StartOnboardingWithPassword(context.Background(), "vps.example.net", "deploy", "machine-1", password)
	if err != nil {
		t.Fatal(err)
	}
	if op != "op-1" || fixture.request == nil || fixture.request.GetHost() != "vps.example.net" || fixture.request.GetUser() != "deploy" || fixture.request.GetMachineId() != "machine-1" || fixture.request.GetSshPassword() != "synthetic-one-time-password" {
		t.Fatalf("onboarding request = %#v, operation=%q", fixture.request, op)
	}
	for _, value := range password {
		if value != 0 {
			t.Fatal("caller password buffer was not cleared after use")
		}
	}
}
