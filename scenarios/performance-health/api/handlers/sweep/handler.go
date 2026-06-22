// Package sweep mounts performance-health's SweepService — the out-of-band
// per-flow capture cadence. For each interaction flow that has a per-flow budget
// declared, it drives a targeted audit, analyzes the trace, persists a
// FLOW-TAGGED trend sample, and reports the per-flow budget verdict. The capture
// is off-cadence (never inside a gated run); the per-flow budget CHECK then runs
// in the test-genie Performance phase, reading the latest persisted sample, so a
// breach fails the suite run (`vrooli scenario test`), not baseline-diff.
package sweep

import (
	"context"
	"fmt"
	"log"
	"sort"

	"connectrpc.com/connect"

	"performance-health/internal/analysis"
	"performance-health/internal/budgets"
	"performance-health/internal/capture"
	"performance-health/internal/perfsample"
	"performance-health/internal/readiness"

	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
	sweepconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep/sweep_v1connect"
)

// Auditor drives a profile-mode capture of one scenario+flow. capture.Service
// satisfies it; tests drive a fake.
type Auditor interface {
	Orchestrate(ctx context.Context, scenario, workflow string, tier readiness.Tier) (capture.Result, error)
}

// Analyzer turns a captured trace into web-vitals + per-component readings.
// analysis.Service satisfies it.
type Analyzer interface {
	Analyze(ctx context.Context, scenario, artifact string) (analysis.Result, error)
}

// SampleWriter persists a flow-tagged trend sample. trend.Store satisfies it.
type SampleWriter interface {
	Insert(ctx context.Context, sample perfsample.Sample) error
}

// FlowGate enumerates and checks per-flow budgets. budgets.Service satisfies it.
type FlowGate interface {
	Get(ctx context.Context, scenario string) (budgets.Budget, bool, error)
	CheckFlow(ctx context.Context, scenario, flow string) (bool, []budgets.Violation, error)
}

// Tierer decides the reachable capture tier for a scenario. readiness.Service
// satisfies it; nil defaults every flow to Tier 0.
type Tierer interface {
	Validate(ctx context.Context, scenario, path string) (readiness.Result, error)
}

// Handler implements the generated SweepServiceHandler.
type Handler struct {
	sweepconnect.UnimplementedSweepServiceHandler
	auditor  Auditor
	analyzer Analyzer
	trend    SampleWriter
	budgets  FlowGate
	tierer   Tierer
	logger   *log.Logger
}

// NewHandler builds a sweep Handler.
func NewHandler(auditor Auditor, analyzer Analyzer, trend SampleWriter, gate FlowGate, tierer Tierer, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{auditor: auditor, analyzer: analyzer, trend: trend, budgets: gate, tierer: tierer, logger: logger}
}

var _ sweepconnect.SweepServiceHandler = (*Handler)(nil)

// RunSweep audits every budgeted flow for a scenario, persists a flow-tagged
// sample per captured flow, and reports the per-flow budget verdict.
func (h *Handler) RunSweep(ctx context.Context, req *connect.Request[sweepv1.RunSweepRequest]) (*connect.Response[sweepv1.RunSweepResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	if h.budgets == nil || h.auditor == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errInvalid("sweep not wired"))
	}

	out := &sweepv1.RunSweepResponse{Scenario: scenario}
	budget, declared, err := h.budgets.Get(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !declared || len(budget.Flows) == 0 {
		return connect.NewResponse(out), nil // nothing budgeted → nothing to sweep
	}

	slugs := make([]string, 0, len(budget.Flows))
	for slug := range budget.Flows {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	tier := h.resolveTier(ctx, scenario)
	for _, slug := range slugs {
		out.Results = append(out.Results, h.sweepFlow(ctx, scenario, slug, tier))
	}
	return connect.NewResponse(out), nil
}

// sweepFlow runs one flow's audit → analyze → persist → check cycle.
func (h *Handler) sweepFlow(ctx context.Context, scenario, slug string, tier readiness.Tier) *sweepv1.FlowSweepResult {
	res := &sweepv1.FlowSweepResult{Flow: slug}

	cap, err := h.auditor.Orchestrate(ctx, scenario, slug, tier)
	if err != nil {
		res.Outcome = "failed"
		res.Reason = err.Error()
		return res
	}
	res.Outcome = outcomeString(cap.Outcome)
	res.Reason = cap.Reason
	if cap.Outcome != capture.OutcomeCaptured {
		return res // skipped / unavailable / failed: nothing to persist
	}

	ares, err := h.analyzer.Analyze(ctx, scenario, cap.TraceArtifact)
	if err != nil {
		res.Outcome = "failed"
		res.Reason = fmt.Sprintf("analyze trace: %v", err)
		return res
	}

	sample := perfsample.Sample{Scenario: scenario, Flow: slug, LCPMs: ares.LCPMs, Note: "sweep"}
	if slowest, ok := slowestComponent(ares.Components); ok {
		sample.SlowestComponent = slowest.Component
		sample.SlowestComponentAvgMs = slowest.AvgMs
		sample.SlowestComponentMaxMs = slowest.MaxMs
	}
	if h.trend != nil && (sample.LCPMs > 0 || sample.SlowestComponent != "") {
		if err := h.trend.Insert(ctx, sample); err != nil {
			h.logger.Printf("sweep(%s/%s): persist flow sample: %v", scenario, slug, err)
		}
	}

	passed, violations, err := h.budgets.CheckFlow(ctx, scenario, slug)
	if err != nil {
		h.logger.Printf("sweep(%s/%s): per-flow budget check degraded: %v", scenario, slug, err)
		return res
	}
	res.WithinBudget = passed
	for _, v := range violations {
		res.Violations = append(res.Violations, fmt.Sprintf("%s measured %d%s exceeds budget %d%s",
			v.Axis, v.Measured, unitSuffix(v.Unit), v.Budget, unitSuffix(v.Unit)))
	}
	return res
}

func (h *Handler) resolveTier(ctx context.Context, scenario string) readiness.Tier {
	if h.tierer == nil {
		return readiness.Tier0
	}
	r, err := h.tierer.Validate(ctx, scenario, "")
	if err != nil {
		h.logger.Printf("sweep(%s): tier detection degraded, defaulting Tier 0: %v", scenario, err)
		return readiness.Tier0
	}
	return r.Tier
}

func slowestComponent(components []analysis.ComponentTiming) (analysis.ComponentTiming, bool) {
	var (
		worst analysis.ComponentTiming
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

func outcomeString(o capture.Outcome) string {
	switch o {
	case capture.OutcomeCaptured:
		return "captured"
	case capture.OutcomeSkipped:
		return "skipped"
	case capture.OutcomeUnavailable:
		return "unavailable"
	case capture.OutcomeFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

func unitSuffix(unit string) string {
	switch unit {
	case "ms":
		return "ms"
	case "bytes":
		return "B"
	default:
		return ""
	}
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
