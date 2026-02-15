package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"
)

func (a *App) cmdBacklogList(args []string) error {
	query := url.Values{}
	if len(args) > 0 {
		kinds := strings.Join(args, ",")
		if strings.TrimSpace(kinds) != "" {
			query.Set("kinds", kinds)
		}
	}

	body, err := a.getV1("/backlog", query)
	if err != nil {
		return err
	}

	var response ListBacklogResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Items) == 0 {
		fmt.Println("No backlog items found.")
		return nil
	}

	fmt.Printf("Found %d backlog item(s):\n\n", len(response.Items))
	for _, item := range response.Items {
		fmt.Printf("  [%s] %s (priority: %d, status: %s)\n", item.Kind, item.Name, item.Priority, item.Status)
		fmt.Printf("    Title: %s\n", item.Title)
		if len(item.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(item.Tags, ", "))
		}
		if item.Kind == "research" && item.ResearchTarget != "" {
			fmt.Printf("    Target: %s\n", item.ResearchTarget)
		}
		fmt.Println()
	}
	return nil
}

func (a *App) cmdBacklogGet(args []string) error {
	if err := requireArgs(args, 2, "backlog get <kind> <name>"); err != nil {
		return err
	}
	kind := args[0]
	name := args[1]

	body, err := a.getV1("/backlog/"+kind+"/"+name, nil)
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	item := response.Item

	fmt.Printf("Name: %s\n", item.Name)
	fmt.Printf("Kind: %s\n", item.Kind)
	fmt.Printf("Title: %s\n", item.Title)
	fmt.Printf("Description: %s\n", item.Description)
	fmt.Printf("Status: %s\n", item.Status)
	fmt.Printf("Priority: %d\n", item.Priority)
	if len(item.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if item.ResearchTarget != "" {
		fmt.Printf("Research Target: %s\n", item.ResearchTarget)
	}
	fmt.Printf("Created: %s\n", item.Created)
	fmt.Printf("Updated: %s\n", item.Updated)
	return nil
}

func (a *App) cmdBacklogCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: backlog create <json-or-@file>\n\nExample:\n  backlog create '{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'")
	}

	payload, err := parseJSONArg(args)
	if err != nil {
		return err
	}

	var req CreateBacklogRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Name == "" || req.Title == "" || req.Kind == "" {
		return fmt.Errorf("name, title, and kind are required fields")
	}

	body, err := a.requestV1("POST", "/backlog", nil, payload)
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	item := response.Item

	fmt.Printf("Created backlog item: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	return nil
}

func (a *App) cmdBacklogUpdate(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: backlog update <kind> <name> <json-or-@file>\n\nExample:\n  backlog update idea my-idea '{\"title\":\"Updated Title\",\"status\":\"ready\"}'")
	}

	kind := args[0]
	name := args[1]
	payload, err := parseJSONArg(args[2:])
	if err != nil {
		return err
	}

	var update map[string]any
	if err := json.Unmarshal(payload, &update); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.requestV1("PUT", "/backlog/"+kind+"/"+name, nil, payload)
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	item := response.Item

	fmt.Printf("Updated backlog item: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	return nil
}

func (a *App) cmdBacklogDelete(args []string) error {
	if err := requireArgs(args, 2, "backlog delete <kind> <name>"); err != nil {
		return err
	}
	kind := args[0]
	name := args[1]

	_, err := a.requestV1("DELETE", "/backlog/"+kind+"/"+name, nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted backlog item: %s (%s)\n", name, kind)
	return nil
}

func (a *App) cmdBacklogFiles(args []string) error {
	if err := requireArgs(args, 2, "backlog files <kind> <name>"); err != nil {
		return err
	}
	kind := strings.TrimSpace(args[0])
	name := strings.TrimSpace(args[1])

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/files", nil)
	if err != nil {
		return err
	}

	var response BacklogFilesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if len(response.Files) == 0 {
		fmt.Println("No files found.")
		return nil
	}

	fmt.Printf("Files for %s/%s:\n", kind, name)
	printTree(response.Files,
		func(item BacklogFile) []BacklogFile { return item.Children },
		func(item BacklogFile) string {
			if item.Type == "directory" {
				return item.Name + "/"
			}
			return fmt.Sprintf("%s (%d bytes)", item.Name, item.Size)
		},
		0,
	)
	return nil
}

func (a *App) cmdBacklogQueue(args []string) error {
	fs := flag.NewFlagSet("backlog queue", flag.ContinueOnError)
	mode := fs.String("mode", "", "Execution mode override: manual|scheduled|yolo (default uses execution policy)")
	delaySeconds := fs.Int64("delay-seconds", 0, "Schedule delay in seconds (scheduled mode)")
	operation := fs.String("operation", "generator", "Operation hint: generator|improver")
	startedBy := fs.String("started-by", "swarm-manager", "Started-by attribution label")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog queue <kind> <name> [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver] [--started-by NAME]")
	}
	kind := fs.Arg(0)
	name := fs.Arg(1)
	modeValue := strings.ToLower(strings.TrimSpace(*mode))
	if modeValue != "" && modeValue != "manual" && modeValue != "scheduled" && modeValue != "yolo" {
		return fmt.Errorf("invalid mode %q (expected manual, scheduled, or yolo)", modeValue)
	}
	operationValue := strings.ToLower(strings.TrimSpace(*operation))
	if operationValue != "generator" && operationValue != "improver" {
		return fmt.Errorf("invalid operation %q (expected generator or improver)", operationValue)
	}
	if *delaySeconds < 0 {
		return fmt.Errorf("delay-seconds must be >= 0")
	}
	payload, err := json.Marshal(map[string]any{
		"operation":     operationValue,
		"mode":          modeValue,
		"delay_seconds": *delaySeconds,
		"started_by":    strings.TrimSpace(*startedBy),
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/queue", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}

	var response QueueBacklogResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Queued backlog item: %s\n", response.Item.Name)
	fmt.Printf("  Kind: %s\n", response.Item.Kind)
	fmt.Printf("  Status: %s\n", response.Item.Status)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	if response.RunID != "" {
		fmt.Printf("  Run ID: %s\n", response.RunID)
	}
	fmt.Printf("  Mode: %s\n", modeValue)
	if modeValue == "scheduled" {
		fmt.Printf("  Delay Seconds: %d\n", *delaySeconds)
	}
	return nil
}

func (a *App) cmdBacklogResearch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: backlog research <kind> <name> [json-or-@file]\n\nExample:\n  backlog research idea my-idea '{\"prompt\":\"Focus on risks\"}'")
	}
	kind := args[0]
	name := args[1]

	var payload json.RawMessage
	if len(args) > 2 {
		parsed, err := parseJSONArg(args[2:])
		if err != nil {
			return err
		}
		payload = parsed
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/research", nil, payload)
	if err != nil {
		return err
	}

	var response ResearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Research started for backlog item: %s\n", name)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	fmt.Printf("  Run ID: %s\n", response.RunID)
	fmt.Printf("  Base URL: %s\n", response.BaseURL)
	return nil
}

func (a *App) cmdBacklogConvert(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: backlog convert <kind> <name> <target-kind> [target-name]")
	}
	kind := args[0]
	name := args[1]
	targetKind := args[2]
	targetName := ""
	if len(args) > 3 {
		targetName = strings.Join(args[3:], " ")
	}

	payload := map[string]string{
		"targetKind": targetKind,
	}
	if strings.TrimSpace(targetName) != "" {
		payload["targetName"] = targetName
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/convert", nil, json.RawMessage(bodyBytes))
	if err != nil {
		return err
	}

	var response BacklogItemResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Converted backlog item: %s → %s/%s\n", name, response.Item.Kind, response.Item.Name)
	return nil
}
