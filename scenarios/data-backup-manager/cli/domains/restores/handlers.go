package restores

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"
	restoresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores/restores_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client restoresconnect.RestoresServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: restoresconnect.NewRestoresServiceClient(httpClient, baseURL),
	}
}

// restorePollInterval is how often the CLI polls a restore/verify for its
// terminal state. The request RPCs are now async (they return immediately), so
// the CLI owns the wait — each poll is a fast GetRestore, no long-lived request.
const restorePollInterval = 2 * time.Second

// restorePollDeadline caps how long the CLI waits for a restore/verify to reach
// a terminal state. A real restore can take many minutes on large targets/slow
// drives; this generous ceiling only guards against a wedged backend (the
// worker always reaches terminal, and startup reconciliation closes orphans).
const restorePollDeadline = 6 * time.Hour

func (h *handlers) restore(ctx cliapp.RunContext) error {
	resp, err := h.client.RestoreTarget(context.Background(), connect.NewRequest(&restoresv1.RestoreTargetRequest{
		TargetId:      ctx.Flag("target"),
		DestinationId: ctx.Flag("destination"),
		SnapshotId:    ctx.Flag("snapshot"),
		Location:      ctx.Flag("location"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("restore target", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Restore == nil {
		return fmt.Errorf("server returned no restore")
	}
	final, err := h.pollToTerminal(resp.Msg.Restore.Id)
	if err != nil {
		return cliapp.WrapAPIError("restore target", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &restoresv1.RestoreTargetResponse{Restore: final}, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Restore %s: %s.", restoreStatusLabel(final.Status), final.Id)},
		Changes: []string{formatRestore(final)},
		NextCommand: []string{
			fmt.Sprintf("`restores get %s` — show restore status", final.Id),
		},
	})
}

func (h *handlers) verify(ctx cliapp.RunContext) error {
	resp, err := h.client.VerifyTarget(context.Background(), connect.NewRequest(&restoresv1.VerifyTargetRequest{
		TargetId:      ctx.Flag("target"),
		DestinationId: ctx.Flag("destination"),
		SnapshotId:    ctx.Flag("snapshot"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("verify target", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Restore == nil {
		return fmt.Errorf("server returned no restore")
	}
	final, err := h.pollToTerminal(resp.Msg.Restore.Id)
	if err != nil {
		return cliapp.WrapAPIError("verify target", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &restoresv1.VerifyTargetResponse{Restore: final}, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Verify %s: %s.", restoreStatusLabel(final.Status), final.Id)},
		Changes: []string{formatRestore(final)},
		NextCommand: []string{
			fmt.Sprintf("`restores get %s` — show verify status", final.Id),
		},
	})
}

// pollToTerminal polls GetRestore until the record reaches a terminal state.
// Restore/verify now run in the background, so the CLI waits here rather than
// holding a long request open — each poll is a quick GetRestore.
func (h *handlers) pollToTerminal(id string) (*restoresv1.Restore, error) {
	deadline := time.Now().Add(restorePollDeadline)
	for {
		resp, err := h.client.GetRestore(context.Background(), connect.NewRequest(&restoresv1.GetRestoreRequest{Id: id}))
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Msg == nil || resp.Msg.Restore == nil {
			return nil, fmt.Errorf("server returned no restore for %q", id)
		}
		if isTerminalRestore(resp.Msg.Restore.Status) {
			return resp.Msg.Restore, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("restore %q did not reach a terminal state within %s (still %s)", id, restorePollDeadline, restoreStatusLabel(resp.Msg.Restore.Status))
		}
		time.Sleep(restorePollInterval)
	}
}

func isTerminalRestore(s restoresv1.RestoreStatus) bool {
	switch s {
	case restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED,
		restoresv1.RestoreStatus_RESTORE_STATUS_RESTORED,
		restoresv1.RestoreStatus_RESTORE_STATUS_FAILED:
		return true
	default:
		return false
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetRestore(context.Background(), connect.NewRequest(&restoresv1.GetRestoreRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get restore %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Restore == nil {
		return fmt.Errorf("server returned no restore")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched restore %s.", resp.Msg.Restore.Id)},
		ResultsHeading: "Restore",
		Results:        []string{formatRestore(resp.Msg.Restore)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRestores(context.Background(), connect.NewRequest(&restoresv1.ListRestoresRequest{
		TargetId: ctx.Flag("target"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list restores", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no restores response")
	}
	results := make([]string, 0, len(resp.Msg.Restores))
	for _, r := range resp.Msg.Restores {
		results = append(results, formatRestore(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d restore(s).", len(resp.Msg.Restores))},
		ResultsHeading: "Restores",
		Results:        results,
		RetrievalHints: []string{
			"`restores get <id>` — show a single restore",
			"`restores restore --target <id> --destination <id> --snapshot <id> --location <path>` — start a restore",
		},
	})
}

func restoreModeLabel(m restoresv1.RestoreMode) string {
	switch m {
	case restoresv1.RestoreMode_RESTORE_MODE_RESTORE:
		return "restore"
	case restoresv1.RestoreMode_RESTORE_MODE_VERIFY:
		return "verify"
	default:
		return "unspecified"
	}
}

func restoreStatusLabel(s restoresv1.RestoreStatus) string {
	switch s {
	case restoresv1.RestoreStatus_RESTORE_STATUS_REQUESTED:
		return "requested"
	case restoresv1.RestoreStatus_RESTORE_STATUS_RESTORING:
		return "restoring"
	case restoresv1.RestoreStatus_RESTORE_STATUS_VERIFYING:
		return "verifying"
	case restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED:
		return "verified"
	case restoresv1.RestoreStatus_RESTORE_STATUS_RESTORED:
		return "restored"
	case restoresv1.RestoreStatus_RESTORE_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func formatRestore(r *restoresv1.Restore) string {
	if r == nil {
		return "(nil)"
	}
	requested := ""
	if r.RequestedAt != nil {
		requested = r.RequestedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — target=%s destination=%s snapshot=%s mode=%s status=%s requested=%s",
		r.Id, r.TargetId, r.DestinationId, r.SnapshotId,
		restoreModeLabel(r.Mode), restoreStatusLabel(r.Status), requested)
}
