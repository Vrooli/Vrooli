package lists

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"elo-swipe/cli/internal/client"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "elo-swipe"

func Register(api *client.Client) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "lists",
		Description: "List lifecycle operations",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List ranking lists", Run: func(args []string) error { return runList(api, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get one ranking list", Run: func(args []string) error { return runGet(api, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a ranking list from a JSON file", Run: func(args []string) error { return runCreate(api, args) }},
		},
	}
}

func runList(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("lists list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	lists, err := api.ListLists()
	if err != nil {
		return err
	}

	results := make([]string, 0, len(lists))
	for _, item := range lists {
		results = append(results, fmt.Sprintf("%s | %s | %d items | created %s", item.ID, item.Name, item.ItemCount, item.CreatedAt))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Lists: %d", len(lists))},
		Results:        results,
		RetrievalHints: []string{cliName + " lists get <list-id>", cliName + " lists create --name \"Product priorities\" --items-file items.json"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			cliapp.ListReport
			Lists []client.List `json:"lists"`
		}{ListReport: report, Lists: lists})
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("lists get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lists get <list-id> [--json]")
	}

	list, err := api.GetList(fs.Arg(0))
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"List: " + list.Name, "ID: " + list.ID},
		ResultsHeading: "Details",
		Results: []string{
			"Description: " + list.Description,
			fmt.Sprintf("Items: %d", list.ItemCount),
			"Created: " + list.CreatedAt,
			"Updated: " + list.UpdatedAt,
		},
		RetrievalHints: []string{cliName + " rankings show --list " + list.ID, cliName + " comparisons next --list " + list.ID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			cliapp.ListReport
			List *client.List `json:"list"`
		}{ListReport: report, List: list})
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("lists create", flag.ContinueOnError)
	name := fs.String("name", "", "List name (required)")
	description := fs.String("description", "", "List description")
	itemsFile := fs.String("items-file", "", "Path to JSON array of items (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	if *itemsFile == "" {
		return fmt.Errorf("--items-file is required")
	}

	data, err := os.ReadFile(*itemsFile)
	if err != nil {
		return fmt.Errorf("read items file: %w", err)
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return fmt.Errorf("parse items file: %w", err)
	}

	resp, err := api.CreateList(*name, *description, rawItems)
	if err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"List created", "List ID: " + resp.ListID},
		Changes: []string{
			fmt.Sprintf("Name: %s", *name),
			fmt.Sprintf("Items imported: %d", resp.ItemCount),
		},
		NextCommand: []string{cliName + " swipe run --list " + resp.ListID, cliName + " rankings show --list " + resp.ListID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			cliapp.MutationReport
			Response *client.CreateListResponse `json:"response"`
		}{MutationReport: report, Response: resp})
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
