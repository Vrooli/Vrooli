package corpusgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

type routeEvidenceFixture struct {
	routingconnect.UnimplementedRoutingServiceHandler
	event *routingv1.RouteEvidence
}

func (f routeEvidenceFixture) GetRouteEvidence(context.Context, *connect.Request[routingv1.GetRouteEvidenceRequest]) (*connect.Response[routingv1.GetRouteEvidenceResponse], error) {
	return connect.NewResponse(&routingv1.GetRouteEvidenceResponse{Event: f.event}), nil
}

func TestDiscoveredGatewayRouteReaderVerifiesExactOwnerEvent(t *testing.T) {
	runID, eventID, taskRole, consumer := "run-1", "source-event-1", "coding", "opencode"
	requestID, err := searcheval.AgentManagerReplayRequestID(runID, eventID, 9, taskRole, consumer)
	if err != nil {
		t.Fatal(err)
	}
	proof := GatewayRouteProof{
		SourceRunID: runID, SourceEventID: eventID, SourceSequence: 9,
		GatewayEventID: "gateway-event", GatewayRequestID: requestID, GatewayRoleEvidenceURI: "gateway-ledger://events/gateway-event",
		TaskRole: taskRole, Consumer: consumer, SelectedResourceRole: "code.local", Status: "succeeded",
		SelectedProvider: "ollama", SelectedModel: "qwen3.5:9b", Locality: "local",
		CapacityClaimID: "claim-1", CapacityRequestedBytes: 1024, CapacityRequiredBytes: 1024,
		CapacityGrantedBytes: 2048, CapacityVerdict: "fit", CapacityEstimateEvidenceURI: "capacity-footprint://claim-1",
	}
	event := &routingv1.RouteEvidence{
		EventId: proof.GatewayEventID, RequestId: proof.GatewayRequestID, Scenario: proof.Consumer,
		TaskRole: proof.TaskRole, SelectedResourceRole: proof.SelectedResourceRole, Status: proof.Status,
		SelectedProvider: proof.SelectedProvider, SelectedModel: proof.SelectedModel, SelectedLocality: proof.Locality,
		CapacityClaimId: proof.CapacityClaimID, CapacityRequiredBytes: proof.CapacityRequiredBytes,
		CapacityGrantedBytes: proof.CapacityGrantedBytes, CapacityVerdict: proof.CapacityVerdict,
	}
	path, handler := routingconnect.NewRoutingServiceHandler(routeEvidenceFixture{event: event})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	reader := DiscoveredGatewayRouteEvidenceReader{HTTPClient: server.Client(), ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }}
	if err := reader.VerifyCodeTaskRoute(context.Background(), proof); err != nil {
		t.Fatalf("exact Gateway route rejected: %v", err)
	}
	proof.SelectedModel = "different-model"
	if err := reader.VerifyCodeTaskRoute(context.Background(), proof); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mutated route identity accepted: %v", err)
	}
}

func TestGatewayRouteProofRejectsMisboundCapacityAndSourceRequest(t *testing.T) {
	event := &routingv1.RouteEvidence{
		EventId: "gateway-event", RequestId: "request-id", Scenario: "opencode", TaskRole: "coding",
		SelectedProvider: "ollama", SelectedModel: "qwen3.5:9b", SelectedResourceRole: "code.local",
		SelectedLocality: "local", Status: "succeeded", CapacityClaimId: "claim-1",
		CapacityRequiredBytes: 1024, CapacityGrantedBytes: 2048, CapacityVerdict: "fit",
	}
	proof := GatewayRouteProof{
		GatewayEventID: "gateway-event", GatewayRequestID: "request-id", GatewayRoleEvidenceURI: "gateway-ledger://events/gateway-event",
		TaskRole: "coding", Consumer: "opencode", SelectedResourceRole: "code.local", Status: "succeeded",
		SelectedProvider: "ollama", SelectedModel: "qwen3.5:9b", Locality: "local", CapacityClaimID: "claim-1",
		CapacityRequestedBytes: 1024, CapacityRequiredBytes: 1024, CapacityGrantedBytes: 2048, CapacityVerdict: "fit",
	}
	if err := VerifyGatewayRouteProof(proof, event); err != nil {
		t.Fatalf("exact route rejected: %v", err)
	}
	proof.CapacityClaimID = "other-claim"
	if err := VerifyGatewayRouteProof(proof, event); err == nil {
		t.Fatal("different capacity claim accepted")
	}
}
