package timeline

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "actions",
		Description: "Inspect timeline, incidents, uptime, and action history",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "history", Description: "Show recent recovery action history", Run: func(args []string) error { return actionHistory(core, args) }},
			{Name: "timeline", Description: "Show recent timeline events", Run: func(args []string) error { return timeline(core, args) }},
			{Name: "incidents", Description: "Show recent incidents", Run: func(args []string) error { return incidents(core, args) }},
			{Name: "uptime", Description: "Show uptime summary", Run: func(args []string) error { return uptime(core, args) }},
			{Name: "trends", Description: "Show check trends", Run: func(args []string) error { return trends(core, args) }},
		},
	}
}

func actionHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("actions history")
	checkID := fs.String("check-id", "", "Filter by check id")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{}
	if *checkID != "" {
		query.Set("checkId", *checkID)
	}
	body, err := core.Get("/actions/history", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.ActionLogsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	results := make([]string, 0, len(resp.Logs))
	for _, log := range resp.Logs {
		results = append(results, fmt.Sprintf("%s %s/%s success=%s: %s", log.Timestamp, log.CheckID, log.ActionID, support.BoolWord(log.Success), log.Message))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recovery action history entries: %d", resp.Total)},
		ResultsHeading: "Actions",
		Results:        results,
		RetrievalHints: []string{"vrooli-autoheal check actions <check-id>"},
	})
}

func timeline(core *cliapp.ScenarioApp, args []string) error {
	return renderJSONOnlyGet(core, "/timeline", args)
}

func incidents(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("actions incidents")
	hours := fs.Int("hours", 24, "Incident window in hours")
	limit := fs.Int("limit", 50, "Maximum incidents")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{
		"hours": []string{strconv.Itoa(*hours)},
		"limit": []string{strconv.Itoa(*limit)},
	}
	body, err := core.Get("/incidents", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}

func uptime(core *cliapp.ScenarioApp, args []string) error {
	return renderJSONOnlyGet(core, "/uptime", args)
}

func trends(core *cliapp.ScenarioApp, args []string) error {
	return renderJSONOnlyGet(core, "/checks/trends", args)
}

func renderJSONOnlyGet(core *cliapp.ScenarioApp, path string, args []string) error {
	fs := support.NewFlagSet(path)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}
