package observability

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/handlers"
	observabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/observability"
)

// service implements observabilityconnect.ObservabilityServiceHandler.
type service struct {
	deps Deps
}

// ---------------------------------------------------------------------------
// Snapshot + refresh
// ---------------------------------------------------------------------------

func (s *service) GetObservability(
	ctx context.Context,
	req *connect.Request[observabilityv1.GetObservabilityRequest],
) (*connect.Response[observabilityv1.GetObservabilityResponse], error) {
	payload, err := s.deps.Proxy.FetchObservability(ctx, strings.TrimSpace(req.Msg.GetDepth()), req.Msg.GetNoCache())
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.GetObservabilityResponse{Snapshot: pb}), nil
}

func (s *service) RefreshObservability(
	ctx context.Context,
	_ *connect.Request[observabilityv1.RefreshObservabilityRequest],
) (*connect.Response[observabilityv1.RefreshObservabilityResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilityRefresh(ctx)
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.RefreshObservabilityResponse{Result: pb}), nil
}

// ---------------------------------------------------------------------------
// Diagnostics
// ---------------------------------------------------------------------------

func (s *service) RunDiagnostics(
	ctx context.Context,
	req *connect.Request[observabilityv1.RunDiagnosticsRequest],
) (*connect.Response[observabilityv1.RunDiagnosticsResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilityDiagnostics(ctx, structToMap(req.Msg.GetOptions()))
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.RunDiagnosticsResponse{Result: pb}), nil
}

// ---------------------------------------------------------------------------
// Sessions / cleanup / metrics
// ---------------------------------------------------------------------------

func (s *service) GetSessionList(
	ctx context.Context,
	_ *connect.Request[observabilityv1.GetSessionListRequest],
) (*connect.Response[observabilityv1.GetSessionListResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilitySessions(ctx)
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.GetSessionListResponse{Result: pb}), nil
}

func (s *service) RunCleanup(
	ctx context.Context,
	_ *connect.Request[observabilityv1.RunCleanupRequest],
) (*connect.Response[observabilityv1.RunCleanupResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilityCleanup(ctx)
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.RunCleanupResponse{Result: pb}), nil
}

func (s *service) GetMetrics(
	ctx context.Context,
	_ *connect.Request[observabilityv1.GetMetricsRequest],
) (*connect.Response[observabilityv1.GetMetricsResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilityMetrics(ctx)
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.GetMetricsResponse{Result: pb}), nil
}

// ---------------------------------------------------------------------------
// Pipeline test
// ---------------------------------------------------------------------------

func (s *service) RunPipelineTest(
	ctx context.Context,
	req *connect.Request[observabilityv1.RunPipelineTestRequest],
) (*connect.Response[observabilityv1.RunPipelineTestResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilityPipelineTest(ctx, structToMap(req.Msg.GetOptions()))
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.RunPipelineTestResponse{Result: pb}), nil
}

// ---------------------------------------------------------------------------
// Runtime config
// ---------------------------------------------------------------------------

func (s *service) GetConfigRuntime(
	ctx context.Context,
	_ *connect.Request[observabilityv1.GetConfigRuntimeRequest],
) (*connect.Response[observabilityv1.GetConfigRuntimeResponse], error) {
	payload, err := s.deps.Proxy.FetchObservabilityConfigRuntime(ctx)
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.GetConfigRuntimeResponse{Result: pb}), nil
}

func (s *service) UpdateConfig(
	ctx context.Context,
	req *connect.Request[observabilityv1.UpdateConfigRequest],
) (*connect.Response[observabilityv1.UpdateConfigResponse], error) {
	envVar := strings.TrimSpace(req.Msg.GetEnvVar())
	if envVar == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEnvVarRequired)
	}
	payload, err := s.deps.Proxy.UpdateObservabilityConfig(ctx, envVar, req.Msg.GetValue())
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.UpdateConfigResponse{Result: pb}), nil
}

func (s *service) ResetConfig(
	ctx context.Context,
	req *connect.Request[observabilityv1.ResetConfigRequest],
) (*connect.Response[observabilityv1.ResetConfigResponse], error) {
	envVar := strings.TrimSpace(req.Msg.GetEnvVar())
	if envVar == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEnvVarRequired)
	}
	payload, err := s.deps.Proxy.ResetObservabilityConfig(ctx, envVar)
	if err != nil {
		return nil, s.mapProxyError(err)
	}
	pb, err := mapToStruct(payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&observabilityv1.ResetConfigResponse{Result: pb}), nil
}

// ---------------------------------------------------------------------------
// Debug mode (in-process state, no proxy round-trip)
// ---------------------------------------------------------------------------

func (s *service) GetDebugMode(
	_ context.Context,
	_ *connect.Request[observabilityv1.GetDebugModeRequest],
) (*connect.Response[observabilityv1.DebugModeState], error) {
	snap := handlers.GetDebugModeSnapshot()
	return connect.NewResponse(snapshotToProto(snap)), nil
}

func (s *service) SetDebugMode(
	ctx context.Context,
	req *connect.Request[observabilityv1.SetDebugModeRequest],
) (*connect.Response[observabilityv1.DebugModeState], error) {
	msg := req.Msg
	snap := handlers.SetDebugModeState(msg.GetEnabled(), msg.GetComponents(), int(msg.GetDurationMinutes()))
	if s.deps.Logger != nil {
		fields := map[string]any{
			"enabled":    snap.Enabled,
			"components": snap.Components,
		}
		if snap.Enabled {
			fields["expires_at"] = snap.ExpiresAt
		}
		s.deps.Logger.WithFields(fields).Info("observability: debug mode toggled")
	}
	_ = ctx
	return connect.NewResponse(snapshotToProto(snap)), nil
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func (s *service) mapProxyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, handlers.ErrUpstreamUnavailable) {
		if s.deps.Logger != nil {
			s.deps.Logger.WithError(err).Warn("observability: playwright-driver unavailable")
		}
		return connect.NewError(connect.CodeUnavailable, err)
	}
	var proxyErr *handlers.ObservabilityProxyError
	if errors.As(err, &proxyErr) {
		code := connect.CodeInternal
		switch {
		case proxyErr.StatusCode == 400:
			code = connect.CodeInvalidArgument
		case proxyErr.StatusCode == 404:
			code = connect.CodeNotFound
		case proxyErr.StatusCode == 409:
			code = connect.CodeFailedPrecondition
		case proxyErr.StatusCode >= 500:
			code = connect.CodeInternal
		}
		return connect.NewError(code, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// ---------------------------------------------------------------------------
// proto conversion helpers
// ---------------------------------------------------------------------------

func mapToStruct(in map[string]any) (*structpb.Struct, error) {
	if in == nil {
		return structpb.NewStruct(map[string]any{})
	}
	return structpb.NewStruct(in)
}

func structToMap(in *structpb.Struct) map[string]any {
	if in == nil {
		return nil
	}
	return in.AsMap()
}

func snapshotToProto(snap handlers.DebugModeSnapshot) *observabilityv1.DebugModeState {
	out := &observabilityv1.DebugModeState{
		Enabled:       snap.Enabled,
		Components:    append([]string(nil), snap.Components...),
		RemainingMins: int32(snap.RemainingMins),
	}
	if !snap.ExpiresAt.IsZero() {
		out.ExpiresAt = snap.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}
