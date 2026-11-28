package chat

import (
	"fmt"
	"os"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `ai-chatbot-manager chat <chatbot-id>` as a single-shot
// send. The bash CLI implemented an interactive REPL, but per the thin-client
// policy we do not orchestrate prompts here — each invocation posts one
// message and prints the response. Callers wanting a conversation can loop.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Chat",
		Commands: []cliapp.Command{
			{
				Name:        "chat",
				Description: "Send one message to a chatbot and print the response",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runChat(core, args) },
			},
		},
	}
}

func runChat(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chat")
	message := fs.String("message", "", "Message to send (required unless --body-file)")
	sessionID := fs.String("session-id", "", "Session ID (optional; API assigns one if empty)")
	bodyFile := fs.String("body-file", "", "Path to full ChatRequest JSON (overrides --message / --session-id)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chat <chatbot-id> --message TEXT [--session-id ID] | --body-file PATH")
	}
	id := fs.Arg(0)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *message == "" {
			return fmt.Errorf("--message TEXT is required (or supply --body-file PATH)")
		}
		req := map[string]interface{}{
			"message": *message,
			"context": map[string]interface{}{"source": "cli"},
		}
		if *sessionID != "" {
			req["session_id"] = *sessionID
		}
		payload = req
	}

	body, err := core.Request("POST", "/chat/"+id, nil, payload)
	if err != nil {
		return err
	}
	var resp support.ChatResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{fmt.Sprintf("Response: %s", resp.Response)}
	if resp.ConversationID != "" {
		results = append(results, fmt.Sprintf("Conversation: %s", resp.ConversationID))
	}
	results = append(results, fmt.Sprintf("Confidence: %.3f", resp.Confidence))
	if resp.ShouldEscalate {
		reason := resp.EscalationReason
		if reason == "" {
			reason = "(no reason provided)"
		}
		results = append(results, fmt.Sprintf("Escalate: true | reason=%s", reason))
	}
	if len(resp.LeadQualification) > 0 {
		results = append(results, "Lead qualification:")
		results = append(results, support.MapRows(resp.LeadQualification)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chat with %s", id)},
		ResultsHeading: "Response",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s chat %s --message \"follow-up\" --session-id <sid>", support.CLIName, id),
			fmt.Sprintf("%s analytics %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
