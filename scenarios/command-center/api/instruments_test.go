package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	memberflowv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow"
	memberflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/memberflow/memberflow_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type promptManagerInstrumentFixture struct {
	memberflowconnect.UnimplementedMemberflowServiceHandler
}

func (promptManagerInstrumentFixture) GetInstruments(context.Context, *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	data, err := structpb.NewStruct(map[string]any{"teams": []any{map[string]any{"teamId": "director-swarm", "instrument": map[string]any{"status": "live", "archetype": "portfolio"}}}})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&memberflowv1.JsonResponse{Data: structpb.NewStructValue(data)}), nil
}

func TestPromptManagerInstrumentProviderChecksTypedTransmitter(t *testing.T) {
	_, handler := memberflowconnect.NewMemberflowServiceHandler(promptManagerInstrumentFixture{})
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := newPromptManagerInstrumentProvider(func() string { return server.URL })
	result := provider.CheckResult(context.Background())
	if result.Status != "available" || result.FeatureStatus["team_instrument"] != "compatible" {
		t.Fatalf("result = %+v, want compatible typed transmitter", result)
	}
	declarations := provider.Declarations(context.Background())
	if declarations["director-swarm"]["status"] != "live" {
		t.Fatalf("declarations = %+v", declarations)
	}
}

func TestPromptManagerInstrumentProviderReportsUnavailableTransport(t *testing.T) {
	provider := newPromptManagerInstrumentProvider(func() string { return "http://127.0.0.1:1" })
	provider.http = &http.Client{}
	result := provider.CheckResult(context.Background())
	if result.Status != "unavailable" || result.ReasonCode != "feature_transmitter_unavailable" {
		t.Fatalf("result = %+v, want unavailable transmitter", result)
	}
}

func TestPromptManagerInstrumentProviderReresolvesAfterTransportFailure(t *testing.T) {
	_, handler := memberflowconnect.NewMemberflowServiceHandler(promptManagerInstrumentFixture{})
	server := httptest.NewServer(handler)
	defer server.Close()

	resolutions := 0
	provider := newPromptManagerInstrumentProvider(func() string {
		resolutions++
		if resolutions == 1 {
			return "http://127.0.0.1:1"
		}
		return server.URL
	})
	result := provider.CheckResult(context.Background())
	if result.Status != "available" || result.ReasonCode != "typed_transmitter_reachable" {
		t.Fatalf("result = %+v, want available typed transmitter", result)
	}
	if resolutions != 2 {
		t.Fatalf("resolver calls = %d, want one retry with re-resolution", resolutions)
	}
}
