package milestones

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "milestones",
		Description: "List and manage strategic milestones",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List milestones", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "create", Description: "Create a milestone", Run: func(args []string) error { return runCreate(deps, args) }},
			{Name: "update", Description: "Update a milestone", Run: func(args []string) error { return runUpdate(deps, args) }},
			{Name: "delete", Description: "Delete a milestone", Run: func(args []string) error { return runDelete(deps, args) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("milestones list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := deps.Get("/milestones", nil)
	if err != nil {
		return err
	}
	var response support.MilestonesResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Milestones returned: %d", len(response.Milestones)),
		},
		ResultsHeading: "Milestones",
		Results:        milestoneRows(response.Milestones),
		RetrievalHints: []string{
			"tech-tree-designer milestones create --name \"...\" --type strategic",
			"tech-tree-designer analyze --resources 8 --timeline 12",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(deps support.Dependencies, args []string) error {
	return runUpsert(deps, args, "")
}

func runUpdate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("milestones update")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: milestones update <milestone-id> [flags]")
	}
	return runUpsert(deps, args[1:], fs.Arg(0))
}

func runUpsert(deps support.Dependencies, args []string, milestoneID string) error {
	commandName := "milestones create"
	if strings.TrimSpace(milestoneID) != "" {
		commandName = "milestones update"
	}
	fs := support.NewFlagSet(commandName)
	name := fs.String("name", "", "Milestone name")
	description := fs.String("description", "", "Description")
	milestoneType := fs.String("type", "", "Milestone type")
	completion := fs.Float64("completion", -1, "Completion percentage")
	businessValue := fs.Int64("value", -1, "Business value estimate")
	confidence := fs.Float64("confidence", -1, "Confidence level 0-1")
	estimatedDate := fs.String("date", "", "Estimated completion date (YYYY-MM-DD)")
	targetSectors := fs.String("sector-ids", "", "Comma-separated sector IDs")
	targetStages := fs.String("stage-ids", "", "Comma-separated stage IDs")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*milestoneType) == "" {
		return fmt.Errorf("name and type are required")
	}

	payload := map[string]interface{}{
		"name":              *name,
		"description":       *description,
		"milestone_type":    *milestoneType,
		"target_sector_ids": support.TrimmedCSV(*targetSectors),
		"target_stage_ids":  support.TrimmedCSV(*targetStages),
	}
	if *completion >= 0 {
		payload["completion_percentage"] = *completion
	}
	if *businessValue >= 0 {
		payload["business_value_estimate"] = *businessValue
	}
	if *confidence >= 0 {
		payload["confidence_level"] = *confidence
	}
	if strings.TrimSpace(*estimatedDate) != "" {
		payload["estimated_completion_date"] = *estimatedDate
	}

	method := "POST"
	path := "/milestones"
	if strings.TrimSpace(milestoneID) != "" {
		method = "PATCH"
		path = "/milestones/" + milestoneID
	}
	body, err := deps.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	var response support.MilestoneMutationResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Saved milestone %s.", response.Milestone.Name)},
		Changes: []string{
			fmt.Sprintf("Type: %s", response.Milestone.MilestoneType),
			fmt.Sprintf("Completion: %s", support.FormatPercent(response.Milestone.CompletionPercentage)),
			fmt.Sprintf("Value: %d", response.Milestone.BusinessValueEstimate),
		},
		NextCommand: []string{
			"tech-tree-designer milestones list",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("milestones delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: milestones delete <milestone-id>")
	}
	if _, err := deps.Request("DELETE", "/milestones/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted milestone %s.", fs.Arg(0))},
		Changes:     []string{"The milestone no longer contributes to strategic timeline projections."},
		NextCommand: []string{"tech-tree-designer milestones list"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, map[string]string{"deleted_id": fs.Arg(0)})
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func milestoneRows(items []support.StrategicMilestone) []string {
	if len(items) == 0 {
		return []string{"No milestones found."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s | %s | completion=%s | value=%s | target=%s", item.Name, item.MilestoneType, support.FormatPercent(item.CompletionPercentage), strconv.FormatInt(item.BusinessValueEstimate, 10), support.FormatDate(item.EstimatedCompletionDate)))
	}
	return rows
}
