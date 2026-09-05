package stats

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"lifestyle-dashboard/cli/internal/query"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "lifestyle-dashboard"

type TimelineEntry struct {
	Day    string `json:"day"`
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type TimelineResponse struct {
	Timeline []TimelineEntry `json:"timeline"`
	Days     string          `json:"days"`
}

type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type SummaryResponse struct {
	TotalEvents    int           `json:"total_events"`
	ActiveDomains  int           `json:"active_domains"`
	EventsByDomain []DomainCount `json:"events_by_domain"`
	LastEventAt    string        `json:"last_event_at"`
}

type DomainScoreEntry struct {
	Domain      string  `json:"domain"`
	DisplayName string  `json:"display_name"`
	Score       int     `json:"score"`
	Weight      float64 `json:"weight"`
	EventCount  int     `json:"event_count"`
}

type LifestyleScore struct {
	Score               int                `json:"score"`
	Date                string             `json:"date"`
	DomainScores        []DomainScoreEntry `json:"domain_scores"`
	Trend               string             `json:"trend"`
	ChangeFromYesterday int                `json:"change_from_yesterday"`
	DataQuality         string             `json:"data_quality"`
	Message             string             `json:"message"`
}

type ScoreHistoryEntry struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

type ScoreResponse struct {
	Current LifestyleScore      `json:"current"`
	History []ScoreHistoryEntry `json:"history"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "stats",
		Description: "Read dashboard timeline, summary, and score metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "timeline", NeedsAPI: true, Description: "Get event timeline", Run: func(args []string) error { return runTimeline(core, args) }},
			{Name: "summary", NeedsAPI: true, Description: "Get aggregated statistics", Run: func(args []string) error { return runSummary(core, args) }},
			{Name: "score", NeedsAPI: true, Description: "Get daily lifestyle score", Run: func(args []string) error { return runScore(core, args) }},
		},
	}
}

func runTimeline(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("stats timeline", flag.ContinueOnError)
	days := fs.Int("days", 0, "Number of days to include (default: 7)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	params := map[string]string{}
	if *days > 0 {
		params["days"] = fmt.Sprintf("%d", *days)
	}
	body, err := core.Get("/stats/timeline", query.ToURLValues(params))
	if err != nil {
		return err
	}
	var resp TimelineResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Timeline window (days): " + resp.Days,
			fmt.Sprintf("Timeline entries: %d", len(resp.Timeline)),
		},
		Results:        renderTimeline(resp.Timeline),
		RetrievalHints: []string{cliName + " stats summary", cliName + " stats score"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSummary(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("stats summary", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/stats/summary", nil)
	if err != nil {
		return err
	}
	var resp SummaryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Total events: %d", resp.TotalEvents),
			fmt.Sprintf("Active domains: %d", resp.ActiveDomains),
		},
		ResultsHeading: "Events By Domain",
		Results:        renderDomainCounts(resp.EventsByDomain),
		RetrievalHints: []string{cliName + " stats timeline", cliName + " stats score"},
	}
	if resp.LastEventAt != "" {
		report.Summary = append(report.Summary, "Last event: "+resp.LastEventAt)
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runScore(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("stats score", flag.ContinueOnError)
	historyDays := fs.Int("history", 0, "Number of history days to include (default: 7)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	params := map[string]string{}
	if *historyDays > 0 {
		params["history_days"] = fmt.Sprintf("%d", *historyDays)
	}
	body, err := core.Get("/stats/score", query.ToURLValues(params))
	if err != nil {
		return err
	}
	var resp ScoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Lifestyle score: %d/100", resp.Current.Score),
			"Date: " + resp.Current.Date,
			fmt.Sprintf("Trend: %s (%+d)", resp.Current.Trend, resp.Current.ChangeFromYesterday),
			"Data quality: " + resp.Current.DataQuality,
			"Message: " + resp.Current.Message,
		},
		ResultsHeading: "Domain Scores",
		Results:        renderScoreBreakdown(resp.Current.DomainScores, resp.History),
		RetrievalHints: []string{cliName + " stats summary", cliName + " stats timeline"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderTimeline(entries []TimelineEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s | %s: %d events", entry.Day, entry.Domain, entry.Count))
	}
	return lines
}

func renderDomainCounts(counts []DomainCount) []string {
	if len(counts) == 0 {
		return nil
	}
	lines := make([]string, 0, len(counts))
	for _, count := range counts {
		lines = append(lines, fmt.Sprintf("%s: %d", count.Domain, count.Count))
	}
	return lines
}

func renderScoreBreakdown(scores []DomainScoreEntry, history []ScoreHistoryEntry) []string {
	lines := make([]string, 0, len(scores)+len(history))
	for _, score := range scores {
		lines = append(lines, fmt.Sprintf("%s: %d/100 (%d events, %.0f%% weight)", score.DisplayName, score.Score, score.EventCount, score.Weight*100))
	}
	for _, item := range history {
		lines = append(lines, fmt.Sprintf("History %s: %d", item.Date, item.Score))
	}
	if len(lines) == 0 {
		return nil
	}
	return lines
}
