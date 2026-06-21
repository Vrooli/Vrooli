// Package budgets mounts performance-health's BudgetService — per-scenario
// performance budgets and the baseline-diff check gate.
package budgets

import (
	"context"
	"log"

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

// SetBudget writes/updates a scenario's budget (dry-run honors X-Dry-Run).
func (h *Handler) SetBudget(ctx context.Context, req *connect.Request[budgetsv1.SetBudgetRequest]) (*connect.Response[budgetsv1.SetBudgetResponse], error) {
	in := budgetFromProto(req.Msg.GetBudget())
	if in.Scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	dryRun := req.Header().Get("X-Dry-Run") == "true"
	b, err := h.svc.Set(ctx, in, dryRun)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&budgetsv1.SetBudgetResponse{Budget: budgetToProto(b), DryRun: dryRun}), nil
}

// CheckBudget evaluates a scenario's latest measurements against its budget.
func (h *Handler) CheckBudget(ctx context.Context, req *connect.Request[budgetsv1.CheckBudgetRequest]) (*connect.Response[budgetsv1.CheckBudgetResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	passed, violations, err := h.svc.Check(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &budgetsv1.CheckBudgetResponse{Scenario: scenario, Passed: passed}
	for _, v := range violations {
		out.Violations = append(out.Violations, &budgetsv1.BudgetViolation{
			Axis: v.Axis, Measured: v.Measured, Budget: v.Budget, Unit: v.Unit,
		})
	}
	return connect.NewResponse(out), nil
}

func budgetToProto(b internalbudgets.Budget) *budgetsv1.Budget {
	return &budgetsv1.Budget{
		Scenario:                b.Scenario,
		GoBuildMaxMs:            b.GoBuildMaxMs,
		UiBuildMaxMs:            b.UIBuildMaxMs,
		BundleMaxBytes:          b.BundleMaxBytes,
		LcpMaxMs:                b.LCPMaxMs,
		StartupMaxMs:            b.StartupMaxMs,
		ComponentCommitAvgMaxMs: b.ComponentCommitAvgMaxMs,
		ComponentCommitMaxMs:    b.ComponentCommitMaxMs,
		Ratchet:                 b.Ratchet,
	}
}

func budgetFromProto(b *budgetsv1.Budget) internalbudgets.Budget {
	if b == nil {
		return internalbudgets.Budget{}
	}
	return internalbudgets.Budget{
		Scenario:                b.GetScenario(),
		GoBuildMaxMs:            b.GetGoBuildMaxMs(),
		UIBuildMaxMs:            b.GetUiBuildMaxMs(),
		BundleMaxBytes:          b.GetBundleMaxBytes(),
		LCPMaxMs:                b.GetLcpMaxMs(),
		StartupMaxMs:            b.GetStartupMaxMs(),
		ComponentCommitAvgMaxMs: b.GetComponentCommitAvgMaxMs(),
		ComponentCommitMaxMs:    b.GetComponentCommitMaxMs(),
		Ratchet:                 b.GetRatchet(),
	}
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
