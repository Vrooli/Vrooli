package rankings

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"elo-swipe/cli/internal/client"
	"elo-swipe/cli/internal/render"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "elo-swipe"

func Register(api *client.Client) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "rankings",
		Description: "Ranking read operations",
		Subcommands: []cliapp.Command{
			{Name: "show", NeedsAPI: true, Description: "Show rankings for a list", Run: func(args []string) error { return runShow(api, args) }},
		},
	}
}

func runShow(api *client.Client, args []string) error {
	fs := flag.NewFlagSet("rankings show", flag.ContinueOnError)
	listID := fs.String("list", "", "List ID (required)")
	format := fs.String("format", "table", "Output format: table, json, csv")
	top := fs.Int("top", 0, "Show only the top N items")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *listID == "" {
		return fmt.Errorf("--list is required")
	}

	rankings, err := api.Rankings(*listID)
	if err != nil {
		return err
	}
	if *top > 0 && *top < len(rankings) {
		rankings = rankings[:*top]
	}

	if *format == "csv" {
		return writeCSV(rankings)
	}

	report := cliapp.ListReport{
		Summary: []string{
			"List: " + *listID,
			fmt.Sprintf("Ranked items: %d", len(rankings)),
		},
		Results:        rankingRows(rankings),
		RetrievalHints: []string{cliName + " comparisons next --list " + *listID, cliName + " rankings show --list " + *listID + " --json"},
	}

	if *jsonOutput || *format == "json" {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			cliapp.ListReport
			Rankings []client.RankedItem `json:"rankings"`
		}{ListReport: report, Rankings: rankings})
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func rankingRows(items []client.RankedItem) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, strings.Join([]string{
			"#" + strconv.Itoa(item.Rank),
			render.ItemLabel(item.Item),
			fmt.Sprintf("rating %.2f", item.EloRating),
			"confidence " + render.ConfidencePercent(item.Confidence),
		}, " | "))
	}
	return rows
}

func writeCSV(items []client.RankedItem) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()
	if err := writer.Write([]string{"Rank", "Item", "Elo Rating", "Confidence"}); err != nil {
		return err
	}
	for _, item := range items {
		if err := writer.Write([]string{
			strconv.Itoa(item.Rank),
			render.ItemLabel(item.Item),
			fmt.Sprintf("%.2f", item.EloRating),
			render.ConfidencePercent(item.Confidence),
		}); err != nil {
			return err
		}
	}
	return nil
}
