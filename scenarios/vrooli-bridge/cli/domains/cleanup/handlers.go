package cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup/cleanup_v1connect"
	"vrooli-bridge/cli/internal/operatorauth"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	client cleanupconnect.CleanupServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	hc, base := session.NewConnectHTTPClient(core)
	return &handlers{client: cleanupconnect.NewCleanupServiceClient(hc, base)}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	resp, err := h.client.StartCleanup(context.Background(), connect.NewRequest(&cleanupv1.StartCleanupRequest{MachineId: ctx.Flag("machine"), NodeId: ctx.Flag("node"), Target: ctx.Flag("target"), Scope: ctx.Flag("scope")}))
	if err != nil {
		return cliapp.WrapAPIError("start cleanup", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "cleanup operation started")
}

func (h *handlers) provisionBreakGlass(ctx cliapp.RunContext) error {
	prepared, err := h.client.PrepareCleanup(context.Background(), connect.NewRequest(&cleanupv1.PrepareCleanupRequest{MachineId: ctx.Flag("machine"), NodeId: ctx.Flag("node"), Target: ctx.Flag("target"), Scope: ctx.Flag("scope")}))
	if err != nil {
		return cliapp.WrapAPIError("prepare break-glass provisioning", err, nil)
	}
	target := prepared.Msg.GetTarget()
	if target == nil || len(target.GetSealingPublicKey()) == 0 || strings.TrimSpace(target.GetOperationId()) == "" {
		return fmt.Errorf("cleanup target did not publish a sealing key and operation id")
	}
	op := &cleanupv1.CleanupOperation{Id: target.GetOperationId(), MachineId: target.GetMachineId(), NodeId: target.GetNodeId(), Target: target.GetTarget(), Scope: target.GetScope(), OperatorId: target.GetOperatorId(), SealingPublicKey: target.GetSealingPublicKey()}
	if operatorID := strings.TrimSpace(ctx.Flag("operator-id")); operatorID != "" {
		op.OperatorId = operatorID
	}
	sealed, _, err := readAuthorization(os.Stdin, op)
	if err != nil {
		return err
	}
	resp, err := h.client.ProvisionBreakGlass(context.Background(), connect.NewRequest(&cleanupv1.ProvisionBreakGlassRequest{MachineId: target.GetMachineId(), NodeId: target.GetNodeId(), Target: target.GetTarget(), Scope: target.GetScope(), OperationId: target.GetOperationId(), SealedPassphrase: sealed, OperatorId: op.GetOperatorId()}))
	if err != nil {
		return cliapp.WrapAPIError("provision target break-glass material", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "target protection provisioning dispatched")
}

func (h *handlers) resetBreakGlass(ctx cliapp.RunContext) error {
	resp, err := h.client.ResetBreakGlass(context.Background(), connect.NewRequest(&cleanupv1.ResetBreakGlassRequest{
		MachineId: ctx.Flag("machine"), NodeId: ctx.Flag("node"), Target: ctx.Flag("target"), Scope: ctx.Flag("scope"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("retire target break-glass material", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "target break-glass retirement dispatched")
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCleanup(context.Background(), connect.NewRequest(&cleanupv1.GetCleanupRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get cleanup", err, nil)
	}
	if ctx.JSON() {
		return writeJSON(ctx.Stdout(), resp.Msg)
	}
	return render(ctx, resp.Msg.GetOperation(), fmt.Sprintf("%d cleanup event(s)", len(resp.Msg.GetEvents())))
}

func (h *handlers) plan(ctx cliapp.RunContext) error {
	resp, err := h.client.PlanCleanup(context.Background(), connect.NewRequest(&cleanupv1.PlanCleanupRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("read cleanup plan", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "frozen cleanup plan")
}

func (h *handlers) confirm(ctx cliapp.RunContext) error {
	lookup, err := h.client.GetCleanup(context.Background(), connect.NewRequest(&cleanupv1.GetCleanupRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("read cleanup authorization target", err, nil)
	}
	op := lookup.Msg.GetOperation()
	if op == nil || len(op.GetSealingPublicKey()) == 0 {
		return fmt.Errorf("cleanup operation does not publish a node sealing key")
	}
	if operatorID := strings.TrimSpace(ctx.Flag("operator-id")); operatorID != "" {
		op.OperatorId = operatorID
	}
	sealed, capability, err := readAuthorization(os.Stdin, op)
	if err != nil {
		return err
	}
	resp, err := h.client.ConfirmCleanup(context.Background(), connect.NewRequest(&cleanupv1.ConfirmCleanupRequest{Id: ctx.Positional("id"), Target: ctx.Flag("target"), PlanHash: ctx.Flag("plan-hash"), SealedPassphrase: sealed, Capability: capability, OperatorId: ctx.Flag("operator-id")}))
	if err != nil {
		return cliapp.WrapAPIError("confirm cleanup", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "cleanup confirmation accepted; application dispatched")
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	resp, err := h.client.ApplyCleanup(context.Background(), connect.NewRequest(&cleanupv1.ApplyCleanupRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("apply cleanup", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "cleanup resume dispatched")
}

func (h *handlers) verify(ctx cliapp.RunContext) error {
	resp, err := h.client.VerifyCleanup(context.Background(), connect.NewRequest(&cleanupv1.VerifyCleanupRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("verify cleanup", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "cleanup verification dispatched")
}

func (h *handlers) cancel(ctx cliapp.RunContext) error {
	resp, err := h.client.CancelCleanup(context.Background(), connect.NewRequest(&cleanupv1.CancelCleanupRequest{Id: ctx.Positional("id"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("cancel cleanup", err, nil)
	}
	return render(ctx, resp.Msg.GetOperation(), "cleanup canceled")
}

func render(ctx cliapp.RunContext, op *cleanupv1.CleanupOperation, message string) error {
	if op == nil {
		return fmt.Errorf("server returned no cleanup operation")
	}
	if ctx.JSON() {
		return writeJSON(ctx.Stdout(), op)
	}
	_, err := fmt.Fprintf(ctx.Stdout(), "%s: %s status=%s target=%s plan_hash=%s\n", message, op.Id, strings.TrimPrefix(op.Status.String(), "CLEANUP_STATUS_"), op.Target, empty(op.PlanHash))
	if len(op.PlanJson) > 0 {
		_, _ = fmt.Fprintf(ctx.Stdout(), "Frozen plan: %s\n", string(op.PlanJson))
	}
	if len(op.ReceiptJson) > 0 {
		_, _ = fmt.Fprintf(ctx.Stdout(), "Receipt: %s\n", string(op.ReceiptJson))
	}
	return err
}

func readAuthorization(input io.Reader, op *cleanupv1.CleanupOperation) ([]byte, []byte, error) {
	return operatorauth.Read(input, operatorauth.Target{
		MachineID: op.GetMachineId(), NodeID: op.GetNodeId(), Target: op.GetTarget(), Scope: op.GetScope(),
		PlanHash: op.GetPlanHash(), OperationID: op.GetId(), OperatorID: op.GetOperatorId(), SealingPublicKey: op.GetSealingPublicKey(),
	})
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func empty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
