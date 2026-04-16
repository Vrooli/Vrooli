package assignments

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "brand-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Assignments",
		Commands: []cliapp.Command{
			{Name: "assign", NeedsAPI: true, Description: "Assign a brand to a scenario", Run: func(args []string) error { return runAssign(core, args) }},
			{Name: "unassign", NeedsAPI: true, Description: "Remove a brand assignment by ID", Run: func(args []string) error { return runUnassign(core, args) }},
			{Name: "scenario-status", NeedsAPI: true, Description: "Check branding status for a scenario", Run: func(args []string) error { return runScenarioStatus(core, args) }},
		},
	}
}

func runAssign(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("assign", flag.ContinueOnError)
	brandID := fs.String("brand", "", "Brand ID to assign (required)")
	scenario := fs.String("scenario", "", "Scenario name to assign to (required)")
	elements := fs.String("elements", "", "Comma-separated elements to apply (e.g. colors,typography)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *brandID == "" || *scenario == "" {
		return fmt.Errorf("--brand and --scenario are required\nUsage: brand-manager assign --brand ID --scenario NAME [--elements colors,typography] [--json]")
	}

	payload := map[string]interface{}{
		"brand_id":      *brandID,
		"scenario_name": *scenario,
	}
	if *elements != "" {
		payload["elements"] = cliutil.ParseCSV(*elements)
	}
	body, err := core.Request("POST", "/assignments", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Brand assigned", fmt.Sprintf("Assignment ID: %v", result["id"])},
		Changes: []string{
			"Brand ID: " + *brandID,
			"Scenario: " + *scenario,
		},
		NextCommand: []string{cliName + " scenario-status " + *scenario, cliName + " unassign " + fmt.Sprintf("%v", result["id"])},
	}
	if *elements != "" {
		report.Changes = append(report.Changes, "Elements: "+*elements)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUnassign(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("unassign", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager unassign <assignment-id> [--json]")
	}

	id := fs.Arg(0)
	if _, err := core.Request("DELETE", "/assignments/"+id, nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Assignment removed", "Assignment ID: " + id},
		Changes:     []string{"Brand-to-scenario linkage removed"},
		NextCommand: []string{cliName + " list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runScenarioStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("scenario-status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: brand-manager scenario-status <scenario-name> [--json]")
	}

	name := fs.Arg(0)
	body, err := core.Get("/scenarios/"+name+"/status", nil)
	if err != nil {
		return err
	}
	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %v", status["scenario"]),
		},
	}
	if hasBrand, ok := status["has_brand"].(bool); ok && hasBrand {
		report.Status = append(report.Status, fmt.Sprintf("Brand ID: %v", status["brand_id"]), fmt.Sprintf("Brand version: %v", status["brand_version"]))
		triage := cliapp.TriageGroup{Heading: "Assignment"}
		if elems, ok := status["elements"].([]interface{}); ok && len(elems) > 0 {
			items := make([]string, 0, len(elems))
			for _, elem := range elems {
				items = append(items, fmt.Sprintf("Element: %v", elem))
			}
			triage.Items = append(triage.Items, items...)
		}
		if applied, ok := status["applied_at"].(string); ok && applied != "" {
			triage.Items = append(triage.Items, "Applied at: "+applied)
		}
		report.Triage = []cliapp.TriageGroup{triage}
		report.NextSteps = []string{cliName + " apply " + fmt.Sprintf("%v", status["brand_id"]) + " --scenario " + name}
	} else {
		report.Triage = []cliapp.TriageGroup{{Heading: "Assignment", Items: []string{"No brand assigned"}}}
		report.NextSteps = []string{cliName + " assign --brand <brand-id> --scenario " + name}
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
