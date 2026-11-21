package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the "metrics" subcommand group bound to the given ScenarioApp.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "metrics",
		Description: "Inspect process metrics for sessions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "tail",
				Description: "Poll session metrics every 2s (Ctrl+C to stop)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runTail(core, args) },
			},
		},
	}
}

func runTail(core *cliapp.ScenarioApp, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: metrics tail <id>")
	}
	id := args[0]
	for {
		body, err := core.Get(fmt.Sprintf("/sessions/%s/metrics", id), nil)
		if err != nil {
			return err
		}
		// The response is either a map[string]string{"status":"no_monitor"} or a procmetrics.Report.
		var generic map[string]any
		if err := json.Unmarshal(body, &generic); err != nil {
			return fmt.Errorf("decoding metrics: %w", err)
		}
		report := cliapp.OperationalReport{
			Status: []string{fmt.Sprintf("session=%s @ %s", id, time.Now().UTC().Format(time.RFC3339))},
		}
		if status, ok := generic["status"].(string); ok && status == "no_monitor" {
			report.Status = append(report.Status, "no monitor attached (no app launched yet)")
		} else {
			pretty, _ := json.MarshalIndent(generic, "", "  ")
			report.Status = append(report.Status, string(pretty))
		}
		if err := cliapp.RenderOperationalReport(os.Stdout, report); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}
