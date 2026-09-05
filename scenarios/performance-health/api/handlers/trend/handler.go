// Package trend mounts performance-health's TrendService — additive, newest-first
// reads of the persisted per-scenario performance trend.
package trend

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internaltrend "performance-health/internal/trend"

	trendv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend"
	trendconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend/trend_v1connect"
)

// Handler implements the generated TrendServiceHandler.
type Handler struct {
	trendconnect.UnimplementedTrendServiceHandler
	svc    *internaltrend.Service
	logger *log.Logger
}

// NewHandler builds a trend Handler.
func NewHandler(svc *internaltrend.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ trendconnect.TrendServiceHandler = (*Handler)(nil)

// GetTrend returns a scenario's persisted samples, newest first.
func (h *Handler) GetTrend(ctx context.Context, req *connect.Request[trendv1.GetTrendRequest]) (*connect.Response[trendv1.GetTrendResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	samples, err := h.svc.Trend(ctx, scenario, int(req.Msg.GetLimit()))
	if err != nil {
		h.logger.Printf("trend.GetTrend(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &trendv1.GetTrendResponse{Scenario: scenario}
	for _, s := range samples {
		out.Samples = append(out.Samples, &trendv1.TrendSample{
			Scenario:              s.Scenario,
			CapturedAt:            s.CapturedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			GoBuildMs:             s.GoBuildMs,
			UiBuildMs:             s.UIBuildMs,
			BundleBytes:           s.BundleBytes,
			LcpMs:                 s.LCPMs,
			Cls:                   s.CLS,
			ResponseEndMs:         s.ResponseEndMs,
			DomInteractiveMs:      s.DOMInteractiveMs,
			DomContentLoadedMs:    s.DOMContentLoadedMs,
			LoadEventEndMs:        s.LoadEventEndMs,
			NavigationType:        s.NavigationType,
			StartupMs:             s.StartupMs,
			SlowestComponent:      s.SlowestComponent,
			SlowestComponentAvgMs: s.SlowestComponentAvgMs,
			SlowestComponentMaxMs: s.SlowestComponentMaxMs,
			Note:                  s.Note,
		})
	}
	return connect.NewResponse(out), nil
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
