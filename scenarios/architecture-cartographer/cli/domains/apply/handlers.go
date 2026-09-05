package apply

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply/apply_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client applyconnect.ApplyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: applyconnect.NewApplyServiceClient(httpClient, baseURL),
	}
}

// suppress writes a durable `// arch:allow` marker into a source file,
// sanctioning a finding as intentional. This is the safe write path —
// it inserts a comment only, never moves or rewrites code.
func (h *handlers) suppress(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	file := ctx.Positional("file")
	id := ctx.Positional("id")
	req := &applyv1.WriteSuppressionRequest{
		Scenario: scenario,
		File:     file,
		Id:       id,
		Reason:   strings.TrimSpace(ctx.Flag("reason")),
		Expires:  strings.TrimSpace(ctx.Flag("expires")),
	}
	if l := strings.TrimSpace(ctx.Flag("line")); l != "" {
		n, err := strconv.ParseInt(l, 10, 32)
		if err != nil {
			return fmt.Errorf("--line must be a 32-bit integer: %w", err)
		}
		if n <= 0 {
			return fmt.Errorf("--line must be greater than zero")
		}
		req.Line = int32(n)
	}
	resp, err := h.client.WriteSuppression(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("write suppression for %q in %q", id, scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no suppression response")
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Wrote suppression marker into %s", resp.Msg.GetFile()),
			resp.Msg.GetMarker(),
		},
		NextSteps: []string{
			fmt.Sprintf("`conflicts detect %s` to confirm the finding now reports as suppressed.", scenario),
		},
	})
}

// plan derives the deterministic operation list that would execute for a
// domain, given the current resolved-conflict state. v0.1 is read-only
// (no toolchain, no mutation); it still honors --dry-run via the
// X-Dry-Run header and echoes the dry-run state.
func (h *handlers) plan(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	domain := ctx.Positional("domain")
	resp, err := h.client.PlanApply(context.Background(), connect.NewRequest(&applyv1.PlanApplyRequest{
		Scenario:    scenario,
		Domain:      domain,
		ConflictIds: cliutil.ParseCSV(ctx.Flag("conflict-id")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("plan apply for %q/%q", scenario, domain), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetPlan() == nil {
		return fmt.Errorf("server returned no plan")
	}
	p := resp.Msg.GetPlan()
	results := make([]string, 0, len(p.GetOperations()))
	for _, op := range p.GetOperations() {
		results = append(results, fmt.Sprintf("%s %s %s -> %s",
			op.GetId(), opKindName(op.GetKind()), op.GetFromPath(), op.GetToPath()))
	}
	summary := fmt.Sprintf("Plan %s for %q/%q: %d operation(s).", p.GetId(), scenario, domain, len(p.GetOperations()))
	if resp.Msg.GetDryRun() {
		summary += " (dry-run)"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Operations",
		Results:        results,
		RetrievalHints: []string{
			"`apply run <plan-id>` executes a plan — unimplemented in v0.1 (see the apply-execution plan).",
		},
	})
}

// run executes a planned Plan. v0.1 always returns CodeUnimplemented; this
// handler translates that into a clean capability-not-available message
// that names the unblocking plan, rather than surfacing a raw error/stack.
func (h *handlers) run(ctx cliapp.RunContext) error {
	planID := ctx.Positional("plan_id")
	_, err := h.client.RunApply(context.Background(), connect.NewRequest(&applyv1.RunApplyRequest{
		PlanId:                      planID,
		AcknowledgeV01Unimplemented: ctx.BoolFlag("acknowledge"),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeUnimplemented {
			return fmt.Errorf("apply execution is not implemented in v0.1 — `apply plan` derives the operation list, but running it lands in the separate apply-execution plan")
		}
		return cliapp.WrapAPIError(fmt.Sprintf("run apply for plan %q", planID), err, nil)
	}
	// Unreachable in v0.1; symmetric for when execution lands.
	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Ran apply for plan %s.", planID)},
	})
}

// history pages the apply-run history (empty in v0.1).
func (h *handlers) history(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := &applyv1.ListApplyHistoryRequest{
		Scenario:  scenario,
		Domain:    ctx.Flag("domain"),
		PageToken: ctx.Flag("page-token"),
	}
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--page-size must be a 32-bit integer: %w", err)
		}
		if n <= 0 {
			return fmt.Errorf("--page-size must be greater than zero")
		}
		req.PageSize = int32(n)
	}
	resp, err := h.client.ListApplyHistory(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list apply history for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no history response")
	}
	results := make([]string, 0, len(resp.Msg.GetRuns()))
	for _, r := range resp.Msg.GetRuns() {
		results = append(results, fmt.Sprintf("%s plan=%s %s/%s status=%s",
			r.GetId(), r.GetPlanId(), r.GetScenario(), r.GetDomain(), applyStatusName(r.GetStatus())))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d apply run(s) for %q.", len(resp.Msg.GetRuns()), scenario)},
		ResultsHeading: "Runs",
		Results:        results,
		RetrievalHints: []string{"v0.1 records no runs because apply execution is unimplemented."},
	})
}

// baseline shows the toolchain build baseline (empty in v0.1).
func (h *handlers) baseline(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetBuildBaseline(context.Background(), connect.NewRequest(&applyv1.GetBuildBaselineRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get build baseline for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no baseline response")
	}
	b := resp.Msg.GetBaseline()
	if b == nil || b.GetToolchain() == "" {
		return ctx.RenderList(cliapp.ListReport{
			Summary: []string{fmt.Sprintf("No build baseline recorded for %q (BuildGuard is unwired in v0.1).", scenario)},
		})
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Build baseline for %q.", scenario)},
		ResultsHeading: "Baseline",
		Results: []string{
			fmt.Sprintf("toolchain: %s", b.GetToolchain()),
			fmt.Sprintf("green: %t", b.GetGreen()),
		},
	})
}

// -------------------------- enum mapping --------------------------

func opKindName(k applyv1.OperationKind) string {
	switch k {
	case applyv1.OperationKind_OPERATION_KIND_MOVE_FILE:
		return "move_file"
	case applyv1.OperationKind_OPERATION_KIND_REWRITE_IMPORT:
		return "rewrite_import"
	case applyv1.OperationKind_OPERATION_KIND_DELETE_FILE:
		return "delete_file"
	case applyv1.OperationKind_OPERATION_KIND_CREATE_FILE:
		return "create_file"
	default:
		return "unspecified"
	}
}

func applyStatusName(s applyv1.ApplyStatus) string {
	switch s {
	case applyv1.ApplyStatus_APPLY_STATUS_PLANNED:
		return "planned"
	case applyv1.ApplyStatus_APPLY_STATUS_RUNNING:
		return "running"
	case applyv1.ApplyStatus_APPLY_STATUS_BUILD_GREEN:
		return "build_green"
	case applyv1.ApplyStatus_APPLY_STATUS_BUILD_RED:
		return "build_red"
	case applyv1.ApplyStatus_APPLY_STATUS_REVERTED:
		return "reverted"
	case applyv1.ApplyStatus_APPLY_STATUS_COMMITTED:
		return "committed"
	default:
		return "unspecified"
	}
}
