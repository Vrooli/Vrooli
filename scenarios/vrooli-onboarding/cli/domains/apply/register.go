package apply

import (
	"fmt"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	"google.golang.org/protobuf/proto"
)

const (
	startProcedure = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply"
	runProcedure   = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyRun"
	planProcedure  = "/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan"
)

// Register exposes the typed long-running apply surface. The run id belongs in
// the request message so callers can use the same contract over Connect and
// do not need to construct a URL path.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "apply",
		Description: "Plan, start, and inspect onboarding configuration",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "start", Description: "Start applying the committed onboarding selection", Run: func(args []string) error { return start(core, args) }},
			{Name: "status", Description: "Show the state of an apply run", Run: func(args []string) error { return status(core, args) }},
			{Name: "plan", Description: "Show the committed onboarding apply plan", Run: func(args []string) error { return plan(core, args) }},
		},
	}
}

func targetFlags(name string, args []string) (*string, *bool, error) {
	fs := support.NewFlagSet(name)
	target := fs.String("target", "local", "Onboarding target scenario")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(*target) == "" {
		return nil, nil, fmt.Errorf("--target is required")
	}
	return target, jsonOutput, nil
}

func request(core *cliapp.ScenarioApp, procedure string, message proto.Message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "apply")
}

func printJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "apply")
}

func start(core *cliapp.ScenarioApp, args []string) error {
	target, jsonOutput, err := targetFlags("apply start", args)
	if err != nil {
		return err
	}
	response := &applyv1.StartApplyResponse{}
	if err := request(core, startProcedure, &applyv1.StartApplyRequest{Target: strings.TrimSpace(*target)}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	if response.GetRun() == nil {
		return fmt.Errorf("apply start returned no run")
	}
	_, err = fmt.Fprintf(os.Stdout, "Apply run %s started with status %s\n", response.GetRun().GetRunId(), response.GetRun().GetStatus().String())
	return err
}

func status(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("apply status")
	target := fs.String("target", "local", "Onboarding target scenario")
	runID := fs.String("run-id", "", "Apply run id")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*target) == "" || strings.TrimSpace(*runID) == "" {
		return fmt.Errorf("--target and --run-id are required")
	}
	response := &applyv1.GetApplyRunResponse{}
	if err := request(core, runProcedure, &applyv1.GetApplyRunRequest{Target: strings.TrimSpace(*target), RunId: strings.TrimSpace(*runID)}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	_, err := fmt.Fprintf(os.Stdout, "Apply run %s: %s\n", response.GetRunId(), response.GetStatus().String())
	return err
}

func plan(core *cliapp.ScenarioApp, args []string) error {
	target, jsonOutput, err := targetFlags("apply plan", args)
	if err != nil {
		return err
	}
	response := &applyv1.GetApplyPlanResponse{}
	if err := request(core, planProcedure, &applyv1.GetApplyPlanRequest{Target: strings.TrimSpace(*target)}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	_, err = fmt.Fprintf(os.Stdout, "Apply plan: %d item(s)\n", len(response.GetItems()))
	for _, item := range response.GetItems() {
		if _, writeErr := fmt.Fprintf(os.Stdout, "  - %s %s (%s)\n", item.GetKind(), item.GetName(), item.GetObservedState()); writeErr != nil && err == nil {
			err = writeErr
		}
	}
	return err
}
