// Package schedule wraps the /api/v1/schedule/* endpoints: optimal-slot search
// and conflict detection.
package schedule

import (
	"fmt"
	"os"
	"strings"
	"time"

	"time-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `schedule` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "schedule",
		Description: "Find optimal meeting slots and detect scheduling conflicts",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "optimal", Description: "Find optimal meeting times for a set of participants", Run: func(args []string) error { return runOptimal(core, args) }},
			{Name: "conflicts", Description: "Detect conflicts in a time window", Run: func(args []string) error { return runConflicts(core, args) }},
		},
	}
}

func runOptimal(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule optimal")
	participants := fs.String("participants", "", "Comma-separated participant identifiers (required)")
	durationMinutes := fs.Int("duration", 60, "Meeting duration in minutes")
	timezone := fs.String("timezone", "UTC", "Timezone for candidate slots")
	earliest := fs.String("earliest", "", "Earliest acceptable date (YYYY-MM-DD, defaults to today)")
	latest := fs.String("latest", "", "Latest acceptable date (YYYY-MM-DD, defaults to today+7d)")
	businessHoursOnly := fs.Bool("business-hours-only", false, "Restrict to business hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*participants) == "" {
		return fmt.Errorf("--participants is required (comma-separated)")
	}

	earliestDate := strings.TrimSpace(*earliest)
	latestDate := strings.TrimSpace(*latest)
	if earliestDate == "" {
		earliestDate = time.Now().UTC().Format("2006-01-02")
	}
	if latestDate == "" {
		latestDate = time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	}

	parts := splitCSV(*participants)
	payload := map[string]interface{}{
		"participants":        parts,
		"duration_minutes":    *durationMinutes,
		"timezone":            *timezone,
		"earliest_date":       earliestDate,
		"latest_date":         latestDate,
		"business_hours_only": *businessHoursOnly,
	}

	body, err := core.Request("POST", "/schedule/optimal", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ScheduleOptimalResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.OptimalSlots))
	for _, slot := range resp.OptimalSlots {
		rows = append(rows, fmt.Sprintf("%s -> %s | score=%.2f | conflicts=%d | free=%s",
			slot.StartTime, slot.EndTime, slot.Score, slot.ConflictCount,
			strings.Join(slot.ParticipantsFree, ",")))
	}
	if len(rows) == 0 {
		rows = []string{"(no optimal slots returned)"}
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Participants: %s", strings.Join(parts, ", ")),
			fmt.Sprintf("Duration: %d minutes", *durationMinutes),
			fmt.Sprintf("Range: %s to %s", earliestDate, latestDate),
		},
		ResultsHeading: "Optimal slots",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s schedule conflicts <start> <end> --organizer <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConflicts(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule conflicts")
	organizerID := fs.String("organizer", "default", "Organizer identifier to check against")
	eventID := fs.String("event", "", "Optional event ID context")
	participants := fs.String("participants", "", "Optional comma-separated participant filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: schedule conflicts <start> <end> [--organizer ID] [--event ID] [--participants a,b]")
	}

	payload := map[string]interface{}{
		"start_time":   fs.Arg(0),
		"end_time":     fs.Arg(1),
		"organizer_id": *organizerID,
	}
	if strings.TrimSpace(*eventID) != "" {
		payload["event_id"] = *eventID
	}
	if strings.TrimSpace(*participants) != "" {
		payload["participants"] = splitCSV(*participants)
	}

	body, err := core.Request("POST", "/schedule/conflicts", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ConflictDetectionResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Conflicts))
	for _, c := range resp.Conflicts {
		rows = append(rows, fmt.Sprintf("%s | severity=%s | overlap=%dm | type=%s",
			c.EventTitle, c.Severity, c.OverlapMinutes, c.ConflictType))
	}
	if len(rows) == 0 {
		rows = []string{"(no conflicts detected)"}
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Range: %s to %s", fs.Arg(0), fs.Arg(1)),
			fmt.Sprintf("Has conflicts: %t (%d)", resp.HasConflicts, resp.ConflictCount),
		},
		ResultsHeading: "Conflicts",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
