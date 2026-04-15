package analytics

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"visited-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "visited-tracker"

func Register(core *cliapp.ScenarioApp, campaignID *string) cliapp.CommandGroup {
	resolver := support.Resolver{Core: core, CampaignID: campaignID}
	return cliapp.CommandGroup{
		Title: "Analytics",
		Commands: []cliapp.Command{
			{Name: "least-visited", NeedsAPI: true, Description: "List least visited files", Run: func(args []string) error { return runLeastVisited(core, &resolver, args) }},
			{Name: "most-stale", NeedsAPI: true, Description: "List most stale files", Run: func(args []string) error { return runMostStale(core, &resolver, args) }},
			{Name: "coverage", NeedsAPI: true, Description: "Show coverage statistics", Run: func(args []string) error { return runCoverage(core, &resolver, args) }},
		},
	}
}

func runLeastVisited(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("least-visited", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.String("limit", "", "Limit results")
	context := fs.String("context", "", "Filter by context")
	includeUnvisited := fs.Bool("include-unvisited", true, "Include never-visited files")
	noUnvisited := fs.Bool("no-unvisited", false, "Exclude never-visited files")
	patterns := fs.String("patterns", "", "Filter patterns (comma-separated, currently ignored)")
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "Campaign pattern")
	name := fs.String("name", "", "Campaign name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*limit) != "" {
		if _, err := strconv.Atoi(*limit); err != nil {
			return errors.New("--limit must be a positive integer")
		}
	}
	_ = patterns
	includeValue := *includeUnvisited
	if *noUnvisited {
		includeValue = false
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{Location: *location, Tag: *tag, Pattern: *pattern, Name: *name}, *jsonOutput)
	if err != nil {
		return err
	}
	query := map[string]string{}
	if strings.TrimSpace(*limit) != "" {
		query["limit"] = strings.TrimSpace(*limit)
	}
	if strings.TrimSpace(*context) != "" {
		query["context"] = strings.TrimSpace(*context)
	}
	if !includeValue {
		query["include_unvisited"] = "false"
	}
	body, err := core.Get("/campaigns/"+campaignID+"/prioritize/least-visited", support.BuildQuery(query))
	if err != nil {
		return err
	}
	var response struct {
		Files []support.TrackedFile `json:"files"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse least-visited response: %w", err)
	}
	report := cliapp.ListReport{
		Summary: []string{"Least visited files", "Campaign ID: " + campaignID},
		Results: renderTrackedFiles(response.Files, func(file support.TrackedFile) string {
			return fmt.Sprintf("%s: %d visits", file.FilePath, file.VisitCount)
		}),
		RetrievalHints: []string{cliName + " visit <file>", cliName + " coverage --campaign-id " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runMostStale(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("most-stale", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.String("limit", "", "Limit results")
	threshold := fs.String("threshold", "", "Minimum staleness score")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*threshold) != "" {
		if _, err := strconv.ParseFloat(*threshold, 64); err != nil {
			return errors.New("--threshold must be numeric")
		}
	}
	if strings.TrimSpace(*limit) != "" {
		if _, err := strconv.Atoi(*limit); err != nil {
			return errors.New("--limit must be a positive integer")
		}
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}
	query := map[string]string{}
	if strings.TrimSpace(*limit) != "" {
		query["limit"] = strings.TrimSpace(*limit)
	}
	if strings.TrimSpace(*threshold) != "" {
		query["threshold"] = strings.TrimSpace(*threshold)
	}
	body, err := core.Get("/campaigns/"+campaignID+"/prioritize/most-stale", support.BuildQuery(query))
	if err != nil {
		return err
	}
	var response struct {
		Files         []support.TrackedFile `json:"files"`
		CriticalCount int                   `json:"critical_count"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse most-stale response: %w", err)
	}
	report := cliapp.ListReport{
		Summary: []string{"Most stale files", "Campaign ID: " + campaignID, fmt.Sprintf("Critical files: %d", response.CriticalCount)},
		Results: renderTrackedFiles(response.Files, func(file support.TrackedFile) string {
			return fmt.Sprintf("%s: staleness=%.0f visits=%d", file.FilePath, file.StalenessScore, file.VisitCount)
		}),
		RetrievalHints: []string{cliName + " files priority --file-id <file-id> --weight 2.0", cliName + " least-visited --campaign-id " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCoverage(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("coverage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	patterns := fs.String("patterns", "", "Comma-separated patterns (optional)")
	groupBy := fs.String("group-by", "", "Group coverage (currently ignored)")
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{Location: *location, Tag: *tag, Pattern: *pattern, Name: *name}, *jsonOutput)
	if err != nil {
		return err
	}
	query := map[string]string{}
	if strings.TrimSpace(*patterns) != "" {
		query["patterns"] = strings.TrimSpace(*patterns)
	}
	if strings.TrimSpace(*groupBy) != "" {
		query["group_by"] = strings.TrimSpace(*groupBy)
	}
	body, err := core.Get("/campaigns/"+campaignID+"/coverage", support.BuildQuery(query))
	if err != nil {
		return err
	}
	var coverage support.CoverageResponse
	if err := json.Unmarshal(body, &coverage); err != nil {
		return fmt.Errorf("parse coverage response: %w", err)
	}
	report := cliapp.OperationalReport{
		Status: []string{
			"Coverage report ready",
			fmt.Sprintf("Campaign ID: %s", campaignID),
		},
		Triage: []cliapp.TriageGroup{{
			Heading: "Coverage",
			Items: []string{
				fmt.Sprintf("Total files: %d", coverage.TotalFiles),
				fmt.Sprintf("Visited: %d (%.0f%%)", coverage.VisitedFiles, coverage.CoveragePercent),
				fmt.Sprintf("Unvisited: %d", coverage.UnvisitedFiles),
				fmt.Sprintf("Average visits: %.0f", coverage.AverageVisits),
				fmt.Sprintf("Average staleness: %.0f", coverage.AverageStaleness),
			},
		}},
		NextSteps: []string{cliName + " least-visited --campaign-id " + campaignID, cliName + " most-stale --campaign-id " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func renderTrackedFiles(files []support.TrackedFile, render func(file support.TrackedFile) string) []string {
	lines := make([]string, 0, len(files))
	for _, file := range files {
		lines = append(lines, render(file))
	}
	return lines
}
