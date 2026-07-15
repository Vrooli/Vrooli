package gate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"
	gateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate/gate_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client gateconnect.GateServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: gateconnect.NewGateServiceClient(httpClient, baseURL),
	}
}

// run fans a scenario's validation out across the target OSes and prints the
// per-OS ledger + aggregate verdict.
func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunGate(context.Background(), connect.NewRequest(&gatev1.RunGateRequest{
		Scenario:       ctx.Flag("scenario"),
		TargetRevision: ctx.Flag("revision"),
		TargetOses:     splitCSV(ctx.Flag("oses")),
		Verb:           ctx.Flag("verb"),
		Args:           splitCSV(ctx.Flag("args")),
		TimeoutSeconds: parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("run cross-OS gate (set a token via `configure token` or $VROOLI_BRIDGE_API_TOKEN if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no gate response")
	}
	msg := resp.Msg

	results := make([]string, 0, len(msg.Results))
	for _, r := range msg.Results {
		results = append(results, formatResult(r))
	}

	if msg.DryRun {
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:      append([]string{fmt.Sprintf("[dry-run] would validate %s on %d OS(es) — %s.", ctx.Flag("scenario"), len(msg.Results), verdictLabel(msg.Verdict))}, results...),
			Changes:     []string{"No gate created, no validation run dispatched."},
			NextCommand: []string{"Re-run without --dry-run to gate for real."},
		})
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  append([]string{fmt.Sprintf("Gate %s — %s.", msg.GateId, verdictLabel(msg.Verdict))}, results...),
		Changes: []string{fmt.Sprintf("gate %s recorded (%d OS target(s))", msg.GateId, len(msg.Results))},
		NextCommand: []string{
			fmt.Sprintf("`gate wait %s` — block once for the final cross-OS verdict", msg.GateId),
			fmt.Sprintf("`gate get %s` — show the per-OS ledger", msg.GateId),
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetGate(context.Background(), connect.NewRequest(&gatev1.GetGateRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get gate %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Gate == nil {
		return fmt.Errorf("server returned no gate")
	}
	results := []string{formatGate(resp.Msg.Gate)}
	for _, r := range resp.Msg.Results {
		results = append(results, "  "+formatResult(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Gate %s — %s (%d OS target(s)).", resp.Msg.Gate.Id, verdictLabel(resp.Msg.Gate.Verdict), len(resp.Msg.Results))},
		ResultsHeading: "Gate",
		Results:        results,
	})
}

// wait blocks once until the gate is terminal, then exits non-zero unless every
// target OS passed — so a deployment-manager / CI caller can gate on the code.
func (h *handlers) wait(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.WaitGate(context.Background(), connect.NewRequest(&gatev1.WaitGateRequest{
		Id:             id,
		TimeoutSeconds: parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("wait for gate %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Gate == nil {
		return fmt.Errorf("server returned no gate")
	}
	g := resp.Msg.Gate
	results := []string{formatGate(g)}
	for _, r := range resp.Msg.Results {
		results = append(results, "  "+formatResult(r))
	}

	if resp.Msg.TimedOut {
		_ = cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Gate %s still %s after the wait window.", g.Id, verdictLabel(g.Verdict))},
			ResultsHeading: "Gate",
			Results:        results,
			RetrievalHints: []string{fmt.Sprintf("`gate wait %s` — re-attach and keep waiting (the gate is durable)", g.Id)},
		})
		return fmt.Errorf("gate %s did not finish within the wait window (still %s)", g.Id, verdictLabel(g.Verdict))
	}

	_ = cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Gate %s finished: %s (%d passed, %d failed).", g.Id, verdictLabel(g.Verdict), g.Passed, g.Failed)},
		ResultsHeading: "Gate",
		Results:        results,
	})
	if g.Verdict != gatev1.GateVerdict_GATE_VERDICT_PASSED {
		return fmt.Errorf("gate %s %s (%d of %d OS target(s) not green)", g.Id, verdictLabel(g.Verdict), g.Failed, g.TotalTargets)
	}
	return nil
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListGates(context.Background(), connect.NewRequest(&gatev1.ListGatesRequest{
		Limit: int32(parseInt(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list gates", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no gates response")
	}
	results := make([]string, 0, len(resp.Msg.Gates))
	for _, g := range resp.Msg.Gates {
		results = append(results, formatGate(g))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d gate(s).", len(resp.Msg.Gates))},
		ResultsHeading: "Gates",
		Results:        results,
		RetrievalHints: []string{"`gate get <id>` — show one gate with its per-OS ledger"},
	})
}

// ---- formatting helpers ----

func formatGate(g *gatev1.Gate) string {
	if g == nil {
		return "(nil)"
	}
	created := ""
	if g.CreatedAt != nil {
		created = g.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — scenario=%s target=%s [verdict=%s targets=%d passed=%d failed=%d pending=%d created=%s]",
		g.Id, g.Scenario, g.TargetRevision, verdictLabel(g.Verdict),
		g.TotalTargets, g.Passed, g.Failed, g.Pending, created)
}

func formatResult(r *gatev1.OSResult) string {
	if r == nil {
		return "(nil)"
	}
	run := ""
	if r.RunId != "" {
		run = " run=" + r.RunId
	}
	exit := ""
	if r.Disposition == gatev1.OSDisposition_OS_DISPOSITION_FAILED {
		exit = fmt.Sprintf(" exit=%d", r.ExitCode)
	}
	return fmt.Sprintf("os=%s [%s]%s%s %s", r.Os, dispositionLabel(r.Disposition), run, exit, emptyDash(r.Detail))
}

func verdictLabel(v gatev1.GateVerdict) string {
	switch v {
	case gatev1.GateVerdict_GATE_VERDICT_PENDING:
		return "pending"
	case gatev1.GateVerdict_GATE_VERDICT_PASSED:
		return "passed"
	case gatev1.GateVerdict_GATE_VERDICT_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func dispositionLabel(d gatev1.OSDisposition) string {
	switch d {
	case gatev1.OSDisposition_OS_DISPOSITION_PENDING:
		return "pending"
	case gatev1.OSDisposition_OS_DISPOSITION_PASSED:
		return "passed"
	case gatev1.OSDisposition_OS_DISPOSITION_FAILED:
		return "failed"
	case gatev1.OSDisposition_OS_DISPOSITION_ABORTED:
		return "aborted"
	case gatev1.OSDisposition_OS_DISPOSITION_NO_NODE:
		return "no-node"
	case gatev1.OSDisposition_OS_DISPOSITION_DISPATCH_FAILED:
		return "dispatch-failed"
	default:
		return "unspecified"
	}
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseInt(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}

func parseInt64(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
