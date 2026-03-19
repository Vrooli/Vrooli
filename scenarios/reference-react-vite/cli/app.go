package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// makeQuery converts a map[string]string to url.Values
func makeQuery(params map[string]string) url.Values {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return q
}

const (
	appName        = "reference-react-vite"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Reference React Vite CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
		},
	}

	// Tasks domain: core work items
	tasks := cliapp.CommandGroup{
		Title: "Tasks",
		Commands: []cliapp.Command{
			{Name: "task list", NeedsAPI: true, Description: "List tasks", Run: a.cmdTaskList},
			{Name: "task get", NeedsAPI: true, Description: "Get task by ID", Run: a.cmdTaskGet},
			{Name: "task create", NeedsAPI: true, Description: "Create a new task", Run: a.cmdTaskCreate},
			{Name: "task update", NeedsAPI: true, Description: "Update an existing task", Run: a.cmdTaskUpdate},
			{Name: "task delete", NeedsAPI: true, Description: "Delete a task", Run: a.cmdTaskDelete},
		},
	}

	// Projects domain: containers for organizing tasks
	projects := cliapp.CommandGroup{
		Title: "Projects",
		Commands: []cliapp.Command{
			{Name: "project list", NeedsAPI: true, Description: "List projects", Run: a.cmdProjectList},
			{Name: "project get", NeedsAPI: true, Description: "Get project by ID", Run: a.cmdProjectGet},
			{Name: "project create", NeedsAPI: true, Description: "Create a new project", Run: a.cmdProjectCreate},
			{Name: "project update", NeedsAPI: true, Description: "Update an existing project", Run: a.cmdProjectUpdate},
			{Name: "project delete", NeedsAPI: true, Description: "Delete a project", Run: a.cmdProjectDelete},
		},
	}

	// Notes domain: annotations attached to tasks
	notes := cliapp.CommandGroup{
		Title: "Notes",
		Commands: []cliapp.Command{
			{Name: "note list", NeedsAPI: true, Description: "List notes for a task", Run: a.cmdNoteList},
			{Name: "note get", NeedsAPI: true, Description: "Get note by ID", Run: a.cmdNoteGet},
			{Name: "note create", NeedsAPI: true, Description: "Create a new note on a task", Run: a.cmdNoteCreate},
			{Name: "note update", NeedsAPI: true, Description: "Update an existing note", Run: a.cmdNoteUpdate},
			{Name: "note delete", NeedsAPI: true, Description: "Delete a note", Run: a.cmdNoteDelete},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, tasks, projects, notes, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

func (a *App) cmdStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		fmt.Printf("Ready: %v\n", parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// ============================================================================
// Task Commands
// ============================================================================

// taskResponse represents a task from the API
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

// listResponse is the generic paginated list response
type listResponse struct {
	Items  json.RawMessage `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func (a *App) cmdTaskList(args []string) error {
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

	params := make(map[string]string)
	if *projectID != "" {
		params["project_id"] = *projectID
	}
	if *status != "" {
		params["status"] = *status
	}
	if *priority != 0 {
		params["priority"] = strconv.Itoa(*priority)
	}
	params["limit"] = strconv.Itoa(*limit)
	params["offset"] = strconv.Itoa(*offset)

	body, err := a.core.APIClient.Get(a.apiPath("/tasks"), makeQuery(params))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	var tasks []taskResponse
	if err := json.Unmarshal(resp.Items, &tasks); err != nil {
		return fmt.Errorf("parse tasks: %w", err)
	}

	fmt.Printf("Tasks (%d total, showing %d-%d)\n", resp.Total, resp.Offset+1, resp.Offset+len(tasks))
	fmt.Println(strings.Repeat("-", 60))
	for _, t := range tasks {
		priorityStr := map[int]string{1: "Low", 2: "Medium", 3: "High"}[t.Priority]
		fmt.Printf("  %s [%s] P:%s - %s\n", t.ID[:8], t.Status, priorityStr, t.Title)
	}
	return nil
}

func (a *App) cmdTaskGet(args []string) error {
	fs := flag.NewFlagSet("task get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/tasks/"+id), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var task taskResponse
	if err := json.Unmarshal(body, &task); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	priorityStr := map[int]string{1: "Low", 2: "Medium", 3: "High"}[task.Priority]
	fmt.Printf("Task: %s\n", task.ID)
	fmt.Printf("Title: %s\n", task.Title)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Priority: %s\n", priorityStr)
	if task.Description != "" {
		fmt.Printf("Description: %s\n", task.Description)
	}
	if task.ProjectID != "" {
		fmt.Printf("Project: %s\n", task.ProjectID)
	}
	if task.DueDate != "" {
		fmt.Printf("Due: %s\n", task.DueDate)
	}
	fmt.Printf("Created: %s\n", task.CreatedAt)
	fmt.Printf("Updated: %s\n", task.UpdatedAt)
	return nil
}

func (a *App) cmdTaskCreate(args []string) error {
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

	input := map[string]interface{}{
		"title": *title,
	}
	if *description != "" {
		input["description"] = *description
	}
	if *projectID != "" {
		input["project_id"] = *projectID
	}
	if *priority != 0 {
		input["priority"] = *priority
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/tasks"), nil, input)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var task taskResponse
	if err := json.Unmarshal(body, &task); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Created task: %s\n", task.ID)
	fmt.Printf("Title: %s\n", task.Title)
	return nil
}

func (a *App) cmdTaskUpdate(args []string) error {
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

	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/tasks/"+id), nil, input)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var task taskResponse
	if err := json.Unmarshal(body, &task); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Updated task: %s\n", task.ID)
	fmt.Printf("Title: %s\n", task.Title)
	fmt.Printf("Status: %s\n", task.Status)
	return nil
}

func (a *App) cmdTaskDelete(args []string) error {
	fs := flag.NewFlagSet("task delete", flag.ContinueOnError)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: task delete <id>")
	}
	id := fs.Arg(0)

	_, err := a.core.APIClient.Request("DELETE", a.apiPath("/tasks/"+id), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted task: %s\n", id)
	return nil
}

// ============================================================================
// Project Commands
// ============================================================================

// projectResponse represents a project from the API
type projectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Color       string `json:"color,omitempty"`
	TaskCount   int    `json:"task_count,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (a *App) cmdProjectList(args []string) error {
	fs := flag.NewFlagSet("project list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status (active, paused, complete, archived)")
	limit := fs.Int("limit", 20, "Maximum number of projects to return")
	offset := fs.Int("offset", 0, "Number of projects to skip")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	params := make(map[string]string)
	if *status != "" {
		params["status"] = *status
	}
	params["limit"] = strconv.Itoa(*limit)
	params["offset"] = strconv.Itoa(*offset)

	body, err := a.core.APIClient.Get(a.apiPath("/projects"), makeQuery(params))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	var projects []projectResponse
	if err := json.Unmarshal(resp.Items, &projects); err != nil {
		return fmt.Errorf("parse projects: %w", err)
	}

	fmt.Printf("Projects (%d total, showing %d-%d)\n", resp.Total, resp.Offset+1, resp.Offset+len(projects))
	fmt.Println(strings.Repeat("-", 60))
	for _, p := range projects {
		colorStr := ""
		if p.Color != "" {
			colorStr = " " + p.Color
		}
		fmt.Printf("  %s [%s]%s - %s (%d tasks)\n", p.ID[:8], p.Status, colorStr, p.Name, p.TaskCount)
	}
	return nil
}

func (a *App) cmdProjectGet(args []string) error {
	fs := flag.NewFlagSet("project get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/projects/"+id), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var project projectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Project: %s\n", project.ID)
	fmt.Printf("Name: %s\n", project.Name)
	fmt.Printf("Status: %s\n", project.Status)
	if project.Description != "" {
		fmt.Printf("Description: %s\n", project.Description)
	}
	if project.Color != "" {
		fmt.Printf("Color: %s\n", project.Color)
	}
	fmt.Printf("Tasks: %d\n", project.TaskCount)
	fmt.Printf("Created: %s\n", project.CreatedAt)
	fmt.Printf("Updated: %s\n", project.UpdatedAt)
	return nil
}

func (a *App) cmdProjectCreate(args []string) error {
	fs := flag.NewFlagSet("project create", flag.ContinueOnError)
	name := fs.String("name", "", "Project name (required)")
	description := fs.String("description", "", "Project description")
	color := fs.String("color", "", "Project color (hex code, e.g., #FF5733)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	input := map[string]interface{}{
		"name": *name,
	}
	if *description != "" {
		input["description"] = *description
	}
	if *color != "" {
		input["color"] = *color
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/projects"), nil, input)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var project projectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Created project: %s\n", project.ID)
	fmt.Printf("Name: %s\n", project.Name)
	return nil
}

func (a *App) cmdProjectUpdate(args []string) error {
	fs := flag.NewFlagSet("project update", flag.ContinueOnError)
	name := fs.String("name", "", "New project name")
	description := fs.String("description", "", "New project description")
	status := fs.String("status", "", "New status (active, paused, complete, archived)")
	color := fs.String("color", "", "New project color (hex code)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project update <id> [--name NAME] [--status STATUS] [--json]")
	}
	id := fs.Arg(0)

	input := make(map[string]interface{})
	if *name != "" {
		input["name"] = *name
	}
	if *description != "" {
		input["description"] = *description
	}
	if *status != "" {
		input["status"] = *status
	}
	if *color != "" {
		input["color"] = *color
	}

	if len(input) == 0 {
		return fmt.Errorf("at least one field must be specified to update")
	}

	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/projects/"+id), nil, input)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var project projectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Updated project: %s\n", project.ID)
	fmt.Printf("Name: %s\n", project.Name)
	fmt.Printf("Status: %s\n", project.Status)
	return nil
}

func (a *App) cmdProjectDelete(args []string) error {
	fs := flag.NewFlagSet("project delete", flag.ContinueOnError)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project delete <id>")
	}
	id := fs.Arg(0)

	_, err := a.core.APIClient.Request("DELETE", a.apiPath("/projects/"+id), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted project: %s\n", id)
	return nil
}

// ============================================================================
// Note Commands
// ============================================================================

// noteResponse represents a note from the API
type noteResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Content   string `json:"content"`
	Author    string `json:"author,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (a *App) cmdNoteList(args []string) error {
	fs := flag.NewFlagSet("note list", flag.ContinueOnError)
	taskID := fs.String("task", "", "Task ID to list notes for (required)")
	limit := fs.Int("limit", 20, "Maximum number of notes to return")
	offset := fs.Int("offset", 0, "Number of notes to skip")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *taskID == "" {
		return fmt.Errorf("--task is required")
	}

	params := make(map[string]string)
	params["limit"] = strconv.Itoa(*limit)
	params["offset"] = strconv.Itoa(*offset)

	body, err := a.core.APIClient.Get(a.apiPath("/tasks/"+*taskID+"/notes"), makeQuery(params))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	var notes []noteResponse
	if err := json.Unmarshal(resp.Items, &notes); err != nil {
		return fmt.Errorf("parse notes: %w", err)
	}

	fmt.Printf("Notes for task %s (%d total, showing %d-%d)\n", *taskID, resp.Total, resp.Offset+1, resp.Offset+len(notes))
	fmt.Println(strings.Repeat("-", 60))
	for _, n := range notes {
		author := n.Author
		if author == "" {
			author = "anonymous"
		}
		// Truncate content for display
		content := n.Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		fmt.Printf("  %s [%s] %s\n", n.ID[:8], author, content)
	}
	return nil
}

func (a *App) cmdNoteGet(args []string) error {
	fs := flag.NewFlagSet("note get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := a.core.APIClient.Get(a.apiPath("/notes/"+id), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var note noteResponse
	if err := json.Unmarshal(body, &note); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Note: %s\n", note.ID)
	fmt.Printf("Task: %s\n", note.TaskID)
	if note.Author != "" {
		fmt.Printf("Author: %s\n", note.Author)
	}
	fmt.Printf("Content:\n%s\n", note.Content)
	fmt.Printf("Created: %s\n", note.CreatedAt)
	fmt.Printf("Updated: %s\n", note.UpdatedAt)
	return nil
}

func (a *App) cmdNoteCreate(args []string) error {
	fs := flag.NewFlagSet("note create", flag.ContinueOnError)
	taskID := fs.String("task", "", "Task ID to attach note to (required)")
	content := fs.String("content", "", "Note content (required)")
	author := fs.String("author", "", "Note author")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *taskID == "" {
		return fmt.Errorf("--task is required")
	}
	if *content == "" {
		return fmt.Errorf("--content is required")
	}

	input := map[string]interface{}{
		"content": *content,
	}
	if *author != "" {
		input["author"] = *author
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/tasks/"+*taskID+"/notes"), nil, input)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var note noteResponse
	if err := json.Unmarshal(body, &note); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Created note: %s\n", note.ID)
	fmt.Printf("Task: %s\n", note.TaskID)
	return nil
}

func (a *App) cmdNoteUpdate(args []string) error {
	fs := flag.NewFlagSet("note update", flag.ContinueOnError)
	content := fs.String("content", "", "New note content")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note update <id> --content CONTENT [--json]")
	}
	id := fs.Arg(0)

	if *content == "" {
		return fmt.Errorf("--content is required")
	}

	input := map[string]interface{}{
		"content": *content,
	}

	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/notes/"+id), nil, input)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var note noteResponse
	if err := json.Unmarshal(body, &note); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Updated note: %s\n", note.ID)
	return nil
}

func (a *App) cmdNoteDelete(args []string) error {
	fs := flag.NewFlagSet("note delete", flag.ContinueOnError)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note delete <id>")
	}
	id := fs.Arg(0)

	_, err := a.core.APIClient.Request("DELETE", a.apiPath("/notes/"+id), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted note: %s\n", id)
	return nil
}
