package sectors

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
		Name:        "sectors",
		Description: "List and manage technology sectors",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List sectors", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "get", Description: "Get one sector", Run: func(args []string) error { return runGet(deps, args) }},
			{Name: "create", Description: "Create a sector", Run: func(args []string) error { return runCreate(deps, args) }},
			{Name: "update", Description: "Update a sector", Run: func(args []string) error { return runUpdate(deps, args) }},
			{Name: "delete", Description: "Delete a sector", Run: func(args []string) error { return runDelete(deps, args) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("sectors list")
	category := fs.String("category", "", "Filter by category")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := deps.Get("/tech-tree/sectors", nil)
	if err != nil {
		return err
	}
	var response support.SectorListResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	filtered := make([]support.Sector, 0, len(response.Sectors))
	for _, sector := range response.Sectors {
		if strings.TrimSpace(*category) != "" && !strings.EqualFold(sector.Category, *category) {
			continue
		}
		filtered = append(filtered, sector)
	}

	report := cliapp.ListReport{
		Summary: []string{
			support.TreeScopeLine(deps.Selector),
			fmt.Sprintf("Sectors returned: %d", len(filtered)),
		},
		ResultsHeading: "Sectors",
		Results:        sectorRows(filtered),
		RetrievalHints: []string{
			"tech-tree-designer sectors get <sector-id>",
			"tech-tree-designer stages create --sector-id <sector-id> --name \"...\"",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("sectors get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sectors get <sector-id>")
	}

	body, err := deps.Get("/tech-tree/sectors/"+fs.Arg(0), nil)
	if err != nil {
		return err
	}
	var response support.SectorResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Sector: %s", response.Sector.Name),
			fmt.Sprintf("Category: %s", response.Sector.Category),
			fmt.Sprintf("Progress: %s", support.FormatPercent(response.Sector.ProgressPercentage)),
		},
		ResultsHeading: "Stages",
		Results:        stageRows(response.Sector.Stages),
		RetrievalHints: []string{
			fmt.Sprintf("tech-tree-designer stages create --sector-id %s --name \"...\"", response.Sector.ID),
			fmt.Sprintf("tech-tree-designer sectors update %s --progress %s", response.Sector.ID, support.FormatRatio(response.Sector.ProgressPercentage)),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("sectors create")
	name := fs.String("name", "", "Sector name")
	category := fs.String("category", "", "Sector category")
	description := fs.String("description", "", "Sector description")
	color := fs.String("color", "", "Sector color")
	positionX := fs.Float64("x", 0, "X position")
	positionY := fs.Float64("y", 0, "Y position")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("name is required")
	}

	body, err := deps.Request("POST", "/tech-tree/sectors", nil, map[string]interface{}{
		"name":        *name,
		"category":    *category,
		"description": *description,
		"color":       *color,
		"position_x":  *positionX,
		"position_y":  *positionY,
	})
	if err != nil {
		return err
	}
	var response support.SectorResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Created sector %s (%s).", response.Sector.Name, response.Sector.ID),
		},
		Changes: []string{
			fmt.Sprintf("Category: %s", response.Sector.Category),
			fmt.Sprintf("Color: %s", response.Sector.Color),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer stages create --sector-id %s --name \"...\"", response.Sector.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("sectors update")
	name := fs.String("name", "", "Sector name")
	category := fs.String("category", "", "Sector category")
	description := fs.String("description", "", "Sector description")
	progress := fs.Float64("progress", -1, "Progress percentage")
	color := fs.String("color", "", "Sector color")
	positionX := fs.Float64("x", 0, "X position")
	positionY := fs.Float64("y", 0, "Y position")
	setPosition := fs.Bool("set-position", false, "Persist x/y position fields")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sectors update <sector-id> [flags]")
	}

	payload := map[string]interface{}{}
	if strings.TrimSpace(*name) != "" {
		payload["name"] = *name
	}
	if strings.TrimSpace(*category) != "" {
		payload["category"] = *category
	}
	if strings.TrimSpace(*description) != "" {
		payload["description"] = *description
	}
	if *progress >= 0 {
		payload["progress_percentage"] = *progress
	}
	if strings.TrimSpace(*color) != "" {
		payload["color"] = *color
	}
	if *setPosition {
		payload["position_x"] = *positionX
		payload["position_y"] = *positionY
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one field must be updated")
	}

	body, err := deps.Request("PATCH", "/tech-tree/sectors/"+fs.Arg(0), nil, payload)
	if err != nil {
		return err
	}
	var response support.SectorResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Updated sector %s.", response.Sector.Name)},
		Changes: []string{
			fmt.Sprintf("Progress: %s", support.FormatPercent(response.Sector.ProgressPercentage)),
			fmt.Sprintf("Category: %s", response.Sector.Category),
		},
		NextCommand: []string{
			fmt.Sprintf("tech-tree-designer sectors get %s", response.Sector.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("sectors delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sectors delete <sector-id>")
	}
	if _, err := deps.Request("DELETE", "/tech-tree/sectors/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Deleted the sector."},
		Changes:     []string{"The sector and any API-visible references were removed from the selected tree."},
		NextCommand: []string{"tech-tree-designer sectors list"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func sectorRows(items []support.Sector) []string {
	if len(items) == 0 {
		return []string{"No sectors found."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s | %s | progress %s | %d stages", item.Name, item.Category, support.FormatPercent(item.ProgressPercentage), len(item.Stages)))
	}
	return rows
}

func stageRows(items []support.Stage) []string {
	if len(items) == 0 {
		return []string{"No stages attached to this sector."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s | type=%s | maturity=%s | progress %s", item.Name, item.StageType, item.Maturity, support.FormatPercent(item.ProgressPercentage)))
	}
	return rows
}
