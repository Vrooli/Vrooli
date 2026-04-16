package capabilities

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `web-console capabilities` as a flat command covering
// both the capability snapshot and the cheaper `--liveness` probe.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Capabilities",
		Commands: []cliapp.Command{
			{
				Name:        "capabilities",
				Description: "Inspect runtime capabilities (use --liveness for a cheap probe)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capabilities")
	liveness := fs.Bool("liveness", false, "Probe /capabilities/liveness instead of the full snapshot")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	path := "/capabilities"
	if *liveness {
		path = "/capabilities/liveness"
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	status := "unknown"
	if v, ok := payload["status"].(string); ok && v != "" {
		status = v
	}

	summary := []string{fmt.Sprintf("Capabilities: %s", status)}
	if *liveness {
		summary = []string{fmt.Sprintf("Capabilities liveness: %s", status)}
	}

	report := cliapp.OperationalReport{
		Status:    summary,
		Triage:    []cliapp.TriageGroup{{Heading: "Findings", Items: support.MapRows(payload)}},
		NextSteps: []string{fmt.Sprintf("%s capabilities --liveness", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
