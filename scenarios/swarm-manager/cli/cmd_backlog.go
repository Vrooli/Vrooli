package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogList(args []string) error {
	fs := flag.NewFlagSet("backlog list", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Comma-separated kinds to filter by")
	statusFlag := fs.String("status", "", "Comma-separated statuses to include, or \"all\" (default: non-archived)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*kindFlag) != "" {
		query.Set("kinds", strings.TrimSpace(*kindFlag))
	}
	if strings.TrimSpace(*statusFlag) != "" {
		query.Set("statuses", strings.TrimSpace(*statusFlag))
	}

	body, err := a.getV1("/backlog", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ListBacklogResponse](body)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No backlog items found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "create", "--data", "'{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d backlog item(s)\n", len(response.Items))
	if kinds := strings.TrimSpace(query.Get("kinds")); kinds != "" {
		fmt.Printf("  Filtered kinds: %s\n", kinds)
	}
	if statuses := strings.TrimSpace(query.Get("statuses")); statuses != "" {
		fmt.Printf("  Filtered statuses: %s\n", statuses)
	}

	printSection("Results")
	for _, item := range response.Items {
		fmt.Printf("  [%s] %s (priority: %d, status: %s)\n", item.Kind, item.Name, item.Priority, item.Status)
		fmt.Printf("    Title: %s\n", item.Title)
		if len(item.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(item.Tags, ", "))
		}
		if len(item.DependsOn) > 0 {
			fmt.Printf("    Depends on: %s\n", strings.Join(item.DependsOn, ", "))
		}
		if item.Initiative != "" {
			fmt.Printf("    Initiative: %s\n", item.Initiative)
		}
		if item.Effort != "" {
			fmt.Printf("    Effort: %s\n", item.Effort)
		}
		if len(item.AcceptanceAllow) > 0 {
			fmt.Printf("    Acceptance Allow: %s\n", strings.Join(item.AcceptanceAllow, ", "))
		}
		if len(item.AcceptanceDeny) > 0 {
			fmt.Printf("    Acceptance Deny: %s\n", strings.Join(item.AcceptanceDeny, ", "))
		}
		fmt.Println()
	}

	first := response.Items[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("backlog", "get", "--kind", "<kind>", "--name", "<name>"),
		cliCommand("backlog", "get", "--kind", first.Kind, "--name", first.Name),
		cliCommand("backlog", "files", "--kind", first.Kind, "--name", first.Name),
		cliCommand("backlog", "queue", "--kind", first.Kind, "--name", first.Name),
	})
	return nil
}

func (a *App) cmdBacklogGet(args []string) error {
	fs := flag.NewFlagSet("backlog get", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog get --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.getV1("/backlog/"+kind+"/"+name, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Summary")
	fmt.Printf("  %s/%s (%s)\n", item.Kind, item.Name, item.Status)

	printSection("Details")
	fmt.Printf("  Name: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Title: %s\n", item.Title)
	fmt.Printf("  Description: %s\n", item.Description)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	if len(item.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if len(item.DependsOn) > 0 {
		fmt.Printf("  Depends On: %s\n", strings.Join(item.DependsOn, ", "))
	}
	if item.Initiative != "" {
		fmt.Printf("  Initiative: %s\n", item.Initiative)
	}
	if item.Effort != "" {
		fmt.Printf("  Effort: %s\n", item.Effort)
	}
	if len(item.AcceptanceAllow) > 0 {
		fmt.Printf("  Acceptance Allow: %s\n", strings.Join(item.AcceptanceAllow, ", "))
	}
	if len(item.AcceptanceDeny) > 0 {
		fmt.Printf("  Acceptance Deny: %s\n", strings.Join(item.AcceptanceDeny, ", "))
	}
	fmt.Printf("  Created: %s\n", item.Created)
	fmt.Printf("  Updated: %s\n", item.Updated)

	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "update", "--kind", item.Kind, "--name", item.Name, "--data", "'{\"status\":\"ready\"}'"),
		cliCommand("backlog", "queue", "--kind", item.Kind, "--name", item.Name),
	})
	return nil
}

func (a *App) cmdBacklogCreate(args []string) error {
	fs := flag.NewFlagSet("backlog create", flag.ContinueOnError)
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("data", *data); err != nil {
		return fmt.Errorf("usage: backlog create --data JSON [--json]\n\nExample:\n  backlog create --data '{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'\n\n%s", err)
	}

	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	var req CreateBacklogRequest
	if err := decodeJSONStrict(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Name == "" || req.Title == "" || req.Kind == "" {
		return fmt.Errorf("name, title, and kind are required fields")
	}

	body, err := a.requestV1("POST", "/backlog", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Result")
	fmt.Printf("  Created backlog item: %s/%s\n", item.Kind, item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "files", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "queue", "--kind", item.Kind, "--name", item.Name),
	})
	return nil
}

func (a *App) cmdBacklogUpdate(args []string) error {
	fs := flag.NewFlagSet("backlog update", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag, "data", *data); err != nil {
		return fmt.Errorf("usage: backlog update --kind KIND --name NAME --data JSON [--json]\n\nExample:\n  backlog update --kind idea --name my-idea --data '{\"status\":\"ready\"}'\n\n%s", err)
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	var update UpdateBacklogRequest
	if err := decodeJSONStrict(payload, &update); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if update.Empty() {
		return fmt.Errorf("at least one field must be provided")
	}

	requestBody, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal update payload: %w", err)
	}

	body, err := a.requestV1("PATCH", "/backlog/"+kind+"/"+name, nil, json.RawMessage(requestBody))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Result")
	fmt.Printf("  Updated backlog item: %s/%s\n", item.Kind, item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", item.Kind, "--name", item.Name),
		cliCommand("backlog", "queue", "--kind", item.Kind, "--name", item.Name),
	})
	return nil
}

func (a *App) cmdBacklogDelete(args []string) error {
	fs := flag.NewFlagSet("backlog delete", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog delete --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.requestV1("DELETE", "/backlog/"+kind+"/"+name, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	fmt.Printf("  Deleted backlog item: %s/%s\n", kind, name)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "list"),
		cliCommand("backlog", "create", "--data", "'{\"name\":\"new-item\",\"title\":\"New Item\",\"kind\":\"idea\"}'"),
	})
	return nil
}

func (a *App) cmdBacklogFiles(args []string) error {
	fs := flag.NewFlagSet("backlog files", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog files --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/files", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogFilesResponse](body)
	if err != nil {
		return err
	}
	if len(response.Files) == 0 {
		printSection("Summary")
		fmt.Printf("  No files found for %s/%s.\n", kind, name)
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "file-upload", "--kind", kind, "--name", name, "--file", "<local-file>"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d file node(s) for %s/%s\n", len(response.Files), kind, name)
	printSection("Results")
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
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("backlog", "file-get", "--kind", kind, "--name", name, "--path", "<path>"),
		cliCommand("backlog", "file-upload", "--kind", kind, "--name", name, "--path", "<path>", "--file", "<local-file>"),
	})
	return nil
}

func (a *App) cmdBacklogProcessPreflight(args []string) error {
	fs := flag.NewFlagSet("backlog process-preflight", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog process-preflight --kind KIND --name NAME [--json]\n\n%s", err)
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	body, err := a.getV1("/backlog/"+kind+"/"+name+"/process-preflight", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ProcessPreflightResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	fmt.Printf("  Readiness: %t\n", response.Preflight.Ready)
	fmt.Printf("  Item: %s/%s\n", response.Item.Kind, response.Item.Name)
	if response.Preflight.ResolvedTargetScenarioID != "" {
		fmt.Printf("  Target Scenario: %s\n", response.Preflight.ResolvedTargetScenarioID)
	}
	fmt.Printf("  Target Exists: %t\n", response.Preflight.TargetScenarioExists)
	if response.Preflight.SuggestedOperation != "" {
		fmt.Printf("  Suggested Operation: %s\n", response.Preflight.SuggestedOperation)
	}
	if response.Preflight.SuggestedSteerProfileID != "" {
		fmt.Printf("  Suggested Steer Profile: %s\n", response.Preflight.SuggestedSteerProfileID)
	}

	if len(response.Preflight.BlockingReasons) > 0 {
		printSection("Blocking Reasons")
		for _, reason := range response.Preflight.BlockingReasons {
			fmt.Printf("  - %s\n", reason)
		}
	}
	if len(response.Preflight.BlockingQuestions) > 0 {
		printSection("Blocking Questions")
		for _, q := range response.Preflight.BlockingQuestions {
			fmt.Printf("  - %s: %s\n", q.ID, q.Question)
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "queue", "--kind", kind, "--name", name, "--execute", "--operation", response.Preflight.SuggestedOperation),
		cliCommand("backlog", "get", "--kind", kind, "--name", name),
	})
	return nil
}

func (a *App) cmdBacklogQueue(args []string) error {
	fs := flag.NewFlagSet("backlog queue", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	executeFlag := fs.Bool("execute", false, "Execute queue mutation (default is preview-only)")
	forceFlag := fs.Bool("force", false, "Override unanswered feedback gates (questions/suggestions)")
	mode, delaySeconds, operation, startedBy := addExecutionOptionsFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog queue --kind KIND --name NAME [--execute] [--force] [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver] [--started-by NAME] [--json]\n\n%s", err)
	}

	opts, err := parseExecutionOptions(mode, delaySeconds, operation, startedBy, false)
	if err != nil {
		return err
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	payloadMap := map[string]any{
		"operation":     opts.operation,
		"delay_seconds": opts.delaySeconds,
		"started_by":    opts.startedBy,
		"confirm":       *executeFlag,
		"force":         *forceFlag,
	}
	if opts.mode != "" {
		payloadMap["mode"] = opts.mode
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/queue", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[QueueBacklogResponse](body)
	if err != nil {
		return err
	}

	if response.DryRun || !response.Queued {
		printSection("Status")
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = "Queue request validated with no mutation."
		}
		fmt.Printf("  %s\n", message)
		fmt.Printf("  Item: %s/%s\n", kind, name)
		fmt.Printf("  Queued: %t\n", response.Queued)
		if response.UnansweredQuestions > 0 || response.PendingSuggestions > 0 {
			fmt.Printf("  Feedback Gates: %d unanswered question(s), %d pending suggestion(s)\n", response.UnansweredQuestions, response.PendingSuggestions)
		}

		printSection("Triage")
		if len(response.BlockingReasons) == 0 {
			fmt.Println("  No blockers detected.")
		} else {
			for _, reason := range response.BlockingReasons {
				fmt.Printf("  - %s\n", reason)
			}
		}

		nextCommands := []string{
			cliCommand("backlog", "queue", "--kind", kind, "--name", name, "--execute", "--operation", opts.operation),
			cliCommand("execution", "list", "--backlog-kind", kind, "--backlog-name", name),
		}
		if len(response.BlockingReasons) > 0 {
			nextCommands = []string{
				cliCommand("backlog", "queue", "--kind", kind, "--name", name, "--execute", "--force", "--operation", opts.operation),
				cliCommand("backlog", "get", "--kind", kind, "--name", name),
			}
		}
		printCommandListSection("Next Steps", nextCommands)
		return nil
	}

	printSection("Result")
	fmt.Printf("  Queued backlog item: %s/%s\n", response.Item.Kind, response.Item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", response.Item.Status)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	if response.RunID != "" {
		fmt.Printf("  Run ID: %s\n", response.RunID)
	}
	fmt.Printf("  Mode: %s\n", opts.mode)
	if opts.mode == "scheduled" {
		fmt.Printf("  Delay Seconds: %d\n", opts.delaySeconds)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "list", "--backlog-kind", response.Item.Kind, "--backlog-name", response.Item.Name),
		cliCommand("execution", "get", "--id", "<execution-id>"),
	})
	return nil
}

func (a *App) cmdBacklogResearch(args []string) error {
	fs := flag.NewFlagSet("backlog research", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	data := fs.String("data", "", "Optional JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog research --kind KIND --name NAME [--data JSON] [--json]\n\nExample:\n  backlog research --kind idea --name my-idea --data '{\"prompt\":\"Focus on risks\"}'\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	var payload json.RawMessage
	if strings.TrimSpace(*data) != "" {
		parsed, err := parseJSONString(*data)
		if err != nil {
			return err
		}
		payload = parsed
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/research", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ResearchResponse](body)
	if err != nil {
		return err
	}

	if response.DryRun {
		printSection("Result")
		fmt.Printf("  Dry-run validated research request for %s/%s\n", kind, name)
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "research", "--kind", kind, "--name", name),
			cliCommand("backlog", "prompt-trace", "--kind", kind, "--name", name),
		})
		return nil
	}

	printSection("Result")
	fmt.Printf("  Started research for %s/%s\n", kind, name)
	printSection("What Changed")
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	fmt.Printf("  Run ID: %s\n", response.RunID)
	fmt.Printf("  Base URL: %s\n", response.BaseURL)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", kind, "--name", name),
		cliCommand("execution", "list", "--backlog-kind", kind, "--backlog-name", name),
	})
	return nil
}

func (a *App) cmdBacklogPromptTrace(args []string) error {
	fs := flag.NewFlagSet("backlog prompt-trace", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog prompt-trace --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/prompt-trace", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptTraceResponse](body)
	if err != nil {
		return err
	}
	printPromptTraceSummary(
		"Summary",
		fmt.Sprintf("Prompt trace for backlog item %s/%s", kind, name),
		response.Trace,
	)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "research", "--kind", kind, "--name", name),
		cliCommand("backlog", "get", "--kind", kind, "--name", name),
	})
	return nil
}

func (a *App) cmdBacklogFileGet(args []string) error {
	fs := flag.NewFlagSet("backlog file-get", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	pathFlag := fs.String("path", "", "File path within backlog item")
	outPath := fs.String("out", "", "Write file content to local path instead of stdout")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag, "path", *pathFlag); err != nil {
		return fmt.Errorf("usage: backlog file-get --kind KIND --name NAME --path PATH [--out local-path] [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	filePath := strings.TrimSpace(*pathFlag)

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/files/"+filePath, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	if strings.TrimSpace(*outPath) == "" {
		fmt.Print(string(body))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil && filepath.Dir(*outPath) != "." {
		return fmt.Errorf("prepare output directory: %w", err)
	}
	if err := os.WriteFile(*outPath, body, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	printSection("Result")
	fmt.Printf("  Saved file to %s\n", *outPath)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", "--kind", kind, "--name", name),
	})
	return nil
}

func (a *App) cmdBacklogFileUpload(args []string) error {
	fs := flag.NewFlagSet("backlog file-upload", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	serverPath := fs.String("path", "", "Full server-side destination path (e.g. workshop/round-001.json)")
	localFile := fs.String("file", "", "Local file path to upload")
	contentStr := fs.String("content", "", "Inline content string to upload (⚠️  prefer --stdin to avoid shell quoting issues)")
	stdinFlag := fs.Bool("stdin", false, "Read content from stdin (safest for content with special characters)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	fileStr := strings.TrimSpace(*localFile)
	contentVal := *contentStr

	// Read from stdin if --stdin is set
	if *stdinFlag {
		if fileStr != "" || contentVal != "" {
			return fmt.Errorf("--stdin cannot be combined with --file or --content")
		}
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		contentVal = string(stdinBytes)
	}

	if fileStr == "" && contentVal == "" {
		return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\neither --stdin, --file, or --content is required")
	}
	if fileStr != "" && contentVal != "" {
		return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\n--file and --content are mutually exclusive")
	}

	sp := strings.TrimSpace(*serverPath)

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)

	if contentVal != "" {
		// --content mode: --path is required
		if sp == "" {
			return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH --content CONTENT [--json]\n\n--path is required when using --content")
		}
		serverDir := filepath.Dir(sp)
		serverFile := filepath.Base(sp)

		part, err := writer.CreateFormFile("file", serverFile)
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, strings.NewReader(contentVal)); err != nil {
			return fmt.Errorf("copy content: %w", err)
		}
		if serverDir != "." && serverDir != "" {
			if err := writer.WriteField("path", serverDir); err != nil {
				return fmt.Errorf("write path field: %w", err)
			}
		}
	} else {
		// --file mode: --path defaults to filepath.Base(localFile)
		if sp == "" {
			sp = filepath.Base(fileStr)
		}
		serverDir := filepath.Dir(sp)
		serverFile := filepath.Base(sp)

		file, err := os.Open(fileStr)
		if err != nil {
			return fmt.Errorf("open local file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("file", serverFile)
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("copy file content: %w", err)
		}
		if serverDir != "." && serverDir != "" {
			if err := writer.WriteField("path", serverDir); err != nil {
				return fmt.Errorf("write path field: %w", err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	respBody, err := a.requestMultipartV1("POST", "/backlog/"+kind+"/"+name+"/files", formBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(respBody)
		return nil
	}

	parsed, err := decodeResponse[BacklogFileResponse](respBody)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Uploaded file: %s\n", parsed.File.Path)
	printSection("What Changed")
	fmt.Printf("  Name: %s\n", parsed.File.Name)
	fmt.Printf("  Size: %d bytes\n", parsed.File.Size)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", "--kind", kind, "--name", name),
		cliCommand("backlog", "file-get", "--kind", kind, "--name", name, "--path", parsed.File.Path),
	})
	return nil
}

func (a *App) cmdBacklogExport(args []string) error {
	fs := flag.NewFlagSet("backlog export", flag.ContinueOnError)
	kinds := fs.String("kinds", "", "Comma-separated kinds to include (default: all)")
	statuses := fs.String("status", "", "Comma-separated statuses to include (default: non-archived)")
	names := fs.String("names", "", "Comma-separated kind/name pairs for specific items")
	priorityMax := fs.Int("priority-max", 0, "Only items with priority <= this value")
	tags := fs.String("tags", "", "Comma-separated tags (any match)")
	noPRD := fs.Bool("no-prd", false, "Exclude PRD content (smaller file)")
	outPath := fs.String("out", "", "Output file path (default: backlog-export-YYYY-MM-DD.md)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	payload := map[string]any{}
	if strings.TrimSpace(*kinds) != "" {
		payload["kinds"] = strings.Split(*kinds, ",")
	}
	if strings.TrimSpace(*statuses) != "" {
		payload["statuses"] = strings.Split(*statuses, ",")
	}
	if strings.TrimSpace(*names) != "" {
		payload["names"] = strings.Split(*names, ",")
	}
	if *priorityMax > 0 {
		payload["priorityMax"] = *priorityMax
	}
	if strings.TrimSpace(*tags) != "" {
		payload["tags"] = strings.Split(*tags, ",")
	}
	if *noPRD {
		includePrd := false
		payload["includePrd"] = includePrd
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/export", nil, json.RawMessage(bodyBytes))
	if err != nil {
		return err
	}

	if *jsonOut {
		// In JSON mode, output metadata about the export
		result := map[string]any{
			"size_bytes": len(body),
			"format":     "markdown",
		}
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}

	output := strings.TrimSpace(*outPath)
	if output == "" {
		output = "backlog-export-" + time.Now().Format("2006-01-02") + ".md"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil && filepath.Dir(output) != "." {
		return fmt.Errorf("prepare output directory: %w", err)
	}
	if err := os.WriteFile(output, body, 0o644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	printSection("Result")
	fmt.Printf("  Exported backlog to %s (%d bytes)\n", output, len(body))
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "import", "--file", output),
		cliCommand("backlog", "import", "--file", output, "--apply"),
	})
	return nil
}

func (a *App) cmdBacklogImport(args []string) error {
	fs := flag.NewFlagSet("backlog import", flag.ContinueOnError)
	fileFlag := fs.String("file", "", "Import file path")
	apply := fs.Bool("apply", false, "Apply changes (default: dry-run only)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("file", *fileFlag); err != nil {
		return fmt.Errorf("usage: backlog import --file FILE [--apply] [--json]\n\n%s", err)
	}

	filePath := strings.TrimSpace(*fileFlag)

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}
	if *apply {
		if err := writer.WriteField("apply", "true"); err != nil {
			return fmt.Errorf("write apply field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	body, err := a.requestMultipartV1("POST", "/backlog/import", formBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ImportBacklogResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	if response.DryRun {
		fmt.Println("  Mode: dry-run (use --apply to apply changes)")
	} else {
		fmt.Println("  Mode: applied")
	}
	fmt.Printf("  %s\n", response.Summary)

	if len(response.Changes) > 0 {
		printSection("Changes")
		for _, change := range response.Changes {
			fmt.Printf("  [%s] %s\n", change.Action, change.Item)
			for _, detail := range change.Details {
				fmt.Printf("    - %s\n", detail)
			}
		}
	}

	if len(response.Errors) > 0 {
		printSection("Errors")
		for _, e := range response.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if response.DryRun {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "import", "--file", filePath, "--apply"),
		})
	} else {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "list"),
		})
	}
	return nil
}
