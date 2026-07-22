package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

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

	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/queue", nil, json.RawMessage(payload))
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
		printBacklogQueueDryRun(response, kind, name, opts.operation)
		return nil
	}

	printBacklogQueueResult(response, opts)
	return nil
}

// printBacklogQueueDryRun renders the preview/blocked output produced when a
// queue request validated without mutating (dry-run or gated).
func printBacklogQueueDryRun(response QueueBacklogResponse, kind, name, operation string) {
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
			forceLabel := ""
			if reason.Forceable {
				forceLabel = " (forceable)"
			}
			fmt.Printf("  - %s%s\n", reason.Message, forceLabel)
		}
	}

	printBacklogQueueAdvisories(response.Advisories)

	nextCommands := []string{
		cliCommand("backlog", "queue", "--kind", kind, "--name", name, "--execute", "--operation", operation),
		cliCommand("execution", "list", "--backlog-kind", kind, "--backlog-name", name),
	}
	if len(response.BlockingReasons) > 0 {
		nextCommands = []string{
			cliCommand("backlog", "queue", "--kind", kind, "--name", name, "--execute", "--force", "--operation", operation),
			cliCommand("backlog", "get", "--kind", kind, "--name", name),
		}
	}
	printCommandListSection("Next Steps", nextCommands)
}

// printBacklogQueueResult renders the success output for an actually-queued item.
func printBacklogQueueResult(response QueueBacklogResponse, opts executionOptions) {
	printSection("Result")
	fmt.Printf("  Queued backlog item: %s/%s\n", response.Item.Kind, response.Item.Name)
	printBacklogQueueAdvisories(response.Advisories)
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
}

// printBacklogQueueAdvisories prints the Advisories section when any are present.
func printBacklogQueueAdvisories(advisories []string) {
	if len(advisories) == 0 {
		return
	}
	printSection("Advisories")
	for _, advisory := range advisories {
		fmt.Printf("  - %s\n", advisory)
	}
}
