// Package startup mounts performance-health's StartupService — startup-perf
// benchmarking + trend reads (axis ②, migrated from structure-health). It is
// decoupled from validation and is NEVER invoked by a test-genie phase.
package startup

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"

	internalstartup "performance-health/internal/startup"

	startupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/startup"
	startupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/startup/startup_v1connect"
)

// Handler implements the generated StartupServiceHandler.
type Handler struct {
	startupconnect.UnimplementedStartupServiceHandler
	svc    *internalstartup.Service
	logger *log.Logger
}

// NewHandler builds a startup Handler.
func NewHandler(svc *internalstartup.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ startupconnect.StartupServiceHandler = (*Handler)(nil)

// BenchmarkStartup restarts the target scenario, measures its startup, persists
// the measurement, and returns it.
func (h *Handler) BenchmarkStartup(ctx context.Context, req *connect.Request[startupv1.BenchmarkStartupRequest]) (*connect.Response[startupv1.BenchmarkStartupResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	timeout := time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	m, err := h.svc.Benchmark(ctx, scenario, timeout)
	if err != nil {
		h.logger.Printf("startup.BenchmarkStartup(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&startupv1.BenchmarkStartupResponse{Measurement: measurementToProto(m)}), nil
}

// GetStartupTrend returns the persisted measurements for a scenario, newest first.
func (h *Handler) GetStartupTrend(ctx context.Context, req *connect.Request[startupv1.GetStartupTrendRequest]) (*connect.Response[startupv1.GetStartupTrendResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	measurements, err := h.svc.History(ctx, scenario, int(req.Msg.GetLimit()))
	if err != nil {
		h.logger.Printf("startup.GetStartupTrend(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &startupv1.GetStartupTrendResponse{Scenario: scenario}
	for _, m := range measurements {
		out.Measurements = append(out.Measurements, measurementToProto(m))
	}
	return connect.NewResponse(out), nil
}

func measurementToProto(m internalstartup.Measurement) *startupv1.StartupMeasurement {
	out := &startupv1.StartupMeasurement{
		Scenario:        m.Scenario,
		CapturedAt:      m.CapturedAt.UTC().Format(time.RFC3339Nano),
		TimeToHealthyMs: m.TimeToHealthyMs,
		Healthy:         m.Healthy,
		Metrics:         m.Metrics,
		Note:            m.Note,
	}
	for _, st := range m.SurfaceTimings {
		out.SurfaceTimings = append(out.SurfaceTimings, &startupv1.SurfaceTiming{
			Surface:         st.Surface,
			TimeToHealthyMs: st.TimeToHealthyMs,
			Healthy:         st.Healthy,
		})
	}
	return out
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
