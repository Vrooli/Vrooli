package telemetry

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

type ConnectService struct {
	domainconnect.UnimplementedTelemetryServiceHandler
	service Service
}

var _ domainconnect.TelemetryServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService {
	if handler == nil {
		return &ConnectService{}
	}
	return &ConnectService{service: handler.service}
}

func (s *ConnectService) require() error {
	if s == nil || s.service == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("telemetry service is not configured"))
	}
	return nil
}

func (s *ConnectService) IngestTelemetry(ctx context.Context, request *connect.Request[domainv1.IngestTelemetryRequest]) (*connect.Response[domainv1.IngestTelemetryResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	events := make([]map[string]interface{}, 0, len(request.Msg.GetEvents()))
	for _, event := range request.Msg.GetEvents() {
		events = append(events, event.AsMap())
	}
	path, count, err := s.service.IngestEvents(ctx, request.Msg.GetScenarioName(), request.Msg.GetDeploymentMode(), request.Msg.GetSource(), events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&domainv1.IngestTelemetryResponse{OutputPath: path, EventsIngested: int32(count)}), nil
}

func (s *ConnectService) GetTelemetrySummary(ctx context.Context, request *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	result, err := s.service.GetSummary(ctx, request.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return telemetryPayload(SummaryResponse{ScenarioName: request.Msg.GetScenarioName(), Exists: result.Exists, FilePath: result.FilePath, FileSizeBytes: result.FileSizeBytes, EventCount: result.EventCount, LastIngestedAt: result.LastIngestedAt})
}

func (s *ConnectService) GetTelemetryInsights(ctx context.Context, request *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	result, err := s.service.GetInsights(ctx, request.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return telemetryPayload(InsightsResponse{ScenarioName: request.Msg.GetScenarioName(), Exists: result.Exists, LastSession: result.LastSession, LastSmokeTest: result.LastSmokeTest, LastError: result.LastError})
}

func (s *ConnectService) GetTelemetryTail(ctx context.Context, request *connect.Request[domainv1.TelemetryTailRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	limit := int(request.Msg.GetLimit())
	if limit <= 0 {
		limit = 200
	}
	result, err := s.service.GetTail(ctx, request.Msg.GetScenarioName(), limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return telemetryPayload(TailResponse{ScenarioName: request.Msg.GetScenarioName(), Exists: result.Exists, Limit: result.Limit, TotalLines: result.TotalLines, Entries: result.Entries})
}

func (s *ConnectService) DeleteTelemetry(ctx context.Context, request *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryDeleteResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	if err := s.service.Delete(ctx, request.Msg.GetScenarioName()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&domainv1.TelemetryDeleteResponse{
		ScenarioName: request.Msg.GetScenarioName(),
		Deleted:      true,
	}), nil
}

func telemetryPayload(value any) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	payload, err := structpb.NewStruct(fields)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.TelemetryPayloadResponse{Payload: payload}), nil
}
