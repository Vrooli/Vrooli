package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Task Command Dispatcher
// =============================================================================

func (a *App) cmdTask(args []string) error {
	if len(args) == 0 {
		return a.taskHelp()
	}

	switch args[0] {
	case "list":
		return a.taskList(args[1:])
	case "get":
		return a.taskGet(args[1:])
	case "create":
		return a.taskCreate(args[1:])
	case "cancel":
		return a.taskCancel(args[1:])
	case "help", "-h", "--help":
		return a.taskHelp()
	default:
		return fmt.Errorf("unknown task subcommand: %s\n\nRun 'agent-manager task help' for usage", args[0])
	}
}

func (a *App) taskHelp() error {
	fmt.Println(`Usage: agent-manager task <subcommand> [options]

Subcommands:
  list              List all tasks
  get <id>          Get task details
  create            Create a new task
  cancel <id>       Cancel a queued or running task

Options:
  --json            Output raw JSON
  --quiet           Output only IDs (for piping)
  --status          Filter by status (queued, running, complete, failed, cancelled)

Examples:
  agent-manager task list
  agent-manager task list --status running
  agent-manager task get abc123
  agent-manager task create --title "Fix bug" --scope-path ./src
  agent-manager task cancel abc123`)
	return nil
}

// =============================================================================
// Task List
// =============================================================================

func (a *App) taskList(args []string) error {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	quiet := fs.Bool("quiet", false, "Output only IDs")
	limit := fs.Int("limit", 0, "Maximum number of tasks to return")
	offset := fs.Int("offset", 0, "Number of tasks to skip")
	status := fs.String("status", "", "Filter by status")

	if err := fs.Parse(args); err != nil {
		return err
	}

	body, tasks, err := a.services.Tasks.List(*limit, *offset, *status)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if tasks == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if *quiet {
		for _, t := range tasks {
			fmt.Println(t.ID)
		}
		return nil
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	fmt.Printf("%-36s  %-30s  %-12s  %-20s\n", "ID", "TITLE", "STATUS", "CREATED")
	fmt.Printf("%-36s  %-30s  %-12s  %-20s\n", strings.Repeat("-", 36), strings.Repeat("-", 30), strings.Repeat("-", 12), strings.Repeat("-", 20))
	for _, t := range tasks {
		title := t.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		created := t.CreatedAt
		if len(created) > 20 {
			created = created[:19]
		}
		fmt.Printf("%-36s  %-30s  %-12s  %-20s\n", t.ID, title, t.Status, created)
	}

	return nil
}

// =============================================================================
// Task Get
// =============================================================================

func (a *App) taskGet(args []string) error {
	fs := flag.NewFlagSet("task get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("usage: agent-manager task get <id>")
	}

	id := remaining[0]
	body, task, err := a.services.Tasks.Get(id)
	if err != nil {
		return err
	}

	if *jsonOutput || task == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:          %s\n", task.ID)
	fmt.Printf("Title:       %s\n", task.Title)
	if task.Description != "" {
		fmt.Printf("Description: %s\n", task.Description)
	}
	fmt.Printf("Status:      %s\n", task.Status)
	if task.ScopePath != "" {
		fmt.Printf("Scope Path:  %s\n", task.ScopePath)
	}
	if task.ProjectRoot != "" {
		fmt.Printf("Project Root: %s\n", task.ProjectRoot)
	}
	if len(task.ContextAttachments) > 0 {
		fmt.Println("Context Attachments:")
		for _, att := range task.ContextAttachments {
			switch att.Type {
			case "file":
				fmt.Printf("  - [file] %s\n", att.Path)
			case "link":
				fmt.Printf("  - [link] %s\n", att.URL)
			case "note":
				fmt.Printf("  - [note] %s\n", truncate(att.Content, 50))
			}
		}
	}
	if task.CreatedBy != "" {
		fmt.Printf("Created By:  %s\n", task.CreatedBy)
	}
	if task.CreatedAt != "" {
		fmt.Printf("Created:     %s\n", task.CreatedAt)
	}
	if task.UpdatedAt != "" {
		fmt.Printf("Updated:     %s\n", task.UpdatedAt)
	}

	return nil
}

// =============================================================================
// Task Create
// =============================================================================

func (a *App) taskCreate(args []string) error {
	fs := flag.NewFlagSet("task create", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	title := fs.String("title", "", "Task title (required)")
	description := fs.String("description", "", "Task description")
	scopePath := fs.String("scope-path", "", "Scope path for agent operations")
	projectRoot := fs.String("project-root", "", "Project root directory")
	createdBy := fs.String("created-by", "", "Creator identifier")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *title == "" {
		return fmt.Errorf("--title is required")
	}

	req := CreateTaskRequest{
		Title:       *title,
		Description: *description,
		ScopePath:   *scopePath,
		ProjectRoot: *projectRoot,
		CreatedBy:   *createdBy,
	}

	body, task, err := a.services.Tasks.Create(req)
	if err != nil {
		return err
	}

	if *jsonOutput || task == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created task: %s (%s)\n", task.Title, task.ID)
	return nil
}

// =============================================================================
// Task Cancel
// =============================================================================

func (a *App) taskCancel(args []string) error {
	fs := flag.NewFlagSet("task cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager task cancel <id>")
	}

	body, err := a.services.Tasks.Cancel(id)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Cancelled task: %s\n", id)
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
