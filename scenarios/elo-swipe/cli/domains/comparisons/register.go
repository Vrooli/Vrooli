package comparisons

import (
	"flag"
	"fmt"
	"os"

	"elo-swipe/cli/internal/client"
	"elo-swipe/cli/internal/render"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "elo-swipe"

func Register(api *client.Client) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "comparisons",
		Description: "Comparison queue and submission operations",
		Subcommands: []cliapp.Command{
			{Name: "next", NeedsAPI: true, Description: "Show the next suggested comparison", Run: func(args []string) error { return runNext(api, args) }},
			{Name: "create", NeedsAPI: true, Description: "Record a comparison result", Run: func(args []string) error { return runCreate(api, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a recorded comparison", Run: func(args []string) error { return runDelete(api, args) }},
		},
	}
}

func runNext(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("comparisons next", flag.ContinueOnError)
	listID := fs.String("list", "", "List ID (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *listID == "" {
		return fmt.Errorf("--list is required")
	}

	next, err := api.NextComparison(*listID)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{"List: " + *listID},
		RetrievalHints: []string{
			cliName + " comparisons create --list " + *listID + " --winner <item-id> --loser <item-id>",
			cliName + " swipe run --list " + *listID,
		},
	}
	if next == nil {
		report.Results = []string{"No more comparisons needed."}
	} else {
		report.Summary = append(report.Summary, fmt.Sprintf("Progress: %d / %d", next.Progress.Completed, next.Progress.Total))
		report.Results = []string{
			"A: " + render.ItemLabel(next.ItemA.Content) + " (" + next.ItemA.ID + ")",
			"B: " + render.ItemLabel(next.ItemB.Content) + " (" + next.ItemB.ID + ")",
		}
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			cliapp.ListReport
			Comparison *client.NextComparison `json:"comparison,omitempty"`
		}{ListReport: report, Comparison: next})
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("comparisons create", flag.ContinueOnError)
	listID := fs.String("list", "", "List ID (required)")
	winnerID := fs.String("winner", "", "Winner item ID (required)")
	loserID := fs.String("loser", "", "Loser item ID (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *listID == "" || *winnerID == "" || *loserID == "" {
		return fmt.Errorf("--list, --winner, and --loser are required")
	}

	resp, err := api.CreateComparison(*listID, *winnerID, *loserID)
	if err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Comparison recorded", "Comparison ID: " + resp.ID},
		Changes: []string{
			"Winner: " + resp.WinnerID,
			"Loser: " + resp.LoserID,
			fmt.Sprintf("Winner rating: %.2f -> %.2f", resp.WinnerRatingBefore, resp.WinnerRatingAfter),
			fmt.Sprintf("Loser rating: %.2f -> %.2f", resp.LoserRatingBefore, resp.LoserRatingAfter),
		},
		NextCommand: []string{cliName + " comparisons next --list " + *listID, cliName + " rankings show --list " + *listID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			cliapp.MutationReport
			Comparison *client.Comparison `json:"comparison"`
		}{MutationReport: report, Comparison: resp})
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("comparisons delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: comparisons delete <comparison-id> [--json]")
	}
	id := fs.Arg(0)
	if err := api.DeleteComparison(id); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Comparison deleted", "Comparison ID: " + id},
		Changes:     []string{"Comparison ratings were reverted in the ranking history."},
		NextCommand: []string{cliName + " comparisons next --list <list-id>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
