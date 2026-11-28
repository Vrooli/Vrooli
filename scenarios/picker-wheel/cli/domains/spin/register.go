package spin

import (
	"fmt"
	"os"

	"picker-wheel/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `picker-wheel spin` as a flat command. The API endpoint
// POST /api/spin is a single action and expects either a wheel_id (to look up
// server-side options) or an explicit options list, so we use --body-file to
// pass the JSON payload straight through rather than composing it client-side.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Spin",
		Commands: []cliapp.Command{
			{
				Name:        "spin",
				Description: "Spin a wheel and record the result (body via --body-file)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runSpin(core, args) },
			},
		},
	}
}

func runSpin(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("spin")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the spin payload (wheel_id and/or options, session_id)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/spin", nil, payload)
	if err != nil {
		return err
	}
	var result support.SpinResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("Result: %s", result.Result)}
	if result.WheelID != "" {
		changes = append(changes, fmt.Sprintf("Wheel: %s", result.WheelID))
	}
	if result.SessionID != "" {
		changes = append(changes, fmt.Sprintf("Session: %s", result.SessionID))
	}
	changes = append(changes, fmt.Sprintf("Timestamp: %s", support.FormatTimeValue(result.Timestamp)))

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Spin result: %s", result.Result)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s history", support.CLIName),
			fmt.Sprintf("%s wheel list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
