package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// BatchCreateRequest is the request body for the batch create endpoint.
type BatchCreateRequest struct {
	Items      []BatchCreateItem `json:"items"`
	Initiative string            `json:"initiative,omitempty"`
}

// BatchCreateItem represents a single item in a batch create request.
type BatchCreateItem struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Kind           string   `json:"kind"`
	Priority       *int32   `json:"priority,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	ResearchTarget *string  `json:"research_target,omitempty"`
	DependsOn      []string `json:"depends_on,omitempty"`
}

// BatchCreateResponse is the response from the batch create endpoint.
type BatchCreateResponse struct {
	Items      []BacklogItem `json:"items"`
	Initiative string        `json:"initiative,omitempty"`
	Count      int           `json:"count"`
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
	initiativeFlag := fs.String("initiative", "", "Initiative to assign all items to")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("file", *fileFlag); err != nil {
		return fmt.Errorf("usage: backlog batch-create --file items.json [--initiative NAME] [--json]\n\n%s", err)
	}

	// Read and parse the JSON file.
	raw, err := parseJSONString("@" + *fileFlag)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// The file may contain just the items array or the full request object.
	// Try parsing as full request first.
	var req BatchCreateRequest
	if jsonErr := json.Unmarshal(raw, &req); jsonErr != nil {
		// Try as items array.
		var items []BatchCreateItem
		if arrErr := json.Unmarshal(raw, &items); arrErr != nil {
			return fmt.Errorf("file must contain a JSON object with 'items' array or a JSON array of items: %w", jsonErr)
		}
		req.Items = items
	}

	// Override initiative from flag if provided.
	if strings.TrimSpace(*initiativeFlag) != "" {
		req.Initiative = strings.TrimSpace(*initiativeFlag)
	}

	if len(req.Items) == 0 {
		return fmt.Errorf("no items found in file")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/batch", nil, json.RawMessage(payload))
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

	printSection("Result")
	fmt.Printf("  Created %d backlog item(s)\n", response.Count)
	if response.Initiative != "" {
		fmt.Printf("  Initiative: %s\n", response.Initiative)
	}

	printSection("Items")
	for _, item := range response.Items {
		fmt.Printf("  [%s] %s (priority: %d, status: %s)\n", item.Kind, item.Name, item.Priority, item.Status)
		if len(item.DependsOn) > 0 {
			fmt.Printf("    Depends on: %s\n", strings.Join(item.DependsOn, ", "))
		}
	}

	if len(response.Items) > 0 {
		first := response.Items[0]
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "get", "--kind", first.Kind, "--name", first.Name),
			cliCommand("backlog", "list"),
		})
	}

	return nil
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

	body, err := a.requestV1("POST", "/backlog/batch/queue", nil, json.RawMessage(payload))
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
