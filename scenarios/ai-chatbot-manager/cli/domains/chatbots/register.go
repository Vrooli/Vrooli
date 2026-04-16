package chatbots

import (
	"fmt"
	"os"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires `ai-chatbot-manager chatbot ...` covering list/get/create/update/delete
// plus the widget embed endpoint. Update/create accept `--body-file` for nested JSON.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "chatbot",
		Description: "Manage chatbots (create, list, get, update, delete, widget)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List chatbots", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one chatbot", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a new chatbot", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update an existing chatbot (--body-file PATH required)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a chatbot", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "widget", Description: "Print the widget embed code for a chatbot", Run: func(args []string) error { return runWidget(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chatbot list")
	activeOnly := fs.Bool("active-only", false, "Show only active chatbots (forwarded as ?active=true)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{})
	if *activeOnly {
		query.Set("active", "true")
	}

	body, err := core.Get("/chatbots", query)
	if err != nil {
		return err
	}
	var bots []support.Chatbot
	if err := support.Decode(body, &bots); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chatbots: %d", len(bots))},
		ResultsHeading: "Chatbots",
		Results:        chatbotRows(bots),
		RetrievalHints: []string{
			fmt.Sprintf("%s chatbot get <chatbot-id>", support.CLIName),
			fmt.Sprintf("%s chatbot widget <chatbot-id>", support.CLIName),
			fmt.Sprintf("%s analytics <chatbot-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chatbot get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chatbot get <chatbot-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/chatbots/"+id, nil)
	if err != nil {
		return err
	}
	var bot support.Chatbot
	if err := support.Decode(body, &bot); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", bot.ID),
		fmt.Sprintf("Name: %s", bot.Name),
		fmt.Sprintf("Active: %t", bot.IsActive),
	}
	if bot.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", bot.Description))
	}
	if bot.Personality != "" {
		results = append(results, fmt.Sprintf("Personality: %s", bot.Personality))
	}
	if bot.TenantID != "" {
		results = append(results, fmt.Sprintf("Tenant: %s", bot.TenantID))
	}
	if model, ok := bot.ModelConfig["model"].(string); ok && model != "" {
		results = append(results, fmt.Sprintf("Model: %s", model))
	}
	results = append(results,
		fmt.Sprintf("Created: %s", support.FormatTimeValue(bot.CreatedAt)),
		fmt.Sprintf("Updated: %s", support.FormatTimeValue(bot.UpdatedAt)),
	)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chatbot: %s", bot.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s chatbot widget %s", support.CLIName, bot.ID),
			fmt.Sprintf("%s analytics %s", support.CLIName, bot.ID),
			fmt.Sprintf("%s escalations list %s", support.CLIName, bot.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chatbot create")
	name := fs.String("name", "", "Chatbot name (required unless --body-file supplies it)")
	personality := fs.String("personality", "", "Chatbot personality instruction")
	description := fs.String("description", "", "Chatbot description")
	knowledgeBase := fs.String("knowledge-base", "", "Knowledge base text")
	model := fs.String("model", "", "Model name (e.g. llama3.2); populates model_config.model")
	bodyFile := fs.String("body-file", "", "Path to full CreateChatbotRequest JSON (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	// Allow a single positional `<name>` for parity with the old bash CLI.
	if *name == "" && fs.NArg() >= 1 {
		*name = fs.Arg(0)
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *name == "" {
			return fmt.Errorf("usage: chatbot create <name> [--personality TEXT] [--description TEXT] [--knowledge-base TEXT] [--model NAME] | --body-file PATH")
		}
		req := map[string]interface{}{"name": *name}
		if *description != "" {
			req["description"] = *description
		}
		if *personality != "" {
			req["personality"] = *personality
		}
		if *knowledgeBase != "" {
			req["knowledge_base"] = *knowledgeBase
		}
		if *model != "" {
			req["model_config"] = map[string]interface{}{
				"model":       *model,
				"temperature": 0.7,
				"max_tokens":  1000,
			}
		}
		payload = req
	}

	body, err := core.Request("POST", "/chatbots", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ChatbotCreateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("Created chatbot %s (%s)", resp.Chatbot.Name, support.ShortID(resp.Chatbot.ID))}
	result := []string{
		fmt.Sprintf("ID: %s", resp.Chatbot.ID),
		fmt.Sprintf("Name: %s", resp.Chatbot.Name),
	}
	if resp.WidgetEmbedCode != "" {
		result = append(result, fmt.Sprintf("Widget embed code: %s", resp.WidgetEmbedCode))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s chatbot get %s", support.CLIName, resp.Chatbot.ID),
			fmt.Sprintf("%s chatbot widget %s", support.CLIName, resp.Chatbot.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chatbot update")
	bodyFile := fs.String("body-file", "", "Path to UpdateChatbotRequest JSON (required)")
	method := fs.String("method", "PUT", "HTTP verb to use (PUT or PATCH)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chatbot update <chatbot-id> --body-file PATH [--method PUT|PATCH]")
	}
	id := fs.Arg(0)
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request(*method, "/chatbots/"+id, nil, payload)
	if err != nil {
		return err
	}
	var bot support.Chatbot
	if err := support.Decode(body, &bot); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("ID: %s", bot.ID),
			fmt.Sprintf("Name: %s", bot.Name),
			fmt.Sprintf("Active: %t", bot.IsActive),
			fmt.Sprintf("Updated: %s", support.FormatTimeValue(bot.UpdatedAt)),
		},
		Changes:     []string{fmt.Sprintf("Updated chatbot %s", id)},
		NextCommand: []string{fmt.Sprintf("%s chatbot get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chatbot delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chatbot delete <chatbot-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/chatbots/"+id, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Chatbot %s deleted", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Deleted chatbot %s", id)},
		NextCommand: []string{fmt.Sprintf("%s chatbot list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runWidget(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chatbot widget")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chatbot widget <chatbot-id>")
	}
	id := fs.Arg(0)

	// The /chatbots/{id}/widget endpoint returns text/html (embed snippet), not JSON.
	// Expose the raw payload as a single-result ListReport.
	body, err := core.Get("/chatbots/"+id+"/widget", nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Widget embed code for chatbot %s", id)},
		ResultsHeading: "Embed",
		Results:        []string{string(body)},
		RetrievalHints: []string{fmt.Sprintf("%s chatbot get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func chatbotRows(bots []support.Chatbot) []string {
	if len(bots) == 0 {
		return []string{"No chatbots registered"}
	}
	rows := make([]string, 0, len(bots))
	for _, b := range bots {
		active := "inactive"
		if b.IsActive {
			active = "active"
		}
		model := ""
		if m, ok := b.ModelConfig["model"].(string); ok {
			model = m
		}
		created := support.FormatTimeValue(b.CreatedAt)
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | model=%s | created=%s",
			b.Name, support.ShortID(b.ID), active, model, created))
	}
	return rows
}
