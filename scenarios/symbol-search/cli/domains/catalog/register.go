package catalog

import (
	"fmt"
	"os"

	"symbol-search/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `catalog` subcommand group. It exposes the read-only
// metadata endpoints that describe what kinds of characters exist in the
// dataset: Unicode categories and Unicode blocks.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "catalog",
		Description: "Inspect the Unicode catalog (categories and blocks)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "categories", Aliases: []string{"cats"}, Description: "List Unicode categories with character counts", Run: func(args []string) error { return runCategories(core, args) }},
			{Name: "blocks", Description: "List Unicode blocks with character ranges and counts", Run: func(args []string) error { return runBlocks(core, args) }},
		},
	}
}

func runCategories(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("catalog categories")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/categories", nil)
	if err != nil {
		return err
	}
	var resp support.CategoriesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Unicode categories: %d", len(resp.Categories))},
		ResultsHeading: "Categories",
		Results:        categoryRows(resp.Categories),
		RetrievalHints: []string{
			fmt.Sprintf("%s characters search <query> --category <code>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runBlocks(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("catalog blocks")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/blocks", nil)
	if err != nil {
		return err
	}
	var resp support.BlocksResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Unicode blocks: %d", len(resp.Blocks))},
		ResultsHeading: "Blocks",
		Results:        blockRows(resp.Blocks),
		RetrievalHints: []string{
			fmt.Sprintf("%s characters range <start> <end>", support.CLIName),
			fmt.Sprintf("%s characters search <query> --block <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func categoryRows(cats []support.Category) []string {
	if len(cats) == 0 {
		return []string{"(no categories)"}
	}
	rows := make([]string, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, fmt.Sprintf("%s | %s | count=%d", c.Code, c.Name, c.CharacterCount))
	}
	return rows
}

func blockRows(blocks []support.CharacterBlock) []string {
	if len(blocks) == 0 {
		return []string{"(no blocks)"}
	}
	rows := make([]string, 0, len(blocks))
	for _, b := range blocks {
		rows = append(rows, fmt.Sprintf("%s | %d..%d | count=%d",
			b.Name, b.StartCodepoint, b.EndCodepoint, b.CharacterCount))
	}
	return rows
}
