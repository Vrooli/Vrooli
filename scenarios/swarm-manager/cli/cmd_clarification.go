package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogClarify(args []string) error {
	fs := flag.NewFlagSet("backlog clarify", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog kind (required)")
	nameFlag := fs.String("name", "", "Backlog item name (required)")
	roundFlag := fs.Int("round", 0, "Workshop round number (required)")
	itemFlag := fs.String("item", "", "Decision item ID (required)")
	messageFlag := fs.String("message", "", "Clarification question (empty = generic explanation)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *kindFlag == "" || *nameFlag == "" || *roundFlag == 0 || *itemFlag == "" {
		return fmt.Errorf("usage: backlog clarify --kind KIND --name NAME --round N --item ITEM_ID [--message MSG]")
	}

	payload := map[string]any{
		"round_number": *roundFlag,
		"item_id":      *itemFlag,
		"message":      *messageFlag,
	}

	path := fmt.Sprintf("/backlog/%s/%s/workshop/clarification", *kindFlag, *nameFlag)
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var resp struct {
		Thread struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			RunID  string `json:"run_id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	printSection("Clarification Started")
	fmt.Printf("  Thread ID: %s\n", resp.Thread.ID)
	fmt.Printf("  Status:    %s\n", resp.Thread.Status)
	fmt.Printf("  Run ID:    %s\n", resp.Thread.RunID)

	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "clarify-get", "--kind", *kindFlag, "--name", *nameFlag, "--thread", resp.Thread.ID),
	})
	return nil
}

func (a *App) cmdBacklogClarifyGet(args []string) error {
	fs := flag.NewFlagSet("backlog clarify-get", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog kind (required)")
	nameFlag := fs.String("name", "", "Backlog item name (required)")
	threadFlag := fs.String("thread", "", "Clarification thread ID (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *kindFlag == "" || *nameFlag == "" || *threadFlag == "" {
		return fmt.Errorf("usage: backlog clarify-get --kind KIND --name NAME --thread THREAD_ID")
	}

	path := fmt.Sprintf("/backlog/%s/%s/workshop/clarification/%s", *kindFlag, *nameFlag, *threadFlag)
	body, err := a.core.Get(path, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var resp struct {
		Thread struct {
			ID       string `json:"id"`
			Status   string `json:"status"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			LatestImpact *struct {
				Level       string `json:"level"`
				Reasoning   string `json:"reasoning"`
				ContextNote string `json:"context_note"`
			} `json:"latest_impact"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	printSection("Clarification Thread")
	fmt.Printf("  Thread ID: %s\n", resp.Thread.ID)
	fmt.Printf("  Status:    %s\n", resp.Thread.Status)
	fmt.Printf("  Messages:  %d\n", len(resp.Thread.Messages))

	printSection("Conversation")
	for _, msg := range resp.Thread.Messages {
		fmt.Printf("\n  [%s]\n", strings.ToUpper(msg.Role))
		for _, line := range strings.Split(msg.Content, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}

	if resp.Thread.LatestImpact != nil {
		printSection("Impact Assessment")
		fmt.Printf("  Level:        %s\n", resp.Thread.LatestImpact.Level)
		fmt.Printf("  Reasoning:    %s\n", resp.Thread.LatestImpact.Reasoning)
		fmt.Printf("  Context Note: %s\n", resp.Thread.LatestImpact.ContextNote)
	}

	if resp.Thread.Status == "active" {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "clarify-continue", "--kind", *kindFlag, "--name", *nameFlag, "--thread", resp.Thread.ID, "--message", "\"follow-up question\""),
			cliCommand("backlog", "clarify-action", "--kind", *kindFlag, "--name", *nameFlag, "--thread", resp.Thread.ID, "--action", "got_it"),
		})
	}
	return nil
}

func (a *App) cmdBacklogClarifyContinue(args []string) error {
	fs := flag.NewFlagSet("backlog clarify-continue", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog kind (required)")
	nameFlag := fs.String("name", "", "Backlog item name (required)")
	threadFlag := fs.String("thread", "", "Clarification thread ID (required)")
	messageFlag := fs.String("message", "", "Follow-up message (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *kindFlag == "" || *nameFlag == "" || *threadFlag == "" || *messageFlag == "" {
		return fmt.Errorf("usage: backlog clarify-continue --kind KIND --name NAME --thread THREAD_ID --message MSG")
	}

	payload := map[string]any{
		"message": *messageFlag,
	}

	path := fmt.Sprintf("/backlog/%s/%s/workshop/clarification/%s/continue", *kindFlag, *nameFlag, *threadFlag)
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	fmt.Printf("Follow-up sent. Poll with:\n  %s\n",
		cliCommand("backlog", "clarify-get", "--kind", *kindFlag, "--name", *nameFlag, "--thread", *threadFlag))
	return nil
}

func (a *App) cmdBacklogClarifyAction(args []string) error {
	fs := flag.NewFlagSet("backlog clarify-action", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog kind (required)")
	nameFlag := fs.String("name", "", "Backlog item name (required)")
	threadFlag := fs.String("thread", "", "Clarification thread ID (required)")
	actionFlag := fs.String("action", "", "Action: got_it, update_decision, remove_decision, invalidate_round (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *kindFlag == "" || *nameFlag == "" || *threadFlag == "" || *actionFlag == "" {
		return fmt.Errorf("usage: backlog clarify-action --kind KIND --name NAME --thread THREAD_ID --action ACTION")
	}

	validActions := map[string]bool{"got_it": true, "update_decision": true, "remove_decision": true, "invalidate_round": true}
	if !validActions[*actionFlag] {
		return fmt.Errorf("invalid action %q: must be one of got_it, update_decision, remove_decision, invalidate_round", *actionFlag)
	}

	payload := map[string]any{
		"action": *actionFlag,
	}

	path := fmt.Sprintf("/backlog/%s/%s/workshop/clarification/%s/action", *kindFlag, *nameFlag, *threadFlag)
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var resp struct {
		Action  string `json:"action"`
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	printSection("Action Result")
	fmt.Printf("  Action:  %s\n", resp.Action)
	fmt.Printf("  Success: %v\n", resp.Success)
	fmt.Printf("  Message: %s\n", resp.Message)
	return nil
}
