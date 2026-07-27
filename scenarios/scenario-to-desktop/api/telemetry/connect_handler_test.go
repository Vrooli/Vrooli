package telemetry

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConnectServiceDeleteTelemetryReturnsTypedAcknowledgement(t *testing.T) {
	service := NewService(t.TempDir())
	handler := NewHandler(service)
	connectService := NewConnectService(handler)
	event, err := structpb.NewStruct(map[string]any{"event": "startup"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connectService.IngestTelemetry(context.Background(), connect.NewRequest(&domainv1.IngestTelemetryRequest{
		ScenarioName: "demo",
		Events:       []*structpb.Struct{event},
	})); err != nil {
		t.Fatalf("IngestTelemetry() error = %v", err)
	}

	response, err := connectService.DeleteTelemetry(context.Background(), connect.NewRequest(&domainv1.TelemetryScenarioRequest{ScenarioName: "demo"}))
	if err != nil {
		t.Fatalf("DeleteTelemetry() error = %v", err)
	}
	if !response.Msg.GetDeleted() || response.Msg.GetScenarioName() != "demo" {
		t.Fatalf("response = %#v, want typed successful acknowledgement", response.Msg)
	}
}
