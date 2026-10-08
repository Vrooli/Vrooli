package reports

import (
	"fmt"
	"os"
	"strconv"

	"fall-foliage-explorer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `reports` subcommand group for user foliage reports
// (`GET /api/reports?region_id=N` and `POST /api/reports`).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "reports",
		Description: "List and submit user foliage reports",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List reports for a region", Run: func(args []string) error { return runList(core, args) }},
			{Name: "submit", Description: "Submit a user foliage report (body from --body-file)", Run: func(args []string) error { return runSubmit(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reports list")
	region := fs.String("region", "", "Region ID to filter by (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *region == "" && fs.NArg() > 0 {
		*region = fs.Arg(0)
	}
	if *region == "" {
		return fmt.Errorf("usage: reports list --region <region-id>")
	}
	if _, err := strconv.Atoi(*region); err != nil {
		return fmt.Errorf("--region must be an integer: %s", *region)
	}

	query := support.BuildQuery(map[string]string{"region_id": *region})
	body, err := core.Get("/reports", query)
	if err != nil {
		return err
	}
	var reports []support.UserReport
	if err := support.Decode(body, &reports); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Reports for region %s: %d", *region, len(reports))},
		ResultsHeading: "Reports",
		Results:        reportRows(reports),
		RetrievalHints: []string{
			fmt.Sprintf("%s reports submit --body-file report.json", support.CLIName),
			fmt.Sprintf("%s foliage status %s", support.CLIName, *region),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSubmit(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reports submit")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/reports", nil, raw)
	if err != nil {
		return err
	}

	var submitted support.UserReport
	if err := support.Decode(body, &submitted); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Report submitted"
	}

	changes := []string{}
	if submitted.ID != 0 {
		changes = append(changes, fmt.Sprintf("Report ID: %d", submitted.ID))
	}
	if submitted.RegionID != 0 {
		changes = append(changes, fmt.Sprintf("Region: %d", submitted.RegionID))
	}
	if submitted.FoliageStatus != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", submitted.FoliageStatus))
	}
	if submitted.ReportDate != "" {
		changes = append(changes, fmt.Sprintf("Date: %s", submitted.ReportDate))
	}

	out := cliapp.MutationReport{
		Result:  []string{message},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s reports list --region %d", support.CLIName, submitted.RegionID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, out)
	}
	return cliapp.RenderMutationReport(os.Stdout, out)
}

func reportRows(reports []support.UserReport) []string {
	if len(reports) == 0 {
		return []string{"No reports for this region"}
	}
	rows := make([]string, 0, len(reports))
	for _, r := range reports {
		line := fmt.Sprintf("#%d | %s | %s | %s", r.ID, r.ReportDate, r.FoliageStatus, r.Description)
		if r.PhotoURL != "" {
			line += " | photo=" + r.PhotoURL
		}
		rows = append(rows, line)
	}
	return rows
}
