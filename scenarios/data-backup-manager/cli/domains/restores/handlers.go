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
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Restore requested: %s.", resp.Msg.Restore.Id)},
		Changes: []string{formatRestore(resp.Msg.Restore)},
		NextCommand: []string{
			fmt.Sprintf("`restores get %s` — show restore status", resp.Msg.Restore.Id),
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
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Verify requested: %s.", resp.Msg.Restore.Id)},
		Changes: []string{formatRestore(resp.Msg.Restore)},
		NextCommand: []string{
			fmt.Sprintf("`restores get %s` — show verify status", resp.Msg.Restore.Id),
		},
	})
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
