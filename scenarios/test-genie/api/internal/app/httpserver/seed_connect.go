package httpserver

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	seedsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds/seeds_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type seedConnectService struct{ server *Server }

var _ seedsconnect.SeedServiceHandler = (*seedConnectService)(nil)

func (h *seedConnectService) ApplySeed(ctx context.Context, req *connect.Request[seedspb.ApplySeedRequest]) (*connect.Response[seedspb.ApplySeedResponse], error) {
	if strings.TrimSpace(req.Msg.GetRequestId()) == "" || len(req.Msg.GetRequestId()) > 200 || len(req.Msg.GetScenario()) > 128 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bounded scenario and request identity are required"))
	}
	if req.Header().Get("X-Dry-Run") == "true" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("seed apply requires effects; dry-run does not acquire a lease"))
	}
	token, record, err := h.server.applySeedSession(ctx, req.Msg.GetScenario(), req.Msg.GetScenarioPath(), req.Msg.GetRequestId(), false)
	if err != nil && (record == nil || record.Session == nil) {
		return nil, err
	}
	evidence := record.Session.SeedEvidence()
	evidence.Scenario = record.Scenario
	params, err := structpb.NewStruct(record.Session.SeedState)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		_, _ = h.server.clearSeedSession(cleanupCtx, record.Scenario, token)
		return nil, connect.NewError(connect.CodeInternal, errors.New("seed values do not satisfy the JSON input contract"))
	}
	return connect.NewResponse(&seedspb.ApplySeedResponse{SessionToken: token, Evidence: evidence, InitialParams: params}), nil
}

func (h *seedConnectService) GetSeed(_ context.Context, req *connect.Request[seedspb.GetSeedRequest]) (*connect.Response[seedspb.GetSeedResponse], error) {
	record, err := h.server.observeSeedSession(req.Msg.GetScenario(), req.Msg.GetSessionToken())
	if err != nil {
		return nil, err
	}
	evidence := record.Session.SeedEvidence()
	evidence.Scenario = record.Scenario
	return connect.NewResponse(&seedspb.GetSeedResponse{Evidence: evidence}), nil
}

func (h *seedConnectService) ClearSeed(ctx context.Context, req *connect.Request[seedspb.ClearSeedRequest]) (*connect.Response[seedspb.ClearSeedResponse], error) {
	if req.Header().Get("X-Dry-Run") == "true" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("seed cleanup requires effects"))
	}
	record, err := h.server.clearSeedSession(ctx, req.Msg.GetScenario(), req.Msg.GetSessionToken())
	if record == nil {
		return nil, err
	}
	evidence := record.Session.SeedEvidence()
	evidence.Scenario = record.Scenario
	if err != nil {
		evidence.State = seedspb.SeedState_SEED_STATE_FAILED
		evidence.CleanupError = "seed cleanup failed; inspect retained owner evidence"
	}
	return connect.NewResponse(&seedspb.ClearSeedResponse{Evidence: evidence}), nil
}
