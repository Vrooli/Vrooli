package launch

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"research-desk/cli/domains/render"

	launchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch"
	launchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch/launch_v1connect"
)

type handlers struct {
	client launchconnect.LaunchServiceClient
}

// specFile reads a spec JSON file; packet.py validates its content.
func specFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read the spec file: %w (a spec is JSON as in docs/reference/FILE-CONTRACT.md §2)", err)
	}
	return string(data), nil
}

func (h *handlers) optionsCall(ctx cliapp.OperationContext) (*launchv1.GetLaunchOptionsResponse, error) {
	return render.Call("read the launch options", h.client.GetLaunchOptions, &launchv1.GetLaunchOptionsRequest{Topics: list(ctx.Flag("topics"))})
}

// list splits a comma-separated flag.
func list(flag string) []string {
	var out []string
	for _, s := range strings.Split(flag, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// blockerChoices reads --supersede and --alongside, comma-separated: the
// commissions this launch supersedes and those it runs alongside. The API
// refuses a commission named in both.
func blockerChoices(ctx cliapp.OperationContext) (supersedes, runAlongside []string) {
	return list(ctx.Flag("supersede")), list(ctx.Flag("alongside"))
}

// preflightCall sends --spec-file, or a target naming a file, as a spec and
// any other target as a commission.
func (h *handlers) preflightCall(ctx cliapp.OperationContext) (*launchv1.PreflightResponse, error) {
	target, file := ctx.Positional("target"), ctx.Flag("spec-file")
	switch {
	case target != "" && file != "":
		return nil, fmt.Errorf("give a commission or --spec-file, not both")
	case file != "":
		target = file
	case target == "":
		return nil, fmt.Errorf("give a prepared commission's folder name, or a spec JSON file (docs/reference/FILE-CONTRACT.md §2)")
	}
	sup, along := blockerChoices(ctx)
	req := &launchv1.PreflightRequest{Supersedes: sup, RunAlongside: along, AssertNeverStarted: ctx.BoolFlag("assert-never-started")}
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		spec, err := specFile(target)
		if err != nil {
			return nil, err
		}
		req.Target = &launchv1.PreflightRequest_SpecJson{SpecJson: spec}
	} else {
		req.Target = &launchv1.PreflightRequest_CommissionId{CommissionId: target}
	}
	return render.Call("preflight "+target, h.client.Preflight, req)
}

func (h *handlers) buildCall(ctx cliapp.OperationContext) (*launchv1.BuildResponse, error) {
	old, file := ctx.Flag("relaunch-of"), ctx.Positional("spec-file")
	sup, along := blockerChoices(ctx)
	switch {
	case old != "" && file != "":
		return nil, fmt.Errorf("give a spec file or --relaunch-of, not both")
	case old != "":
		return render.Call("relaunch "+old, h.client.Build, &launchv1.BuildRequest{Source: &launchv1.BuildRequest_RelaunchOf{RelaunchOf: old}, Supersedes: sup, RunAlongside: along})
	case file == "":
		return nil, fmt.Errorf("give a spec JSON file (docs/reference/FILE-CONTRACT.md §2) or --relaunch-of <commission-id>")
	}
	spec, err := specFile(file)
	if err != nil {
		return nil, err
	}
	return render.Call("build a packet", h.client.Build, &launchv1.BuildRequest{Source: &launchv1.BuildRequest_SpecJson{SpecJson: spec}, Supersedes: sup, RunAlongside: along})
}

func (h *handlers) launchCall(ctx cliapp.OperationContext) (*launchv1.LaunchResponse, error) {
	sup, along := blockerChoices(ctx)
	req := &launchv1.LaunchRequest{CommissionId: ctx.Positional("commission-id"), Supersedes: sup, RunAlongside: along, AssertNeverStarted: ctx.BoolFlag("assert-never-started")}
	return render.Call("launch "+req.CommissionId, h.client.Launch, req)
}

func (h *handlers) messagesCall(ctx cliapp.OperationContext) (*launchv1.GetLaunchMessagesResponse, error) {
	id := ctx.Positional("commission-id")
	return render.Call("read the launch messages of "+id, h.client.GetLaunchMessages, &launchv1.GetLaunchMessagesRequest{CommissionId: id})
}

func (h *handlers) statusCall(ctx cliapp.OperationContext) (*launchv1.GetLaunchResponse, error) {
	id := ctx.Positional("commission-id")
	return render.Call("read the launch of "+id, h.client.GetLaunch, &launchv1.GetLaunchRequest{CommissionId: id})
}

var actionWords = map[string]launchv1.ControlAction{
	"pause":         launchv1.ControlAction_CONTROL_ACTION_PAUSE,
	"resume":        launchv1.ControlAction_CONTROL_ACTION_RESUME,
	"stop":          launchv1.ControlAction_CONTROL_ACTION_STOP,
	"discard":       launchv1.ControlAction_CONTROL_ACTION_DISCARD,
	"resend-stance": launchv1.ControlAction_CONTROL_ACTION_RESEND_STANCE,
}

func (h *handlers) controlCall(ctx cliapp.OperationContext) (*launchv1.ControlResponse, error) {
	word := ctx.Positional("action")
	action, ok := actionWords[word]
	if !ok {
		if word == "wind-down" {
			return nil, fmt.Errorf("wind down is an OPERATOR.md entry, not a keystroke: `say <commission-id> wind-down`")
		}
		return nil, fmt.Errorf("the action is pause, resume, stop, discard or resend-stance, not %q", word)
	}
	req := &launchv1.ControlRequest{CommissionId: ctx.Positional("commission-id"), Action: action, Reason: strings.TrimSpace(ctx.Flag("reason"))}
	return render.Call(word+" "+req.CommissionId, h.client.Control, req)
}

func (h *handlers) testCall(cliapp.OperationContext) (*launchv1.LaunchTestReport, error) {
	return render.Call("start a launch test", h.client.TestLaunch, &launchv1.TestLaunchRequest{})
}

func (h *handlers) testStatusCall(cliapp.OperationContext) (*launchv1.LaunchTestReport, error) {
	return render.Call("read the newest launch test", h.client.GetLaunchTest, &launchv1.GetLaunchTestRequest{})
}
