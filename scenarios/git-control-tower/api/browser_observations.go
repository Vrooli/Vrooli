package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

// A narrow same-origin facade. UI Health owns retention/readback; GCT accepts
// only its own bounded geometry reports and exposes no general-purpose proxy.
type browserObservationFacade struct {
	rpc.UnimplementedVisualHealthServiceHandler
	base BaseClient
}

func (s browserObservationFacade) RecordBrowserObservation(ctx context.Context, req *connect.Request[pb.RecordBrowserObservationRequest]) (*connect.Response[pb.RecordBrowserObservationResponse], error) {
	row := req.Msg.GetObservation()
	if row == nil || len(row.GetLayoutJson()) > 16384 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bounded observation required"))
	}
	row.Scenario = "git-control-tower"
	url, err := s.base.resolveBaseURL(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return rpc.NewVisualHealthServiceClient(s.base.httpClient, url).RecordBrowserObservation(ctx, req)
}

func browserObservationHandler() (string, http.Handler) {
	return rpc.NewVisualHealthServiceHandler(browserObservationFacade{base: NewBaseClient("ui-health", 5*time.Second)}, connect.WithReadMaxBytes(24*1024))
}
