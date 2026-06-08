// Package tasks provides CLI commands for task management.
package tasks

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/internal/format"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// TaskCreateRequest represents the request body for creating a task.
type TaskCreateRequest struct {
	Title            string      `json:"title"`
	Type             string      `json:"type"`
	Operation        string      `json:"operation"`
	Target           string      `json:"target,omitempty"`
	Category         string      `json:"category,omitempty"`
	Priority         string      `json:"priority,omitempty"`
	Notes            string      `json:"notes,omitempty"`
	Origin           *TaskOrigin `json:"origin,omitempty"`
	SteerMode        string      `json:"steer_mode,omitempty"`
	SteeringQueue    []string    `json:"steering_queue,omitempty"`
	AutoSteerProfile string      `json:"auto_steer_profile_id,omitempty"`
}

type TaskOrigin struct {
	Source                 string `json:"source,omitempty"`
	BacklogItem            string `json:"backlog_item,omitempty"`
	ItemFolder             string `json:"item_folder,omitempty"`
	HandoffDir             string `json:"handoff_dir,omitempty"`
	HandoffBriefPath       string `json:"handoff_brief_path,omitempty"`
	HandoffManifestPath    string `json:"handoff_manifest_path,omitempty"`
	HandoffSourceIndexPath string `json:"handoff_source_index_path,omitempty"`
}

// TaskResponse represents a task from the API.
type TaskResponse struct {
	ID                 string      `json:"id"`
	Title              string      `json:"title"`
	Type               string      `json:"type"`
	Operation          string      `json:"operation"`
	AutoSteerProfileID string      `json:"auto_steer_profile_id,omitempty"`
	Category           string      `json:"category"`
	Priority           string      `json:"priority"`
	Status             string      `json:"status"`
	CurrentPhase       string      `json:"current_phase,omitempty"`
	CompletionCount    int         `json:"completion_count"`
	LastCompletedAt    string      `json:"last_completed_at,omitempty"`
	CreatedAt          string      `json:"created_at,omitempty"`
	UpdatedAt          string      `json:"updated_at,omitempty"`
	SteerMode          string      `json:"steer_mode,omitempty"`
	Notes              string      `json:"notes,omitempty"`
	Origin             *TaskOrigin `json:"origin,omitempty"`
}

// TaskListResponse represents a list of tasks.
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Count int            `json:"count"`
	Total int            `json:"total,omitempty"`
}

// TaskDetailResponse includes runtime information.
type TaskDetailResponse struct {
	TaskResponse
	CurrentProcess      json.RawMessage `json:"current_process,omitempty"`
	AutoSteerPhaseIndex *int            `json:"auto_steer_phase_index,omitempty"`
}

// TaskCreateResponse wraps the API envelope for task creation.
type TaskCreateResponse struct {
	Success   bool         `json:"success"`
	DryRun    bool         `json:"dry_run,omitempty"`
	Task      TaskResponse `json:"task"`
	NextSteps []string     `json:"next_steps,omitempty"`
}

// TaskActionResponse represents the result of an action on a task.
type TaskActionResponse struct {
	Success   bool     `json:"success"`
	DryRun    bool     `json:"dry_run,omitempty"`
	Message   string   `json:"message,omitempty"`
	TaskID    string   `json:"task_id,omitempty"`
	Error     string   `json:"error,omitempty"`
	NextSteps []string `json:"next_steps,omitempty"`
}

// StatusUpdateRequest represents the request body for updating task status.
type StatusUpdateRequest struct {
	Status       string `json:"status,omitempty"`
	CurrentPhase string `json:"current_phase,omitempty"`
}

// Commands returns the task command groups.
func Commands(ctx appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Tasks",
			Commands: []cliapp.Command{
				{
					Name:        "task",
					Aliases:     []string{"tasks"},
					NeedsAPI:    true,
					Description: "Manage tasks (add|improve|list|show|status|delete)",
					Run: func(args []string) error {
						return route(ctx, args)
					},
				},
			},
		},
	}
}

func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "help":
		return printUsage()
	case "add":
		return cmdAdd(ctx, subArgs)
	case "improve":
		return cmdImprove(ctx, subArgs)
	case "list", "ls":
		return cmdList(ctx, subArgs)
	case "show", "get":
		return cmdShow(ctx, subArgs)
	case "status":
		return cmdStatus(ctx, subArgs)
	case "delete", "rm":
		return cmdDelete(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: ecosystem-manager task <subcommand> [args]

Subcommands:
  add [resource|scenario] <name>    Create a generation task
  improve [resource|scenario] <name> Create an improvement task
  list, ls                          List tasks
  show, get <id>                    Show task details
  status <id>                       Update task status
  delete, rm <id>                   Delete a task

Examples:
  ecosystem-manager task add scenario my-app --steer-profile balanced
  ecosystem-manager task add scenario my-app --handoff-dir /tmp/handoff --origin-source swarm-manager --origin-backlog-item idea/my-app
  ecosystem-manager task improve scenario my-app --steer-profile production-ready
  ecosystem-manager task list --status pending --type scenario
  ecosystem-manager task show <task-id> --json`
}

func cmdAdd(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	steerProfile := fs.String("steer-profile", "", "Auto-steer profile ID")
	steerMode := fs.String("steer-mode", "", "Single steer mode (e.g. test)")
	steerQueue := fs.String("steer-queue", "", "Comma-separated ordered list of steer modes (improver tasks only)")
	priority := fs.String("priority", "medium", "Priority (low, medium, high, critical)")
	category := fs.String("category", "general", "Category")
	notes := fs.String("notes", "", "Inline notes to persist on the task")
	notesFile := fs.String("notes-file", "", "Read task notes from a file")
	originSource := fs.String("origin-source", "", "Upstream system creating this task (e.g. swarm-manager)")
	originBacklogItem := fs.String("origin-backlog-item", "", "Upstream backlog item reference (e.g. idea/my-item)")
	originItemFolder := fs.String("origin-item-folder", "", "Absolute path to the upstream backlog item folder")
	handoffDir := fs.String("handoff-dir", "", "Absolute path to an upstream handoff directory")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: task add [resource|scenario] <name> [--steer-profile ID] [--steer-mode mode] [--steer-queue modes] [--priority P] [--category C] [--json]")
	}

	typeName := fs.Arg(0)
	if typeName != "resource" && typeName != "scenario" {
		return fmt.Errorf("type must be 'resource' or 'scenario', got %q", typeName)
	}
	name := fs.Arg(1)

	req := TaskCreateRequest{
		Title:     fmt.Sprintf("Generate %s %s", name, typeName),
		Type:      typeName,
		Operation: "generator",
		Target:    name,
		Category:  *category,
		Priority:  *priority,
	}
	if err := applyTaskContextFlags(&req, *notes, *notesFile, taskOriginInput{
		source:      *originSource,
		backlogItem: *originBacklogItem,
		itemFolder:  *originItemFolder,
		handoffDir:  *handoffDir,
	}); err != nil {
		return err
	}

	if *steerProfile != "" {
		req.AutoSteerProfile = *steerProfile
	}
	if *steerMode != "" {
		req.SteerMode = *steerMode
	}
	if *steerQueue != "" {
		req.SteeringQueue = parseSteerQueue(*steerQueue)
	}

	var resp TaskCreateResponse
	if err := ctx.Post("/tasks", req, &resp); err != nil {
		return format.WrapAPIError("Failed to create task", err)
	}
	task := resp.Task

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if resp.DryRun {
		return format.MutationResult(
			fmt.Sprintf("[DRY RUN] Would create task: %s [%s]", task.Title, task.ID),
			"", nil,
		)
	} else {
		return format.MutationResult(
			fmt.Sprintf("Created task: %s [%s]", task.Title, task.ID),
			"", resp.NextSteps,
		)
	}
}

func cmdImprove(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("improve", flag.ContinueOnError)
	steerProfile := fs.String("steer-profile", "", "Auto-steer profile ID")
	steerMode := fs.String("steer-mode", "", "Single steer mode (e.g. test)")
	steerQueue := fs.String("steer-queue", "", "Comma-separated ordered list of steer modes (improver tasks only)")
	priority := fs.String("priority", "medium", "Priority (low, medium, high, critical)")
	notes := fs.String("notes", "", "Inline notes to persist on the task")
	notesFile := fs.String("notes-file", "", "Read task notes from a file")
	originSource := fs.String("origin-source", "", "Upstream system creating this task (e.g. swarm-manager)")
	originBacklogItem := fs.String("origin-backlog-item", "", "Upstream backlog item reference (e.g. idea/my-item)")
	originItemFolder := fs.String("origin-item-folder", "", "Absolute path to the upstream backlog item folder")
	handoffDir := fs.String("handoff-dir", "", "Absolute path to an upstream handoff directory")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: task improve [resource|scenario] <name> [--steer-profile ID] [--steer-mode mode] [--steer-queue modes] [--priority P] [--json]")
	}

	typeName := fs.Arg(0)
	if typeName != "resource" && typeName != "scenario" {
		return fmt.Errorf("type must be 'resource' or 'scenario', got %q", typeName)
	}
	name := fs.Arg(1)

	req := TaskCreateRequest{
		Title:     fmt.Sprintf("Improve %s %s", name, typeName),
		Type:      typeName,
		Operation: "improver",
		Target:    name,
		Category:  "improvement",
		Priority:  *priority,
	}
	if err := applyTaskContextFlags(&req, *notes, *notesFile, taskOriginInput{
		source:      *originSource,
		backlogItem: *originBacklogItem,
		itemFolder:  *originItemFolder,
		handoffDir:  *handoffDir,
	}); err != nil {
		return err
	}

	if *steerProfile != "" {
		req.AutoSteerProfile = *steerProfile
	}
	if *steerMode != "" {
		req.SteerMode = *steerMode
	}
	if *steerQueue != "" {
		req.SteeringQueue = parseSteerQueue(*steerQueue)
	}

	var resp TaskCreateResponse
	if err := ctx.Post("/tasks", req, &resp); err != nil {
		return format.WrapAPIError("Failed to create improvement task", err)
	}
	task := resp.Task

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if resp.DryRun {
		return format.MutationResult(
			fmt.Sprintf("[DRY RUN] Would create improvement task: %s [%s]", task.Title, task.ID),
			"", nil,
		)
	} else {
		return format.MutationResult(
			fmt.Sprintf("Created improvement task: %s [%s]", task.Title, task.ID),
			"", resp.NextSteps,
		)
	}
}

type taskOriginInput struct {
	source      string
	backlogItem string
	itemFolder  string
	handoffDir  string
}

func applyTaskContextFlags(req *TaskCreateRequest, inlineNotes string, notesFile string, origin taskOriginInput) error {
	if req == nil {
		return fmt.Errorf("task request is required")
	}
	if strings.TrimSpace(inlineNotes) != "" && strings.TrimSpace(notesFile) != "" {
		return fmt.Errorf("--notes and --notes-file are mutually exclusive")
	}

	if strings.TrimSpace(notesFile) != "" {
		data, err := os.ReadFile(strings.TrimSpace(notesFile))
		if err != nil {
			return fmt.Errorf("read notes file: %w", err)
		}
		req.Notes = strings.TrimSpace(string(data))
	} else {
		req.Notes = strings.TrimSpace(inlineNotes)
	}

	itemFolder, err := normalizeOptionalAbsPath(origin.itemFolder)
	if err != nil {
		return fmt.Errorf("resolve origin item folder: %w", err)
	}
	handoffDir, err := normalizeOptionalAbsPath(origin.handoffDir)
	if err != nil {
		return fmt.Errorf("resolve handoff dir: %w", err)
	}

	if strings.TrimSpace(origin.source) == "" &&
		strings.TrimSpace(origin.backlogItem) == "" &&
		itemFolder == "" &&
		handoffDir == "" {
		return nil
	}

	req.Origin = &TaskOrigin{
		Source:      strings.TrimSpace(origin.source),
		BacklogItem: strings.TrimSpace(origin.backlogItem),
		ItemFolder:  itemFolder,
		HandoffDir:  handoffDir,
	}
	if req.Origin.HandoffDir != "" {
		req.Origin.HandoffBriefPath = filepath.Join(req.Origin.HandoffDir, "brief.md")
		req.Origin.HandoffManifestPath = filepath.Join(req.Origin.HandoffDir, "manifest.json")
		req.Origin.HandoffSourceIndexPath = filepath.Join(req.Origin.HandoffDir, "source-index.json")
		for _, path := range []string{
			req.Origin.HandoffBriefPath,
			req.Origin.HandoffManifestPath,
			req.Origin.HandoffSourceIndexPath,
		} {
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("invalid handoff package: %s: %w", path, err)
			}
		}
		if req.Notes == "" {
			data, err := os.ReadFile(req.Origin.HandoffBriefPath)
			if err != nil {
				return fmt.Errorf("read handoff brief: %w", err)
			}
			req.Notes = strings.TrimSpace(string(data))
		}
	}
	return nil
}

func normalizeOptionalAbsPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	return absolute, nil
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status")
	typeName := fs.String("type", "", "Filter by type (resource, scenario)")
	operation := fs.String("operation", "", "Filter by operation (generator, improver)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *status != "" {
		query.Set("status", *status)
	}
	if *typeName != "" {
		query.Set("type", *typeName)
	}
	if *operation != "" {
		query.Set("operation", *operation)
	}

	var resp TaskListResponse
	if err := ctx.GetWithQuery("/tasks", query, &resp); err != nil {
		return format.WrapAPIError("Failed to list tasks", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if len(resp.Tasks) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"No tasks found"},
			RetrievalHints: []string{"ecosystem-manager task add scenario <name>"},
		})
	}

	results := make([]string, 0, len(resp.Tasks))
	for _, t := range resp.Tasks {
		phase := ""
		if t.CurrentPhase != "" {
			phase = fmt.Sprintf(" [%s]", t.CurrentPhase)
		}
		results = append(results, fmt.Sprintf("%-25s %-8s %-10s %-8s %s%s", t.ID, t.Type, t.Operation, t.Priority, t.Title, phase))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tasks returned: %d", resp.Count)},
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager task show <task-id>"},
	})
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task show <id>")
	}
	taskID := fs.Arg(0)

	var task TaskDetailResponse
	if err := ctx.Get(fmt.Sprintf("/tasks/%s", taskID), &task); err != nil {
		return format.WrapAPIError("Failed to get task", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, task)
	}

	summary := []string{
		fmt.Sprintf("Task: %s", task.Title),
		fmt.Sprintf("ID: %s", task.ID),
		fmt.Sprintf("Type: %s", task.Type),
		fmt.Sprintf("Operation: %s", task.Operation),
		fmt.Sprintf("Status: %s", task.Status),
		fmt.Sprintf("Priority: %s", task.Priority),
	}
	results := []string{}
	if task.Category != "" {
		results = append(results, fmt.Sprintf("Category: %s", task.Category))
	}
	if task.CurrentPhase != "" {
		results = append(results, fmt.Sprintf("Phase: %s", task.CurrentPhase))
	}
	if task.SteerMode != "" {
		results = append(results, fmt.Sprintf("Steer mode: %s", task.SteerMode))
	}
	if task.AutoSteerProfileID != "" {
		results = append(results, fmt.Sprintf("Auto-steer profile: %s", task.AutoSteerProfileID))
	}
	results = append(results, fmt.Sprintf("Completions: %d", task.CompletionCount))
	if task.LastCompletedAt != "" {
		results = append(results, fmt.Sprintf("Last completed: %s", task.LastCompletedAt))
	}
	if task.CreatedAt != "" {
		results = append(results, fmt.Sprintf("Created: %s", task.CreatedAt))
	}
	if task.UpdatedAt != "" {
		results = append(results, fmt.Sprintf("Updated: %s", task.UpdatedAt))
	}
	if task.Notes != "" {
		results = append(results, fmt.Sprintf("Notes: %s", task.Notes))
	}
	if task.Origin != nil {
		if task.Origin.Source != "" {
			results = append(results, fmt.Sprintf("Origin source: %s", task.Origin.Source))
		}
		if task.Origin.BacklogItem != "" {
			results = append(results, fmt.Sprintf("Origin backlog item: %s", task.Origin.BacklogItem))
		}
		if task.Origin.ItemFolder != "" {
			results = append(results, fmt.Sprintf("Origin item folder: %s", task.Origin.ItemFolder))
		}
		if task.Origin.HandoffDir != "" {
			results = append(results, fmt.Sprintf("Origin handoff dir: %s", task.Origin.HandoffDir))
		}
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        summary,
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager task status <task-id> --status <new-status>"},
	})
}

func cmdStatus(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	status := fs.String("status", "", "New status")
	phase := fs.String("phase", "", "Current phase")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task status <id> [--status S] [--phase P] [--json]")
	}
	taskID := fs.Arg(0)

	if *status == "" && *phase == "" {
		return fmt.Errorf("must specify --status or --phase")
	}

	req := StatusUpdateRequest{}
	if *status != "" {
		req.Status = *status
	}
	if *phase != "" {
		req.CurrentPhase = *phase
	}

	var resp TaskActionResponse
	if err := ctx.Put(fmt.Sprintf("/tasks/%s/status", taskID), req, &resp); err != nil {
		return format.WrapAPIError("Failed to update task status", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if resp.DryRun {
		fmt.Printf("[DRY RUN] Would update task\n")
		parts := []string{}
		if *status != "" {
			parts = append(parts, fmt.Sprintf("Status: %s", *status))
		}
		if *phase != "" {
			parts = append(parts, fmt.Sprintf("Phase: %s", *phase))
		}
		fmt.Printf("  %s\n", strings.Join(parts, ", "))
	} else if resp.Success {
		parts := []string{}
		if *status != "" {
			parts = append(parts, fmt.Sprintf("Status: %s", *status))
		}
		if *phase != "" {
			parts = append(parts, fmt.Sprintf("Phase: %s", *phase))
		}
		return format.MutationResult(
			"Task updated successfully",
			strings.Join(parts, ", "),
			resp.NextSteps,
		)
	} else {
		fmt.Printf("Failed to update task: %s\n", resp.Error)
	}
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task delete <id> [--json]")
	}
	taskID := fs.Arg(0)

	var resp TaskActionResponse
	if err := ctx.DeleteWithResult(fmt.Sprintf("/tasks/%s", taskID), &resp); err != nil {
		return format.WrapAPIError("Failed to delete task", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if resp.DryRun {
		fmt.Printf("[DRY RUN] Would delete task: %s\n", taskID)
	} else {
		return format.MutationResult(
			fmt.Sprintf("Task deleted: %s", taskID),
			"", resp.NextSteps,
		)
	}
	return nil
}

// parseSteerQueue splits a comma-separated steer queue string into normalized mode entries.
func parseSteerQueue(raw string) []string {
	parts := strings.Split(raw, ",")
	modes := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(p))
		if trimmed != "" {
			modes = append(modes, trimmed)
		}
	}
	return modes
}
