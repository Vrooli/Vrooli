package projects

import (
	"fmt"
	"os"
	"strings"

	"vrooli-bridge/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes project discovery and integration lifecycle as the
// `project` subcommand group. The API is the source of truth for scan +
// integrate + remove; this package is a thin wrapper over /api/v1/projects*.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "project",
		Description: "Discover and manage Vrooli integrations for external projects",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List discovered projects", Run: func(args []string) error { return runList(core, args) }},
			{Name: "scan", Description: "Scan a directory for projects", Run: func(args []string) error { return runScan(core, args) }},
			{Name: "integrate", Description: "Install Vrooli integration files for a project", Run: func(args []string) error { return runIntegrate(core, args) }},
			{Name: "remove", Aliases: []string{"rm"}, Description: "Remove a project's Vrooli integration", Run: func(args []string) error { return runRemove(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("project list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/projects", nil)
	if err != nil {
		return err
	}
	var resp support.ProjectsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Discovered projects: %d", len(resp.Projects))},
		ResultsHeading: "Projects",
		Results:        projectRows(resp.Projects),
		RetrievalHints: []string{
			fmt.Sprintf("%s project scan <path> [--depth N]", support.CLIName),
			fmt.Sprintf("%s project integrate <project-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("project scan")
	depth := fs.Int("depth", 3, "Maximum directory scan depth")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full scan request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	switch {
	case strings.TrimSpace(*bodyFile) != "":
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	default:
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: project scan <path> [--depth N] [--body-file PATH]")
		}
		payload = map[string]interface{}{
			"directories": []string{fs.Arg(0)},
			"depth":       *depth,
		}
	}

	body, err := core.Request("POST", "/projects/scan", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ScanResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Projects found: %d", resp.Found),
		fmt.Sprintf("New projects registered: %d", resp.New),
	}
	result := []string{fmt.Sprintf("Scan complete: %d found, %d new", resp.Found, resp.New)}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s project list", support.CLIName),
			fmt.Sprintf("%s project integrate <project-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runIntegrate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("project integrate")
	force := fs.Bool("force", false, "Overwrite existing integration files")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full integrate request body (overrides --force)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project integrate <project-id> [--force] [--body-file PATH]")
	}
	id := fs.Arg(0)

	var payload interface{}
	switch {
	case strings.TrimSpace(*bodyFile) != "":
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	default:
		payload = map[string]interface{}{"force": *force}
	}

	body, err := core.Request("POST", "/projects/"+id+"/integrate", nil, payload)
	if err != nil {
		return err
	}
	var resp support.IntegrateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	message := strings.TrimSpace(resp.Message)
	if message == "" {
		message = support.EnvelopeMessage(body)
	}
	if message == "" {
		if resp.Success {
			message = fmt.Sprintf("Integration completed for %s", id)
		} else {
			message = fmt.Sprintf("Integration failed for %s", id)
		}
	}

	changes := make([]string, 0, len(resp.FilesCreated)+len(resp.FilesUpdated))
	for _, f := range resp.FilesCreated {
		changes = append(changes, "created: "+f)
	}
	for _, f := range resp.FilesUpdated {
		changes = append(changes, "updated: "+f)
	}
	if len(changes) == 0 {
		changes = []string{"no file changes reported"}
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s project list", support.CLIName),
			fmt.Sprintf("%s project remove %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	if !resp.Success && message != "" {
		return cliapp.RenderMutationReport(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRemove(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("project remove")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: project remove <project-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/projects/"+id, nil, nil)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Integration removed for %s", id)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Project %s: integration removed", id)},
		NextCommand: []string{
			fmt.Sprintf("%s project list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func projectRows(projects []support.Project) []string {
	if len(projects) == 0 {
		return []string{"No projects discovered"}
	}
	rows := make([]string, 0, len(projects))
	for _, p := range projects {
		version := support.PtrString(p.VrooliVersion)
		if version == "" {
			version = "-"
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | type=%s | status=%s | vrooli=%s | path=%s",
			p.Name, support.ShortID(p.ID), p.Type, p.IntegrationStatus, version, p.Path))
	}
	return rows
}
