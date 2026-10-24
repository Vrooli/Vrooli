package handlers

// DOC: docs/reference/api-endpoints.md#alerts

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AlertSource is the evaluator surface the alerts handler reads.
type AlertSource interface {
	ListAlerts(ctx context.Context, query services.AlertQuery) ([]*models.Alert, error)
	RuleStatuses() []services.RuleStatus
	EvaluatedAt() time.Time
	SustainDuration() time.Duration
	MonitoringActive() bool
}

// AlertAcknowledger records operator acknowledgements.
type AlertAcknowledger interface {
	AcknowledgeAlert(ctx context.Context, id string, ackedBy string) error
	GetAlert(ctx context.Context, id string) (*models.Alert, error)
}

// AlertsHandler serves server-owned alert state over Connect.
type AlertsHandler struct {
	source AlertSource
	acks   AlertAcknowledger
	log    *slog.Logger
}

// NewAlertsHandler creates the alerts handler.
func NewAlertsHandler(source AlertSource, acks AlertAcknowledger, log *slog.Logger) *AlertsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &AlertsHandler{source: source, acks: acks, log: log}
}

// ListAlerts returns persisted alerts and the evaluator's rule states.
func (h *AlertsHandler) ListAlerts(ctx context.Context, req *connect.Request[alertspb.ListAlertsRequest]) (*connect.Response[alertspb.ListAlertsResponse], error) {
	status := strings.TrimSpace(req.Msg.GetStatus())
	switch status {
	case "", "active", "resolved", "all":
	default:
		return nil, connectError(apierrors.Validation("status", "must be active, resolved, or all"))
	}
	alerts, err := h.source.ListAlerts(ctx, services.AlertQuery{Status: status, IncludeEvents: req.Msg.GetIncludeEvents(), Limit: int(req.Msg.GetLimit())})
	if err != nil {
		return nil, connectError(apierrors.Internal("failed to list alerts", err))
	}
	resp := &alertspb.ListAlertsResponse{
		Rules:            convert.RuleStatusesToProto(h.source.RuleStatuses()),
		SustainSeconds:   int32(h.source.SustainDuration() / time.Second),
		MonitoringActive: h.source.MonitoringActive(),
	}
	for _, alert := range alerts {
		resp.Alerts = append(resp.Alerts, convert.AlertToProto(alert))
	}
	if evaluated := h.source.EvaluatedAt(); !evaluated.IsZero() {
		resp.EvaluatedAt = timestamppb.New(evaluated)
	}
	return connect.NewResponse(resp), nil
}

// AcknowledgeAlert records an acknowledgement without resolving the alert.
func (h *AlertsHandler) AcknowledgeAlert(ctx context.Context, req *connect.Request[alertspb.AcknowledgeAlertRequest]) (*connect.Response[alertspb.AcknowledgeAlertResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connectError(apierrors.Validation("id", "is required"))
	}
	ackedBy := strings.TrimSpace(req.Msg.GetAckedBy())
	if ackedBy == "" {
		ackedBy = "operator"
	}
	if err := h.acks.AcknowledgeAlert(ctx, id, ackedBy); err != nil {
		return nil, connectError(apierrors.NotFound("alert", id))
	}
	alert, err := h.acks.GetAlert(ctx, id)
	if err != nil {
		return nil, connectError(apierrors.NotFound("alert", id))
	}
	return connect.NewResponse(&alertspb.AcknowledgeAlertResponse{Alert: convert.AlertToProto(alert)}), nil
}
