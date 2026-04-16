package contacts

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
		Name:        "contacts",
		Description: "Manage profile contacts",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List contacts for the current profile", Run: func(args []string) error { return runList(d, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a contact for the current profile", Run: func(args []string) error { return runCreate(d, args) }},
		},
	}
}

type contact struct {
	ID         string  `json:"id"`
	ProfileID  string  `json:"profile_id"`
	ExternalID *string `json:"external_id,omitempty"`
	Identifier string  `json:"identifier"`
	FirstName  *string `json:"first_name,omitempty"`
	LastName   *string `json:"last_name,omitempty"`
	Timezone   string  `json:"timezone,omitempty"`
	Locale     string  `json:"locale,omitempty"`
	Status     string  `json:"status,omitempty"`
}

type contactsListResponse struct {
	Contacts []contact `json:"contacts"`
}

type contactsListReport struct {
	cliapp.ListReport
	ProfileID string    `json:"profile_id"`
	Contacts  []contact `json:"contacts"`
}

type contactMutationReport struct {
	cliapp.MutationReport
	ProfileID string  `json:"profile_id"`
	Contact   contact `json:"contact"`
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

	body, err := d.ScopedGet(profileID, "/contacts", nil)
	if err != nil {
		return err
	}

	var resp contactsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := contactsListReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{"Profile ID: " + profileID, fmt.Sprintf("Contacts: %d", len(resp.Contacts))},
			Results:        renderContacts(resp.Contacts),
			RetrievalHints: []string{"notification-hub contacts create --profile-id " + profileID + " --identifier user@example.com"},
		},
		ProfileID: profileID,
		Contacts:  resp.Contacts,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func runCreate(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	profileFlag := fs.String("profile-id", "", "Profile ID override")
	identifier := fs.String("identifier", "", "Primary contact identifier, typically email or phone")
	externalID := fs.String("external-id", "", "External contact ID")
	firstName := fs.String("first-name", "", "First name")
	lastName := fs.String("last-name", "", "Last name")
	timezone := fs.String("timezone", "UTC", "Timezone")
	locale := fs.String("locale", "en-US", "Locale")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	profileID, err := d.ResolveProfileID(*profileFlag)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*identifier) == "" {
		return fmt.Errorf("usage: contacts create --identifier <email-or-phone> [--profile-id <id>]")
	}

	payload := map[string]interface{}{
		"identifier": strings.TrimSpace(*identifier),
		"timezone":   strings.TrimSpace(*timezone),
		"locale":     strings.TrimSpace(*locale),
	}
	if strings.TrimSpace(*externalID) != "" {
		payload["external_id"] = strings.TrimSpace(*externalID)
	}
	if strings.TrimSpace(*firstName) != "" {
		payload["first_name"] = strings.TrimSpace(*firstName)
	}
	if strings.TrimSpace(*lastName) != "" {
		payload["last_name"] = strings.TrimSpace(*lastName)
	}

	body, err := d.ScopedRequest(profileID, "POST", "/contacts", nil, payload)
	if err != nil {
		return err
	}

	var created contact
	if err := json.Unmarshal(body, &created); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := contactMutationReport{
		MutationReport: cliapp.MutationReport{
			Result: []string{"Contact created", "Contact ID: " + created.ID},
			Changes: []string{
				"Profile ID: " + profileID,
				"Identifier: " + created.Identifier,
				"Timezone: " + support.DefaultString(created.Timezone, "UTC"),
				"Locale: " + support.DefaultString(created.Locale, "en-US"),
			},
			NextCommand: []string{
				"notification-hub contacts list --profile-id " + profileID,
				"notification-hub notifications send --profile-id " + profileID + " --contact-id " + created.ID,
			},
		},
		ProfileID: profileID,
		Contact:   created,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report.MutationReport)
}

func renderContacts(items []contact) []string {
	if len(items) == 0 {
		return nil
	}
	rows := make([]string, 0, len(items))
	for i, item := range items {
		name := strings.TrimSpace(strings.Join([]string{support.DerefString(item.FirstName), support.DerefString(item.LastName)}, " "))
		if name == "" {
			name = "(unnamed)"
		}
		rows = append(rows, fmt.Sprintf("%d. %s | %s | %s | %s", i+1, item.Identifier, name, support.DefaultString(item.Locale, "en-US"), support.DefaultString(item.Status, "unknown")))
	}
	return rows
}
