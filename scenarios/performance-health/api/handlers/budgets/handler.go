// Package budgets mounts performance-health's BudgetService — per-scenario
// performance budgets and the suite-run budget check gate (a breach fails the
// test-genie Performance phase, not baseline-diff).
package budgets

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"

	internalbudgets "performance-health/internal/budgets"

	budgetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets"
	budgetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets/budgets_v1connect"
)

// Handler implements the generated BudgetServiceHandler.
type Handler struct {
	budgetsconnect.UnimplementedBudgetServiceHandler
	svc    *internalbudgets.Service
	logger *log.Logger
}

// NewHandler builds a budgets Handler.
func NewHandler(svc *internalbudgets.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ budgetsconnect.BudgetServiceHandler = (*Handler)(nil)

// GetBudget returns the declared budget for a scenario.
func (h *Handler) GetBudget(ctx context.Context, req *connect.Request[budgetsv1.GetBudgetRequest]) (*connect.Response[budgetsv1.GetBudgetResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	b, declared, err := h.svc.Get(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&budgetsv1.GetBudgetResponse{Budget: budgetToProto(b), Declared: declared}), nil
}

// SetBudget writes/updates a scenario's budget (dry-run honors X-Dry-Run). When
// req.flow is set, the incoming per-flow axes are written to budget.flows[flow]
// preserving scenario-level axes and sibling flows; otherwise the scenario-level
// axes are written, preserving existing flows. Both are a read-modify-write
// against the persisted budget.
func (h *Handler) SetBudget(ctx context.Context, req *connect.Request[budgetsv1.SetBudgetRequest]) (*connect.Response[budgetsv1.SetBudgetResponse], error) {
	in := budgetFromProto(req.Msg.GetBudget())
	if in.Scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	dryRun := req.Header().Get("X-Dry-Run") == "true"

	existing, _, err := h.svc.Get(ctx, in.Scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	merged := mergeBudgetWrite(existing, in, strings.TrimSpace(req.Msg.GetFlow()))

	b, err := h.svc.Set(ctx, merged, dryRun)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&budgetsv1.SetBudgetResponse{Budget: budgetToProto(b), DryRun: dryRun}), nil
}

// mergeBudgetWrite composes the complete desired budget from the persisted one
// and an incoming partial (see SetBudget).
func mergeBudgetWrite(existing, in internalbudgets.Budget, flow string) internalbudgets.Budget {
	if flow != "" {
		merged := existing
		merged.Scenario = in.Scenario
		cp := make(map[string]internalbudgets.FlowBudget, len(existing.Flows)+1)
		for k, v := range existing.Flows {
			cp[k] = v
		}
		cp[flow] = internalbudgets.FlowBudget{
			LCPMaxMs:                in.LCPMaxMs,
			CLSMax:                  in.CLSMax,
			ComponentCommitAvgMaxMs: in.ComponentCommitAvgMaxMs,
			ComponentCommitMaxMs:    in.ComponentCommitMaxMs,
			DrawnFPSMin:             in.DrawnFPSMin,
			DroppedFrameRateMax:     in.DroppedFrameRateMax,
			LongTaskTotalMaxMs:      in.LongTaskTotalMaxMs,
			LongTaskMaxMs:           in.LongTaskMaxMs,
			RasterTotalMaxMs:        in.RasterTotalMaxMs,
			LayoutTotalMaxMs:        in.LayoutTotalMaxMs,
			PaintTotalMaxMs:         in.PaintTotalMaxMs,
			InputEventCountMin:      in.InputEventCountMin,
			LoadOnly:                in.LoadOnly,
		}
		merged.Flows = cp
		merged.Ratchet = existing.Ratchet || in.Ratchet
		return merged
	}
	// Scenario-level write: incoming scalar axes are authoritative; preserve
	// existing per-flow budgets.
	in.Flows = existing.Flows
	return in
}

// CheckBudget evaluates a scenario's latest measurements against its budget, or
// (when req.flow is set) one interaction flow's per-flow budget against its
// latest flow-tagged sample.
func (h *Handler) CheckBudget(ctx context.Context, req *connect.Request[budgetsv1.CheckBudgetRequest]) (*connect.Response[budgetsv1.CheckBudgetResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	flow := strings.TrimSpace(req.Msg.GetFlow())

	var (
		passed     bool
		violations []internalbudgets.Violation
		err        error
	)
	if flow != "" {
		passed, violations, err = h.svc.CheckFlow(ctx, scenario, flow)
	} else {
		passed, violations, err = h.svc.Check(ctx, scenario)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &budgetsv1.CheckBudgetResponse{Scenario: scenario, Passed: passed}
	for _, v := range violations {
		out.Violations = append(out.Violations, &budgetsv1.BudgetViolation{
			Axis: v.Axis, Measured: v.Measured, Budget: v.Budget,
			MeasuredValue: v.MeasuredValue, BudgetValue: v.BudgetValue,
			Unit: v.Unit, Flow: flow, Mode: v.Mode, Detail: v.Detail,
		})
	}
	return connect.NewResponse(out), nil
}

func budgetToProto(b internalbudgets.Budget) *budgetsv1.Budget {
	out := &budgetsv1.Budget{
		Scenario:                b.Scenario,
		GoBuildMaxMs:            b.GoBuildMaxMs,
		UiBuildMaxMs:            b.UIBuildMaxMs,
		BundleMaxBytes:          b.BundleMaxBytes,
		LcpMaxMs:                b.LCPMaxMs,
		ClsMax:                  b.CLSMax,
		ResponseEndMaxMs:        b.ResponseEndMaxMs,
		DomInteractiveMaxMs:     b.DOMInteractiveMaxMs,
		DomContentLoadedMaxMs:   b.DOMContentLoadedMaxMs,
		LoadEventEndMaxMs:       b.LoadEventEndMaxMs,
		StartupMaxMs:            b.StartupMaxMs,
		ComponentCommitAvgMaxMs: b.ComponentCommitAvgMaxMs,
		ComponentCommitMaxMs:    b.ComponentCommitMaxMs,
		DrawnFpsMin:             b.DrawnFPSMin,
		DroppedFrameRateMax:     b.DroppedFrameRateMax,
		LongTaskTotalMaxMs:      b.LongTaskTotalMaxMs,
		LongTaskMaxMs:           b.LongTaskMaxMs,
		RasterTotalMaxMs:        b.RasterTotalMaxMs,
		LayoutTotalMaxMs:        b.LayoutTotalMaxMs,
		PaintTotalMaxMs:         b.PaintTotalMaxMs,
		InputEventCountMin:      b.InputEventCountMin,
		LoadOnly:                b.LoadOnly,
		Ratchet:                 b.Ratchet,
	}
	if len(b.Flows) > 0 {
		out.Flows = make(map[string]*budgetsv1.FlowBudget, len(b.Flows))
		for slug, fb := range b.Flows {
			out.Flows[slug] = &budgetsv1.FlowBudget{
				LcpMaxMs:                fb.LCPMaxMs,
				ClsMax:                  fb.CLSMax,
				ComponentCommitAvgMaxMs: fb.ComponentCommitAvgMaxMs,
				ComponentCommitMaxMs:    fb.ComponentCommitMaxMs,
				DrawnFpsMin:             fb.DrawnFPSMin,
				DroppedFrameRateMax:     fb.DroppedFrameRateMax,
				LongTaskTotalMaxMs:      fb.LongTaskTotalMaxMs,
				LongTaskMaxMs:           fb.LongTaskMaxMs,
				RasterTotalMaxMs:        fb.RasterTotalMaxMs,
				LayoutTotalMaxMs:        fb.LayoutTotalMaxMs,
				PaintTotalMaxMs:         fb.PaintTotalMaxMs,
				InputEventCountMin:      fb.InputEventCountMin,
				LoadOnly:                fb.LoadOnly,
			}
		}
	}
	return out
}

func budgetFromProto(b *budgetsv1.Budget) internalbudgets.Budget {
	if b == nil {
		return internalbudgets.Budget{}
	}
	out := internalbudgets.Budget{
		Scenario:                b.GetScenario(),
		GoBuildMaxMs:            b.GetGoBuildMaxMs(),
		UIBuildMaxMs:            b.GetUiBuildMaxMs(),
		BundleMaxBytes:          b.GetBundleMaxBytes(),
		LCPMaxMs:                b.GetLcpMaxMs(),
		CLSMax:                  b.GetClsMax(),
		ResponseEndMaxMs:        b.GetResponseEndMaxMs(),
		DOMInteractiveMaxMs:     b.GetDomInteractiveMaxMs(),
		DOMContentLoadedMaxMs:   b.GetDomContentLoadedMaxMs(),
		LoadEventEndMaxMs:       b.GetLoadEventEndMaxMs(),
		StartupMaxMs:            b.GetStartupMaxMs(),
		ComponentCommitAvgMaxMs: b.GetComponentCommitAvgMaxMs(),
		ComponentCommitMaxMs:    b.GetComponentCommitMaxMs(),
		DrawnFPSMin:             b.GetDrawnFpsMin(),
		DroppedFrameRateMax:     b.GetDroppedFrameRateMax(),
		LongTaskTotalMaxMs:      b.GetLongTaskTotalMaxMs(),
		LongTaskMaxMs:           b.GetLongTaskMaxMs(),
		RasterTotalMaxMs:        b.GetRasterTotalMaxMs(),
		LayoutTotalMaxMs:        b.GetLayoutTotalMaxMs(),
		PaintTotalMaxMs:         b.GetPaintTotalMaxMs(),
		InputEventCountMin:      b.GetInputEventCountMin(),
		LoadOnly:                b.GetLoadOnly(),
		Ratchet:                 b.GetRatchet(),
	}
	if len(b.GetFlows()) > 0 {
		out.Flows = make(map[string]internalbudgets.FlowBudget, len(b.GetFlows()))
		for slug, fb := range b.GetFlows() {
			out.Flows[slug] = internalbudgets.FlowBudget{
				LCPMaxMs:                fb.GetLcpMaxMs(),
				CLSMax:                  fb.GetClsMax(),
				ComponentCommitAvgMaxMs: fb.GetComponentCommitAvgMaxMs(),
				ComponentCommitMaxMs:    fb.GetComponentCommitMaxMs(),
				DrawnFPSMin:             fb.GetDrawnFpsMin(),
				DroppedFrameRateMax:     fb.GetDroppedFrameRateMax(),
				LongTaskTotalMaxMs:      fb.GetLongTaskTotalMaxMs(),
				LongTaskMaxMs:           fb.GetLongTaskMaxMs(),
				RasterTotalMaxMs:        fb.GetRasterTotalMaxMs(),
				LayoutTotalMaxMs:        fb.GetLayoutTotalMaxMs(),
				PaintTotalMaxMs:         fb.GetPaintTotalMaxMs(),
				InputEventCountMin:      fb.GetInputEventCountMin(),
				LoadOnly:                fb.GetLoadOnly(),
			}
		}
	}
	return out
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
