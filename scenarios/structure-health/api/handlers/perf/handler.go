// Package perf mounts structure-health's PerfService — startup-performance
// benchmarking + trend reads. It is decoupled from validation and is NEVER
// invoked by a test-genie phase.
package perf

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"

	internalperf "structure-health/internal/perf"

	perfv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf"
	perfconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf/perf_v1connect"
)

// Handler implements the generated PerfServiceHandler.
type Handler struct {
	perfconnect.UnimplementedPerfServiceHandler
	svc    *internalperf.Service
	logger *log.Logger
}

// NewHandler builds a perf Handler.
func NewHandler(svc *internalperf.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ perfconnect.PerfServiceHandler = (*Handler)(nil)

// BenchmarkStartup restarts the target scenario, measures its startup, persists
// the measurement, and returns it.
func (h *Handler) BenchmarkStartup(ctx context.Context, req *connect.Request[perfv1.BenchmarkStartupRequest]) (*connect.Response[perfv1.BenchmarkStartupResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	timeout := time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	m, err := h.svc.Benchmark(ctx, scenario, timeout)
	if err != nil {
		// A measurement is still returned for context (e.g. timeout/restart note),
		// but the call surfaces the error.
		h.logger.Printf("perf.BenchmarkStartup(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&perfv1.BenchmarkStartupResponse{Measurement: measurementToProto(m)}), nil
}

// GetPerfTrend returns the persisted measurements for a scenario, newest first.
func (h *Handler) GetPerfTrend(ctx context.Context, req *connect.Request[perfv1.GetPerfTrendRequest]) (*connect.Response[perfv1.GetPerfTrendResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	measurements, err := h.svc.Trend(ctx, scenario, int(req.Msg.GetLimit()))
	if err != nil {
		h.logger.Printf("perf.GetPerfTrend(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &perfv1.GetPerfTrendResponse{Scenario: scenario}
	for _, m := range measurements {
		out.Measurements = append(out.Measurements, measurementToProto(m))
	}
	return connect.NewResponse(out), nil
}

func measurementToProto(m internalperf.Measurement) *perfv1.PerfMeasurement {
	out := &perfv1.PerfMeasurement{
		Scenario:        m.Scenario,
		CapturedAt:      m.CapturedAt.UTC().Format(time.RFC3339Nano),
		TimeToHealthyMs: m.TimeToHealthyMs,
		Healthy:         m.Healthy,
		Metrics:         m.Metrics,
		Note:            m.Note,
	}
	for _, st := range m.SurfaceTimings {
		out.SurfaceTimings = append(out.SurfaceTimings, &perfv1.SurfaceTiming{
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
