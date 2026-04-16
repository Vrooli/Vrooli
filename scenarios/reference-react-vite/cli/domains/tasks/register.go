package tasks

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "reference-react-vite"

type taskResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	DueDate     string `json:"due_date,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type listResponse struct {
	Items  json.RawMessage `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "task",
		Description: "Task operations",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List tasks", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get task by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a new task", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update an existing task", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a task", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	projectID := fs.String("project", "", "Filter by project ID")
	status := fs.String("status", "", "Filter by status (pending, in_progress, completed, archived)")
	priority := fs.Int("priority", 0, "Filter by priority (1=low, 2=medium, 3=high)")
	limit := fs.Int("limit", 20, "Maximum number of tasks to return")
	offset := fs.Int("offset", 0, "Number of tasks to skip")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *projectID != "" {
		query.Set("project_id", *projectID)
	}
	if *status != "" {
		query.Set("status", *status)
	}
	if *priority != 0 {
		query.Set("priority", strconv.Itoa(*priority))
	}
	query.Set("limit", strconv.Itoa(*limit))
	query.Set("offset", strconv.Itoa(*offset))

	body, err := core.Get("/tasks", query)
	if err != nil {
		return err
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	var tasks []taskResponse
	if err := json.Unmarshal(resp.Items, &tasks); err != nil {
		return fmt.Errorf("parse tasks: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Total tasks: %d", resp.Total),
			fmt.Sprintf("Window: %d-%d", resp.Offset+1, resp.Offset+len(tasks)),
		},
		Results:        renderTaskRows(tasks),
		RetrievalHints: []string{cliName + " task get <task-id>", cliName + " task list --status pending"},
	}
	if len(tasks) == 0 {
		report.Summary[1] = "Window: 0-0"
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("task get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := core.Get("/tasks/"+id, nil)
	if err != nil {
		return err
	}

	var task taskResponse
	if err := json.Unmarshal(body, &task); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Task: %s", task.ID), fmt.Sprintf("Status: %s", task.Status)},
		ResultsHeading: "Details",
		Results:        taskDetails(task),
		RetrievalHints: []string{cliName + " task update " + task.ID + " --status completed", cliName + " note list --task " + task.ID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("task create", flag.ContinueOnError)
	title := fs.String("title", "", "Task title (required)")
	description := fs.String("description", "", "Task description")
	projectID := fs.String("project", "", "Project ID to assign task to")
	priority := fs.Int("priority", 0, "Priority (1=low, 2=medium, 3=high)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *title == "" {
		return fmt.Errorf("--title is required")
	}

	input := map[string]interface{}{"title": *title}
	if *description != "" {
		input["description"] = *description
	}
	if *projectID != "" {
		input["project_id"] = *projectID
	}
	if *priority != 0 {
		input["priority"] = *priority
	}

	body, err := core.Request("POST", "/tasks", nil, input)
	if err != nil {
		return err
	}

	var task taskResponse
	if err := json.Unmarshal(body, &task); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Task created", "Task ID: " + task.ID},
		Changes: []string{
			"Title: " + task.Title,
			"Status: " + task.Status,
			"Priority: " + priorityLabel(task.Priority),
		},
		NextCommand: []string{cliName + " task get " + task.ID, cliName + " note create --task " + task.ID + " --content \"...\""},
	}
	if task.ProjectID != "" {
		report.Changes = append(report.Changes, "Project: "+task.ProjectID)
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("task update", flag.ContinueOnError)
	title := fs.String("title", "", "New task title")
	description := fs.String("description", "", "New task description")
	status := fs.String("status", "", "New status (pending, in_progress, completed, archived)")
	priority := fs.Int("priority", 0, "New priority (1=low, 2=medium, 3=high)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task update <id> [--title TITLE] [--status STATUS] [--json]")
	}
	id := fs.Arg(0)

	input := make(map[string]interface{})
	if *title != "" {
		input["title"] = *title
	}
	if *description != "" {
		input["description"] = *description
	}
	if *status != "" {
		input["status"] = *status
	}
	if *priority != 0 {
		input["priority"] = *priority
	}
	if len(input) == 0 {
		return fmt.Errorf("at least one field must be specified to update")
	}

	body, err := core.Request("PATCH", "/tasks/"+id, nil, input)
	if err != nil {
		return err
	}

	var task taskResponse
	if err := json.Unmarshal(body, &task); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Task updated", "Task ID: " + task.ID},
		Changes: []string{
			"Title: " + task.Title,
			"Status: " + task.Status,
			"Priority: " + priorityLabel(task.Priority),
		},
		NextCommand: []string{cliName + " task get " + task.ID, cliName + " task list --status " + task.Status},
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("task delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task delete <id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/tasks/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Task deleted", "Task ID: " + id},
		Changes:     []string{"Removed task from active task list"},
		NextCommand: []string{cliName + " task list", cliName + " project list"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderTaskRows(tasks []taskResponse) []string {
	if len(tasks) == 0 {
		return nil
	}
	rows := make([]string, 0, len(tasks))
	for _, task := range tasks {
		rows = append(rows, fmt.Sprintf("%s [%s] P:%s - %s", shortID(task.ID), task.Status, priorityLabel(task.Priority), task.Title))
	}
	return rows
}

func taskDetails(task taskResponse) []string {
	lines := []string{
		"Title: " + task.Title,
		"Priority: " + priorityLabel(task.Priority),
		"Created: " + task.CreatedAt,
		"Updated: " + task.UpdatedAt,
	}
	if task.Description != "" {
		lines = append(lines, "Description: "+task.Description)
	}
	if task.ProjectID != "" {
		lines = append(lines, "Project: "+task.ProjectID)
	}
	if task.DueDate != "" {
		lines = append(lines, "Due: "+task.DueDate)
	}
	return lines
}

func priorityLabel(priority int) string {
	if label, ok := map[int]string{1: "Low", 2: "Medium", 3: "High"}[priority]; ok {
		return label
	}
	if priority == 0 {
		return "Unspecified"
	}
	return strconv.Itoa(priority)
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
