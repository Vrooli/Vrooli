package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"scenario-completeness-scoring/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "scenario-completeness-scoring"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Server Configuration",
		Commands: []cliapp.Command{
			{Name: "config", NeedsAPI: true, Description: "Show server scoring configuration", Run: func(args []string) error { return runConfig(core, args) }},
		},
	}
}

func runConfig(core *cliapp.ScenarioApp, args []string) error {
	if len(args) > 0 && args[0] == "set" {
		return runConfigSet(core, args[1:])
	}

	fs, jsonOut, err := support.ParseFlags("config", args)
	if err != nil {
		return err
	}
	_ = fs

	body, err := core.Get("/config", nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scoring configuration loaded"},
		ResultsHeading: "Configuration Payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{cliName + " config set --file ./config.json", cliName + " configure api_base <url>"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to JSON config file")
	inline := fs.String("json", "", "Inline JSON payload")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var payload map[string]interface{}
	switch {
	case *filePath != "":
		data, err := os.ReadFile(*filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	case *inline != "":
		if err := json.Unmarshal([]byte(*inline), &payload); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	default:
		return fmt.Errorf("config set requires --file or --json")
	}

	body, err := core.Request("PUT", "/config", nil, payload)
	if err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Scoring configuration updated"},
		Changes:     []string{"Server-side scoring weights and components were updated."},
		NextCommand: []string{cliName + " config", cliName + " scores"},
	}
	if *filePath != "" {
		report.Changes = append(report.Changes, "Config file: "+*filePath)
	}
	if *inline != "" {
		report.Changes = append(report.Changes, "Inline JSON payload applied.")
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	_ = body
	return cliapp.RenderMutationReport(os.Stdout, report)
}
