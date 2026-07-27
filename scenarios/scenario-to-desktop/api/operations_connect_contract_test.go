package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scenario-to-desktop-api/scenario"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

func operationsString(value string) *string { return &value }
func operationsInt32(value int32) *int32    { return &value }

func TestOperationsConnectServiceListsDesktopStatusAndProbesEndpoints(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"web-only", "desktop"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "desktop", "platforms", "electron"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "desktop", "platforms", "electron", "package.json"), []byte(`{"name":"Desktop","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	operations := operationsConnectService{scenarioHandler: scenario.NewHandler(root, nil, slog.Default())}
	status, err := operations.ListDesktopScenarioStatus(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if err != nil || status.Msg.GetStats().GetTotal() != 2 || status.Msg.GetStats().GetWithDesktop() != 1 {
		t.Fatalf("ListDesktopScenarioStatus() = %#v, %v", status, err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer upstream.Close()
	server := NewServer(0)
	t.Cleanup(func() { shutdownServer(t, server) })
	operations.server = server
	probe, err := operations.ProbeEndpoints(context.Background(), connect.NewRequest(&domainv1.ProbeEndpointsRequest{ServerUrl: operationsString(upstream.URL), TimeoutMs: operationsInt32(1000)}))
	if err != nil || probe.Msg.GetServer().GetStatus() != "ok" || probe.Msg.GetServer().GetStatusCode() != http.StatusNoContent || probe.Msg.GetApi().GetStatus() != "skipped" {
		t.Fatalf("ProbeEndpoints() = %#v, %v", probe, err)
	}
}

func TestOperationsConnectServiceMapsConfigurationFailures(t *testing.T) {
	service := operationsConnectService{}
	if _, err := service.ListDesktopScenarioStatus(context.Background(), connect.NewRequest(&emptypb.Empty{})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing scenario handler code = %v", connect.CodeOf(err))
	}
	if _, err := service.ProbeEndpoints(context.Background(), connect.NewRequest(&domainv1.ProbeEndpointsRequest{})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing server code = %v", connect.CodeOf(err))
	}
	server := NewServer(0)
	t.Cleanup(func() { shutdownServer(t, server) })
	service.server = server
	if _, err := service.ProbeEndpoints(context.Background(), connect.NewRequest(&domainv1.ProbeEndpointsRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing endpoints code = %v", connect.CodeOf(err))
	}
	if _, err := service.ResolveScenarioPort(context.Background(), connect.NewRequest(&domainv1.ScenarioPortRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing port fields code = %v", connect.CodeOf(err))
	}
}
