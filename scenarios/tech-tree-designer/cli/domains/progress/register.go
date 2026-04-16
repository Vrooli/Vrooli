package progress

import (
	"fmt"
	"os"
	"strings"

	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "progress",
		Description: "Track scenario-to-stage progress mappings",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List scenario mappings", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "link", Description: "Create or update a scenario mapping", Run: func(args []string) error { return runLink(deps, args) }},
			{Name: "set-status", Description: "Update scenario status for matching mappings", Run: func(args []string) error { return runSetStatus(deps, args) }},
			{Name: "unlink", Description: "Delete a scenario mapping", Run: func(args []string) error { return runUnlink(deps, args) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("progress list")
	scenario := fs.String("scenario", "", "Filter by scenario name")
	stageID := fs.String("stage-id", "", "Filter by stage ID")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := support.BuildQuery(map[string]string{
		"scenario": *scenario,
		"stage_id": *stageID,
	})
	body, err := deps.Get("/progress/scenarios", query)
	if err != nil {
		return err
	}
	var response support.ScenarioMappingsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Mappings returned: %d", len(response.ScenarioMappings)),
		},
		ResultsHeading: "Mappings",
		Results:        mappingRows(response.ScenarioMappings),
		RetrievalHints: []string{
			"tech-tree-designer progress link --stage-id <stage-id> --scenario <scenario>",
			"tech-tree-designer progress set-status --scenario <scenario> --status completed",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runLink(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("progress link")
	stageID := fs.String("stage-id", "", "Stage ID")
	scenario := fs.String("scenario", "", "Scenario name")
	status := fs.String("status", "not_started", "Scenario completion status")
	weight := fs.Float64("weight", 1, "Contribution weight")
	priority := fs.Int("priority", 1, "Priority rank")
	impact := fs.Float64("impact", 0, "Estimated impact")
	notes := fs.String("notes", "", "Notes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*stageID) == "" || strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("stage-id and scenario are required")
	}
	body, err := deps.Request("POST", "/progress/scenarios", nil, map[string]interface{}{
		"stage_id":            *stageID,
		"scenario_name":       *scenario,
		"completion_status":   *status,
		"contribution_weight": *weight,
		"priority":            *priority,
		"estimated_impact":    *impact,
		"notes":               *notes,
	})
	if err != nil {
		return err
	}
	var response support.ScenarioMappingMutationResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Linked %s to stage %s.", response.Mapping.ScenarioName, response.Mapping.StageID)},
		Changes: []string{
			fmt.Sprintf("Status: %s", response.Mapping.CompletionStatus),
			fmt.Sprintf("Priority: %d | impact %.2f", response.Mapping.Priority, response.Mapping.EstimatedImpact),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer progress list --scenario %s", response.Mapping.ScenarioName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSetStatus(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("progress set-status")
	scenario := fs.String("scenario", "", "Scenario name")
	status := fs.String("status", "", "New status")
	impact := fs.Float64("impact", 0, "Estimated impact override")
	notes := fs.String("notes", "Updated via tech-tree-designer CLI", "Notes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" || strings.TrimSpace(*status) == "" {
		return fmt.Errorf("scenario and status are required")
	}
	body, err := deps.Request("PUT", "/progress/scenarios/"+*scenario, nil, map[string]interface{}{
		"completion_status": *status,
		"estimated_impact":  *impact,
		"notes":             *notes,
	})
	if err != nil {
		return err
	}
	var response support.ScenarioStatusMutationResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Updated %s to %s.", response.Scenario, response.Status)},
		Changes: []string{
			fmt.Sprintf("Tree: %s", response.Tree.Name),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer progress list --scenario %s", response.Scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUnlink(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("progress unlink")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: progress unlink <mapping-id>")
	}
	body, err := deps.Request("DELETE", "/progress/scenarios/"+fs.Arg(0), nil, nil)
	if err != nil {
		return err
	}
	var response support.ScenarioStatusMutationResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Deleted mapping %s.", response.ID)},
		Changes: []string{
			fmt.Sprintf("Tree: %s", response.Tree.Name),
		},
		NextCommand: []string{
			"tech-tree-designer progress list",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func mappingRows(items []support.ScenarioMappingEntry) []string {
	if len(items) == 0 {
		return []string{"No scenario mappings found."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s -> %s / %s | status=%s | impact=%.2f | priority=%d", item.Mapping.ScenarioName, item.SectorName, item.StageName, item.Mapping.CompletionStatus, item.Mapping.EstimatedImpact, item.Mapping.Priority))
	}
	return rows
}
