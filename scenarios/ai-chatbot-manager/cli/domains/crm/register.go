package crm

import (
	"fmt"
	"os"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires `ai-chatbot-manager crm ...` covering CRM integration create
// and conversation sync endpoints. `integrate` takes --body-file because
// provider configs vary (salesforce/hubspot/pipedrive/webhook).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "crm",
		Description: "Manage CRM integrations and conversation sync",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "integrate", Description: "Create a CRM integration (--body-file PATH required)", Run: func(args []string) error { return runIntegrate(core, args) }},
			{Name: "sync", Description: "Sync a conversation's lead to CRM", Run: func(args []string) error { return runSync(core, args) }},
		},
	}
}

func runIntegrate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("crm integrate")
	bodyFile := fs.String("body-file", "", "Path to CRM integration JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/crm-integrations", nil, payload)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		if m, ok := resp["message"].(string); ok && m != "" {
			message = m
		} else {
			message = "CRM integration created"
		}
	}

	report := cliapp.MutationReport{
		Result:  append([]string{message}, support.MapRows(resp)...),
		Changes: []string{message},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSync(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("crm sync")
	bodyFile := fs.String("body-file", "", "Optional JSON payload with sync overrides")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: crm sync <conversation-id> [--body-file PATH]")
	}
	convID := fs.Arg(0)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	}

	body, err := core.Request("POST", "/conversations/"+convID+"/sync-crm", nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Conversation %s CRM sync issued", convID)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Conversation %s -> CRM sync", convID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
