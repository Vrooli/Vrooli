package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdQueueList(args []string) error {
	fs := flag.NewFlagSet("queue list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Get("/queue", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[QueueListResponse](body)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No queue items found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("queue", "create", "--kind", "backlog"),
			cliCommand("backlog", "list"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d queue item(s)\n", len(response.Items))
	printSection("Results")
	for _, item := range response.Items {
		fmt.Printf("  %s (%s)\n", item.ID, item.Kind)
		fmt.Printf("    Created: %s\n", item.Created)
		if len(item.Payload) > 0 {
			fmt.Printf("    Payload: %s\n", string(item.Payload))
		}
		fmt.Println()
	}
	first := response.Items[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("queue", "delete", "--id", "<id>"),
		cliCommand("queue", "delete", "--id", first.ID),
	})
	return nil
}

func (a *App) cmdQueueCreate(args []string) error {
	fs := flag.NewFlagSet("queue create", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Queue item kind")
	data := fs.String("data", "", "Optional JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("kind", *kindFlag); err != nil {
		return fmt.Errorf("usage: queue create --kind KIND [--data JSON] [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)

	payload := map[string]any{"kind": kind}
	if strings.TrimSpace(*data) != "" {
		raw, err := parseJSONString(*data)
		if err != nil {
			return fmt.Errorf("invalid payload JSON: %w", err)
		}
		payload["payload"] = raw
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	body, err := a.core.Request("POST", "/queue", nil, json.RawMessage(requestBody))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	printSection("Result")
	fmt.Printf("  Created queue item for kind: %s\n", kind)
	printSection("What Changed")
	cliutil.PrintJSON(body)
	printCommandListSection("Next Steps", []string{
		cliCommand("queue", "list"),
		cliCommand("queue", "delete", "--id", "<id>"),
	})
	return nil
}

func (a *App) cmdQueueDelete(args []string) error {
	fs := flag.NewFlagSet("queue delete", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Queue item ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: queue delete --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.core.Request("DELETE", "/queue/"+id, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	fmt.Printf("  Deleted queue item: %s\n", id)
	printCommandListSection("Next Steps", []string{
		cliCommand("queue", "list"),
	})
	return nil
}
