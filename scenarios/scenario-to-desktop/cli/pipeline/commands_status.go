package pipeline

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// Status gets pipeline status.
func (c *Commands) Status(args []string) error {
	fs := flag.NewFlagSet("pipeline-status", flag.ContinueOnError)
	verbose := fs.Bool("verbose", false, "Include detailed stage logs")
	showOutput := fs.Bool("show-output", false, "Show app stdout/stderr on failure (useful for debugging)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-status <id> [--verbose]")
	}

	pipelineID := fs.Args()[0]
	query := make(map[string]string)
	if *verbose {
		query["verbose"] = "true"
	}

	body, err := c.apiGet("/pipeline/"+pipelineID, query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp pipelineStatus
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	// Also parse progress for display
	var progressResp struct {
		Progress int `json:"progress_percent"`
	}
	_ = json.Unmarshal(body, &progressResp)

	fmt.Printf("Pipeline: %s\n", resp.PipelineID)
	fmt.Printf("Status: %s (%d%% complete)\n", resp.Status, progressResp.Progress)
	if resp.ScenarioName != "" {
		fmt.Printf("Scenario: %s\n", resp.ScenarioName)
	}
	if resp.Config != nil && resp.Config.Version != "" {
		fmt.Printf("Version: %s\n", resp.Config.Version)
	}
	if len(resp.Stages) > 0 {
		fmt.Println("Stages:")
		for name, stage := range resp.Stages {
			fmt.Printf("  %-12s %s\n", name+":", stage.Status)
		}
	}

	// Show error details for failed pipelines
	if resp.Status == "failed" {
		fmt.Println()
		printPipelineError(&resp, *showOutput)
	}

	return nil
}

func (c *Commands) fetchPipelineStatus(pipelineID string, verbose bool) (*pipelineStatus, error) {
	query := map[string]string{}
	if verbose {
		query["verbose"] = "true"
	}
	body, err := c.apiGet("/pipeline/"+pipelineID, query)
	if err != nil {
		return nil, err
	}
	var resp pipelineStatus
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
