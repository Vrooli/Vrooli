package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// BatchCreateRequest is the request body for the batch create endpoint.
type BatchCreateRequest struct {
	Items      []BatchCreateItem      `json:"items"`
	Milestones []BatchCreateMilestone `json:"milestones,omitempty"`
	Preview    bool                   `json:"preview,omitempty"`
}

// BatchCreateItem represents a single item in a batch create request.
type BatchCreateItem struct {
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Kind            string   `json:"kind"`
	Priority        *int32   `json:"priority,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Milestone       string   `json:"milestone,omitempty"`
	Effort          *string  `json:"effort,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
	Creates         []string `json:"creates,omitempty"`
}

type BatchCreateMilestone struct {
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	DependsOn   *[]string `json:"depends_on,omitempty"`
}

type BatchCreateMilestoneResult struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Action      string   `json:"action"`
}

// BatchCreateResponse is the response from the batch create endpoint.
type BatchCreateResponse struct {
	Items      []BacklogItem                `json:"items"`
	Milestones []BatchCreateMilestoneResult `json:"milestones,omitempty"`
	Count      int                          `json:"count"`
	Preview    bool                         `json:"preview,omitempty"`
	Warnings   []string                     `json:"warnings,omitempty"`
}

// BatchQueueRequest is the request body for the batch queue endpoint.
type BatchQueueRequest struct {
	Items   []string `json:"items"`
	Mode    string   `json:"mode,omitempty"`
	Confirm bool     `json:"confirm"`
	Force   bool     `json:"force,omitempty"`
}

// BatchQueueResponse is the response from the batch queue endpoint.
type BatchQueueResponse struct {
	Results        []BatchQueueItemResult `json:"results"`
	ExecutionOrder []string               `json:"execution_order"`
}

// BatchQueueItemResult reports the outcome of queuing a single item.
type BatchQueueItemResult struct {
	Item              string   `json:"item"`
	Queued            bool     `json:"queued"`
	Message           string   `json:"message"`
	ExecutionID       string   `json:"execution_id,omitempty"`
	UnmetDependencies []string `json:"unmet_dependencies,omitempty"`
}

func (a *App) cmdBacklogBatchCreate(args []string) error {
	fs := flag.NewFlagSet("backlog batch-create", flag.ContinueOnError)
	fileFlag := fs.String("file", "", "Path to JSON file containing items (or @file for inline)")
	previewFlag := fs.Bool("preview", false, "Preview the batch import without creating items")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("file", *fileFlag); err != nil {
		return fmt.Errorf("usage: backlog batch-create --file items.json [--preview] [--json]\n\n%s", err)
	}

	// Read and parse the JSON file.
	raw, err := parseJSONString("@" + *fileFlag)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var req BatchCreateRequest
	if err := decodeJSONStrict(raw, &req); err != nil {
		return fmt.Errorf("file must contain a JSON object with an 'items' array: %w", err)
	}
	req.Preview = req.Preview || *previewFlag

	if len(req.Items) == 0 {
		return fmt.Errorf("no items found in file")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/batch", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BatchCreateResponse](body)
	if err != nil {
		return err
	}

	if len(response.Warnings) > 0 {
		fmt.Fprintf(os.Stderr, "Warnings:\n")
		for _, w := range response.Warnings {
			fmt.Fprintf(os.Stderr, "  - %s\n", w)
		}
	}

	printSection("Result")
	if response.Preview {
		fmt.Printf("  Previewed %d backlog item(s)\n", response.Count)
	} else {
		fmt.Printf("  Created %d backlog item(s)\n", response.Count)
	}

	printBatchCreateItems(response.Items)
	printBatchCreateMilestones(response.Milestones)

	if len(response.Items) > 0 {
		first := response.Items[0]
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "get", "--kind", first.Kind, "--name", first.Name),
			cliCommand("backlog", "list"),
		})
	}

	return nil
}

// printBatchCreateItems renders the "Items" section for a batch-create result.
func printBatchCreateItems(items []BacklogItem) {
	printSection("Items")
	for _, item := range items {
		effortStr := ""
		if item.Effort != "" {
			effortStr = fmt.Sprintf(", effort: %s", item.Effort)
		}
		fmt.Printf("  [%s] %s (priority: %d, status: %s%s)\n", item.Kind, item.Name, item.Priority, item.Status, effortStr)
		if len(item.DependsOn) > 0 {
			fmt.Printf("    Depends on: %s\n", strings.Join(item.DependsOn, ", "))
		}
		if item.Milestone != "" {
			fmt.Printf("    Milestone: %s\n", item.Milestone)
		}
	}
}

// printBatchCreateMilestones renders the "Milestones" section when any
// milestones were created or updated by the batch.
func printBatchCreateMilestones(milestones []BatchCreateMilestoneResult) {
	if len(milestones) == 0 {
		return
	}
	printSection("Milestones")
	for _, milestone := range milestones {
		fmt.Printf("  [%s] %s (%s)\n", strings.ToUpper(milestone.Action), milestone.Name, milestone.Status)
		fmt.Printf("    Title: %s\n", milestone.Title)
		if milestone.Description != "" {
			fmt.Printf("    Description: %s\n", milestone.Description)
		}
		if milestone.Priority > 0 {
			fmt.Printf("    Priority: %d\n", milestone.Priority)
		}
		if len(milestone.DependsOn) > 0 {
			fmt.Printf("    Depends on: %s\n", strings.Join(milestone.DependsOn, ", "))
		}
	}
}

func (a *App) cmdBacklogBatchQueue(args []string) error {
	fs := flag.NewFlagSet("backlog batch-queue", flag.ContinueOnError)
	itemsFlag := fs.String("items", "", "Comma-separated list of kind/name references")
	executeFlag := fs.Bool("execute", false, "Execute queue mutations (default is preview-only)")
	forceFlag := fs.Bool("force", false, "Override forceable blocking reasons")
	modeFlag := fs.String("mode", "", "Execution mode (manual, yolo, scheduled)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: backlog batch-queue --items kind/name,kind/name [--execute] [--force] [--mode MODE] [--json]\n\n%s", err)
	}

	// Parse comma-separated items.
	rawItems := strings.Split(*itemsFlag, ",")
	items := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	if len(items) == 0 {
		return fmt.Errorf("at least one item is required")
	}

	req := BatchQueueRequest{
		Items:   items,
		Mode:    strings.TrimSpace(*modeFlag),
		Confirm: *executeFlag,
		Force:   *forceFlag,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/batch/queue", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BatchQueueResponse](body)
	if err != nil {
		return err
	}

	printSection("Execution Order")
	for i, ref := range response.ExecutionOrder {
		fmt.Printf("  %d. %s\n", i+1, ref)
	}

	printSection("Results")
	queuedCount := 0
	for _, result := range response.Results {
		status := "SKIP"
		if result.Queued {
			status = "QUEUED"
			queuedCount++
		}
		fmt.Printf("  [%s] %s — %s\n", status, result.Item, result.Message)
		if result.ExecutionID != "" {
			fmt.Printf("    Execution ID: %s\n", result.ExecutionID)
		}
		if len(result.UnmetDependencies) > 0 {
			fmt.Printf("    Unmet deps: %s\n", strings.Join(result.UnmetDependencies, ", "))
		}
	}

	printSection("Summary")
	fmt.Printf("  Total: %d, Queued: %d, Skipped: %d\n",
		len(response.Results), queuedCount, len(response.Results)-queuedCount)

	if !*executeFlag && queuedCount == 0 {
		itemsList := strings.Join(items, ",")
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "batch-queue", "--items", itemsList, "--execute"),
			cliCommand("backlog", "batch-queue", "--items", itemsList, "--execute", "--force"),
		})
	}

	return nil
}
