package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdLeastVisited(args []string) error {
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

	if err := fs.Parse(args); err != nil {
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

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{
		location: *location,
		tag:      *tag,
		pattern:  *pattern,
		name:     *name,
	}, *jsonOutput)
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

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns/"+campaignID+"/prioritize/least-visited"), buildQuery(query))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response struct {
		Files []trackedFile `json:"files"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse least-visited response: %w", err)
	}

	fmt.Println("Least Visited Files")
	if len(response.Files) == 0 {
		fmt.Println("  No tracked files yet")
		return nil
	}
	for _, file := range response.Files {
		fmt.Printf("  %s: %d visits\n", file.FilePath, file.VisitCount)
	}
	return nil
}

func (a *App) cmdMostStale(args []string) error {
	fs := flag.NewFlagSet("most-stale", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.String("limit", "", "Limit results")
	threshold := fs.String("threshold", "", "Minimum staleness score")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, *jsonOutput)
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

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns/"+campaignID+"/prioritize/most-stale"), buildQuery(query))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response struct {
		Files         []trackedFile `json:"files"`
		CriticalCount int           `json:"critical_count"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse most-stale response: %w", err)
	}

	fmt.Println("Most Stale Files")
	if len(response.Files) == 0 {
		fmt.Println("  No stale files detected")
	} else {
		for _, file := range response.Files {
			fmt.Printf("  %s: staleness=%.0f, visits=%d\n", file.FilePath, file.StalenessScore, file.VisitCount)
		}
	}
	fmt.Printf("Critical files: %d\n", response.CriticalCount)
	return nil
}

func (a *App) cmdCoverage(args []string) error {
	fs := flag.NewFlagSet("coverage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	patterns := fs.String("patterns", "", "Comma-separated patterns (optional)")
	groupBy := fs.String("group-by", "", "Group coverage (currently ignored)")
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{
		location: *location,
		tag:      *tag,
		pattern:  *pattern,
		name:     *name,
	}, *jsonOutput)
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

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns/"+campaignID+"/coverage"), buildQuery(query))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var coverage coverageResponse
	if err := json.Unmarshal(body, &coverage); err != nil {
		return fmt.Errorf("parse coverage response: %w", err)
	}

	fmt.Println("Coverage Report")
	fmt.Printf("Total files: %d\n", coverage.TotalFiles)
	fmt.Printf("Visited: %d (%.0f%%)\n", coverage.VisitedFiles, coverage.CoveragePercent)
	fmt.Printf("Unvisited: %d\n", coverage.UnvisitedFiles)
	fmt.Printf("Average visits: %.0f\n", coverage.AverageVisits)
	fmt.Printf("Average staleness: %.0f\n", coverage.AverageStaleness)
	return nil
}
