package machines

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines/machines_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client machinesconnect.MachineServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: machinesconnect.NewMachineServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	response, err := h.client.List(context.Background(), connect.NewRequest(&machinesv1.ListRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("machine list", err, nil)
	}
	msg := response.Msg
	summary := []string{fmt.Sprintf("Fleet: %s (%d machines, %d asking to join)", msg.GetState(), len(msg.GetMachines()), len(msg.GetJoinRequests()))}
	if control := msg.GetControlPlane(); control != nil && !control.GetReachable() {
		summary = append(summary, fmt.Sprintf("Control plane: unreachable (%s)", defaultText(control.GetDetail(), "no detail reported")))
	}
	if note := strings.TrimSpace(msg.GetMessage()); note != "" {
		summary = append(summary, note)
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Machines", Results: machineRows(msg.GetMachines()),
		RetrievalHints: []string{
			fmt.Sprintf("%s machine decide <request-id> --approve --words \"amber dolphin quartz\" --preset read-only", support.CLIName),
			fmt.Sprintf("%s machine grant <machine-id> --preset operate", support.CLIName),
		},
	}
	if requests := joinRows(msg.GetJoinRequests()); len(requests) > 0 {
		report.Summary = append(report.Summary, "")
		report.Summary = append(report.Summary, "Asking to join:")
		report.Summary = append(report.Summary, requests...)
	}
	if presets := presetRows(msg.GetPresets()); len(presets) > 0 {
		report.Summary = append(report.Summary, "")
		report.Summary = append(report.Summary, "Permission presets:")
		report.Summary = append(report.Summary, presets...)
	}
	return cliapp.RenderProtoList(ctx, msg, report)
}

func (h *handlers) issueCode(ctx cliapp.RunContext) error {
	response, err := h.client.IssueCode(context.Background(), connect.NewRequest(&machinesv1.IssueCodeRequest{
		Label: strings.TrimSpace(ctx.Flag("label")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("machine code", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{
		Result: []string{"Single-use join code issued. It is shown once."},
		Changes: []string{
			response.Msg.GetCode(),
			fmt.Sprintf("Expires in %s", humanDuration(response.Msg.GetExpiresInSeconds())),
		},
		NextCommand: []string{fmt.Sprintf("%s machine list", support.CLIName)},
	})
}

func (h *handlers) decide(ctx cliapp.RunContext) error {
	requestID := strings.TrimSpace(ctx.Positional("request-id"))
	if requestID == "" {
		return fmt.Errorf("usage: machine decide <request-id> [--approve --words \"one two three\" --preset <name>]")
	}
	approve := ctx.BoolFlag("approve")
	words := splitList(ctx.Flag("words"))
	if approve && len(words) == 0 {
		return fmt.Errorf("approving a machine requires the confirmation words it is showing: --words \"amber dolphin quartz\"")
	}
	response, err := h.client.Decide(context.Background(), connect.NewRequest(&machinesv1.DecideRequest{
		RequestId:         requestID,
		Approve:           approve,
		ConfirmationWords: words,
		Preset:            strings.TrimSpace(ctx.Flag("preset")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("machine decide", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{
		Result:      []string{response.Msg.GetMessage()},
		Changes:     machineDetails(response.Msg.GetMachine()),
		NextCommand: []string{fmt.Sprintf("%s machine list", support.CLIName)},
	})
}

func (h *handlers) setGrant(ctx cliapp.RunContext) error {
	machineID, err := machineID(ctx, "grant")
	if err != nil {
		return err
	}
	response, callErr := h.client.SetGrant(context.Background(), connect.NewRequest(&machinesv1.SetGrantRequest{
		MachineId: machineID,
		Preset:    strings.TrimSpace(ctx.Flag("preset")),
		Scopes:    splitList(ctx.Flag("scopes")),
	}))
	if callErr != nil {
		return cliapp.WrapAPIError("machine grant", callErr, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{
		Result:      []string{"Grant applied."},
		Changes:     machineDetails(response.Msg.GetMachine()),
		NextCommand: []string{fmt.Sprintf("%s machine list", support.CLIName)},
	})
}

func (h *handlers) forget(ctx cliapp.RunContext) error {
	id, err := machineID(ctx, "forget")
	if err != nil {
		return err
	}
	response, callErr := h.client.Forget(context.Background(), connect.NewRequest(&machinesv1.ForgetRequest{MachineId: id}))
	if callErr != nil {
		return cliapp.WrapAPIError("machine forget", callErr, nil)
	}
	return cliapp.RenderProtoMutation(ctx, response.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Forgot %s.", response.Msg.GetForgottenMachineId())},
		NextCommand: []string{fmt.Sprintf("%s machine list", support.CLIName)},
	})
}

// splitList accepts the way a person supplies a short list on a command line:
// the words as they read them off the other machine's screen, separated by
// spaces or commas. Empty elements are dropped rather than sent as blanks.
func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' })
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func machineID(ctx cliapp.RunContext, command string) (string, error) {
	id := strings.TrimSpace(ctx.Positional("machine-id"))
	if id == "" {
		return "", fmt.Errorf("usage: machine %s <machine-id>", command)
	}
	return id, nil
}

func machineRows(machines []*machinesv1.Machine) []string {
	if len(machines) == 0 {
		return []string{"(no machines)"}
	}
	rows := make([]string, 0, len(machines))
	for _, machine := range machines {
		target := machine.GetTarget()
		platform := strings.Trim(strings.Join([]string{target.GetOs(), target.GetArch()}, "/"), "/")
		rows = append(rows, fmt.Sprintf("%s | %s | %s | %s | %s",
			target.GetId(), target.GetLabel(), defaultText(platform, "platform unknown"),
			reachability(machine), machine.GetGrant().GetSummary()))
	}
	return rows
}

// reachability states age rather than implying it. A machine that stopped
// answering must say when, because a surface that hides age reads exactly like
// one that is answering.
func reachability(machine *machinesv1.Machine) string {
	if machine.GetTarget().GetDispatchable() {
		return fmt.Sprintf("reachable (%s)", humanAge(machine.GetHeartbeatAgeSeconds()))
	}
	if machine.GetHeartbeatAgeSeconds() <= 0 {
		return "never responded"
	}
	return fmt.Sprintf("not responding (last %s)", humanAge(machine.GetHeartbeatAgeSeconds()))
}

func joinRows(requests []*machinesv1.JoinRequest) []string {
	rows := make([]string, 0, len(requests))
	for _, request := range requests {
		rows = append(rows, fmt.Sprintf("  %s | %s | %s/%s | words: %s | %s",
			request.GetId(), request.GetName(), request.GetOs(), request.GetArch(),
			defaultText(strings.Join(request.GetConfirmationWords(), " "), "unavailable"),
			defaultText(request.GetKeyFingerprint(), "no fingerprint")))
	}
	return rows
}

func presetRows(presets []*machinesv1.PermissionPreset) []string {
	rows := make([]string, 0, len(presets))
	for _, preset := range presets {
		// The control plane expands a preset to one scope per app per effect.
		// Printing that list would bury the answer, so state the shape and
		// leave the enumeration to `machine grant --scope`.
		line := fmt.Sprintf("  %s — %s | %s | %s across %s", preset.GetName(), preset.GetDescription(),
			preset.GetSummary(), defaultText(strings.Join(preset.GetEffects(), "+"), "nothing"),
			appBreadth(preset.GetAppCount(), false))
		if withholds := strings.Join(preset.GetWithholds(), " "); withholds != "" {
			line += fmt.Sprintf(" | withholds %s", withholds)
		}
		rows = append(rows, line)
	}
	return rows
}

func machineDetails(machine *machinesv1.Machine) []string {
	if machine == nil {
		return []string{"No machine was returned."}
	}
	target := machine.GetTarget()
	return []string{
		fmt.Sprintf("ID: %s", target.GetId()),
		fmt.Sprintf("Name: %s", target.GetLabel()),
		fmt.Sprintf("Platform: %s/%s", target.GetOs(), target.GetArch()),
		fmt.Sprintf("Reachability: %s", reachability(machine)),
		fmt.Sprintf("Permission: %s", machine.GetGrant().GetSummary()),
		fmt.Sprintf("Reach: %s across %s", defaultText(strings.Join(machine.GetGrant().GetEffects(), "+"), "nothing"),
			appBreadth(machine.GetGrant().GetAppCount(), machine.GetGrant().GetCoversAllApps())),
		fmt.Sprintf("Preset: %s", defaultText(machine.GetGrant().GetPreset(), "custom")),
		fmt.Sprintf("Scopes: %s", defaultText(strings.Join(machine.GetGrant().GetScopes(), " "), "none")),
	}
}

// appBreadth states reach the way the grant actually works. A wildcard reaches
// apps that do not exist yet, so it is never reported as a number.
func appBreadth(count int32, coversAll bool) string {
	if coversAll {
		return "every app"
	}
	if count == 1 {
		return "1 app"
	}
	return fmt.Sprintf("%d apps", count)
}

// humanDuration renders a remaining lifetime the way a person reads a
// countdown, so a short-lived credential states its window rather than a raw
// second count.
func humanDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	return fmt.Sprintf("%d minutes", seconds/60)
}

// humanAge renders an age the way a person says it. The unit changes with the
// magnitude so a seven-day silence never reads as a large number of seconds.
func humanAge(seconds int64) string {
	switch {
	case seconds <= 0:
		return "just now"
	case seconds < 60:
		return fmt.Sprintf("%ds ago", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm ago", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("%dh ago", seconds/3600)
	default:
		return fmt.Sprintf("%dd ago", seconds/86400)
	}
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
