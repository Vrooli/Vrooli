package data

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"visited-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "visited-tracker"

func Register(core *cliapp.ScenarioApp, campaignID *string) cliapp.CommandGroup {
	resolver := support.Resolver{Core: core, CampaignID: campaignID}
	return cliapp.CommandGroup{
		Title: "Data",
		Commands: []cliapp.Command{
			{Name: "export", NeedsAPI: true, Description: "Export campaign data to a file", Run: func(args []string) error { return runExport(core, &resolver, args) }},
			{Name: "import", NeedsAPI: true, Description: "Import campaign data from a file", Run: func(args []string) error { return runImport(core, args) }},
		},
	}
}

func runExport(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: export <file> [--format json] [--include-history] [--patterns PATTERN]")
	}
	filePath, err := support.EnsureFilePath(args[0])
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "json", "Export format (json only)")
	includeHistory := fs.Bool("include-history", false, "Include visit history")
	patterns := fs.String("patterns", "", "Comma-separated patterns")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{}, true)
	if err != nil {
		return err
	}
	query := map[string]string{}
	if strings.TrimSpace(*patterns) != "" {
		query["patterns"] = strings.TrimSpace(*patterns)
	}
	if strings.TrimSpace(*format) != "" {
		query["format"] = strings.TrimSpace(*format)
	}
	if *includeHistory {
		query["include_history"] = "true"
	}
	body, err := core.Get("/campaigns/"+campaignID+"/export", support.BuildQuery(query))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, body, 0o600); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	report := cliapp.MutationReport{
		Result: []string{"Export completed", "File: " + filePath},
		Changes: []string{
			"Campaign ID: " + campaignID,
		},
		NextCommand: []string{cliName + " import " + filePath},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runImport(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: import <file> [--merge true|false] [--format json] [--json]")
	}
	filePath, err := support.EnsureFilePath(args[0])
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	mergeValue := fs.String("merge", "", "Merge with existing campaign (true/false)")
	format := fs.String("format", "json", "Import format (json only)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	_ = format
	payload, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}
	var campaignData map[string]interface{}
	if err := json.Unmarshal(payload, &campaignData); err != nil {
		return fmt.Errorf("invalid campaign data: %w", err)
	}
	query := map[string]string{}
	mergeVal := strings.TrimSpace(*mergeValue)
	if mergeVal != "" {
		if mergeVal == "true" || mergeVal == "false" {
			query["merge"] = mergeVal
		} else {
			query["merge"] = "true"
		}
	}
	body, err := core.Request("POST", "/campaigns/import", support.BuildQuery(query), campaignData)
	if err != nil {
		return err
	}
	var response support.ImportResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse import response: %w", err)
	}
	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = "Campaign imported"
	}
	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{"File: " + filePath, "ID: " + response.Campaign.ID},
		NextCommand: []string{cliName + " campaigns get " + response.Campaign.ID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
