package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdCampaignList(args []string) error {
	fs := flag.NewFlagSet("campaigns list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns"), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response campaignListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse campaigns response: %w", err)
	}

	if len(response.Campaigns) == 0 {
		fmt.Println("No campaigns found")
		return nil
	}

	fmt.Printf("Campaigns (%d):\n", len(response.Campaigns))
	for _, c := range response.Campaigns {
		fmt.Printf("- %s (ID: %s)\n", c.Name, c.ID)
		if c.Location != nil || c.Tag != nil {
			fmt.Printf("  Location: %s  Tag: %s\n", valueOrDash(c.Location), valueOrDash(c.Tag))
		}
		if len(c.Patterns) > 0 {
			fmt.Printf("  Patterns: %s\n", strings.Join(c.Patterns, ", "))
		}
		if c.TotalFiles > 0 {
			fmt.Printf("  Coverage: %.0f%% (%d/%d)\n", c.CoveragePercent, c.VisitedFiles, c.TotalFiles)
		}
	}

	return nil
}

func (a *App) cmdCampaignCreate(args []string) error {
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
	patterns = normalizePathList(patterns)
	if len(patterns) == 0 {
		return errors.New("at least one --pattern or --patterns value is required")
	}

	metadataMap, err := parseJSONInput(*metadata)
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

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var created campaign
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("parse campaign response: %w", err)
	}
	if created.ID != "" {
		a.setCampaignID(created.ID)
	}

	fmt.Printf("Campaign created: %s\n", created.Name)
	fmt.Printf("ID: %s\n", created.ID)
	fmt.Printf("Patterns: %s\n", strings.Join(patterns, ", "))
	return nil
}

func (a *App) cmdCampaignGet(args []string) error {
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

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns/"+campaignID), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var campaignResp campaign
	if err := json.Unmarshal(body, &campaignResp); err != nil {
		return fmt.Errorf("parse campaign response: %w", err)
	}

	fmt.Printf("Campaign: %s\n", campaignResp.Name)
	fmt.Printf("ID: %s\n", campaignResp.ID)
	if campaignResp.Location != nil || campaignResp.Tag != nil {
		fmt.Printf("Location: %s  Tag: %s\n", valueOrDash(campaignResp.Location), valueOrDash(campaignResp.Tag))
	}
	if len(campaignResp.Patterns) > 0 {
		fmt.Printf("Patterns: %s\n", strings.Join(campaignResp.Patterns, ", "))
	}
	if campaignResp.Notes != nil && strings.TrimSpace(*campaignResp.Notes) != "" {
		fmt.Printf("Notes: %s\n", strings.TrimSpace(*campaignResp.Notes))
	}
	return nil
}

func (a *App) cmdCampaignUpdate(args []string) error {
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

	payload := map[string]interface{}{"notes": finalNote}
	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/campaigns/"+campaignID), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Campaign updated")
	fmt.Printf("ID: %s\n", campaignID)
	fmt.Printf("Notes: %s\n", finalNote)
	return nil
}

func (a *App) cmdCampaignNote(args []string) error {
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

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{
		location: *location,
		tag:      *tag,
		pattern:  *pattern,
		name:     *name,
	}, *jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{"notes": strings.TrimSpace(*note)}
	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/campaigns/"+campaignID), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Campaign note updated")
	fmt.Printf("ID: %s\n", campaignID)
	fmt.Printf("Note: %s\n", strings.TrimSpace(*note))
	return nil
}

func (a *App) cmdCampaignReset(args []string) error {
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

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{
		location: *location,
		tag:      *tag,
		pattern:  *pattern,
		name:     *name,
	}, *jsonOutput)
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/"+campaignID+"/reset"), nil, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Campaign reset")
	fmt.Printf("ID: %s\n", campaignID)
	return nil
}

func (a *App) cmdCampaignDelete(args []string) error {
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

	body, err := a.core.APIClient.Request("DELETE", a.apiPath("/campaigns/"+campaignID), nil, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Campaign deleted")
	fmt.Printf("ID: %s\n", campaignID)
	return nil
}

func (a *App) cmdCampaignFindOrCreate(args []string) error {
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

	opts := campaignAutoOptions{
		location: *location,
		tag:      *tag,
		pattern:  *pattern,
		name:     *name,
	}
	if !opts.enabled() {
		return errors.New("--location and --tag are required")
	}
	if err := opts.validate(); err != nil {
		return err
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/find-or-create"), nil, map[string]interface{}{
		"location": strings.TrimSpace(opts.location),
		"tag":      strings.TrimSpace(opts.tag),
		"patterns": []string{defaultPattern(opts.pattern)},
		"name":     strings.TrimSpace(opts.name),
	})
	if err != nil {
		return err
	}

	var resp findOrCreateResponse
	parseErr := json.Unmarshal(body, &resp)
	if resp.Campaign.ID != "" {
		a.setCampaignID(resp.Campaign.ID)
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	if parseErr != nil {
		return fmt.Errorf("parse response: %w", parseErr)
	}
	if resp.Created {
		fmt.Printf("Campaign created: %s\n", resp.Campaign.Name)
	} else {
		fmt.Printf("Campaign found: %s\n", resp.Campaign.Name)
	}
	fmt.Printf("ID: %s\n", resp.Campaign.ID)
	return nil
}

func defaultPattern(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "**/*"
	}
	return value
}

func valueOrDash(value *string) string {
	if value == nil {
		return "-"
	}
	trim := strings.TrimSpace(*value)
	if trim == "" {
		return "-"
	}
	return trim
}
