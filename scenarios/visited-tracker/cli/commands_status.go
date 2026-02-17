package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get("/health", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp healthResponse
	if err := json.Unmarshal(body, &resp); err == nil && resp.Status != "" {
		fmt.Println("Visited Tracker Status")
		if resp.Service != "" {
			fmt.Printf("Service: %s\n", resp.Service)
		}
		if resp.Version != "" {
			fmt.Printf("Version: %s\n", resp.Version)
		}
		if resp.Status != "" {
			fmt.Printf("Status: %s\n", resp.Status)
		}
		if resp.Metrics.UptimeSeconds > 0 {
			fmt.Printf("Uptime: %ds\n", int(resp.Metrics.UptimeSeconds))
		}
		fmt.Println()
	} else {
		fmt.Println("Visited Tracker Status")
		fmt.Println(string(body))
		fmt.Println()
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{
		location: *location,
		tag:      *tag,
		pattern:  *pattern,
		name:     *name,
	}, true)
	if err != nil {
		fmt.Println("Coverage Overview:")
		fmt.Println("  Coverage data not available (no campaigns found)")
		return nil
	}

	coverageBody, err := a.core.APIClient.Get(a.apiPath(fmt.Sprintf("/campaigns/%s/coverage", campaignID)), nil)
	if err != nil {
		fmt.Println("Coverage Overview:")
		fmt.Println("  Coverage data not available (campaign request failed)")
		return nil
	}

	var coverage coverageResponse
	if err := json.Unmarshal(coverageBody, &coverage); err != nil {
		fmt.Println("Coverage Overview:")
		fmt.Println("  Coverage data not available (invalid response)")
		return nil
	}

	fmt.Println("Coverage Overview:")
	fmt.Printf("  Total files: %d\n", coverage.TotalFiles)
	fmt.Printf("  Visited: %d (%.0f%%)\n", coverage.VisitedFiles, coverage.CoveragePercent)
	fmt.Printf("  Unvisited: %d\n", coverage.UnvisitedFiles)

	return nil
}
