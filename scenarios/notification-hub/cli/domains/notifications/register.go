package notifications

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
		Name:        "notifications",
		Description: "Send and inspect notifications",
		Subcommands: []cliapp.Command{
			{Name: "send", NeedsAPI: true, Description: "Send one or more notifications", Run: func(args []string) error { return runSend(d, args) }},
			{Name: "list", NeedsAPI: true, Description: "List notifications for the current profile", Run: func(args []string) error { return runList(d, args) }},
		},
	}
}

type notificationListItem struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Subject   string   `json:"subject"`
	Priority  string   `json:"priority"`
	Channels  []string `json:"channels_requested"`
	ContactID string   `json:"contact_id"`
}

type notificationsListResponse struct {
	Notifications []notificationListItem `json:"notifications"`
}

type sendResponse struct {
	Notifications []string `json:"notifications"`
	Message       string   `json:"message"`
}

type notificationsListReport struct {
	cliapp.ListReport
	ProfileID     string                 `json:"profile_id"`
	Notifications []notificationListItem `json:"notifications"`
}

type sendReport struct {
	cliapp.MutationReport
	ProfileID     string      `json:"profile_id"`
	Notifications []string    `json:"notifications"`
	Request       interface{} `json:"request"`
}

func runSend(d support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	profileFlag := fs.String("profile-id", "", "Profile ID override")
	templateID := fs.String("template-id", "", "Template ID")
	contactID := fs.String("contact-id", "", "Single recipient contact ID")
	subject := fs.String("subject", "", "Notification subject when sending raw content")
	message := fs.String("message", "", "Notification message body when sending raw content")
	channels := fs.String("channels", "email", "Comma-separated channels")
	priority := fs.String("priority", "normal", "Notification priority")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	profileID, err := d.ResolveProfileID(*profileFlag)
	if err != nil {
		return err
	}
	recipientID := strings.TrimSpace(*contactID)
	if recipientID == "" {
		return fmt.Errorf("usage: notifications send --contact-id <contact-id> [--template-id <id> | --subject <text> --message <text>]")
	}

	payload := map[string]interface{}{
		"recipients": []map[string]interface{}{
			{
				"contact_id": recipientID,
				"variables":  map[string]interface{}{},
			},
		},
		"channels": cliutil.ParseCSV(*channels),
		"priority": strings.TrimSpace(*priority),
	}
	if len(payload["channels"].([]string)) == 0 {
		payload["channels"] = []string{"email"}
	}

	if tid := strings.TrimSpace(*templateID); tid != "" {
		payload["template_id"] = tid
	} else {
		subj := strings.TrimSpace(*subject)
		msg := strings.TrimSpace(*message)
		if subj == "" {
			subj = "Test Notification"
		}
		if msg == "" {
			msg = "Test message from Notification Hub"
		}
		payload["subject"] = subj
		payload["content"] = map[string]interface{}{
			"email": map[string]interface{}{
				"text": msg,
				"html": "<p>" + msg + "</p>",
			},
		}
	}

	body, err := d.ScopedRequest(profileID, "POST", "/notifications/send", nil, payload)
	if err != nil {
		return err
	}

	var resp sendResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := sendReport{
		MutationReport: cliapp.MutationReport{
			Result: []string{
				"Notifications created",
				fmt.Sprintf("Created: %d", len(resp.Notifications)),
			},
			Changes: []string{
				"Profile ID: " + profileID,
				"Contact ID: " + recipientID,
				"Channels: " + strings.Join(payload["channels"].([]string), ", "),
				"Priority: " + strings.TrimSpace(*priority),
			},
			NextCommand: []string{
				"notification-hub notifications list --profile-id " + profileID,
				"notification-hub analytics delivery-stats --profile-id " + profileID,
			},
		},
		ProfileID:     profileID,
		Notifications: resp.Notifications,
		Request:       payload,
	}
	if msg := strings.TrimSpace(resp.Message); msg != "" {
		report.Changes = append(report.Changes, "API: "+msg)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report.MutationReport)
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

	body, err := d.ScopedGet(profileID, "/notifications", nil)
	if err != nil {
		return err
	}

	var resp notificationsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := notificationsListReport{
		ListReport: cliapp.ListReport{
			Summary:        []string{"Profile ID: " + profileID, fmt.Sprintf("Notifications: %d", len(resp.Notifications))},
			Results:        renderNotifications(resp.Notifications),
			RetrievalHints: []string{"notification-hub notifications send --profile-id " + profileID + " --contact-id <contact-id>"},
		},
		ProfileID:     profileID,
		Notifications: resp.Notifications,
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report.ListReport)
}

func renderNotifications(items []notificationListItem) []string {
	if len(items) == 0 {
		return nil
	}
	rows := make([]string, 0, len(items))
	for i, item := range items {
		rows = append(rows, fmt.Sprintf("%d. %s | %s | %s | %s | contact %s", i+1, support.DefaultString(item.Subject, item.ID), support.DefaultString(item.Status, "unknown"), support.DefaultString(item.Priority, "normal"), strings.Join(item.Channels, ", "), support.DefaultString(item.ContactID, "(unknown)")))
	}
	return rows
}
