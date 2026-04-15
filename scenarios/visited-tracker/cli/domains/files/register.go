package files

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

func Register(core *cliapp.ScenarioApp, campaignID *string) cliapp.SubcommandGroup {
	resolver := support.Resolver{Core: core, CampaignID: campaignID}
	return cliapp.SubcommandGroup{
		Name:        "files",
		Description: "Manage tracked files",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get-by-path", Description: "Lookup a tracked file by path", Run: func(args []string) error { return runGetByPath(core, &resolver, args) }},
			{Name: "note", Description: "Update notes for a tracked file", Run: func(args []string) error { return runNote(core, &resolver, args) }},
			{Name: "priority", Description: "Update priority weight for a tracked file", Run: func(args []string) error { return runPriority(core, &resolver, args) }},
			{Name: "exclude", Description: "Toggle exclusion for a tracked file", Run: func(args []string) error { return runExclude(core, &resolver, args) }},
		},
	}
}

func runGetByPath(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("files get-by-path", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", "", "File path to lookup")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	path := strings.TrimSpace(*pathFlag)
	if path == "" && fs.NArg() > 0 {
		path = strings.TrimSpace(fs.Arg(0))
	}
	if path == "" {
		return errors.New("--path is required")
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}
	body, err := core.Get("/campaigns/"+campaignID+"/files/by-path", support.BuildQuery(map[string]string{"path": path}))
	if err != nil {
		return err
	}
	var file support.TrackedFile
	if err := json.Unmarshal(body, &file); err != nil {
		return fmt.Errorf("parse file response: %w", err)
	}
	report := cliapp.ListReport{
		Summary:        []string{"File loaded", "File ID: " + file.ID},
		ResultsHeading: "File Details",
		Results: []string{
			"Path: " + file.FilePath,
			fmt.Sprintf("Visits: %d", file.VisitCount),
			fmt.Sprintf("Staleness: %.0f", file.StalenessScore),
			fmt.Sprintf("Excluded: %t", file.Excluded),
		},
		RetrievalHints: []string{cliName + " files note --file-id " + file.ID + " --note \"...\"", cliName + " files priority --file-id " + file.ID + " --weight 1.5"},
	}
	if file.Notes != nil && strings.TrimSpace(*file.Notes) != "" {
		report.Results = append(report.Results, "Notes: "+strings.TrimSpace(*file.Notes))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runNote(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("files note", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileID := fs.String("file-id", "", "Tracked file ID")
	note := fs.String("note", "", "Note text")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*fileID) == "" {
		return errors.New("--file-id is required")
	}
	if strings.TrimSpace(*note) == "" {
		return errors.New("--note is required")
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/campaigns/"+campaignID+"/files/"+strings.TrimSpace(*fileID)+"/notes", nil, map[string]interface{}{"notes": strings.TrimSpace(*note)}); err != nil {
		return err
	}
	report := cliapp.MutationReport{Result: []string{"File notes updated", "File ID: " + strings.TrimSpace(*fileID)}, Changes: []string{"Notes: " + strings.TrimSpace(*note)}, NextCommand: []string{cliName + " files get-by-path --path <file-path>"}}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runPriority(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("files priority", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileID := fs.String("file-id", "", "Tracked file ID")
	weight := fs.String("weight", "", "Priority weight (float)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*fileID) == "" {
		return errors.New("--file-id is required")
	}
	if strings.TrimSpace(*weight) == "" {
		return errors.New("--weight is required")
	}
	parsedWeight, err := strconv.ParseFloat(strings.TrimSpace(*weight), 64)
	if err != nil {
		return errors.New("--weight must be numeric")
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/campaigns/"+campaignID+"/files/"+strings.TrimSpace(*fileID)+"/priority", nil, map[string]interface{}{"priority_weight": parsedWeight}); err != nil {
		return err
	}
	report := cliapp.MutationReport{Result: []string{"File priority updated", "File ID: " + strings.TrimSpace(*fileID)}, Changes: []string{fmt.Sprintf("Priority weight: %.2f", parsedWeight)}, NextCommand: []string{cliName + " most-stale"}}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExclude(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("files exclude", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileID := fs.String("file-id", "", "Tracked file ID")
	excluded := fs.Bool("excluded", true, "Set excluded state")
	include := fs.Bool("include", false, "Clear exclusion")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*fileID) == "" {
		return errors.New("--file-id is required")
	}
	excludedValue := *excluded
	if *include {
		excludedValue = false
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/campaigns/"+campaignID+"/files/"+strings.TrimSpace(*fileID)+"/exclude", nil, map[string]interface{}{"excluded": excludedValue}); err != nil {
		return err
	}
	report := cliapp.MutationReport{Result: []string{"File exclusion updated", "File ID: " + strings.TrimSpace(*fileID)}, Changes: []string{fmt.Sprintf("Excluded: %t", excludedValue)}, NextCommand: []string{cliName + " files get-by-path --path <file-path>"}}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
