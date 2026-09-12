package schedule

import (
	"fmt"
	"math"
	"os"
	"strings"

	"calendar/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `schedule` subcommand group for AI-driven scheduling.
// Both endpoints accept a natural-language message and return suggested actions.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "schedule",
		Description: "AI-assisted scheduling (chat interface and optimization)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "chat", Description: "Natural-language scheduling request", Run: func(args []string) error { return runChat(core, args) }},
			{Name: "optimize", Description: "AI-powered schedule optimization", Run: func(args []string) error { return runOptimize(core, args) }},
		},
	}
}

func runChat(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule chat")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	message := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if message == "" {
		return fmt.Errorf("usage: schedule chat <message>")
	}

	body, err := core.Request("POST", "/schedule/chat", nil, map[string]interface{}{
		"message": message,
	})
	if err != nil {
		return err
	}
	var resp support.ChatResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{fmt.Sprintf("Assistant: %s", resp.Response)}
	if len(resp.SuggestedActions) > 0 {
		results = append(results, "", "Suggested actions:")
		for _, action := range resp.SuggestedActions {
			results = append(results, fmt.Sprintf("- %s (confidence: %d%%)",
				action.Action, int(math.Floor(action.Confidence*100))))
		}
	}
	if resp.RequiresConfirmation {
		results = append(results, "", "Requires confirmation — re-run with a specific command to proceed.")
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Processed: %s", message)},
		ResultsHeading: "Response",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s event create --title ... --start ... --end ...", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runOptimize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule optimize")
	preserveHighPriority := fs.Bool("preserve-high-priority", false, "Protect high-priority events during optimization")
	minBufferMinutes := fs.Int("min-buffer-minutes", 0, "Minimum gap between events in minutes")
	businessHoursOnly := fs.Bool("business-hours-only", false, "Limit suggestions to business hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	request := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if request == "" {
		return fmt.Errorf("usage: schedule optimize <request>")
	}

	payload := map[string]interface{}{
		"request": request,
		"constraints": map[string]interface{}{
			"preserve_high_priority": *preserveHighPriority,
			"min_buffer_minutes":     *minBufferMinutes,
			"business_hours_only":    *businessHoursOnly,
		},
	}
	body, err := core.Request("POST", "/schedule/optimize", nil, payload)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Optimization request: %s", request)},
		ResultsHeading: "Response",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s schedule optimize \"%s\" --json", support.CLIName, request)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
