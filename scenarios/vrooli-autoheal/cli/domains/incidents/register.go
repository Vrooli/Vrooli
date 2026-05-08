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
			{Name: "remediations", Description: "List remediation candidates for one incident", Run: func(args []string) error { return remediations(core, args) }},
			{Name: "remediation", Description: "Generate or update incident remediation artifacts", Run: func(args []string) error { return remediation(core, args) }},
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
			{Heading: "Remediation Candidates", Items: remediationLines(incident.RemediationCandidates)},
		},
		NextSteps: []string{
			fmt.Sprintf("vrooli-autoheal incidents remediations %s", incident.ID),
			fmt.Sprintf("vrooli-autoheal incidents acknowledge %s --note \"reviewing\"", incident.ID),
			fmt.Sprintf("vrooli-autoheal incidents resolve %s --note \"fixed\"", incident.ID),
		},
	})
}

func remediations(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("incidents remediations")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal incidents remediations <incident-id> [--json]")
	}
	body, err := core.Get("/incidents/"+url.PathEscape(fs.Arg(0))+"/remediations", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.RemediationsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Remediation candidates: %d", resp.Total)},
		ResultsHeading: "Candidates",
		Results:        remediationLines(resp.Remediations),
		RetrievalHints: []string{fmt.Sprintf("vrooli-autoheal incidents remediation generate %s <remediation-id>", resp.IncidentID)},
	})
}

func remediation(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: vrooli-autoheal incidents remediation <generate|outcome> ...")
	}
	switch args[0] {
	case "generate":
		return generateRemediation(core, args[1:])
	case "outcome":
		return recordRemediationOutcome(core, args[1:])
	default:
		return fmt.Errorf("usage: vrooli-autoheal incidents remediation <generate|outcome> ...")
	}
}

func generateRemediation(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("incidents remediation generate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: vrooli-autoheal incidents remediation generate <incident-id> <remediation-id> [--json]")
	}
	path := "/incidents/" + url.PathEscape(fs.Arg(0)) + "/remediations/" + url.PathEscape(fs.Arg(1)) + "/generate"
	body, err := core.Request("POST", path, nil, bytes.NewBufferString(`{}`))
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.RemediationGenerateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	artifactPath := ""
	if pathValue, ok := resp.Artifact["path"].(string); ok {
		artifactPath = pathValue
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{"Generated remediation artifact"},
		Changes: []string{
			fmt.Sprintf("incident=%s", resp.IncidentID),
			fmt.Sprintf("remediation=%s", resp.Candidate.ID),
			fmt.Sprintf("risk=%s", resp.Candidate.RiskLevel),
			fmt.Sprintf("requiresPrivilege=%t", resp.Candidate.RequiresPrivilege),
			fmt.Sprintf("artifactPath=%s", artifactPath),
		},
		NextCommand: []string{
			fmt.Sprintf("sudo %s/remediation.sh", artifactPath),
			fmt.Sprintf("vrooli-autoheal incidents show %s", resp.IncidentID),
		},
	})
}

func recordRemediationOutcome(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("incidents remediation outcome")
	status := fs.String("status", "", "Outcome status: generated, operator_ran, verified, failed, abandoned")
	note := fs.String("note", "", "Operator note")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 || *status == "" {
		return fmt.Errorf("usage: vrooli-autoheal incidents remediation outcome <incident-id> <remediation-id> --status <status> [--note ...] [--json]")
	}
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf(`{"status":%q,"note":%q}`, *status, *note))
	path := "/incidents/" + url.PathEscape(fs.Arg(0)) + "/remediations/" + url.PathEscape(fs.Arg(1)) + "/outcome"
	resp, err := core.Request("POST", path, nil, &body)
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
	outcomeStatus := *status
	if incident.Outcome != nil {
		outcomeStatus = incident.Outcome.Status
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{"Recorded remediation outcome"},
		Changes: []string{
			fmt.Sprintf("incident=%s", incident.ID),
			fmt.Sprintf("remediation=%s", fs.Arg(1)),
			fmt.Sprintf("status=%s", outcomeStatus),
		},
		NextCommand: []string{
			fmt.Sprintf("vrooli-autoheal incidents show %s", incident.ID),
			"vrooli-autoheal incidents latest --json",
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

func remediationLines(candidates []support.RemediationCandidate) []string {
	lines := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		lines = append(lines, fmt.Sprintf("%s %s risk=%s privilege=%t: %s", candidate.ID, candidate.Applicability, candidate.RiskLevel, candidate.RequiresPrivilege, candidate.Title))
	}
	if len(lines) == 0 {
		return []string{"No remediation candidates are available for this incident."}
	}
	return lines
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
