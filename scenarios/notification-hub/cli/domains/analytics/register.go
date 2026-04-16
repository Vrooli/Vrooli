package analytics

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"notification-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(d support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analytics",
		Description: "View delivery analytics",
		Subcommands: []cliapp.Command{
			{Name: "delivery-stats", NeedsAPI: true, Description: "Show aggregate delivery statistics", Run: func(args []string) error { return runDeliveryStats(d, args) }},
			{Name: "daily-stats", NeedsAPI: true, Description: "Show daily delivery statistics", Run: func(args []string) error { return runDailyStats(d, args) }},
		},
	}
}

type deliveryStatsResponse struct {
	Stats map[string]interface{} `json:"stats"`
}

type dailyStatsResponse struct {
	DailyStats []map[string]interface{} `json:"daily_stats"`
}

type deliveryStatsReport struct {
	cliapp.ListReport
	ProfileID string                 `json:"profile_id"`
	Stats     map[string]interface{} `json:"stats"`
}

type dailyStatsReport struct {
	cliapp.ListReport
	ProfileID  string                   `json:"profile_id"`
	DailyStats []map[string]interface{} `json:"daily_stats"`
}

func runDeliveryStats(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("delivery-stats", flag.ContinueOnError)
	profileFlag := fs.String("profile-id", "", "Profile ID override")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	profileID, err := d.ResolveProfileID(*profileFlag)
	if err != nil {
		return err
	}

	body, err := d.ScopedGet(profileID, "/analytics/delivery-stats", nil)
	if err != nil {
		return err
	}

	var resp deliveryStatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := deliveryStatsReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{"Profile ID: " + profileID, fmt.Sprintf("Stat keys: %d", len(resp.Stats))},
			Results:        renderStatsMap(resp.Stats),
			RetrievalHints: []string{"notification-hub analytics daily-stats --profile-id " + profileID},
		},
		ProfileID: profileID,
		Stats:     resp.Stats,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func runDailyStats(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("daily-stats", flag.ContinueOnError)
	profileFlag := fs.String("profile-id", "", "Profile ID override")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	profileID, err := d.ResolveProfileID(*profileFlag)
	if err != nil {
		return err
	}

	body, err := d.ScopedGet(profileID, "/analytics/daily-stats", nil)
	if err != nil {
		return err
	}

	var resp dailyStatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := dailyStatsReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{"Profile ID: " + profileID, fmt.Sprintf("Days returned: %d", len(resp.DailyStats))},
			Results:        renderDaily(resp.DailyStats),
			RetrievalHints: []string{"notification-hub analytics delivery-stats --profile-id " + profileID},
		},
		ProfileID:  profileID,
		DailyStats: resp.DailyStats,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func renderStatsMap(stats map[string]interface{}) []string {
	if len(stats) == 0 {
		return nil
	}
	keys := make([]string, 0, len(stats))
	for key := range stats {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, fmt.Sprintf("%s: %v", key, stats[key]))
	}
	return rows
}

func renderDaily(items []map[string]interface{}) []string {
	if len(items) == 0 {
		return nil
	}
	rows := make([]string, 0, len(items))
	for i, item := range items {
		day := fmt.Sprintf("day %d", i+1)
		if value, ok := item["date"]; ok {
			day = fmt.Sprintf("%v", value)
		}
		rows = append(rows, day)
		stats := renderStatsMap(item)
		for _, stat := range stats {
			rows = append(rows, "  "+stat)
		}
	}
	return rows
}
