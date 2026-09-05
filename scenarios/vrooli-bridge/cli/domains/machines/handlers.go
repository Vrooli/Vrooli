package machines

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"

	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
)

type handlers struct {
	client machinesconnect.MachineServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{client: machinesconnect.NewMachineServiceClient(httpClient, baseURL)}
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	locators, err := parseLocators(ctx.Flag("locators"))
	if err != nil {
		return err
	}
	resp, err := h.client.CreateMachine(context.Background(), connect.NewRequest(&machinesv1.CreateMachineRequest{Locators: locators, DesiredProfileId: ctx.Flag("profile"), DesiredProfileVersion: ctx.Flag("profile-version")}))
	if err != nil {
		return cliapp.WrapAPIError("create machine", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Machine == nil {
		return fmt.Errorf("server returned no machine")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Created machine %s.", resp.Msg.Machine.Id)}, Changes: []string{formatMachine(resp.Msg.Machine)}, NextCommand: []string{fmt.Sprintf("`machines get %s` — show machine intent and lineage", resp.Msg.Machine.Id)}})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetMachine(context.Background(), connect.NewRequest(&machinesv1.GetMachineRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get machine %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Machine == nil {
		return fmt.Errorf("server returned no machine")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Fetched machine %s.", id)}, ResultsHeading: "Machine", Results: []string{formatMachine(resp.Msg.Machine)}})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListMachines(context.Background(), connect.NewRequest(&machinesv1.ListMachinesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list machines", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no machines response")
	}
	results := make([]string, 0, len(resp.Msg.Machines))
	for _, m := range resp.Msg.Machines {
		results = append(results, formatMachine(m))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%d machine(s).", len(results))}, ResultsHeading: "Machines", Results: results, RetrievalHints: []string{"`machines get <id>` — show one Machine", "`machines create --locators hostname=host.example` — add intent before contact"}})
}
func (h *handlers) archive(ctx cliapp.RunContext) error { return h.lifecycle(ctx, "archive") }
func (h *handlers) remove(ctx cliapp.RunContext) error  { return h.lifecycle(ctx, "remove") }
func (h *handlers) lifecycle(ctx cliapp.RunContext, action string) error {
	id := ctx.Positional("id")
	version, err := parseVersion(ctx.Flag("version"))
	if err != nil {
		return err
	}
	var machine *machinesv1.Machine
	if action == "archive" {
		resp, e := h.client.ArchiveMachine(context.Background(), connect.NewRequest(&machinesv1.ArchiveMachineRequest{Id: id, Version: version}))
		err = e
		if resp != nil && resp.Msg != nil {
			machine = resp.Msg.Machine
		}
	} else {
		resp, e := h.client.RemoveMachine(context.Background(), connect.NewRequest(&machinesv1.RemoveMachineRequest{Id: id, Version: version}))
		err = e
		if resp != nil && resp.Msg != nil {
			machine = resp.Msg.Machine
		}
	}
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("%s machine %q", action, id), err, nil)
	}
	if machine == nil {
		return fmt.Errorf("server returned no machine")
	}
	actionLabel := action
	if actionLabel != "" {
		actionLabel = strings.ToUpper(actionLabel[:1]) + actionLabel[1:]
	}
	return cliapp.RenderProtoMutation(ctx, machine, cliapp.MutationReport{Result: []string{fmt.Sprintf("%sd machine %s.", actionLabel, id)}, Changes: []string{formatMachine(machine)}, NextCommand: []string{"`machines list` — inspect remaining machine intent"}})
}

func (h *handlers) getTrust(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetMachineTrust(context.Background(), connect.NewRequest(&machinesv1.GetMachineTrustRequest{MachineId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get Machine trust %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Trust == nil {
		return fmt.Errorf("server returned no Machine trust")
	}
	t := resp.Msg.Trust
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Trust for Machine %s.", id)}, ResultsHeading: "Trust", Results: []string{fmt.Sprintf("host=%s state=%s client=%s", t.HostKeyFingerprint, t.HostKeyState, t.ClientKeyFingerprint)}})
}

func (h *handlers) reviewHostKey(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.ReviewMachineHostKey(context.Background(), connect.NewRequest(&machinesv1.ReviewMachineHostKeyRequest{MachineId: id, ReplacementHostKeyFingerprint: ctx.Flag("fingerprint")}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("review Machine host key %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Trust == nil {
		return fmt.Errorf("server returned no reviewed Machine trust")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Approved replacement host key for Machine %s.", id)}, Changes: []string{fmt.Sprintf("host=%s state=%s", resp.Msg.Trust.HostKeyFingerprint, resp.Msg.Trust.HostKeyState)}})
}

func (h *handlers) requestSSHCleanup(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RequestMachineSSHCleanup(context.Background(), connect.NewRequest(&machinesv1.RequestMachineSSHCleanupRequest{MachineId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("request SSH cleanup for Machine %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Cleanup == nil {
		return fmt.Errorf("server returned no cleanup record")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded legacy cleanup intent %s.", resp.Msg.Cleanup.Id)}, Changes: []string{fmt.Sprintf("status=%s", resp.Msg.Cleanup.Status)}})
}

func (h *handlers) updateCleanup(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.UpdateMachineCleanup(context.Background(), connect.NewRequest(&machinesv1.UpdateMachineCleanupRequest{Id: id, Status: ctx.Flag("status"), Detail: ctx.Flag("detail")}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("update cleanup %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Cleanup == nil {
		return fmt.Errorf("server returned no cleanup record")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated cleanup record %s.", id)}, Changes: []string{fmt.Sprintf("status=%s", resp.Msg.Cleanup.Status)}})
}

func (h *handlers) applyPolicy(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	version, err := parseVersion(ctx.Flag("version"))
	if err != nil {
		return err
	}
	overrides := map[string]string{}
	if preset := strings.TrimSpace(ctx.Flag("preset")); preset != "" {
		overrides["preset"] = preset
	}
	resp, err := h.client.ApplyMachinePolicy(context.Background(), connect.NewRequest(&machinesv1.ApplyMachinePolicyRequest{MachineId: id, Version: version, ProfileId: ctx.Flag("profile"), ProfileVersion: ctx.Flag("profile-version"), Overrides: overrides, Reason: ctx.Flag("reason"), ConfirmRemoval: ctx.Flag("confirm-removal") == "true"}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("apply policy to Machine %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Machine == nil || resp.Msg.Policy == nil {
		return fmt.Errorf("server returned no Machine policy")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Applied %s@%s to Machine %s.", resp.Msg.Policy.ProfileId, resp.Msg.Policy.ProfileVersion, id)}, Changes: []string{formatMachine(resp.Msg.Machine), fmt.Sprintf("preset=%s suggested scopes=%s", resp.Msg.Policy.SetupPreset, strings.Join(resp.Msg.Policy.SuggestedScopes, ","))}})
}

func (h *handlers) revokeNode(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RevokeMachineNode(context.Background(), connect.NewRequest(&machinesv1.RevokeMachineNodeRequest{MachineId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("revoke current Node for Machine %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no node revocation")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Revoked Node %s for Machine %s.", resp.Msg.RevokedNodeId, id)}, Changes: []string{"Durable Node identity, credential, and live channel were revoked locally. SSH cleanup remains a separate explicit effect."}})
}

func (h *handlers) repair(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RepairMachine(context.Background(), connect.NewRequest(&machinesv1.RepairMachineRequest{MachineId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("repair machine %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Machine == nil {
		return fmt.Errorf("server returned no repaired machine")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Repair started for Machine %s.", id)},
		Changes:     []string{formatMachine(resp.Msg.Machine), "reused the durable Machine identity and Bridge-managed SSH key", fmt.Sprintf("onboarding op=%s enrollment attempt=%s", resp.Msg.OnboardingOpId, resp.Msg.EnrollmentAttemptId)},
		NextCommand: []string{fmt.Sprintf("`onboard watch %s` — follow repair to ONLINE", resp.Msg.OnboardingOpId)},
	})
}

func (h *handlers) merge(ctx cliapp.RunContext) error {
	fromID := ctx.Positional("from")
	intoID := ctx.Positional("into")
	resp, err := h.client.MergeMachines(context.Background(), connect.NewRequest(&machinesv1.MergeMachinesRequest{FromMachineId: fromID, IntoMachineId: intoID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("merge machine %q into %q", fromID, intoID), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Machine == nil {
		return fmt.Errorf("server returned no merged machine")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Merged Machine %s into %s.", fromID, intoID)},
		Changes: []string{formatMachine(resp.Msg.Machine), fmt.Sprintf("archived source machine=%s; lineage, locators, attempts, and audit history remain durable", resp.Msg.ArchivedMachineId)},
	})
}

func parseLocators(raw string) ([]*machinesv1.ConnectionLocator, error) {
	parts := strings.Split(raw, ",")
	out := make([]*machinesv1.ConnectionLocator, 0, len(parts))
	for i, part := range parts {
		kind, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("--locators must be comma-separated kind=value entries (for example hostname=host.example,ip=192.0.2.10)")
		}
		out = append(out, &machinesv1.ConnectionLocator{Kind: strings.TrimSpace(kind), Value: strings.TrimSpace(value), Ordinal: int32(i)})
	}
	return out, nil
}

func parseVersion(raw string) (int64, error) {
	version, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || version < 1 {
		return 0, fmt.Errorf("--version must be a positive integer")
	}
	return version, nil
}

func formatMachine(m *machinesv1.Machine) string {
	if m == nil {
		return "(nil)"
	}
	created := ""
	if m.CreatedAt != nil {
		created = m.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — lifecycle=%s version=%d locators=%d lineage=%d profile=%s@%s created=%s", m.Id, m.Lifecycle, m.Version, len(m.Locators), len(m.NodeLineage), m.DesiredProfileId, m.DesiredProfileVersion, created)
}
