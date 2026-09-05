package provision

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	provisionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision/provision_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client provisionconnect.ProvisionServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: provisionconnect.NewProvisionServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) sync(ctx cliapp.RunContext) error {
	nodeID := ctx.Positional("node-id")
	resp, err := h.client.SyncToRevision(context.Background(), connect.NewRequest(&provisionv1.SyncToRevisionRequest{
		NodeId:           nodeID,
		TargetRevision:   ctx.Flag("revision"),
		RollbackRevision: ctx.Flag("rollback"),
		TimeoutSeconds:   parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("sync node to revision (set a token via `configure token` or $VROOLI_BRIDGE_API_TOKEN if unauthenticated)", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no provisioning response")
	}
	msg := resp.Msg

	if msg.DryRun {
		return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("[dry-run] node %s would be synced to %s (validated, nothing run).", msg.NodeId, msg.TargetRevision)},
			Changes:     []string{"No op created, no audit written, nothing pushed."},
			NextCommand: []string{"Re-run without --dry-run to provision for real."},
		})
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Provisioning op %s — node %s → %s.", msg.OpId, msg.NodeId, msg.TargetRevision)},
		Changes: []string{fmt.Sprintf("op %s queued (target=%s rollback=%s)", msg.OpId, msg.TargetRevision, emptyDash(msg.RollbackRevision))},
		NextCommand: []string{
			fmt.Sprintf("`provision wait %s` — block until it finishes", msg.OpId),
			fmt.Sprintf("`provision get %s` — show its event history", msg.OpId),
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetProvisioningOp(context.Background(), connect.NewRequest(&provisionv1.GetProvisioningOpRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get provisioning op %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Op == nil {
		return fmt.Errorf("server returned no op")
	}
	results := []string{formatOp(resp.Msg.Op)}
	for _, ev := range resp.Msg.Events {
		results = append(results, "  "+formatEvent(ev))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Op %s — %s (%d event(s)).", resp.Msg.Op.Id, statusLabel(resp.Msg.Op.Status), len(resp.Msg.Events))},
		ResultsHeading: "Provisioning op",
		Results:        results,
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListProvisioningOps(context.Background(), connect.NewRequest(&provisionv1.ListProvisioningOpsRequest{
		NodeId: ctx.Flag("node"),
		Limit:  int32(parseInt(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list provisioning ops", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no ops response")
	}
	results := make([]string, 0, len(resp.Msg.Ops))
	for _, op := range resp.Msg.Ops {
		results = append(results, formatOp(op))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d provisioning op(s).", len(resp.Msg.Ops))},
		ResultsHeading: "Provisioning ops",
		Results:        results,
		RetrievalHints: []string{
			"`provision get <id>` — show one op with its event history",
			"`provision wait <id>` — block until an op finishes",
		},
	})
}

// wait blocks server-side until the op is terminal (or the wait deadline
// elapses) and returns an error on a non-completed/timed-out op so the process
// exits non-zero, mirroring the runs wait verb.
func (h *handlers) wait(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.WaitProvisioningOp(context.Background(), connect.NewRequest(&provisionv1.WaitProvisioningOpRequest{
		Id:             id,
		TimeoutSeconds: parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("wait for provisioning op %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Op == nil {
		return fmt.Errorf("server returned no op")
	}
	op := resp.Msg.Op

	if resp.Msg.TimedOut {
		_ = cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Op %s still %s after the wait window.", op.Id, statusLabel(op.Status))},
			ResultsHeading: "Provisioning op",
			Results:        []string{formatOp(op)},
			RetrievalHints: []string{fmt.Sprintf("`provision wait %s` — re-attach and keep waiting (the op is durable)", op.Id)},
		})
		return fmt.Errorf("provisioning op %s did not finish within the wait window (still %s)", op.Id, statusLabel(op.Status))
	}

	_ = cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Op %s finished: %s (resulting revision %s).", op.Id, statusLabel(op.Status), emptyDash(op.ResultingRevision))},
		ResultsHeading: "Provisioning op",
		Results:        []string{formatOp(op)},
	})
	if op.Status != provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED {
		return fmt.Errorf("provisioning op %s %s (exit %d)", op.Id, statusLabel(op.Status), op.ExitCode)
	}
	return nil
}

func (h *handlers) version(ctx cliapp.RunContext) error {
	nodeID := ctx.Positional("node-id")
	resp, err := h.client.GetNodeVersion(context.Background(), connect.NewRequest(&provisionv1.GetNodeVersionRequest{NodeId: nodeID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get version for node %q", nodeID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no version response")
	}
	if !resp.Msg.HasVersion {
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Node %s has never been provisioned (no recorded version).", nodeID)},
			ResultsHeading: "Node version",
			Results:        []string{"(none)"},
			RetrievalHints: []string{fmt.Sprintf("`provision sync %s --revision <rev>` — bring it to a revision", nodeID)},
		})
	}
	v := resp.Msg.Version
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Node %s is at revision %s (op %s).", v.NodeId, v.Revision, emptyDash(v.OpId))},
		ResultsHeading: "Node version",
		Results:        []string{formatVersion(v)},
	})
}

// ---- formatting helpers ----

func formatOp(op *provisionv1.ProvisioningOp) string {
	if op == nil {
		return "(nil)"
	}
	created := ""
	if op.CreatedAt != nil {
		created = op.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — node=%s target=%s rollback=%s [status=%s resulting=%s exit=%d created=%s]",
		op.Id, op.NodeId, op.TargetRevision, emptyDash(op.RollbackRevision),
		statusLabel(op.Status), emptyDash(op.ResultingRevision), op.ExitCode, created)
}

func formatVersion(v *provisionv1.NodeVersion) string {
	if v == nil {
		return "(nil)"
	}
	reported := ""
	if v.ReportedAt != nil {
		reported = v.ReportedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("node=%s revision=%s op=%s reported=%s", v.NodeId, v.Revision, emptyDash(v.OpId), reported)
}

func formatEvent(ev *provisionv1.ProvisionEvent) string {
	if ev == nil {
		return "(nil event)"
	}
	switch ev.Kind {
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_LOG:
		return strings.TrimRight(ev.LogChunk, "\n")
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_STATUS:
		return fmt.Sprintf("[status] %s", ev.Status)
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION:
		return fmt.Sprintf("[version] %s", ev.Revision)
	case provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT:
		return fmt.Sprintf("[exit] %d", ev.ExitCode)
	default:
		return fmt.Sprintf("[event] seq=%d", ev.Sequence)
	}
}

func statusLabel(s provisionv1.ProvisioningStatus) string {
	switch s {
	case provisionv1.ProvisioningStatus_PROVISIONING_STATUS_QUEUED:
		return "queued"
	case provisionv1.ProvisioningStatus_PROVISIONING_STATUS_RUNNING:
		return "running"
	case provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED:
		return "completed"
	case provisionv1.ProvisioningStatus_PROVISIONING_STATUS_FAILED:
		return "failed"
	case provisionv1.ProvisioningStatus_PROVISIONING_STATUS_ROLLED_BACK:
		return "rolled_back"
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
