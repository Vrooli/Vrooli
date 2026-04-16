package scenarios

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"maintenance-orchestrator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `scenario` subcommand group for interacting with the
// orchestrator's view of maintenance scenarios. Each subcommand is a thin
// wrapper over a single API endpoint; complex orchestration stays server-side.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenario",
		Description: "Manage maintenance scenarios tracked by the orchestrator",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List discovered maintenance scenarios", Run: func(args []string) error { return runList(core, args) }},
			{Name: "list-all", Description: "List all scenarios on the host (via vrooli CLI)", Run: func(args []string) error { return runListAll(core, args) }},
			{Name: "statuses", Description: "Show runtime status for all scenarios", Run: func(args []string) error { return runStatuses(core, args) }},
			{Name: "activate", Description: "Activate a maintenance scenario", Run: func(args []string) error { return runStateChange(core, args, "activate", "Activate") }},
			{Name: "deactivate", Description: "Deactivate a maintenance scenario", Run: func(args []string) error { return runStateChange(core, args, "deactivate", "Deactivate") }},
			{Name: "start", Description: "Start a scenario process via vrooli CLI", Run: func(args []string) error { return runStateChange(core, args, "start", "Start") }},
			{Name: "stop", Description: "Stop a scenario process via vrooli CLI", Run: func(args []string) error { return runStateChange(core, args, "stop", "Stop") }},
			{Name: "add-tag", Description: "Add the maintenance tag to a scenario", Run: func(args []string) error { return runTag(core, args, "add-tag", "Added maintenance tag to") }},
			{Name: "remove-tag", Description: "Remove the maintenance tag from a scenario", Run: func(args []string) error { return runTag(core, args, "remove-tag", "Removed maintenance tag from") }},
			{Name: "port", Description: "Resolve a port for a scenario", Run: func(args []string) error { return runPort(core, args) }},
			{Name: "preset-assignments", Description: "Show preset assignments for a scenario", Run: func(args []string) error { return runGetAssignments(core, args) }},
			{Name: "update-preset-assignments", Description: "Update preset assignments for a scenario", Run: func(args []string) error { return runUpdateAssignments(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenario list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/scenarios", nil)
	if err != nil {
		return err
	}
	var resp support.ScenariosResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	active := 0
	for _, s := range resp.Scenarios {
		if s.IsActive {
			active++
		}
	}

	rows := make([]string, 0, len(resp.Scenarios))
	for _, s := range resp.Scenarios {
		state := "INACTIVE"
		if s.IsActive {
			state = "ACTIVE"
		}
		display := s.DisplayName
		if display == "" {
			display = s.Name
		}
		row := fmt.Sprintf("[%s] %s - %s", state, s.Name, display)
		if len(s.Tags) > 0 {
			row += fmt.Sprintf(" | tags=%s", strings.Join(s.Tags, ","))
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = []string{"(no scenarios discovered)"}
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios: %d total, %d active", len(resp.Scenarios), active),
		},
		ResultsHeading: "Scenarios",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s scenario activate <id>", support.CLIName),
			fmt.Sprintf("%s scenario statuses", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runListAll(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenario list-all")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/all-scenarios", nil)
	if err != nil {
		return err
	}
	var resp support.AllScenariosResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Scenarios))
	for _, s := range resp.Scenarios {
		display := s.DisplayName
		if display == "" {
			display = s.Name
		}
		rows = append(rows, fmt.Sprintf("%s - %s", s.Name, display))
	}
	if len(rows) == 0 {
		rows = []string{"(no scenarios on host)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenarios on host: %d", len(resp.Scenarios))},
		ResultsHeading: "All scenarios",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s scenario list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStatuses(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenario statuses")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/scenario-statuses", nil)
	if err != nil {
		return err
	}
	var resp support.ScenarioStatusesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	names := make([]string, 0, len(resp.Statuses))
	for name := range resp.Statuses {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]string, 0, len(names))
	for _, name := range names {
		st := resp.Statuses[name]
		rows = append(rows, fmt.Sprintf("%s | status=%s | processes=%d", name, st.Status, st.ProcessCount))
	}
	if len(rows) == 0 {
		rows = []string{"(no runtime statuses available)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Runtime statuses: %d scenarios", len(resp.Statuses))},
		ResultsHeading: "Statuses",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s scenario list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStateChange(core *cliapp.ScenarioApp, args []string, verb, display string) error {
	fs := support.NewFlagSet("scenario " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenario %s <scenario-id>", verb)
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/scenarios/"+id+"/"+verb, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("%s issued for %s", display, id)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Scenario %s: %s", id, verb)},
		NextCommand: []string{
			fmt.Sprintf("%s scenario list", support.CLIName),
			fmt.Sprintf("%s orchestrator overview", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runTag(core *cliapp.ScenarioApp, args []string, endpoint, display string) error {
	fs := support.NewFlagSet("scenario " + endpoint)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenario %s <scenario-name>", endpoint)
	}
	name := fs.Arg(0)

	body, err := core.Request("POST", "/scenarios/"+name+"/"+endpoint, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("%s %s", display, name)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Scenario %s: %s", name, endpoint)},
		NextCommand: []string{fmt.Sprintf("%s scenario list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runPort(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenario port")
	portType := fs.String("type", "", "Port type (default: UI_PORT)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenario port <scenario-name> [--type PORT_TYPE]")
	}
	name := fs.Arg(0)

	query := support.BuildQuery(map[string]string{
		"type": *portType,
	})
	body, err := core.Get("/scenarios/"+name+"/port", query)
	if err != nil {
		return err
	}
	var resp support.PortResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Scenario: %s", name),
	}
	if resp.Type != "" {
		results = append(results, fmt.Sprintf("Type: %s", resp.Type))
	}
	if len(resp.Port) > 0 {
		results = append(results, fmt.Sprintf("Port: %s", strings.TrimSpace(string(resp.Port))))
	} else {
		results = append(results, "Port: (unavailable)")
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Port resolution for %s", name)},
		ResultsHeading: "Port",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGetAssignments(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenario preset-assignments")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenario preset-assignments <scenario-name>")
	}
	name := fs.Arg(0)

	body, err := core.Get("/scenarios/"+name+"/preset-assignments", nil)
	if err != nil {
		return err
	}
	var resp struct {
		Assignments map[string]bool `json:"assignments"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	keys := make([]string, 0, len(resp.Assignments))
	for k := range resp.Assignments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%s: %t", k, resp.Assignments[k]))
	}
	if len(rows) == 0 {
		rows = []string{"(no preset assignments)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Preset assignments for %s: %d", name, len(resp.Assignments))},
		ResultsHeading: "Assignments",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s scenario update-preset-assignments %s --body-file ./assignments.json", support.CLIName, name),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdateAssignments(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenario update-preset-assignments")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing {\"assignments\": {<preset-id>: <bool>, ...}}")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenario update-preset-assignments <scenario-name> --body-file PATH")
	}
	name := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/scenarios/"+name+"/preset-assignments", nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Preset assignments updated for %s", name)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Scenario %s: preset-assignments updated", name)},
		NextCommand: []string{
			fmt.Sprintf("%s scenario preset-assignments %s", support.CLIName, name),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
