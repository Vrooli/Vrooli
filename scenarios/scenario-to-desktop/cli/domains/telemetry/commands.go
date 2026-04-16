// Package telemetry provides CLI commands for telemetry management.
package telemetry

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides telemetry CLI commands.
type Commands struct {
	deps support.Dependencies
}

// New creates a new telemetry Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{deps: deps}
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.deps.Get(path, query)
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.deps.Request("POST", path, nil, body)
}

func (c *Commands) apiDelete(path string) ([]byte, error) {
	return c.deps.Request("DELETE", path, nil, nil)
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "telemetry",
		Description: "Deployment telemetry (run 'telemetry help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "ingest", Description: "Ingest telemetry from file: ingest <scenario> --file <path>", Run: cmds.Ingest},
			{Name: "summary", Description: "Get telemetry summary: summary <scenario>", Run: cmds.Summary},
			{Name: "insights", Description: "Get AI-generated insights: insights <scenario>", Run: cmds.Insights},
			{Name: "tail", Description: "Get recent telemetry: tail <scenario> [--limit N]", Run: cmds.Tail},
			{Name: "download", Description: "Download telemetry file: download <scenario> [--output <path>]", Run: cmds.Download},
			{Name: "delete", Description: "Delete telemetry: delete <scenario>", Run: cmds.Delete},
		},
	}
}

// Ingest ingests telemetry from file.
func (c *Commands) Ingest(args []string) error {
	fs := flag.NewFlagSet("telemetry-ingest", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to telemetry JSONL file")
	source := fs.String("source", "cli", "Source identifier")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 || *filePath == "" {
		return fmt.Errorf("usage: telemetry-ingest <scenario> --file <path>")
	}

	scenario := fs.Args()[0]

	// Read file
	data, err := os.ReadFile(*filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSONL lines into events
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	events := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		return fmt.Errorf("no valid events found in file")
	}

	req := map[string]interface{}{
		"scenario_name": scenario,
		"source":        *source,
		"events":        events,
	}

	body, err := c.apiPost("/deployment/telemetry", req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Status         string `json:"status"`
		EventsIngested int    `json:"events_ingested"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Ingested %d events for %s\n", resp.EventsIngested, scenario)
	return nil
}

// Summary gets telemetry summary.
func (c *Commands) Summary(args []string) error {
	fs := flag.NewFlagSet("telemetry-summary", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: telemetry-summary <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiGet("/deployment/telemetry/"+scenario+"/summary", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Insights gets telemetry insights.
func (c *Commands) Insights(args []string) error {
	fs := flag.NewFlagSet("telemetry-insights", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: telemetry-insights <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiGet("/deployment/telemetry/"+scenario+"/insights", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Tail gets recent telemetry.
func (c *Commands) Tail(args []string) error {
	fs := flag.NewFlagSet("telemetry-tail", flag.ContinueOnError)
	limit := fs.Int("limit", 200, "Number of entries to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: telemetry-tail <scenario> [--limit N]")
	}

	scenario := fs.Args()[0]
	query := map[string]string{"limit": fmt.Sprintf("%d", *limit)}

	body, err := c.apiGet("/deployment/telemetry/"+scenario+"/tail", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Download downloads telemetry file.
func (c *Commands) Download(args []string) error {
	fs := flag.NewFlagSet("telemetry-download", flag.ContinueOnError)
	output := fs.String("output", "", "Output file path (default: stdout)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: telemetry-download <scenario> [--output <path>]")
	}

	scenario := fs.Args()[0]
	body, err := c.apiGet("/deployment/telemetry/"+scenario+"/download", nil)
	if err != nil {
		return err
	}

	if *output != "" {
		if err := os.WriteFile(*output, body, 0o644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Downloaded to %s\n", *output)
		return nil
	}

	fmt.Println(string(body))
	return nil
}

// Delete deletes telemetry.
func (c *Commands) Delete(args []string) error {
	fs := flag.NewFlagSet("telemetry-delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: telemetry-delete <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiDelete("/deployment/telemetry/" + scenario)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Telemetry deleted for %s\n", scenario)
	return nil
}
