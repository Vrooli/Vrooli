// Package time wraps the /api/v1/time/* endpoints: timezone conversion,
// duration, format, parse, add, subtract, and a convenience `now` helper.
package time

import (
	"fmt"
	"os"
	stdtime "time"

	"time-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `time` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "time",
		Description: "Timezone, duration, formatting, parsing, and arithmetic helpers",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "convert", Description: "Convert a timestamp between timezones", Run: func(args []string) error { return runConvert(core, args) }},
			{Name: "duration", Description: "Calculate duration between two timestamps", Run: func(args []string) error { return runDuration(core, args) }},
			{Name: "format", Description: "Format a timestamp in a named or custom style", Run: func(args []string) error { return runFormat(core, args) }},
			{Name: "parse", Description: "Parse a free-form timestamp into RFC3339/Unix", Run: func(args []string) error { return runParse(core, args) }},
			{Name: "add", Description: "Add a duration to a timestamp", Run: func(args []string) error { return runArithmetic(core, args, "/time/add", "add") }},
			{Name: "subtract", Description: "Subtract a duration from a timestamp", Run: func(args []string) error { return runArithmetic(core, args, "/time/subtract", "subtract") }},
			{Name: "now", Description: "Show the current time, optionally in a given timezone", Run: func(args []string) error { return runNow(core, args) }},
		},
	}
}

func runConvert(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("time convert")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 3 {
		return fmt.Errorf("usage: time convert <time> <from-timezone> <to-timezone>")
	}
	payload := map[string]interface{}{
		"time":          fs.Arg(0),
		"from_timezone": fs.Arg(1),
		"to_timezone":   fs.Arg(2),
	}

	body, err := core.Request("POST", "/time/convert", nil, payload)
	if err != nil {
		return err
	}
	var resp support.TimezoneConversionResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Original:  %s (%s)", resp.OriginalTime, resp.FromTimezone),
		fmt.Sprintf("Converted: %s (%s)", resp.ConvertedTime, resp.ToTimezone),
		fmt.Sprintf("Offset:    %d minutes", resp.OffsetMinutes),
		fmt.Sprintf("DST:       %t", resp.IsDST),
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Converted %s -> %s", resp.FromTimezone, resp.ToTimezone)},
		ResultsHeading: "Conversion",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s time now %s", support.CLIName, resp.ToTimezone),
			fmt.Sprintf("%s time format %s human --timezone %s", support.CLIName, resp.ConvertedTime, resp.ToTimezone),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDuration(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("time duration")
	excludeWeekends := fs.Bool("exclude-weekends", false, "Exclude weekends from calculations")
	excludeHolidays := fs.Bool("exclude-holidays", false, "Exclude holidays from calculations")
	businessHoursOnly := fs.Bool("business-hours-only", false, "Use only business hours")
	timezone := fs.String("timezone", "", "Timezone for interpretation")
	entityID := fs.String("entity", "", "Entity ID for holiday/calendar context")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: time duration <start> <end> [--exclude-weekends] [--exclude-holidays] [--business-hours-only] [--timezone TZ] [--entity ID]")
	}
	payload := map[string]interface{}{
		"start_time":          fs.Arg(0),
		"end_time":            fs.Arg(1),
		"timezone":            *timezone,
		"exclude_weekends":    *excludeWeekends,
		"exclude_holidays":    *excludeHolidays,
		"business_hours_only": *businessHoursOnly,
		"entity_id":           *entityID,
	}

	body, err := core.Request("POST", "/time/duration", nil, payload)
	if err != nil {
		return err
	}
	var resp support.DurationCalculationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Start:         %s", resp.StartTime),
		fmt.Sprintf("End:           %s", resp.EndTime),
		fmt.Sprintf("Total Minutes: %d", resp.TotalMinutes),
		fmt.Sprintf("Total Hours:   %.2f", resp.TotalHours),
		fmt.Sprintf("Total Days:    %.2f", resp.TotalDays),
		fmt.Sprintf("Calendar Days: %d", resp.CalendarDays),
	}
	if *businessHoursOnly || *excludeWeekends {
		results = append(results,
			fmt.Sprintf("Business Hours: %.2f", resp.BusinessHours),
			fmt.Sprintf("Business Days:  %d", resp.BusinessDays),
		)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Duration: %.2f hours (%.2f days)", resp.TotalHours, resp.TotalDays)},
		ResultsHeading: "Duration",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s time duration %s %s --business-hours-only", support.CLIName, resp.StartTime, resp.EndTime),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runFormat(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("time format")
	timezone := fs.String("timezone", "", "Render the timestamp in this timezone")
	locale := fs.String("locale", "", "Locale hint for the server")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: time format <time> <format> [--timezone TZ] [--locale LL]\n  formats: iso8601, unix, date, time, datetime, human, relative, or a Go layout")
	}
	payload := map[string]interface{}{
		"time":     fs.Arg(0),
		"format":   fs.Arg(1),
		"timezone": *timezone,
		"locale":   *locale,
	}

	body, err := core.Request("POST", "/time/format", nil, payload)
	if err != nil {
		return err
	}
	var resp support.TimeFormatResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Original:  %s", resp.Original),
		fmt.Sprintf("Format:    %s", resp.Format),
	}
	if resp.Timezone != "" {
		results = append(results, fmt.Sprintf("Timezone:  %s", resp.Timezone))
	}
	results = append(results, fmt.Sprintf("Formatted: %s", resp.Formatted))

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Formatted as %s", resp.Format)},
		ResultsHeading: "Formatted time",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runParse(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("time parse")
	timezone := fs.String("timezone", "", "Apply this timezone to the parsed result")
	format := fs.String("format", "", "Go layout hint to try first")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: time parse <input> [--timezone TZ] [--format LAYOUT]")
	}
	payload := map[string]interface{}{
		"input":    fs.Arg(0),
		"timezone": *timezone,
		"format":   *format,
	}

	body, err := core.Request("POST", "/time/parse", nil, payload)
	if err != nil {
		return err
	}
	var resp support.TimeParseResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("RFC3339:    %s", resp.RFC3339),
		fmt.Sprintf("Unix:       %d", resp.Unix),
		fmt.Sprintf("Timezone:   %s", resp.Timezone),
		fmt.Sprintf("Ambiguous:  %t", resp.IsAmbiguous),
		fmt.Sprintf("Confidence: %s", resp.Confidence),
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Parsed with %s confidence", resp.Confidence)},
		ResultsHeading: "Parsed time",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runArithmetic(core *cliapp.ScenarioApp, args []string, path, operation string) error {
	fs := support.NewFlagSet("time " + operation)
	unit := fs.String("unit", "", "Explicit unit when duration is numeric (seconds|minutes|hours|days|weeks)")
	timezone := fs.String("timezone", "", "Render result in this timezone")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: time %s <time> <duration> [--unit UNIT] [--timezone TZ]", operation)
	}
	payload := map[string]interface{}{
		"time":     fs.Arg(0),
		"duration": fs.Arg(1),
		"unit":     *unit,
		"timezone": *timezone,
	}

	body, err := core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	var resp support.TimeArithmeticResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Original: %s", resp.OriginalTime),
			fmt.Sprintf("Duration: %s", resp.Duration),
			fmt.Sprintf("Result:   %s", resp.ResultTime),
		},
		Changes: []string{fmt.Sprintf("Operation: %s", resp.Operation)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runNow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("time now")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	timezone := "UTC"
	if fs.NArg() >= 1 {
		timezone = fs.Arg(0)
	}

	payload := map[string]interface{}{
		"time":          stdtime.Now().UTC().Format(stdtime.RFC3339),
		"from_timezone": "UTC",
		"to_timezone":   timezone,
	}
	body, err := core.Request("POST", "/time/convert", nil, payload)
	if err != nil {
		return err
	}
	var resp support.TimezoneConversionResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Timezone: %s", resp.ToTimezone),
		fmt.Sprintf("Time:     %s", resp.ConvertedTime),
		fmt.Sprintf("Offset:   %d minutes", resp.OffsetMinutes),
		fmt.Sprintf("DST:      %t", resp.IsDST),
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Current time in %s", resp.ToTimezone)},
		ResultsHeading: "Now",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
