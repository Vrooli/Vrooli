package foliage

import (
	"fmt"
	"os"
	"strconv"

	"fall-foliage-explorer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `foliage` subcommand group covering status/predict/weather.
// These share a region-scoped read/mutate surface against the foliage API.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "foliage",
		Description: "Query foliage status, predictions, and weather for a region",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "Show current foliage status for a region", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "predict", Description: "Trigger a foliage peak prediction for a region", Run: func(args []string) error { return runPredict(core, args) }},
			{Name: "weather", Description: "Show weather data for a region on a date", Run: func(args []string) error { return runWeather(core, args) }},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("foliage status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: foliage status <region-id>")
	}
	regionID := fs.Arg(0)
	if _, err := strconv.Atoi(regionID); err != nil {
		return fmt.Errorf("region-id must be an integer: %s", regionID)
	}

	query := support.BuildQuery(map[string]string{"region_id": regionID})
	body, err := core.Get("/foliage", query)
	if err != nil {
		return err
	}
	var data support.FoliageData
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Region ID: %d", data.RegionID),
		fmt.Sprintf("Observation date: %s", data.ObservationDate),
		fmt.Sprintf("Foliage: %d%%", data.FoliagePercent),
		fmt.Sprintf("Color intensity: %d/10", data.ColorIntensity),
		fmt.Sprintf("Peak status: %s", data.PeakStatus),
	}
	if data.PredictedPeak != "" {
		results = append(results, fmt.Sprintf("Predicted peak: %s", data.PredictedPeak))
	}
	if data.ConfidenceScore > 0 {
		results = append(results, fmt.Sprintf("Confidence: %.2f", data.ConfidenceScore))
	}
	if data.DataSource != "" {
		results = append(results, fmt.Sprintf("Source: %s", data.DataSource))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Foliage status for region %s", regionID)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s foliage predict %s", support.CLIName, regionID),
			fmt.Sprintf("%s foliage weather %s", support.CLIName, regionID),
			fmt.Sprintf("%s reports list --region %s", support.CLIName, regionID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPredict(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("foliage predict")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: foliage predict <region-id>")
	}
	regionID := fs.Arg(0)
	rid, err := strconv.Atoi(regionID)
	if err != nil {
		return fmt.Errorf("region-id must be an integer: %s", regionID)
	}

	reqBody := map[string]interface{}{"region_id": rid}
	body, err := core.Request("POST", "/predict", nil, reqBody)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Prediction generated for region %s", regionID)
	}

	changes := []string{}
	if v, ok := data["predicted_peak"].(string); ok && v != "" {
		changes = append(changes, fmt.Sprintf("Predicted peak: %s", v))
	}
	if v, ok := data["confidence"].(float64); ok {
		changes = append(changes, fmt.Sprintf("Confidence: %.2f", v))
	}
	if v, ok := data["method"].(string); ok && v != "" {
		changes = append(changes, fmt.Sprintf("Method: %s", v))
	}
	if v, ok := data["region_name"].(string); ok && v != "" {
		changes = append(changes, fmt.Sprintf("Region: %s", v))
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s foliage status %s", support.CLIName, regionID),
			fmt.Sprintf("%s regions", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runWeather(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("foliage weather")
	date := fs.String("date", "", "Observation date (YYYY-MM-DD, defaults to today server-side)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: foliage weather <region-id> [--date YYYY-MM-DD]")
	}
	regionID := fs.Arg(0)
	if _, err := strconv.Atoi(regionID); err != nil {
		return fmt.Errorf("region-id must be an integer: %s", regionID)
	}

	query := support.BuildQuery(map[string]string{
		"region_id": regionID,
		"date":      *date,
	})
	body, err := core.Get("/weather", query)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Weather for region %s", regionID)}
	if msg := support.EnvelopeMessage(body); msg != "" {
		summary = append(summary, msg)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Readings",
		Results:        weatherRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s foliage status %s", support.CLIName, regionID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func weatherRows(data map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{"(no weather data)"}
	}
	// Render known fields in a stable order; fall back to generic rendering for extras.
	ordered := []string{"region_id", "date", "temperature_high", "temperature_low", "precipitation_mm", "humidity_percent"}
	rows := make([]string, 0, len(data))
	seen := map[string]bool{}
	for _, k := range ordered {
		if v, ok := data[k]; ok {
			rows = append(rows, fmt.Sprintf("%s: %s", k, support.RenderValue(v)))
			seen[k] = true
		}
	}
	for k, v := range data {
		if seen[k] {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s: %s", k, support.RenderValue(v)))
	}
	return rows
}
