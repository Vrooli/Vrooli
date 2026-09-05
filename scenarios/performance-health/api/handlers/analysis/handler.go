// Package analysis mounts performance-health's AnalysisService — deterministic
// trace analysis (per-component table + located findings) and trace comparison.
package analysis

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internalanalysis "performance-health/internal/analysis"
	"performance-health/internal/perfsample"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis"
	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis/analysis_v1connect"
)

// SampleWriter persists a captured trace's web-vitals + slowest-component
// readings into the shared perf_samples trend (lcp_ms + component-commit axes)
// so the LCP and component-commit budgets have a producer. The trend store
// satisfies it; nil disables persistence (analysis still returns its findings).
type SampleWriter interface {
	Insert(ctx context.Context, sample perfsample.Sample) error
}

// Handler implements the generated AnalysisServiceHandler.
type Handler struct {
	analysisconnect.UnimplementedAnalysisServiceHandler
	svc    *internalanalysis.Service
	trend  SampleWriter
	logger *log.Logger
}

// NewHandler builds an analysis Handler. A nil trend writer disables trend
// persistence (analysis still parses and returns findings).
func NewHandler(svc *internalanalysis.Service, trendWriter SampleWriter, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, trend: trendWriter, logger: logger}
}

var _ analysisconnect.AnalysisServiceHandler = (*Handler)(nil)

// AnalyzeTrace turns one captured trace into a located, quantified findings set.
func (h *Handler) AnalyzeTrace(ctx context.Context, req *connect.Request[analysisv1.AnalyzeTraceRequest]) (*connect.Response[analysisv1.AnalyzeTraceResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	res, err := h.svc.Analyze(ctx, scenario, req.Msg.GetTraceArtifact())
	if err != nil {
		h.logger.Printf("analysis.AnalyzeTrace(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &analysisv1.AnalyzeTraceResponse{
		Scenario:   res.Scenario,
		LongTaskMs: res.LongTaskMs,
		LcpMs:      res.LCPMs,
		FcpMs:      res.FCPMs,
		Cls:        res.CLS,

		ResponseEndMs:      res.ResponseEndMs,
		DomInteractiveMs:   res.DOMInteractiveMs,
		DomContentLoadedMs: res.DOMContentLoadedMs,
		LoadEventEndMs:     res.LoadEventEndMs,
		NavigationType:     res.NavigationType,
		FrameSummary:       mapFrameSummary(res.FrameSummary),
	}
	for _, c := range res.Components {
		out.Components = append(out.Components, &analysisv1.ComponentTiming{
			Component:   c.Component,
			CommitCount: int32(c.CommitCount),
			AvgMs:       c.AvgMs,
			MaxMs:       c.MaxMs,
			Definition:  c.Definition,
		})
	}
	for _, f := range res.Findings {
		out.Findings = append(out.Findings, &analysisv1.PerfFinding{
			Code:       f.Code,
			Component:  f.Component,
			Definition: f.Definition,
			Message:    f.Message,
			Evidence:   f.Evidence,
			Severity:   f.Severity,
		})
	}
	out.BrowserWork = mapEventSummaries(res.BrowserWork)
	out.InputEvents = mapEventSummaries(res.InputEvents)
	// Persist the capture's web-vitals + slowest-component readings into the
	// shared perf_samples trend so the LCP and component-commit budget axes have
	// a producer (capture-fed: only when an audit/analysis has run). Best-effort:
	// a persistence failure must not fail the analysis itself.
	if h.trend != nil {
		sample := perfsample.Sample{
			Scenario: res.Scenario,
			LCPMs:    res.LCPMs,
			CLS:      res.CLS,
			Note:     "analysis",
		}
		fillNavigationSample(&sample, res)
		fillInteractionSample(&sample, res)
		if slowest, ok := slowestComponent(res.Components); ok {
			sample.SlowestComponent = slowest.Component
			sample.SlowestComponentAvgMs = slowest.AvgMs
			sample.SlowestComponentMaxMs = slowest.MaxMs
		}
		if sample.LCPMs > 0 || sample.CLS > 0 || sample.LoadEventEndMs > 0 || sample.SlowestComponent != "" || sample.HasInteractionMetrics() {
			if err := h.trend.Insert(ctx, sample); err != nil {
				h.logger.Printf("analysis.AnalyzeTrace(%s): persist trend sample: %v", scenario, err)
			}
		}
	}
	return connect.NewResponse(out), nil
}

// fillNavigationSample copies the PerformanceNavigationTiming phases onto the
// persisted sample so the navigation budget axes have a producer.
func fillNavigationSample(sample *perfsample.Sample, res internalanalysis.Result) {
	sample.ResponseEndMs = res.ResponseEndMs
	sample.DOMInteractiveMs = res.DOMInteractiveMs
	sample.DOMContentLoadedMs = res.DOMContentLoadedMs
	sample.LoadEventEndMs = res.LoadEventEndMs
	sample.NavigationType = res.NavigationType
}

func fillInteractionSample(sample *perfsample.Sample, res internalanalysis.Result) {
	sample.DrawnFPS = res.FrameSummary.ApproxDrawnFPS
	sample.DroppedFrameRate = res.FrameSummary.DroppedFrameRate
	sample.LongTaskTotalMs = res.LongTaskMs
	sample.LongTaskMaxMs = res.LongTaskMaxMs
	sample.InputEventCount = int64(totalEventCount(res.InputEvents))
	sample.RasterTotalMs = eventTotal(res.BrowserWork, "RasterTask")
	sample.LayoutTotalMs = eventTotal(res.BrowserWork, "Layout") + eventTotal(res.BrowserWork, "UpdateLayoutTree")
	sample.PaintTotalMs = eventTotal(res.BrowserWork, "Paint")
}

func totalEventCount(events []internalanalysis.EventSummary) int {
	total := 0
	for _, e := range events {
		total += e.Count
	}
	return total
}

func eventTotal(events []internalanalysis.EventSummary, name string) float64 {
	var total float64
	for _, e := range events {
		if e.Name == name {
			total += e.TotalMs
		}
	}
	return total
}

// slowestComponent returns the component with the highest average commit time
// (the budget axes cap the worst offender). Returns ok=false for an empty set.
func slowestComponent(components []internalanalysis.ComponentTiming) (internalanalysis.ComponentTiming, bool) {
	var (
		worst internalanalysis.ComponentTiming
		found bool
	)
	for _, c := range components {
		if !found || c.AvgMs > worst.AvgMs {
			worst = c
			found = true
		}
	}
	return worst, found
}

// CompareTraces diffs two traces of the same interaction.
func (h *Handler) CompareTraces(ctx context.Context, req *connect.Request[analysisv1.CompareTracesRequest]) (*connect.Response[analysisv1.CompareTracesResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	cmp, err := h.svc.Compare(ctx, scenario, req.Msg.GetBaselineArtifact(), req.Msg.GetCandidateArtifact())
	if err != nil {
		h.logger.Printf("analysis.CompareTraces(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &analysisv1.CompareTracesResponse{
		Scenario:        cmp.Scenario,
		LongTaskDeltaMs: cmp.LongTaskDeltaMs,
		LcpDeltaMs:      cmp.LCPDeltaMs,
		FrameDelta:      mapFrameDelta(cmp.FrameDelta),
	}
	for _, d := range cmp.Components {
		out.Components = append(out.Components, &analysisv1.ComponentDelta{
			Component:      d.Component,
			BaselineAvgMs:  d.BaselineAvgMs,
			CandidateAvgMs: d.CandidateAvgMs,
			DeltaMs:        d.DeltaMs,
			BaselineCount:  int32(d.BaselineCount),
			CandidateCount: int32(d.CandidateCount),
			CountDelta:     int32(d.CountDelta),
			BaselineMaxMs:  d.BaselineMaxMs,
			CandidateMaxMs: d.CandidateMaxMs,
			MaxDeltaMs:     d.MaxDeltaMs,
		})
	}
	out.BrowserWork = mapEventDeltas(cmp.BrowserWork)
	out.InputEvents = mapEventDeltas(cmp.InputEvents)
	return connect.NewResponse(out), nil
}

func mapFrameSummary(f internalanalysis.FrameSummary) *analysisv1.FrameSummary {
	return &analysisv1.FrameSummary{
		TraceDurationMs:   f.TraceDurationMs,
		BeginFrameCount:   int32(f.BeginFrameCount),
		DrawnFrameCount:   int32(f.DrawnFrameCount),
		DroppedFrameCount: int32(f.DroppedFrameCount),
		ApproxDrawnFps:    f.ApproxDrawnFPS,
		DroppedFrameRate:  f.DroppedFrameRate,
	}
}

func mapEventSummaries(in []internalanalysis.EventSummary) []*analysisv1.EventSummary {
	out := make([]*analysisv1.EventSummary, 0, len(in))
	for _, e := range in {
		out = append(out, &analysisv1.EventSummary{
			Name:    e.Name,
			Count:   int32(e.Count),
			TotalMs: e.TotalMs,
			MaxMs:   e.MaxMs,
			AvgMs:   e.AvgMs,
		})
	}
	return out
}

func mapFrameDelta(f internalanalysis.FrameDelta) *analysisv1.FrameDelta {
	return &analysisv1.FrameDelta{
		TraceDurationDeltaMs:   f.TraceDurationDeltaMs,
		BeginFrameCountDelta:   int32(f.BeginFrameCountDelta),
		DrawnFrameCountDelta:   int32(f.DrawnFrameCountDelta),
		DroppedFrameCountDelta: int32(f.DroppedFrameCountDelta),
		ApproxDrawnFpsDelta:    f.ApproxDrawnFPSDelta,
		DroppedFrameRateDelta:  f.DroppedFrameRateDelta,
	}
}

func mapEventDeltas(in []internalanalysis.EventDelta) []*analysisv1.EventDelta {
	out := make([]*analysisv1.EventDelta, 0, len(in))
	for _, d := range in {
		out = append(out, &analysisv1.EventDelta{
			Name:             d.Name,
			BaselineCount:    int32(d.BaselineCount),
			CandidateCount:   int32(d.CandidateCount),
			CountDelta:       int32(d.CountDelta),
			BaselineTotalMs:  d.BaselineTotalMs,
			CandidateTotalMs: d.CandidateTotalMs,
			TotalDeltaMs:     d.TotalDeltaMs,
			BaselineMaxMs:    d.BaselineMaxMs,
			CandidateMaxMs:   d.CandidateMaxMs,
			MaxDeltaMs:       d.MaxDeltaMs,
			BaselineAvgMs:    d.BaselineAvgMs,
			CandidateAvgMs:   d.CandidateAvgMs,
			AvgDeltaMs:       d.AvgDeltaMs,
		})
	}
	return out
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
