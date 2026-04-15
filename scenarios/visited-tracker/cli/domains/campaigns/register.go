package campaigns

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

func Register(core *cliapp.ScenarioApp, campaignID *string) cliapp.SubcommandGroup {
	resolver := support.Resolver{Core: core, CampaignID: campaignID}
	return cliapp.SubcommandGroup{
		Name:        "campaigns",
		Description: "Manage visit campaigns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List campaigns", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a campaign", Run: func(args []string) error { return runCreate(core, &resolver, args) }},
			{Name: "get", Description: "Get a campaign by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Description: "Update campaign notes", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "note", Description: "Update campaign notes", Run: func(args []string) error { return runNote(core, &resolver, args) }},
			{Name: "reset", Description: "Reset campaign visits", Run: func(args []string) error { return runReset(core, &resolver, args) }},
			{Name: "delete", Description: "Delete a campaign", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "find-or-create", Description: "Find or create by location and tag", Run: func(args []string) error { return runFindOrCreate(core, &resolver, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("campaigns list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/campaigns", nil)
	if err != nil {
		return err
	}
	var response support.CampaignListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse campaigns response: %w", err)
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Campaigns: %d", len(response.Campaigns))},
		Results:        renderCampaigns(response.Campaigns),
		RetrievalHints: []string{cliName + " campaigns get <campaign-id>", cliName + " campaigns create --name \"...\" --pattern \"**/*\""},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("campaigns create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "Campaign name")
	fromAgent := fs.String("from-agent", "cli", "Agent name")
	description := fs.String("description", "", "Campaign description")
	metadata := fs.String("metadata", "", "Campaign metadata JSON or @file")
	var patternFlags cliutil.StringList
	fs.Var(&patternFlags, "pattern", "File pattern (repeatable)")
	patternsCSV := fs.String("patterns", "", "Comma-separated file patterns")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("--name is required")
	}
	patterns := patternFlags.Values()
	patterns = append(patterns, cliutil.ParseCSV(*patternsCSV)...)
	patterns = support.NormalizePathList(patterns)
	if len(patterns) == 0 {
		return errors.New("at least one --pattern or --patterns value is required")
	}
	metadataMap, err := support.ParseJSONInput(*metadata)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"name":       strings.TrimSpace(*name),
		"from_agent": strings.TrimSpace(*fromAgent),
		"patterns":   patterns,
	}
	if strings.TrimSpace(*description) != "" {
		payload["description"] = strings.TrimSpace(*description)
	}
	if metadataMap != nil {
		payload["metadata"] = metadataMap
	}
	body, err := core.Request("POST", "/campaigns", nil, payload)
	if err != nil {
		return err
	}
	var created support.Campaign
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("parse campaign response: %w", err)
	}
	if created.ID != "" {
		resolver.SetCampaignID(created.ID)
	}
	report := cliapp.MutationReport{
		Result: []string{"Campaign created", "ID: " + created.ID},
		Changes: []string{
			"Name: " + created.Name,
			"Patterns: " + strings.Join(patterns, ", "),
		},
		NextCommand: []string{cliName + " campaigns get " + created.ID, cliName + " coverage"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: campaigns get <campaign-id> [--json]")
	}
	campaignID := strings.TrimSpace(args[0])
	if campaignID == "" {
		return errors.New("campaign id is required")
	}
	fs := flag.NewFlagSet("campaigns get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := core.Get("/campaigns/"+campaignID, nil)
	if err != nil {
		return err
	}
	var campaignResp support.Campaign
	if err := json.Unmarshal(body, &campaignResp); err != nil {
		return fmt.Errorf("parse campaign response: %w", err)
	}
	report := cliapp.ListReport{
		Summary:        []string{"Campaign loaded", "ID: " + campaignResp.ID},
		ResultsHeading: "Campaign Details",
		Results:        renderCampaignDetail(campaignResp),
		RetrievalHints: []string{cliName + " campaigns update " + campaignResp.ID + " --note \"...\"", cliName + " campaigns reset --campaign-id " + campaignResp.ID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: campaigns update <campaign-id> --note <text>")
	}
	campaignID := strings.TrimSpace(args[0])
	if campaignID == "" {
		return errors.New("campaign id is required")
	}
	fs := flag.NewFlagSet("campaigns update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	note := fs.String("note", "", "Campaign notes")
	notes := fs.String("notes", "", "Campaign notes (alias)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	finalNote := strings.TrimSpace(*note)
	if finalNote == "" {
		finalNote = strings.TrimSpace(*notes)
	}
	if finalNote == "" {
		return errors.New("--note is required")
	}
	if _, err := core.Request("PATCH", "/campaigns/"+campaignID, nil, map[string]interface{}{"notes": finalNote}); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Campaign updated", "ID: " + campaignID},
		Changes:     []string{"Notes: " + finalNote},
		NextCommand: []string{cliName + " campaigns get " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runNote(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("campaigns note", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	note := fs.String("note", "", "Campaign notes")
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*note) == "" {
		return errors.New("--note is required")
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{Location: *location, Tag: *tag, Pattern: *pattern, Name: *name}, *jsonOutput)
	if err != nil {
		return err
	}
	if _, err := core.Request("PATCH", "/campaigns/"+campaignID, nil, map[string]interface{}{"notes": strings.TrimSpace(*note)}); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Campaign note updated", "ID: " + campaignID},
		Changes:     []string{"Note: " + strings.TrimSpace(*note)},
		NextCommand: []string{cliName + " campaigns get " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runReset(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("campaigns reset", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
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
	if _, err := core.Request("POST", "/campaigns/"+campaignID+"/reset", nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Campaign reset", "ID: " + campaignID},
		Changes:     []string{"Visit counters cleared for the campaign."},
		NextCommand: []string{cliName + " coverage --campaign-id " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: campaigns delete <campaign-id> [--json]")
	}
	campaignID := strings.TrimSpace(args[0])
	if campaignID == "" {
		return errors.New("campaign id is required")
	}
	fs := flag.NewFlagSet("campaigns delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	if _, err := core.Request("DELETE", "/campaigns/"+campaignID, nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{"Campaign deleted", "ID: " + campaignID},
		Changes:     []string{"Campaign data removed."},
		NextCommand: []string{cliName + " campaigns list"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runFindOrCreate(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs := flag.NewFlagSet("campaigns find-or-create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	opts := support.CampaignAutoOptions{Location: *location, Tag: *tag, Pattern: *pattern, Name: *name}
	if !opts.Enabled() {
		return errors.New("--location and --tag are required")
	}
	campaign, created, err := resolver.FindOrCreateCampaign(opts)
	if err != nil {
		return err
	}
	status := "Campaign found"
	if created {
		status = "Campaign created"
	}
	report := cliapp.MutationReport{
		Result:      []string{status, "ID: " + campaign.ID},
		Changes:     []string{"Name: " + campaign.Name, "Location: " + support.ValueOrDash(campaign.Location), "Tag: " + support.ValueOrDash(campaign.Tag)},
		NextCommand: []string{cliName + " campaigns get " + campaign.ID, cliName + " coverage --campaign-id " + campaign.ID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderCampaigns(items []support.Campaign) []string {
	lines := make([]string, 0, len(items))
	for _, c := range items {
		line := fmt.Sprintf("%s (ID: %s)", c.Name, c.ID)
		if c.TotalFiles > 0 {
			line += fmt.Sprintf(" coverage=%.0f%% (%d/%d)", c.CoveragePercent, c.VisitedFiles, c.TotalFiles)
		}
		lines = append(lines, line)
	}
	return lines
}

func renderCampaignDetail(c support.Campaign) []string {
	lines := []string{
		"Name: " + c.Name,
		"Location: " + support.ValueOrDash(c.Location),
		"Tag: " + support.ValueOrDash(c.Tag),
	}
	if len(c.Patterns) > 0 {
		lines = append(lines, "Patterns: "+strings.Join(c.Patterns, ", "))
	}
	if c.Notes != nil && strings.TrimSpace(*c.Notes) != "" {
		lines = append(lines, "Notes: "+strings.TrimSpace(*c.Notes))
	}
	return lines
}
