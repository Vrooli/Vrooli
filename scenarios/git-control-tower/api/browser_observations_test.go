package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type observationReceiver struct {
	rpc.UnimplementedVisualHealthServiceHandler
	scenario string
}

func (s *observationReceiver) RecordBrowserObservation(_ context.Context, req *connect.Request[pb.RecordBrowserObservationRequest]) (*connect.Response[pb.RecordBrowserObservationResponse], error) {
	s.scenario = req.Msg.GetObservation().GetScenario()
	return connect.NewResponse(&pb.RecordBrowserObservationResponse{Observation: req.Msg.Observation}), nil
}

func TestBrowserObservationFacadeForwardsOnlyOwnScenario(t *testing.T) {
	receiver := &observationReceiver{}
	_, handler := rpc.NewVisualHealthServiceHandler(receiver)
	server := httptest.NewServer(handler)
	defer server.Close()
	facade := browserObservationFacade{base: NewBaseClient("ui-health", 0)}
	facade.base.resolver = discovery.NewStaticResolver(server.URL)
	_, err := facade.RecordBrowserObservation(context.Background(), connect.NewRequest(&pb.RecordBrowserObservationRequest{Observation: &pb.BrowserObservation{Scenario: "forged", LayoutJson: "{}"}}))
	if err != nil {
		t.Fatal(err)
	}
	if receiver.scenario != "git-control-tower" {
		t.Fatalf("scenario=%s", receiver.scenario)
	}
	_, err = facade.RecordBrowserObservation(context.Background(), connect.NewRequest(&pb.RecordBrowserObservationRequest{Observation: &pb.BrowserObservation{LayoutJson: strings.Repeat("x", 16385)}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("oversized report: %v", err)
	}
	_, err = facade.ListBrowserObservations(context.Background(), connect.NewRequest(&pb.ListBrowserObservationsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("facade exposed readback: %v", err)
	}
}
