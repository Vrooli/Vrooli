package projects

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

type listResponse struct {
	Items  json.RawMessage `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "project",
		Description: "Project operations",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List projects", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get project by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a new project", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update an existing project", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a project", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("project list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status (active, paused, complete, archived)")
	limit := fs.Int("limit", 20, "Maximum number of projects to return")
	offset := fs.Int("offset", 0, "Number of projects to skip")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *status != "" {
		query.Set("status", *status)
	}
	query.Set("limit", strconv.Itoa(*limit))
	query.Set("offset", strconv.Itoa(*offset))

	body, err := core.Get("/projects", query)
	if err != nil {
		return err
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	var projects []projectResponse
	if err := json.Unmarshal(resp.Items, &projects); err != nil {
		return fmt.Errorf("parse projects: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Total projects: %d", resp.Total),
			fmt.Sprintf("Window: %d-%d", resp.Offset+1, resp.Offset+len(projects)),
		},
		Results:        renderProjectRows(projects),
		RetrievalHints: []string{cliName + " project get <project-id>", cliName + " task list --project <project-id>"},
	}
	if len(projects) == 0 {
		report.Summary[1] = "Window: 0-0"
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("project get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := core.Get("/projects/"+id, nil)
	if err != nil {
		return err
	}

	var project projectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Project: %s", project.ID), fmt.Sprintf("Status: %s", project.Status)},
		ResultsHeading: "Details",
		Results:        projectDetails(project),
		RetrievalHints: []string{cliName + " task list --project " + project.ID, cliName + " project update " + project.ID + " --status active"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
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

	input := map[string]interface{}{"name": *name}
	if *description != "" {
		input["description"] = *description
	}
	if *color != "" {
		input["color"] = *color
	}

	body, err := core.Request("POST", "/projects", nil, input)
	if err != nil {
		return err
	}

	var project projectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Project created", "Project ID: " + project.ID},
		Changes: []string{
			"Name: " + project.Name,
			"Status: " + project.Status,
		},
		NextCommand: []string{cliName + " project get " + project.ID, cliName + " task create --project " + project.ID + " --title \"...\""},
	}
	if project.Color != "" {
		report.Changes = append(report.Changes, "Color: "+project.Color)
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
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

	body, err := core.Request("PATCH", "/projects/"+id, nil, input)
	if err != nil {
		return err
	}

	var project projectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Project updated", "Project ID: " + project.ID},
		Changes: []string{
			"Name: " + project.Name,
			"Status: " + project.Status,
		},
		NextCommand: []string{cliName + " project get " + project.ID, cliName + " task list --project " + project.ID},
	}
	if project.Color != "" {
		report.Changes = append(report.Changes, "Color: "+project.Color)
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("project delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project delete <id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/projects/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Project deleted", "Project ID: " + id},
		Changes:     []string{"Removed project from active project list"},
		NextCommand: []string{cliName + " project list", cliName + " task list"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderProjectRows(projects []projectResponse) []string {
	if len(projects) == 0 {
		return nil
	}
	rows := make([]string, 0, len(projects))
	for _, project := range projects {
		color := ""
		if project.Color != "" {
			color = " " + project.Color
		}
		rows = append(rows, fmt.Sprintf("%s [%s]%s - %s (%d tasks)", shortID(project.ID), project.Status, color, project.Name, project.TaskCount))
	}
	return rows
}

func projectDetails(project projectResponse) []string {
	lines := []string{
		"Name: " + project.Name,
		fmt.Sprintf("Tasks: %d", project.TaskCount),
		"Created: " + project.CreatedAt,
		"Updated: " + project.UpdatedAt,
	}
	if project.Description != "" {
		lines = append(lines, "Description: "+project.Description)
	}
	if project.Color != "" {
		lines = append(lines, "Color: "+project.Color)
	}
	return lines
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
