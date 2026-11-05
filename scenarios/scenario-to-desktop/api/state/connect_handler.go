package state

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

// ConnectService is the typed transport for durable desktop-generator state.
// The persisted state intentionally retains extensible form and stage-result
// documents; conversion through Struct preserves their complete JSON shape.
type ConnectService struct {
	domainconnect.UnimplementedStateServiceHandler
	service *Service
}

var _ domainconnect.StateServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService {
	if handler == nil {
		return &ConnectService{}
	}
	return &ConnectService{service: handler.service}
}

func (s *ConnectService) require() error {
	if s == nil || s.service == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("state service is not configured"))
	}
	return nil
}

func (s *ConnectService) LoadScenarioState(ctx context.Context, request *connect.Request[domainv1.LoadScenarioStateRequest]) (*connect.Response[domainv1.StateResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	result, err := s.service.LoadState(ctx, request.Msg.GetScenarioName(), LoadStateRequest{IncludeLogs: request.Msg.GetIncludeLogs(), ValidateManifest: request.Msg.GetValidateManifest(), ManifestPath: request.Msg.GetManifestPath()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	payload, err := responseStruct(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.StateResponse{Payload: payload, Found: result.Found}), nil
}

func (s *ConnectService) SaveScenarioState(ctx context.Context, request *connect.Request[domainv1.SaveScenarioStateRequest]) (*connect.Response[domainv1.StateOperationResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	var input SaveStateRequest
	if err := structInto(request.Msg.GetPayload(), &input); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := s.service.SaveState(ctx, request.Msg.GetScenarioName(), input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return operationResponse(result)
}

func (s *ConnectService) DeleteScenarioState(ctx context.Context, request *connect.Request[domainv1.StateRequest]) (*connect.Response[domainv1.StateOperationResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	result, err := s.service.ClearState(ctx, request.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return operationResponse(result)
}

func (s *ConnectService) CheckScenarioState(ctx context.Context, request *connect.Request[domainv1.CheckScenarioStateRequest]) (*connect.Response[domainv1.StateOperationResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	var current InputFingerprint
	if err := structInto(request.Msg.GetCurrentConfig(), &current); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := s.service.CheckStaleness(ctx, request.Msg.GetScenarioName(), CheckStalenessRequest{CurrentConfig: current})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return operationResponse(result)
}

func (s *ConnectService) GetScenarioStateLog(ctx context.Context, request *connect.Request[domainv1.ScenarioStateLogRequest]) (*connect.Response[domainv1.ScenarioStateLogResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	result, err := s.service.GetLogs(ctx, request.Msg.GetScenarioName(), request.Msg.GetServiceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if result == nil {
		return connect.NewResponse(&domainv1.ScenarioStateLogResponse{Found: false}), nil
	}
	payload, err := responseStruct(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.ScenarioStateLogResponse{Payload: payload, Found: true}), nil
}

func (s *ConnectService) InvalidateScenarioState(ctx context.Context, request *connect.Request[domainv1.InvalidateScenarioStateRequest]) (*connect.Response[domainv1.StateOperationResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	reason := request.Msg.GetReason()
	if reason == "" {
		reason = "Manual invalidation"
	}
	if err := s.service.InvalidateStagesFrom(ctx, request.Msg.GetScenarioName(), request.Msg.GetFromStage(), reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := s.service.GetValidationStatus(ctx, request.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return operationResponse(result)
}

func operationResponse(value any) (*connect.Response[domainv1.StateOperationResponse], error) {
	payload, err := responseStruct(value)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.StateOperationResponse{Payload: payload}), nil
}

func responseStruct(value any) (*structpb.Struct, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	return structpb.NewStruct(fields)
}

func structInto(value *structpb.Struct, destination any) error {
	if value == nil {
		return fmt.Errorf("payload is required")
	}
	encoded, err := json.Marshal(value.AsMap())
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, destination)
}
