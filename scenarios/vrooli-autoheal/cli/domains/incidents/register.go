package incidents

import (
	"bytes"
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
		Name:        "incidents",
		Description: "Inspect and update durable incidents",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List incidents", Run: func(args []string) error { return list(core, args, false) }},
			{Name: "latest", Description: "List latest open incidents", Run: func(args []string) error { return list(core, args, true) }},
			{Name: "show", Description: "Show one incident", Run: func(args []string) error { return show(core, args) }},
			{Name: "acknowledge", Description: "Acknowledge one incident", Run: func(args []string) error { return mutate(core, "acknowledge", args) }},
			{Name: "resolve", Description: "Resolve one incident", Run: func(args []string) error { return mutate(core, "resolve", args) }},
			{Name: "ignore", Description: "Ignore one incident", Run: func(args []string) error { return mutate(core, "ignore", args) }},
		},
	}
}

func list(core *cliapp.ScenarioApp, args []string, latest bool) error {
	fs := support.NewFlagSet("incidents list")
	status := fs.String("status", "", "Filter by status")
	severity := fs.String("severity", "", "Filter by severity")
	incidentType := fs.String("type", "", "Filter by incident type")
	limit := fs.Int("limit", 50, "Maximum incidents")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{"limit": []string{strconv.Itoa(*limit)}}
	if *status != "" {
		query.Set("status", *status)
	}
	if *severity != "" {
		query.Set("severity", *severity)
	}
	if *incidentType != "" {
		query.Set("type", *incidentType)
	}
	path := "/incidents"
	if latest {
		path = "/incidents/latest"
		if query.Get("status") == "" {
			query.Set("status", "open")
		}
	}
	body, err := core.Get(path, query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.IncidentsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Incidents: %d", resp.Total)},
		ResultsHeading: "Incidents",
		Results:        incidentLines(resp.Incidents),
		RetrievalHints: []string{"vrooli-autoheal incidents show <incident-id>", "vrooli-autoheal host inventory"},
	})
}

func show(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("incidents show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal incidents show <incident-id>")
	}
	body, err := core.Get("/incidents/"+url.PathEscape(fs.Arg(0)), nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var incident support.Incident
	if err := support.Decode(body, &incident); err != nil {
		return err
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("%s %s (%s/%s)", incident.ID, incident.Title, incident.Type, incident.Severity),
			fmt.Sprintf("Status: %s | events: %d | observations: %d", incident.Status, incident.EventCount, incident.ObservationCount),
			incident.Summary,
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Source Checks", Items: incident.SourceCheckIDs},
			{Heading: "Recommendations", Items: incident.Recommendations},
		},
		NextSteps: []string{
			fmt.Sprintf("vrooli-autoheal incidents acknowledge %s --note \"reviewing\"", incident.ID),
			fmt.Sprintf("vrooli-autoheal incidents resolve %s --note \"fixed\"", incident.ID),
		},
	})
}

func mutate(core *cliapp.ScenarioApp, action string, args []string) error {
	fs := support.NewFlagSet("incidents " + action)
	note := fs.String("note", "", "Operator note")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal incidents %s <incident-id> [--note ...]", action)
	}
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf(`{"note":%q}`, *note))
	resp, err := core.Request("POST", "/incidents/"+url.PathEscape(fs.Arg(0))+"/"+action, nil, &body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(resp))
		return nil
	}
	var incident support.Incident
	if err := support.Decode(resp, &incident); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("%s is now %s.", incident.ID, incident.Status)},
	})
}

func incidentLines(incidents []support.Incident) []string {
	lines := make([]string, 0, len(incidents))
	for _, incident := range incidents {
		lines = append(lines, fmt.Sprintf("%s %s %s/%s events=%d observations=%d: %s", incident.ID, incident.Status, incident.Type, incident.Severity, incident.EventCount, incident.ObservationCount, incident.Summary))
	}
	if len(lines) == 0 {
		return []string{"No incidents match the selected filters."}
	}
	return lines
}
