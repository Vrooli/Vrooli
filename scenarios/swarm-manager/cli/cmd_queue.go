package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdQueueList(_ []string) error {
	body, err := a.getV1("/queue", nil)
	if err != nil {
		return err
	}

	var response QueueListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Items) == 0 {
		fmt.Println("No queue items found.")
		return nil
	}

	fmt.Printf("Found %d queue item(s):\n\n", len(response.Items))
	for _, item := range response.Items {
		fmt.Printf("  %s (%s)\n", item.ID, item.Kind)
		fmt.Printf("    Created: %s\n", item.Created)
		if len(item.Payload) > 0 {
			fmt.Printf("    Payload: %s\n", string(item.Payload))
		}
		fmt.Println()
	}
	return nil
}

func (a *App) cmdQueueCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: queue create <kind> [payload-json-or-@file]")
	}
	kind := strings.TrimSpace(args[0])
	if kind == "" {
		return fmt.Errorf("kind is required")
	}

	payload := map[string]any{"kind": kind}
	if len(args) > 1 {
		raw, err := parseJSONArg(args[1:])
		if err != nil {
			return fmt.Errorf("invalid payload JSON: %w", err)
		}
		payload["payload"] = raw
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	body, err := a.requestV1("POST", "/queue", nil, json.RawMessage(requestBody))
	if err != nil {
		return err
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdQueueDelete(args []string) error {
	if err := requireArgs(args, 1, "queue delete <id>"); err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("id is required")
	}

	_, err := a.requestV1("DELETE", "/queue/"+id, nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted queue item: %s\n", id)
	return nil
}
