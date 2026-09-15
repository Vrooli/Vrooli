package trips

import (
	"fmt"
	"os"
	"strings"

	"fall-foliage-explorer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `trips` subcommand group for trip plans
// (`GET /api/trips` and `POST /api/trips`).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "trips",
		Description: "List and save foliage trip plans",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List saved trip plans", Run: func(args []string) error { return runList(core, args) }},
			{Name: "save", Description: "Save a new trip plan (body from --body-file)", Run: func(args []string) error { return runSave(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("trips list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/trips", nil)
	if err != nil {
		return err
	}
	var trips []support.TripPlan
	if err := support.Decode(body, &trips); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Trip plans: %d", len(trips))},
		ResultsHeading: "Trips",
		Results:        tripRows(trips),
		RetrievalHints: []string{
			fmt.Sprintf("%s trips save --body-file trip.json", support.CLIName),
			fmt.Sprintf("%s regions", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSave(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("trips save")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/trips", nil, raw)
	if err != nil {
		return err
	}

	var saved support.TripPlan
	if err := support.Decode(body, &saved); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Trip saved"
	}

	changes := []string{}
	if saved.ID != 0 {
		changes = append(changes, fmt.Sprintf("Trip ID: %d", saved.ID))
	}
	if saved.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", saved.Name))
	}
	if saved.StartDate != "" || saved.EndDate != "" {
		changes = append(changes, fmt.Sprintf("Dates: %s → %s", saved.StartDate, saved.EndDate))
	}
	if len(saved.Regions) > 0 {
		changes = append(changes, "Regions: "+intSliceToCSV(saved.Regions))
	}
	if saved.CreatedAt != "" {
		changes = append(changes, fmt.Sprintf("Created: %s", saved.CreatedAt))
	}

	out := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s trips list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, out)
	}
	return cliapp.RenderMutationReport(os.Stdout, out)
}

func tripRows(trips []support.TripPlan) []string {
	if len(trips) == 0 {
		return []string{"No trip plans saved"}
	}
	rows := make([]string, 0, len(trips))
	for _, t := range trips {
		line := fmt.Sprintf("#%d | %s | %s → %s | regions=%s",
			t.ID, t.Name, t.StartDate, t.EndDate, intSliceToCSV(t.Regions))
		if t.Notes != "" {
			line += " | " + t.Notes
		}
		rows = append(rows, line)
	}
	return rows
}

func intSliceToCSV(values []int) string {
	if len(values) == 0 {
		return "(none)"
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}
