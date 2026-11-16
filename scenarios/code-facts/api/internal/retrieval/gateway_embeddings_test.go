package retrieval

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type fakeEmbeddingRoute struct {
	request  *routingv1.ExecuteRouteRequest
	response *routingv1.ExecuteRouteResponse
}

func (f *fakeEmbeddingRoute) ExecuteRoute(_ context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(f.response), nil
}

func TestGatewayEmbedderUsesLocalConfidentialRoleContract(t *testing.T) {
	client := &fakeEmbeddingRoute{response: &routingv1.ExecuteRouteResponse{
		Valid: true, OutputText: `{"embedding":[0.25,-0.5,1]}`,
		Evidence: &routingv1.RouteEvidence{SelectedLocality: "local"},
	}}
	embedder := &GatewayEmbedder{Client: client, Dimensions: 3, TaskPrefix: "search_document: "}
	vectors, err := embedder.Embed(context.Background(), []string{"repository source excerpt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 3 || vectors[0][1] != -0.5 {
		t.Fatalf("unexpected vectors: %#v", vectors)
	}
	req := client.request.GetRequest()
	if req.GetKind() != sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING || req.GetRole() != EmbeddingRole {
		t.Fatalf("embedding task contract mismatch: %+v", req)
	}
	if req.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || req.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || req.GetPrivacyProvenance() != codeFactsProvenance {
		t.Fatalf("embedding privacy contract mismatch: %+v", req)
	}
	if client.request.GetInputText() != "search_document: repository source excerpt" {
		t.Fatalf("document task prefix missing: %q", client.request.GetInputText())
	}
}

func TestGatewayEmbedderRejectsRemoteRoute(t *testing.T) {
	client := &fakeEmbeddingRoute{response: &routingv1.ExecuteRouteResponse{
		Valid: true, OutputText: `{"embedding":[1,2]}`, Evidence: &routingv1.RouteEvidence{SelectedLocality: "remote"},
	}}
	if _, err := (&GatewayEmbedder{Client: client, Dimensions: 2}).Embed(context.Background(), []string{"private source"}); err == nil {
		t.Fatal("remote embedding route must be rejected")
	}
}
