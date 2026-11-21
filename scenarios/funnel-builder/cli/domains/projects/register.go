package projects

import (
	"fmt"
	"os"
	"strings"

	"funnel-builder/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "funnel-builder"

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "projects",
		Description: "Manage funnel projects",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List projects", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a project", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update a project", Run: func(args []string) error { return runUpdate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("projects list")
	tenantID := fs.String("tenant", "", "Tenant ID filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/projects", support.BuildQuery(map[string]string{
		"tenant_id": *tenantID,
	}))
	if err != nil {
		return err
	}

	var projects []support.Project
	if err := support.Decode(body, &projects); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Projects: %d", len(projects)),
		},
		ResultsHeading: "Projects",
		Results:        projectRows(projects),
		RetrievalHints: []string{fmt.Sprintf("%s funnels create --name \"Lead Magnet\" --project <project-id>", cliName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("projects create")
	name := fs.String("name", "", "Project name")
	description := fs.String("description", "", "Project description")
	tenantID := fs.String("tenant", "", "Tenant ID")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("usage: projects create --name <name> [--description <text>] [--tenant <id>]")
	}

	body, err := core.Request("POST", "/projects", nil, map[string]interface{}{
		"name":        strings.TrimSpace(*name),
		"description": strings.TrimSpace(*description),
		"tenant_id":   emptyToNil(*tenantID),
	})
	if err != nil {
		return err
	}

	var project support.Project
	if err := support.Decode(body, &project); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Project %s created", project.ID),
			fmt.Sprintf("Name: %s", project.Name),
		},
		Changes: []string{
			fmt.Sprintf("Description: %s", blankFallback(project.Description, "(none)")),
			fmt.Sprintf("Funnels attached: %d", len(project.Funnels)),
		},
		NextCommand: []string{
			fmt.Sprintf("%s funnels create --name \"New Funnel\" --project %s", cliName, project.ID),
			fmt.Sprintf("%s projects update %s --description \"Refined positioning\"", cliName, project.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("projects update")
	name := fs.String("name", "", "Project name")
	description := fs.String("description", "", "Project description")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: projects update <project-id> [--name <name>] [--description <text>]")
	}
	projectID := fs.Arg(0)

	project, err := fetchProject(core, projectID)
	if err != nil {
		return err
	}

	resolvedName := project.Name
	if strings.TrimSpace(*name) != "" {
		resolvedName = strings.TrimSpace(*name)
	}
	resolvedDescription := project.Description
	if fs.Lookup("description") != nil && fs.Lookup("description").Value.String() != "" {
		resolvedDescription = strings.TrimSpace(*description)
	}

	body, err := core.Request("PUT", "/projects/"+projectID, nil, map[string]interface{}{
		"name":        resolvedName,
		"description": resolvedDescription,
	})
	if err != nil {
		return err
	}

	var updated support.Project
	if err := support.Decode(body, &updated); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Project %s updated", updated.ID),
		},
		Changes: []string{
			fmt.Sprintf("Name: %s", updated.Name),
			fmt.Sprintf("Description: %s", blankFallback(updated.Description, "(none)")),
			fmt.Sprintf("Funnels attached: %d", len(updated.Funnels)),
		},
		NextCommand: []string{
			fmt.Sprintf("%s projects list", cliName),
			fmt.Sprintf("%s funnels list", cliName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func fetchProject(core *cliapp.ScenarioApp, projectID string) (support.Project, error) {
	body, err := core.Get("/projects", nil)
	if err != nil {
		return support.Project{}, err
	}
	var projects []support.Project
	if err := support.Decode(body, &projects); err != nil {
		return support.Project{}, err
	}
	for _, project := range projects {
		if project.ID == projectID {
			return project, nil
		}
	}
	return support.Project{}, fmt.Errorf("project %s not found", projectID)
}

func projectRows(projects []support.Project) []string {
	if len(projects) == 0 {
		return []string{"No projects found"}
	}
	rows := make([]string, 0, len(projects))
	for _, project := range projects {
		rows = append(rows, fmt.Sprintf("%s | %s | funnels=%d | updated=%s", project.ID, project.Name, len(project.Funnels), support.FormatTimeValue(project.UpdatedAt)))
	}
	return rows
}

func emptyToNil(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func blankFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
