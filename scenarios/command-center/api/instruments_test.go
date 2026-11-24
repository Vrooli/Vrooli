package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
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

type changingPromptManagerInstrumentFixture struct {
	memberflowconnect.UnimplementedMemberflowServiceHandler
	status    string
	fail      bool
	malformed bool
}

func (f *changingPromptManagerInstrumentFixture) GetInstruments(context.Context, *connect.Request[memberflowv1.EmptyRequest]) (*connect.Response[memberflowv1.JsonResponse], error) {
	if f.fail {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("synthetic Prompt Manager loss"))
	}
	if f.malformed {
		data, err := structpb.NewStruct(map[string]any{"teams": "not-a-list"})
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&memberflowv1.JsonResponse{Data: structpb.NewStructValue(data)}), nil
	}
	data, err := structpb.NewStruct(map[string]any{"teams": []any{map[string]any{"teamId": "director-swarm", "instrument": map[string]any{"status": f.status, "archetype": "portfolio"}}}})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&memberflowv1.JsonResponse{Data: structpb.NewStructValue(data)}), nil
}

func TestPromptManagerDeclarationsRetainIdentityAcrossProviderLossAndRefresh(t *testing.T) {
	fixture := &changingPromptManagerInstrumentFixture{status: "live"}
	_, handler := memberflowconnect.NewMemberflowServiceHandler(fixture)
	server := httptest.NewServer(handler)
	defer server.Close()
	provider := newPromptManagerInstrumentProvider(func() string { return server.URL })

	project := func() SourceBinding {
		server := &Server{
			registry:            &Registry{},
			cache:               NewCache(),
			integrationRegistry: capreg.New(nil, nil, 5*time.Second),
			instrumentProvider:  provider,
		}
		entry := MetricEntry{Coverage: CoverageMissing, Source: SourceBinding{Team: "director-swarm"}}
		readings, _ := server.readings(context.Background(), []MetricEntry{entry})
		return readings[0].Source
	}
	if got := project(); got.InstrumentStatus != "live" || got.InstrumentArchetype != "portfolio" {
		t.Fatalf("initial projected source = %+v, want live portfolio", got)
	}
	fixture.fail = true
	if got := project(); got.InstrumentStatus != "unknown" || got.InstrumentArchetype != "portfolio" {
		t.Fatalf("projected source during provider loss = %+v, want retained identity with unknown status", got)
	}
	fixture.fail = false
	fixture.malformed = true
	if got := project(); got.InstrumentStatus != "unknown" || got.InstrumentArchetype != "portfolio" {
		t.Fatalf("projected source during malformed response = %+v, want retained identity with unknown status", got)
	}
	fixture.malformed = false
	fixture.status = "paused"
	if got := project(); got.InstrumentStatus != "paused" || got.InstrumentArchetype != "portfolio" {
		t.Fatalf("recovered projected source = %+v, want current owner state", got)
	}
}
