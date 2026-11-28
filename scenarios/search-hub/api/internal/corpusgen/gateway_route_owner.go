package corpusgen

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

// GatewayRouteEvidenceReader is the privacy-safe AI Gateway route owner read
// surface used to verify an exact event ref. It returns no prompts or output.
type GatewayRouteEvidenceReader interface {
	GetRouteEvidence(context.Context, string) (*routingv1.RouteEvidence, error)
}

// DiscoveredGatewayRouteEvidenceReader reads exact event receipts through the
// existing RoutingService Connect API. URL resolution and HTTP authentication
// remain owned by Vrooli discovery and the caller's configured client.
type DiscoveredGatewayRouteEvidenceReader struct {
	HTTPClient *http.Client
	ResolveURL func(context.Context, string) (string, error)
}

func (r DiscoveredGatewayRouteEvidenceReader) GetRouteEvidence(ctx context.Context, eventID string) (*routingv1.RouteEvidence, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("AI Gateway route event ID is required")
	}
	resolve := r.ResolveURL
	if resolve == nil {
		resolve = discovery.ResolveScenarioURLDefault
	}
	baseURL, err := resolve(ctx, "ai-gateway")
	if err != nil {
		return nil, fmt.Errorf("resolve AI Gateway route owner: %w", err)
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("AI Gateway route owner discovery returned an empty URL")
	}
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	owner := routingconnect.NewRoutingServiceClient(client, strings.TrimRight(baseURL, "/"))
	response, err := owner.GetRouteEvidence(ctx, connect.NewRequest(&routingv1.GetRouteEvidenceRequest{EventId: eventID}))
	if err != nil {
		return nil, fmt.Errorf("read AI Gateway route evidence %q: %w", eventID, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetEvent() == nil {
		return nil, fmt.Errorf("AI Gateway returned no route evidence for %q", eventID)
	}
	return response.Msg.GetEvent(), nil
}

// VerifyCodeTaskRoute checks every selection and capacity field against the
// authoritative Gateway event. The deterministic request ID ties this event
// to the exact completed Agent Manager source, role and consumer.
func (r DiscoveredGatewayRouteEvidenceReader) VerifyCodeTaskRoute(ctx context.Context, proof GatewayRouteProof) error {
	if proof.SourceRunID == "" || proof.SourceEventID == "" || proof.TaskRole == "" || proof.Consumer == "" {
		return fmt.Errorf("source, task role, and consumer are required to verify a Gateway route")
	}
	wantRequestID, err := searcheval.AgentManagerReplayRequestID(proof.SourceRunID, proof.SourceEventID, int64(proof.SourceSequence), proof.TaskRole, proof.Consumer)
	if err != nil || proof.GatewayRequestID != wantRequestID {
		return fmt.Errorf("Gateway request ID does not bind the source event and route scope")
	}
	event, err := r.GetRouteEvidence(ctx, proof.GatewayEventID)
	if err != nil {
		return err
	}
	return VerifyGatewayRouteProof(proof, event)
}

func VerifyGatewayRouteProof(proof GatewayRouteProof, event *routingv1.RouteEvidence) error {
	if event == nil || event.GetEventId() != proof.GatewayEventID || event.GetRequestId() != proof.GatewayRequestID ||
		event.GetTaskRole() != proof.TaskRole || event.GetScenario() != proof.Consumer ||
		event.GetStatus() != proof.Status || event.GetStatus() != "succeeded" ||
		event.GetSelectedProvider() != proof.SelectedProvider || event.GetSelectedModel() != proof.SelectedModel ||
		event.GetSelectedResourceRole() != proof.SelectedResourceRole || event.GetSelectedLocality() != proof.Locality || proof.Locality != "local" ||
		event.GetCapacityVerdict() != proof.CapacityVerdict || proof.CapacityVerdict != "fit" ||
		event.GetCapacityClaimId() != proof.CapacityClaimID || event.GetCapacityRequiredBytes() != proof.CapacityRequiredBytes ||
		event.GetCapacityGrantedBytes() != proof.CapacityGrantedBytes || proof.CapacityRequestedBytes != proof.CapacityRequiredBytes ||
		proof.CapacityGrantedBytes < proof.CapacityRequiredBytes {
		return fmt.Errorf("AI Gateway route receipt does not match the candidate's exact route and capacity identity")
	}
	if proof.GatewayRoleEvidenceURI != "gateway-ledger://events/"+event.GetEventId() {
		return fmt.Errorf("Gateway role evidence URI does not identify the verified event")
	}
	return nil
}
