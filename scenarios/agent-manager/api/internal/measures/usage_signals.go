package measures

import (
	"context"
	"errors"
	"strings"
	"time"

	"agent-manager/internal/interventions"

	"connectrpc.com/connect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// SignalControl is the operator surface over the operator-signal switch.
// interventions.SignalControl implements it.
type SignalControl interface {
	Status(ctx context.Context) (interventions.SignalStatus, error)
	Set(enabled bool, by string) error
	Purge(ctx context.Context) (int64, error)
}

// SetSignalControl connects the switch. Without it the signal procedures report unavailable.
func (h *Handler) SetSignalControl(control SignalControl) { h.signalControl = control }

func protoSignalStatus(status interventions.SignalStatus) *measurepb.UsageSignalsStatusResponse {
	response := &measurepb.UsageSignalsStatusResponse{Enabled: status.Enabled, Source: status.Source, ChangedBy: status.ChangedBy, Stored: status.Stored, Note: status.Note}
	if !status.ChangedAt.IsZero() {
		response.ChangedAt = status.ChangedAt.UTC().Format(time.RFC3339)
	}
	return response
}

func (h *Handler) signalsControl() (SignalControl, error) {
	if h.signalControl == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("the operator-signal switch is unavailable"))
	}
	return h.signalControl, nil
}

func (h *Handler) UsageSignalsStatus(ctx context.Context, _ *connect.Request[measurepb.UsageSignalsStatusRequest]) (*connect.Response[measurepb.UsageSignalsStatusResponse], error) {
	control, err := h.signalsControl()
	if err != nil {
		return nil, err
	}
	status, err := control.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(protoSignalStatus(status)), nil
}

func (h *Handler) SetUsageSignals(ctx context.Context, req *connect.Request[measurepb.SetUsageSignalsRequest]) (*connect.Response[measurepb.SetUsageSignalsResponse], error) {
	operator, err := requireModeOperator(ctx)
	if err != nil {
		return nil, err
	}
	control, err := h.signalsControl()
	if err != nil {
		return nil, err
	}
	response := &measurepb.SetUsageSignalsResponse{}
	switch action := strings.ToLower(strings.TrimSpace(req.Msg.GetAction())); action {
	case "off", "on":
		if err := control.Set(action == "on", operator); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	case "purge":
		purged, err := control.Purge(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response.Purged = purged
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("action must be off, on or purge"))
	}
	status, err := control.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response.Status = protoSignalStatus(status)
	return connect.NewResponse(response), nil
}
