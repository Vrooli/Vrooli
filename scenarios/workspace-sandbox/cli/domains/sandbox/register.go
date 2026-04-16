package sandbox

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "sandbox",
		Description: "Create, inspect, and manage sandbox lifecycles",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a sandbox", Run: func(args []string) error { return runCreate(deps, args) }},
			{Name: "list", Description: "List sandboxes", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "inspect", Description: "Show sandbox details", Run: func(args []string) error { return runInspect(deps, args) }},
			{Name: "stop", Description: "Stop a sandbox while preserving it", Run: func(args []string) error { return runStop(deps, args) }},
			{Name: "delete", Description: "Delete a sandbox and all data", Run: func(args []string) error { return runDelete(deps, args) }},
			{Name: "workspace", Description: "Show the mounted workspace path", Run: func(args []string) error { return runWorkspace(deps, args) }},
		},
	}
}

func runCreate(deps support.Dependencies, args []string) error {
	var scope, project, owner string
	var reservedPaths []string
	var jsonOut bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--scope="):
			scope = strings.TrimPrefix(arg, "--scope=")
		case strings.HasPrefix(arg, "-s="):
			scope = strings.TrimPrefix(arg, "-s=")
		case strings.HasPrefix(arg, "--reserve="):
			reservedPaths = append(reservedPaths, strings.TrimPrefix(arg, "--reserve="))
		case strings.HasPrefix(arg, "--reserved="):
			reservedPaths = append(reservedPaths, strings.TrimPrefix(arg, "--reserved="))
		case strings.HasPrefix(arg, "-r="):
			reservedPaths = append(reservedPaths, strings.TrimPrefix(arg, "-r="))
		case strings.HasPrefix(arg, "--reserved-paths="):
			for _, path := range strings.Split(strings.TrimPrefix(arg, "--reserved-paths="), ",") {
				path = strings.TrimSpace(path)
				if path != "" {
					reservedPaths = append(reservedPaths, path)
				}
			}
		case strings.HasPrefix(arg, "--project="):
			project = strings.TrimPrefix(arg, "--project=")
		case strings.HasPrefix(arg, "-p="):
			project = strings.TrimPrefix(arg, "-p=")
		case strings.HasPrefix(arg, "--owner="):
			owner = strings.TrimPrefix(arg, "--owner=")
		case strings.HasPrefix(arg, "-o="):
			owner = strings.TrimPrefix(arg, "-o=")
		case arg == "--json":
			jsonOut = true
		}
	}

	if scope == "" && project != "" {
		scope = project
	}

	reqBody := map[string]any{}
	if scope != "" {
		reqBody["scopePath"] = scope
	}
	if project != "" {
		reqBody["projectRoot"] = project
	}
	if owner != "" {
		reqBody["owner"] = owner
	}
	if len(reservedPaths) > 0 {
		reqBody["reservedPath"] = reservedPaths[0]
		reqBody["reservedPaths"] = reservedPaths
	}

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes", nil, reqBody)
	if err != nil {
		return err
	}

	var sandbox support.SandboxResponse
	if err := json.Unmarshal(body, &sandbox); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Sandbox created",
			"Sandbox ID: " + sandbox.ID,
		},
		Changes: []string{
			"Status: " + sandbox.Status,
			"Scope: " + sandbox.ScopePath,
		},
		NextCommand: []string{
			support.CLIName + " sandbox inspect " + sandbox.ID,
			support.CLIName + " process shell " + sandbox.ID,
		},
	}
	if sandbox.ProjectRoot != "" {
		report.Changes = append(report.Changes, "Project root: "+sandbox.ProjectRoot)
	}
	if reserved := reservedSummary(sandbox); reserved != "" {
		report.Changes = append(report.Changes, "Reserved: "+reserved)
	}
	if sandbox.MergedDir != "" {
		report.Changes = append(report.Changes, "Workspace: "+sandbox.MergedDir)
	}

	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("sandbox list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by sandbox status")
	owner := fs.String("owner", "", "Filter by owner ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *status != "" {
		query.Set("status", *status)
	}
	if *owner != "" {
		query.Set("owner", *owner)
	}

	body, err := deps.ScenarioApp().Get("/sandboxes", query)
	if err != nil {
		return err
	}

	var resp support.ListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Sandboxes found: %d", resp.TotalCount),
		},
		Results:        renderSandboxRows(resp.Sandboxes),
		RetrievalHints: []string{support.CLIName + " sandbox inspect <sandbox-id>", support.CLIName + " process list <sandbox-id>"},
	}
	if *status != "" {
		report.Summary = append(report.Summary, "Status filter: "+*status)
	}
	if *owner != "" {
		report.Summary = append(report.Summary, "Owner filter: "+*owner)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runInspect(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("sandbox inspect", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s sandbox inspect <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Get("/sandboxes/"+sandboxID, nil)
	if err != nil {
		return err
	}

	var sandbox support.SandboxResponse
	if err := json.Unmarshal(body, &sandbox); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Sandbox ID: " + sandbox.ID,
			"Status: " + sandbox.Status,
		},
		ResultsHeading: "Details",
		Results: []string{
			"Scope: " + sandbox.ScopePath,
			"Project root: " + sandbox.ProjectRoot,
			"Owner: " + support.DisplayOwner(sandbox.Owner),
			"Created: " + sandbox.CreatedAt.Format(time.RFC3339),
			fmt.Sprintf("Files: %d", sandbox.FileCount),
			fmt.Sprintf("Size: %s", support.FormatBytes(sandbox.SizeBytes)),
		},
		RetrievalHints: []string{
			support.CLIName + " change diff " + sandbox.ID,
			support.CLIName + " process list " + sandbox.ID,
		},
	}
	if reserved := reservedSummary(sandbox); reserved != "" {
		report.Results = append(report.Results, "Reserved: "+reserved)
	}
	if sandbox.MergedDir != "" {
		report.Results = append(report.Results, "Workspace: "+sandbox.MergedDir)
	}
	if sandbox.ErrorMsg != "" {
		report.Results = append(report.Results, "Error: "+sandbox.ErrorMsg)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStop(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("sandbox stop", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s sandbox stop <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Request("POST", "/sandboxes/"+sandboxID+"/stop", nil, nil)
	if err != nil {
		return err
	}

	var sandbox support.SandboxResponse
	if err := json.Unmarshal(body, &sandbox); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Sandbox stopped", "Sandbox ID: " + sandbox.ID},
		Changes:     []string{"Status: " + sandbox.Status},
		NextCommand: []string{support.CLIName + " sandbox inspect " + sandbox.ID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("sandbox delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s sandbox delete <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}
	if _, err := deps.ScenarioApp().Request("DELETE", "/sandboxes/"+sandboxID, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Sandbox deleted", "Sandbox ID: " + sandboxID},
		Changes:     []string{"Removed sandbox data and process state"},
		NextCommand: []string{support.CLIName + " sandbox list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runWorkspace(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("sandbox workspace", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: %s sandbox workspace <sandbox-id> [--json]", support.CLIName)
	}

	sandboxID, err := support.ResolveSandboxID(deps.ScenarioApp(), fs.Arg(0))
	if err != nil {
		return err
	}

	body, err := deps.ScenarioApp().Get("/sandboxes/"+sandboxID+"/workspace", nil)
	if err != nil {
		return err
	}

	var resp struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Sandbox ID: " + sandboxID},
		ResultsHeading: "Workspace",
		Results:        []string{resp.Path},
		RetrievalHints: []string{support.CLIName + " process shell " + sandboxID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func reservedSummary(sandbox support.SandboxResponse) string {
	if len(sandbox.ReservedPaths) > 0 {
		head := sandbox.ReservedPaths[0]
		if len(sandbox.ReservedPaths) > 1 {
			return fmt.Sprintf("%s (+%d)", head, len(sandbox.ReservedPaths)-1)
		}
		if head != "" && head != sandbox.ScopePath {
			return head
		}
	}
	if sandbox.ReservedPath != "" && sandbox.ReservedPath != sandbox.ScopePath {
		return sandbox.ReservedPath
	}
	return ""
}

func renderSandboxRows(sandboxes []support.SandboxResponse) []string {
	if len(sandboxes) == 0 {
		return nil
	}
	rows := make([]string, 0, len(sandboxes))
	for _, sandbox := range sandboxes {
		rows = append(rows, fmt.Sprintf(
			"%s | %s | reserved=%s | owner=%s | created=%s | files=%d",
			support.TruncateID(sandbox.ID),
			sandbox.Status,
			support.TailTruncate(firstNonEmpty(reservedSummary(sandbox), sandbox.ScopePath), 40),
			support.DisplayOwner(sandbox.Owner),
			sandbox.CreatedAt.Format("2006-01-02 15:04"),
			sandbox.FileCount,
		))
	}
	return rows
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
