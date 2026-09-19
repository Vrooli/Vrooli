package webhooks

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `webhooks` subcommands covering the webhook subscription
// lifecycle (list, create, delete, test).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "webhooks",
		Description: "Manage webhook subscriptions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List webhook subscriptions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a new webhook subscription", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Description: "Delete a webhook subscription", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "test", Description: "Send a test event to a webhook", Run: func(args []string) error { return runTest(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("webhooks list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := core.Get("/webhooks", nil)
	if err != nil {
		return err
	}
	var hooks []support.WebhookSubscription
	if err := support.Decode(raw, &hooks); err != nil {
		return err
	}
	rows := make([]string, 0, len(hooks))
	for _, h := range hooks {
		active := "disabled"
		if h.Active {
			active = "active"
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | events=%s | failures=%d",
			h.URL, support.ShortID(h.ID), active, strings.Join(h.Events, ","), h.FailureCount))
	}
	if len(rows) == 0 {
		rows = []string{"(no webhooks)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Webhooks: %d", len(hooks))},
		ResultsHeading: "Subscriptions",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("webhooks create")
	url := fs.String("url", "", "Webhook URL")
	events := fs.String("events", "", "Comma-separated event list (e.g. api.created,api.updated)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full subscription body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if strings.TrimSpace(*url) == "" || strings.TrimSpace(*events) == "" {
			return fmt.Errorf("usage: webhooks create --url URL --events evt1,evt2 [--body-file PATH]")
		}
		body = map[string]interface{}{
			"url":    *url,
			"events": splitCSV(*events),
		}
	}

	raw, err := core.Request("POST", "/webhooks", nil, body)
	if err != nil {
		return err
	}
	var created support.WebhookSubscription
	if err := support.Decode(raw, &created); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created webhook %s for %s", created.ID, created.URL)},
		Changes:     []string{"POST /webhooks"},
		NextCommand: []string{fmt.Sprintf("%s webhooks test %s", support.CLIName, created.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("webhooks delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: webhooks delete <webhook-id>")
	}
	id := fs.Arg(0)
	if _, err := core.Request("DELETE", "/webhooks/"+id, nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Deleted webhook %s", id)},
		Changes: []string{fmt.Sprintf("DELETE /webhooks/%s", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runTest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("webhooks test")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: webhooks test <webhook-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Request("POST", "/webhooks/"+id+"/test", nil, nil)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	_ = support.Decode(raw, &resp)
	report := cliapp.MutationReport{
		Result:  support.MapRows(resp),
		Changes: []string{fmt.Sprintf("POST /webhooks/%s/test", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
