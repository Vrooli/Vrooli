package profiles

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"notification-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(d support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "profiles",
		Description: "Manage notification profiles",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List profiles", Run: func(args []string) error { return runList(d, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a profile", Run: func(args []string) error { return runCreate(d, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get a profile", Run: func(args []string) error { return runGet(d, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update a profile", Run: func(args []string) error { return runUpdate(d, args) }},
		},
	}
}

type profile struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	APIKey       string                 `json:"api_key,omitempty"`
	APIKeyPrefix string                 `json:"api_key_prefix,omitempty"`
	Plan         string                 `json:"plan,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

type profilesListResponse struct {
	Profiles []profile `json:"profiles"`
}

type profilesListReport struct {
	cliapp.ListReport
	Profiles []profile `json:"profiles"`
}

type profileMutationReport struct {
	cliapp.MutationReport
	Profile profile `json:"profile"`
}

type profileDetailReport struct {
	cliapp.ListReport
	Profile profile `json:"profile"`
}

func runList(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := d.ScenarioApp().Get("/admin/profiles", nil)
	if err != nil {
		return err
	}

	var resp profilesListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := profilesListReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Profiles: %d", len(resp.Profiles))},
			Results:        renderProfiles(resp.Profiles),
			RetrievalHints: []string{"notification-hub profiles get <profile-id>", "notification-hub profiles create --name \"My Team\""},
		},
		Profiles: resp.Profiles,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func runCreate(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	name := fs.String("name", "", "Profile name")
	slug := fs.String("slug", "", "Profile slug")
	plan := fs.String("plan", "free", "Profile plan")
	setDefault := fs.Bool("set-default", false, "Save the created profile as the default profile_id")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("usage: profiles create --name <name> [--slug <slug>] [--plan <plan>] [--set-default]")
	}

	payload := map[string]interface{}{
		"name": strings.TrimSpace(*name),
		"plan": strings.TrimSpace(*plan),
	}
	if strings.TrimSpace(*slug) != "" {
		payload["slug"] = strings.TrimSpace(*slug)
	}

	body, err := d.ScenarioApp().Request("POST", "/admin/profiles", nil, payload)
	if err != nil {
		return err
	}

	var created profile
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if *setDefault && strings.TrimSpace(created.ID) != "" {
		defaults := d.DefaultConfig()
		defaults.ProfileID = created.ID
		if err := d.DefaultsStore().Save(defaults); err != nil {
			return err
		}
	}

	report := profileMutationReport{
		MutationReport: cliapp.MutationReport{
			Result: []string{
				"Profile created",
				"Profile ID: " + created.ID,
			},
			Changes: []string{
				"Name: " + created.Name,
				"Slug: " + created.Slug,
				"Plan: " + support.DefaultString(created.Plan, "free"),
				"API key: " + support.DefaultString(created.APIKey, "(not returned)"),
			},
			NextCommand: []string{
				"notification-hub configure token <api-key>",
				"notification-hub configure profile_id " + created.ID,
				"notification-hub profiles get " + created.ID,
			},
		},
		Profile: created,
	}
	if *setDefault {
		report.Changes = append(report.Changes, "Saved as default profile_id")
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report.MutationReport)
}

func runGet(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	id := strings.TrimSpace(firstArg(fs.Args()))
	if id == "" {
		id = d.ProfileID()
	}
	if id == "" {
		return fmt.Errorf("usage: profiles get <profile-id>")
	}

	body, err := d.ScenarioApp().Get("/admin/profiles/"+id, nil)
	if err != nil {
		return err
	}

	var item profile
	if err := json.Unmarshal(body, &item); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := profileDetailReport{
		ListReport: cliapp.ListReport{
			Summary: []string{
				"Profile: " + item.Name,
				"ID: " + item.ID,
			},
			Results: []string{
				"Slug: " + item.Slug,
				"Plan: " + support.DefaultString(item.Plan, "(unset)"),
				"Status: " + support.DefaultString(item.Status, "(unknown)"),
				"API key prefix: " + support.DefaultString(item.APIKeyPrefix, "(unknown)"),
				"Settings keys: " + fmt.Sprintf("%d", len(item.Settings)),
			},
			RetrievalHints: []string{
				"notification-hub profiles update " + item.ID + " --name \"" + support.DefaultString(item.Name, item.ID) + "\"",
				"notification-hub configure profile_id " + item.ID,
			},
		},
		Profile: item,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func runUpdate(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "Updated profile name")
	plan := fs.String("plan", "", "Updated profile plan")
	status := fs.String("status", "", "Updated profile status")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	id := strings.TrimSpace(firstArg(fs.Args()))
	if id == "" {
		id = d.ProfileID()
	}
	if id == "" {
		return fmt.Errorf("usage: profiles update <profile-id> [--name <name>] [--plan <plan>] [--status <status>]")
	}

	payload := map[string]interface{}{}
	if strings.TrimSpace(*name) != "" {
		payload["name"] = strings.TrimSpace(*name)
	}
	if strings.TrimSpace(*plan) != "" {
		payload["plan"] = strings.TrimSpace(*plan)
	}
	if strings.TrimSpace(*status) != "" {
		payload["status"] = strings.TrimSpace(*status)
	}
	if len(payload) == 0 {
		return fmt.Errorf("provide at least one field to update")
	}

	body, err := d.ScenarioApp().Request("PUT", "/admin/profiles/"+id, nil, payload)
	if err != nil {
		return err
	}

	var updated profile
	if err := json.Unmarshal(body, &updated); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	changes := make([]string, 0, len(payload))
	if _, ok := payload["name"]; ok {
		changes = append(changes, "Name: "+updated.Name)
	}
	if _, ok := payload["plan"]; ok {
		changes = append(changes, "Plan: "+updated.Plan)
	}
	if _, ok := payload["status"]; ok {
		changes = append(changes, "Status: "+updated.Status)
	}

	report := profileMutationReport{
		MutationReport: cliapp.MutationReport{
			Result:      []string{"Profile updated", "Profile ID: " + updated.ID},
			Changes:     changes,
			NextCommand: []string{"notification-hub profiles get " + updated.ID},
		},
		Profile: updated,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report.MutationReport)
}

func renderProfiles(items []profile) []string {
	if len(items) == 0 {
		return nil
	}
	rows := make([]string, 0, len(items))
	for i, item := range items {
		rows = append(rows, fmt.Sprintf("%d. %s | %s | %s | %s", i+1, support.DefaultString(item.Name, item.ID), item.ID, support.DefaultString(item.Plan, "free"), support.DefaultString(item.Status, "unknown")))
	}
	return rows
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
