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

	body, err := a.getV1("/queue", nil)
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
			cliCommand("queue", "create", "backlog"),
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
		cliCommand("queue", "delete", "<id>"),
		cliCommand("queue", "delete", first.ID),
	})
	return nil
}

func (a *App) cmdQueueCreate(args []string) error {
	fs := flag.NewFlagSet("queue create", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: queue create <kind> [payload-json-or-@file] [--json]")
	}
	kind := strings.TrimSpace(fs.Arg(0))
	if kind == "" {
		return fmt.Errorf("kind is required")
	}

	payload := map[string]any{"kind": kind}
	if fs.NArg() > 1 {
		raw, err := parseJSONArg(fs.Args()[1:])
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
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	printSection("Result")
	fmt.Printf("  Created queue item for kind: %s\n", kind)
	printSection("What Changed")
	cliutil.PrintJSON(body)
	printCommandListSection("Next Steps", []string{
		cliCommand("queue", "list"),
		cliCommand("queue", "delete", "<id>"),
	})
	return nil
}

func (a *App) cmdQueueDelete(args []string) error {
	fs := flag.NewFlagSet("queue delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: queue delete <id> [--json]")
	}
	id := strings.TrimSpace(fs.Arg(0))
	if id == "" {
		return fmt.Errorf("id is required")
	}

	body, err := a.requestV1("DELETE", "/queue/"+id, nil, nil)
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
