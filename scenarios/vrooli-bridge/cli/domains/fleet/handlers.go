package fleet

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet/fleet_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client fleetconnect.FleetServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: fleetconnect.NewFleetServiceClient(httpClient, baseURL),
	}
}

// roll pins the fleet (or a named subset) to a target revision and prints the
// per-node ledger.
func (h *handlers) roll(ctx cliapp.RunContext) error {
	resp, err := h.client.RollFleet(context.Background(), connect.NewRequest(&fleetv1.RollFleetRequest{
		TargetRevision: ctx.Flag("revision"),
		NodeIds:        splitCSV(ctx.Flag("nodes")),
		TimeoutSeconds: parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("roll fleet to revision (set a token via `configure token` or $VROOLI_BRIDGE_API_TOKEN if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no rollout response")
	}
	msg := resp.Msg

	results := make([]string, 0, len(msg.Results))
	for _, r := range msg.Results {
		results = append(results, formatResult(r))
	}

	if msg.DryRun {
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:      append([]string{fmt.Sprintf("[dry-run] would roll %d node(s) to %s — %s.", len(msg.Results), ctx.Flag("revision"), rolloutStatusLabel(msg.Status))}, results...),
			Changes:     []string{"No rollout created, no provisioning op dispatched."},
			NextCommand: []string{"Re-run without --dry-run to roll for real."},
		})
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  append([]string{fmt.Sprintf("Rollout %s — %s.", msg.RolloutId, rolloutStatusLabel(msg.Status))}, results...),
		Changes: []string{fmt.Sprintf("rollout %s recorded (%d node(s))", msg.RolloutId, len(msg.Results))},
		NextCommand: []string{
			fmt.Sprintf("`fleet get %s` — show the per-node ledger", msg.RolloutId),
			"`provision wait <op-id>` — block on a dispatched node's provisioning op",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetRollout(context.Background(), connect.NewRequest(&fleetv1.GetRolloutRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get rollout %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Rollout == nil {
		return fmt.Errorf("server returned no rollout")
	}
	results := []string{formatRollout(resp.Msg.Rollout)}
	for _, r := range resp.Msg.Results {
		results = append(results, "  "+formatResult(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Rollout %s — %s (%d node(s)).", resp.Msg.Rollout.Id, rolloutStatusLabel(resp.Msg.Rollout.Status), len(resp.Msg.Results))},
		ResultsHeading: "Rollout",
		Results:        results,
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRollouts(context.Background(), connect.NewRequest(&fleetv1.ListRolloutsRequest{
		Limit: int32(parseInt(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list rollouts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no rollouts response")
	}
	results := make([]string, 0, len(resp.Msg.Rollouts))
	for _, r := range resp.Msg.Rollouts {
		results = append(results, formatRollout(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d rollout(s).", len(resp.Msg.Rollouts))},
		ResultsHeading: "Rollouts",
		Results:        results,
		RetrievalHints: []string{"`fleet get <id>` — show one rollout with its per-node ledger"},
	})
}

// ---- formatting helpers ----

func formatRollout(r *fleetv1.Rollout) string {
	if r == nil {
		return "(nil)"
	}
	created := ""
	if r.CreatedAt != nil {
		created = r.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — target=%s [status=%s total=%d dispatched=%d skipped=%d failed=%d created=%s]",
		r.Id, r.TargetRevision, rolloutStatusLabel(r.Status),
		r.TotalNodes, r.Dispatched, r.Skipped, r.Failed, created)
}

func formatResult(r *fleetv1.NodeRolloutResult) string {
	if r == nil {
		return "(nil)"
	}
	op := ""
	if r.OpId != "" {
		op = " op=" + r.OpId
	}
	return fmt.Sprintf("node=%s [%s]%s %s", r.NodeId, dispositionLabel(r.Disposition), op, emptyDash(r.Detail))
}

func rolloutStatusLabel(s fleetv1.RolloutStatus) string {
	switch s {
	case fleetv1.RolloutStatus_ROLLOUT_STATUS_DISPATCHED:
		return "dispatched"
	case fleetv1.RolloutStatus_ROLLOUT_STATUS_PARTIAL:
		return "partial"
	case fleetv1.RolloutStatus_ROLLOUT_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func dispositionLabel(d fleetv1.NodeRolloutDisposition) string {
	switch d {
	case fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_DISPATCHED:
		return "dispatched"
	case fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_SKIPPED_OFFLINE:
		return "skipped:offline"
	case fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_SKIPPED_NEEDS_UPDATE:
		return "skipped:needs-update"
	case fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_SKIPPED_REVOKED:
		return "skipped:revoked"
	case fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_FAILED:
		return "failed"
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
