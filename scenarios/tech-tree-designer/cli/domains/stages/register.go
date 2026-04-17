package stages

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "stages",
		Description: "Inspect and manage progression stages",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get one stage", Run: func(args []string) error { return runGet(deps, args) }},
			{Name: "children", Description: "List direct child stages", Run: func(args []string) error { return runChildren(deps, args) }},
			{Name: "create", Description: "Create a stage", Run: func(args []string) error { return runCreate(deps, args) }},
			{Name: "update", Description: "Update a stage", Run: func(args []string) error { return runUpdate(deps, args) }},
			{Name: "delete", Description: "Delete a stage", Run: func(args []string) error { return runDelete(deps, args) }},
			{Name: "set-maturity", Description: "Update stage maturity", Run: func(args []string) error { return runSetMaturity(deps, args) }},
		},
	}
}

func runGet(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("stages get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stages get <stage-id>")
	}
	body, err := deps.Get("/tech-tree/stages/"+fs.Arg(0), nil)
	if err != nil {
		return err
	}
	var response support.Stage
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Stage: %s", response.Name),
			fmt.Sprintf("Type: %s", response.StageType),
			fmt.Sprintf("Maturity: %s", response.Maturity),
			fmt.Sprintf("Progress: %s", support.FormatPercent(response.ProgressPercentage)),
		},
		ResultsHeading: "Scenario mappings",
		Results:        mappingRows(response.ScenarioMappings),
		RetrievalHints: []string{
			fmt.Sprintf("tech-tree-designer stages children %s", response.ID),
			fmt.Sprintf("tech-tree-designer progress link --stage-id %s --scenario <scenario>", response.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runChildren(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("stages children")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stages children <stage-id>")
	}
	body, err := deps.Get("/tech-tree/stages/"+fs.Arg(0)+"/children", nil)
	if err != nil {
		return err
	}
	var response support.StageChildrenResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Children returned: %d", response.Count),
			support.TreeScopeLine(deps.Selector),
		},
		ResultsHeading: "Child stages",
		Results:        childRows(response.Children),
		RetrievalHints: []string{
			"tech-tree-designer stages get <child-stage-id>",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("stages create")
	sectorID := fs.String("sector-id", "", "Sector ID")
	parentStageID := fs.String("parent-stage-id", "", "Parent stage ID")
	stageType := fs.String("type", "", "Stage type")
	stageOrder := fs.Int("order", 0, "Stage ordering")
	name := fs.String("name", "", "Stage name")
	description := fs.String("description", "", "Stage description")
	progress := fs.Float64("progress", 0, "Progress percentage")
	maturity := fs.String("maturity", "", "Ignored at create time; use set-maturity after creation if needed")
	examples := fs.String("examples", "", "Comma-separated examples")
	positionX := fs.Float64("x", 0, "X position")
	positionY := fs.Float64("y", 0, "Y position")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	_ = maturity
	if strings.TrimSpace(*sectorID) == "" || strings.TrimSpace(*name) == "" {
		return fmt.Errorf("sector-id and name are required")
	}

	payload := map[string]interface{}{
		"sector_id":           *sectorID,
		"stage_type":          *stageType,
		"stage_order":         *stageOrder,
		"name":                *name,
		"description":         *description,
		"progress_percentage": *progress,
		"position_x":          *positionX,
		"position_y":          *positionY,
		"examples":            support.TrimmedCSV(*examples),
	}
	if strings.TrimSpace(*parentStageID) != "" {
		payload["parent_stage_id"] = *parentStageID
	}

	body, err := deps.Request("POST", "/tech-tree/stages", nil, payload)
	if err != nil {
		return err
	}
	var response support.StageResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Created stage %s (%s).", response.Stage.Name, response.Stage.ID)},
		Changes: []string{
			fmt.Sprintf("Type: %s", response.Stage.StageType),
			fmt.Sprintf("Order: %d", response.Stage.StageOrder),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer stages get %s", response.Stage.ID),
			fmt.Sprintf("tech-tree-designer stages set-maturity %s --maturity building", response.Stage.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("stages update")
	sectorID := fs.String("sector-id", "", "Sector ID")
	stageType := fs.String("type", "", "Stage type")
	stageOrder := fs.Int("order", -1, "Stage ordering")
	name := fs.String("name", "", "Stage name")
	description := fs.String("description", "", "Stage description")
	progress := fs.Float64("progress", -1, "Progress percentage")
	examples := fs.String("examples", "", "Comma-separated examples")
	positionX := fs.Float64("x", 0, "X position")
	positionY := fs.Float64("y", 0, "Y position")
	setPosition := fs.Bool("set-position", false, "Persist x/y position fields")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stages update <stage-id> [flags]")
	}

	payload := map[string]interface{}{}
	if strings.TrimSpace(*sectorID) != "" {
		payload["sector_id"] = *sectorID
	}
	if strings.TrimSpace(*stageType) != "" {
		payload["stage_type"] = *stageType
	}
	if *stageOrder >= 0 {
		payload["stage_order"] = *stageOrder
	}
	if strings.TrimSpace(*name) != "" {
		payload["name"] = *name
	}
	if strings.TrimSpace(*description) != "" {
		payload["description"] = *description
	}
	if *progress >= 0 {
		payload["progress_percentage"] = *progress
	}
	if strings.TrimSpace(*examples) != "" {
		payload["examples"] = support.TrimmedCSV(*examples)
	}
	if *setPosition {
		payload["position_x"] = *positionX
		payload["position_y"] = *positionY
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one field must be updated")
	}

	body, err := deps.Request("PATCH", "/tech-tree/stages/"+fs.Arg(0), nil, payload)
	if err != nil {
		return err
	}
	var response support.StageResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Updated stage %s.", response.Stage.Name)},
		Changes: []string{
			fmt.Sprintf("Type: %s", response.Stage.StageType),
			fmt.Sprintf("Progress: %s", support.FormatPercent(response.Stage.ProgressPercentage)),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer stages get %s", response.Stage.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("stages delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stages delete <stage-id>")
	}
	if _, err := deps.Request("DELETE", "/tech-tree/stages/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Deleted the stage."},
		Changes:     []string{"Any attached mappings now need review if they depended on this stage."},
		NextCommand: []string{"tech-tree-designer progress list"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSetMaturity(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("stages set-maturity")
	maturity := fs.String("maturity", "", "planned|building|live|scaled")
	changedBy := fs.String("changed-by", "", "Audit actor")
	notes := fs.String("notes", "", "Change notes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || strings.TrimSpace(*maturity) == "" {
		return fmt.Errorf("usage: stages set-maturity <stage-id> --maturity planned|building|live|scaled")
	}
	body, err := deps.Request("PUT", "/stages/"+fs.Arg(0)+"/maturity", nil, map[string]interface{}{
		"maturity":   *maturity,
		"changed_by": *changedBy,
		"notes":      *notes,
	})
	if err != nil {
		return err
	}
	var response map[string]interface{}
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Updated stage maturity to %s.", *maturity)},
		Changes: []string{
			fmt.Sprintf("Stage ID: %s", fs.Arg(0)),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer stages get %s", fs.Arg(0)),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func mappingRows(items []support.ScenarioMap) []string {
	if len(items) == 0 {
		return []string{"No linked scenarios."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s | status=%s | impact=%.2f | priority=%d", item.ScenarioName, item.CompletionStatus, item.EstimatedImpact, item.Priority))
	}
	return rows
}

func childRows(items []support.Stage) []string {
	if len(items) == 0 {
		return []string{"No child stages found."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		examples := ""
		if len(item.Examples) > 0 {
			var decoded []string
			if err := json.Unmarshal(item.Examples, &decoded); err == nil && len(decoded) > 0 {
				examples = " | examples=" + strings.Join(decoded, ", ")
			}
		}
		rows = append(rows, fmt.Sprintf("%s | %s | maturity=%s%s", item.Name, item.StageType, item.Maturity, examples))
	}
	return rows
}
