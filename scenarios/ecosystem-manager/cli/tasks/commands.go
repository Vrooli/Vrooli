// Package tasks provides CLI commands for task management.
package tasks

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/internal/format"
)

// TaskCreateRequest represents the request body for creating a task.
type TaskCreateRequest struct {
	Title            string   `json:"title"`
	Type             string   `json:"type"`
	Operation        string   `json:"operation"`
	Target           string   `json:"target,omitempty"`
	Category         string   `json:"category,omitempty"`
	Priority         string   `json:"priority,omitempty"`
	SteerMode        string   `json:"steer_mode,omitempty"`
	SteeringQueue    []string `json:"steering_queue,omitempty"`
	AutoSteerProfile string   `json:"auto_steer_profile_id,omitempty"`
}

// TaskResponse represents a task from the API.
type TaskResponse struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Type               string `json:"type"`
	Operation          string `json:"operation"`
	AutoSteerProfileID string `json:"auto_steer_profile_id,omitempty"`
	Category           string `json:"category"`
	Priority           string `json:"priority"`
	Status             string `json:"status"`
	CurrentPhase       string `json:"current_phase,omitempty"`
	CompletionCount    int    `json:"completion_count"`
	LastCompletedAt    string `json:"last_completed_at,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
	SteerMode          string `json:"steer_mode,omitempty"`
	Notes              string `json:"notes,omitempty"`
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.DryRun {
		format.MutationResult(
			fmt.Sprintf("[DRY RUN] Would create task: %s [%s]", task.Title, task.ID),
			"", nil,
		)
	} else {
		format.MutationResult(
			fmt.Sprintf("Created task: %s [%s]", task.Title, task.ID),
			"", resp.NextSteps,
		)
	}
	return nil
}

func cmdImprove(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("improve", flag.ContinueOnError)
	steerProfile := fs.String("steer-profile", "", "Auto-steer profile ID")
	steerMode := fs.String("steer-mode", "", "Single steer mode (e.g. test)")
	steerQueue := fs.String("steer-queue", "", "Comma-separated ordered list of steer modes (improver tasks only)")
	priority := fs.String("priority", "medium", "Priority (low, medium, high, critical)")
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.DryRun {
		format.MutationResult(
			fmt.Sprintf("[DRY RUN] Would create improvement task: %s [%s]", task.Title, task.ID),
			"", nil,
		)
	} else {
		format.MutationResult(
			fmt.Sprintf("Created improvement task: %s [%s]", task.Title, task.ID),
			"", resp.NextSteps,
		)
	}
	return nil
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Tasks) == 0 {
		fmt.Println("No tasks found")
		fmt.Println("  Hint: ecosystem-manager task add scenario <name>")
		return nil
	}

	fmt.Printf("Tasks (%d):\n", resp.Count)
	for _, t := range resp.Tasks {
		phase := ""
		if t.CurrentPhase != "" {
			phase = fmt.Sprintf(" [%s]", t.CurrentPhase)
		}
		fmt.Printf("  %-25s %-8s %-10s %-8s %s%s\n", t.ID, t.Type, t.Operation, t.Priority, t.Title, phase)
	}
	return nil
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(task)
	}

	fmt.Printf("ID: %s\n", task.ID)
	fmt.Printf("Title: %s\n", task.Title)
	fmt.Printf("Type: %s\n", task.Type)
	fmt.Printf("Operation: %s\n", task.Operation)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Priority: %s\n", task.Priority)
	if task.Category != "" {
		fmt.Printf("Category: %s\n", task.Category)
	}
	if task.CurrentPhase != "" {
		fmt.Printf("Phase: %s\n", task.CurrentPhase)
	}
	if task.SteerMode != "" {
		fmt.Printf("Steer Mode: %s\n", task.SteerMode)
	}
	if task.AutoSteerProfileID != "" {
		fmt.Printf("Auto Steer Profile: %s\n", task.AutoSteerProfileID)
	}
	fmt.Printf("Completions: %d\n", task.CompletionCount)
	if task.LastCompletedAt != "" {
		fmt.Printf("Last Completed: %s\n", task.LastCompletedAt)
	}
	if task.CreatedAt != "" {
		fmt.Printf("Created: %s\n", task.CreatedAt)
	}
	if task.UpdatedAt != "" {
		fmt.Printf("Updated: %s\n", task.UpdatedAt)
	}
	if task.Notes != "" {
		fmt.Printf("Notes: %s\n", task.Notes)
	}
	return nil
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
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
		format.MutationResult(
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.DryRun {
		fmt.Printf("[DRY RUN] Would delete task: %s\n", taskID)
	} else {
		format.MutationResult(
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
