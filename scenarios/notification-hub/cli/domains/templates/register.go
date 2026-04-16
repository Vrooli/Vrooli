package templates

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
		Name:        "templates",
		Description: "Inspect notification templates",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List templates for the current profile", Run: func(args []string) error { return runList(d, args) }},
		},
	}
}

type template struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Slug     string   `json:"slug"`
	Channels []string `json:"channels"`
	Status   string   `json:"status"`
}

type templatesListResponse struct {
	Templates []template `json:"templates"`
}

type templatesListReport struct {
	cliapp.ListReport
	ProfileID string     `json:"profile_id"`
	Templates []template `json:"templates"`
}

func runList(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	profileFlag := fs.String("profile-id", "", "Profile ID override")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	profileID, err := d.ResolveProfileID(*profileFlag)
	if err != nil {
		return err
	}

	body, err := d.ScopedGet(profileID, "/templates", nil)
	if err != nil {
		return err
	}

	var resp templatesListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := templatesListReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{"Profile ID: " + profileID, fmt.Sprintf("Templates: %d", len(resp.Templates))},
			Results:        renderTemplates(resp.Templates),
			RetrievalHints: []string{"notification-hub notifications send --profile-id " + profileID + " --template-id <template-id> --contact-id <contact-id>"},
		},
		ProfileID: profileID,
		Templates: resp.Templates,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func renderTemplates(items []template) []string {
	if len(items) == 0 {
		return nil
	}
	rows := make([]string, 0, len(items))
	for i, item := range items {
		rows = append(rows, fmt.Sprintf("%d. %s | %s | %s", i+1, support.DefaultString(item.Name, item.ID), support.DefaultString(item.Slug, "(no slug)"), strings.Join(item.Channels, ", ")))
	}
	return rows
}
