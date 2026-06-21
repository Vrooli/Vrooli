// Package analysis mounts performance-health's AnalysisService — deterministic
// trace analysis (per-component table + located findings) and trace comparison.
package analysis

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internalanalysis "performance-health/internal/analysis"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis"
	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis/analysis_v1connect"
)

// Handler implements the generated AnalysisServiceHandler.
type Handler struct {
	analysisconnect.UnimplementedAnalysisServiceHandler
	svc    *internalanalysis.Service
	logger *log.Logger
}

// NewHandler builds an analysis Handler.
func NewHandler(svc *internalanalysis.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
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
	return connect.NewResponse(out), nil
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
	return connect.NewResponse(out), nil
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
