package extension

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"scenario-to-extension/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `extension` subcommand group that wraps
// `/api/v1/extension/*` endpoints. The CLI is a thin HTTP client over the
// API; local build tooling (npm, node, zip) is handled by the API or by the
// user directly.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "extension",
		Description: "Generate, inspect, test, and download browser extensions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Description: "Generate a browser extension for a scenario", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "status", Description: "Show status for a build", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "download", Description: "Download a completed build as a ZIP archive", Run: func(args []string) error { return runDownload(core, args) }},
			{Name: "test", Description: "Run extension tests against a build path", Run: func(args []string) error { return runTest(core, args) }},
			{Name: "templates", Aliases: []string{"list-templates"}, Description: "List available extension templates", Run: func(args []string) error { return runTemplates(core, args) }},
			{Name: "builds", Aliases: []string{"list"}, Description: "List recent extension builds", Run: func(args []string) error { return runBuilds(core, args) }},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("extension generate")
	template := fs.String("template", "full", "Extension template (full, content-script-only, background-only, popup-only)")
	permissions := fs.String("permissions", "storage,activeTab", "Comma-separated browser permissions")
	hostPermissions := fs.String("host-permissions", "<all_urls>", "Comma-separated host permission patterns")
	apiEndpoint := fs.String("api-endpoint", "http://localhost:3000", "Scenario API endpoint baked into the generated extension")
	appName := fs.String("app-name", "", "Extension display name (defaults to '<scenario> Extension')")
	description := fs.String("description", "", "Extension description")
	version := fs.String("version", "1.0.0", "Extension version")
	authorName := fs.String("author", "Vrooli Scenario Generator", "Author name for manifest/package metadata")
	license := fs.String("license", "MIT", "License identifier")
	bodyFile := fs.String("body-file", "", "Path to a JSON file providing the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 && strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("usage: extension generate <scenario-name> [flags] (or --body-file PATH)")
	}

	var payload interface{}
	scenarioName := ""
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("parse --body-file: %w", err)
		}
		if name, ok := parsed["scenario_name"].(string); ok {
			scenarioName = name
		}
		payload = parsed
	} else {
		scenarioName = fs.Arg(0)
		resolvedAppName := strings.TrimSpace(*appName)
		if resolvedAppName == "" {
			resolvedAppName = fmt.Sprintf("%s Extension", scenarioName)
		}
		resolvedDescription := strings.TrimSpace(*description)
		if resolvedDescription == "" {
			resolvedDescription = fmt.Sprintf("Browser extension for %s scenario", scenarioName)
		}
		payload = map[string]interface{}{
			"scenario_name": scenarioName,
			"template_type": *template,
			"config": map[string]interface{}{
				"app_name":         resolvedAppName,
				"app_description":  resolvedDescription,
				"api_endpoint":     *apiEndpoint,
				"permissions":      support.SplitCommaList(*permissions),
				"host_permissions": support.SplitCommaList(*hostPermissions),
				"version":          *version,
				"author_name":      *authorName,
				"license":          *license,
			},
		}
	}

	body, err := core.Request("POST", "/extension/generate", nil, payload)
	if err != nil {
		return err
	}
	var resp support.GenerateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Build ID: %s", resp.BuildID),
		fmt.Sprintf("Scenario: %s", scenarioName),
		fmt.Sprintf("Status: %s", resp.Status),
	}
	if resp.ExtensionPath != "" {
		results = append(results, fmt.Sprintf("Extension path: %s", resp.ExtensionPath))
	}
	if resp.TestCommand != "" {
		results = append(results, fmt.Sprintf("Test command: %s", resp.TestCommand))
	}
	if strings.TrimSpace(resp.InstallInstructions) != "" {
		results = append(results, "")
		results = append(results, "Installation instructions:")
		for _, line := range strings.Split(resp.InstallInstructions, "\n") {
			results = append(results, "  "+line)
		}
	}

	nextCommands := []string{
		fmt.Sprintf("%s extension status %s", support.CLIName, resp.BuildID),
	}
	if resp.ExtensionPath != "" {
		nextCommands = append(nextCommands, fmt.Sprintf("%s extension test %s", support.CLIName, resp.ExtensionPath))
	}
	nextCommands = append(nextCommands, fmt.Sprintf("%s extension download %s --output %s.zip", support.CLIName, resp.BuildID, resp.BuildID))

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Extension generation started for %s", scenarioName)},
		Changes:     results,
		NextCommand: nextCommands,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("extension status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: extension status <build-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/extension/status/"+id, nil)
	if err != nil {
		return err
	}
	var status support.BuildStatus
	if err := support.Decode(body, &status); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Build ID: %s", status.BuildID),
		fmt.Sprintf("Scenario: %s", status.ScenarioName),
		fmt.Sprintf("Status: %s", status.Status),
	}
	if status.ExtensionPath != "" {
		results = append(results, fmt.Sprintf("Extension path: %s", status.ExtensionPath))
	}
	if status.CreatedAt != nil {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(*status.CreatedAt)))
	}
	if status.CompletedAt != nil {
		results = append(results, fmt.Sprintf("Completed: %s", support.FormatTimeValue(*status.CompletedAt)))
	}
	if len(status.BuildLog) > 0 {
		results = append(results, "", "Build log:")
		for _, line := range status.BuildLog {
			results = append(results, "  "+line)
		}
	}
	if len(status.ErrorLog) > 0 {
		results = append(results, "", "Error log:")
		for _, line := range status.ErrorLog {
			results = append(results, "  "+line)
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Build %s: %s", support.ShortID(status.BuildID), status.Status)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s extension download %s --output %s.zip", support.CLIName, status.BuildID, status.BuildID),
			fmt.Sprintf("%s extension builds", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDownload(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("extension download")
	output := fs.String("output", "", "Output path for the ZIP archive (defaults to <build-id>.zip in CWD)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: extension download <build-id> [--output PATH]")
	}
	id := fs.Arg(0)

	body, err := core.Get("/extension/download/"+id, nil)
	if err != nil {
		return err
	}

	outPath := strings.TrimSpace(*output)
	if outPath == "" {
		outPath = id + ".zip"
	}
	if err := support.WriteOutput(outPath, body); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Downloaded build %s", id)},
		Changes: []string{fmt.Sprintf("Wrote %d bytes to %s", len(body), outPath)},
		NextCommand: []string{
			fmt.Sprintf("unzip -l %s", outPath),
			fmt.Sprintf("%s extension status %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runTest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("extension test")
	sites := fs.String("sites", "https://example.com", "Comma-separated list of test URLs")
	headless := fs.Bool("headless", false, "Run tests in headless mode")
	screenshot := fs.Bool("screenshot", true, "Capture screenshots during testing")
	bodyFile := fs.String("body-file", "", "Path to a JSON file providing the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 && strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("usage: extension test <extension-path> [flags] (or --body-file PATH)")
	}

	var payload interface{}
	extensionPath := ""
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("parse --body-file: %w", err)
		}
		if p, ok := parsed["extension_path"].(string); ok {
			extensionPath = p
		}
		payload = parsed
	} else {
		extensionPath = fs.Arg(0)
		testSites := support.SplitCommaList(*sites)
		if len(testSites) == 0 {
			testSites = []string{"https://example.com"}
		}
		payload = map[string]interface{}{
			"extension_path": extensionPath,
			"test_sites":     testSites,
			"headless":       *headless,
			"screenshot":     *screenshot,
		}
	}

	body, err := core.Request("POST", "/extension/test", nil, payload)
	if err != nil {
		return err
	}
	var result support.TestResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	overall := "FAIL"
	if result.Success {
		overall = "PASS"
	}
	summary := []string{
		fmt.Sprintf("Extension: %s", extensionPath),
		fmt.Sprintf("Overall: %s", overall),
		fmt.Sprintf("Total: %d, Passed: %d, Failed: %d, Success rate: %.1f%%",
			result.Summary.TotalTests, result.Summary.Passed, result.Summary.Failed, result.Summary.SuccessRate),
	}

	results := make([]string, 0, len(result.TestResults))
	for _, r := range result.TestResults {
		status := "FAIL"
		if r.Loaded {
			status = "PASS"
		}
		line := fmt.Sprintf("%s: %s (%dms)", r.Site, status, r.LoadTime)
		if len(r.Errors) > 0 {
			line += fmt.Sprintf(" | errors: %s", strings.Join(r.Errors, ", "))
		}
		if r.ScreenshotPath != "" {
			line += fmt.Sprintf(" | screenshot: %s", r.ScreenshotPath)
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = []string{"(no test sites returned)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Site results",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s extension test %s --sites https://example.com,https://google.com", support.CLIName, extensionPath),
		},
	}
	if *jsonOutput {
		if err := cliapp.PrintReportJSON(os.Stdout, report); err != nil {
			return err
		}
	} else {
		if err := cliapp.RenderListReport(os.Stdout, report); err != nil {
			return err
		}
	}
	if !result.Success {
		return fmt.Errorf("extension tests failed: %d of %d site(s) did not pass", result.Summary.Failed, result.Summary.TotalTests)
	}
	return nil
}

func runTemplates(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("extension templates")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/extension/templates", nil)
	if err != nil {
		return err
	}
	var resp support.TemplatesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Templates)*2)
	for _, tpl := range resp.Templates {
		header := tpl.Name
		if tpl.DisplayName != "" {
			header = fmt.Sprintf("%s (%s)", tpl.Name, tpl.DisplayName)
		}
		if tpl.Source != "" {
			header += fmt.Sprintf(" [source=%s]", tpl.Source)
		}
		results = append(results, header)
		if tpl.Description != "" {
			results = append(results, "  "+tpl.Description)
		}
		if len(tpl.Files) > 0 {
			results = append(results, "  files: "+strings.Join(tpl.Files, ", "))
		}
	}
	if len(results) == 0 {
		results = []string{"(no templates available)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Templates: %d", resp.Count)},
		ResultsHeading: "Available templates",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s extension generate <scenario> --template <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runBuilds(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("extension builds")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/extension/builds", nil)
	if err != nil {
		return err
	}
	var resp support.BuildsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Builds))
	for _, b := range resp.Builds {
		created := "unknown"
		if b.CreatedAt != nil {
			created = support.FormatTimeValue(*b.CreatedAt)
		}
		line := fmt.Sprintf("%s | %s | status=%s | created=%s", support.ShortID(b.BuildID), b.ScenarioName, b.Status, created)
		if b.TemplateType != "" {
			line += fmt.Sprintf(" | template=%s", b.TemplateType)
		}
		if b.ExtensionPath != "" {
			line += fmt.Sprintf(" | path=%s", b.ExtensionPath)
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = []string{"(no builds found)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Builds: %d", resp.Count)},
		ResultsHeading: "Recent builds",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s extension status <build-id>", support.CLIName),
			fmt.Sprintf("%s extension download <build-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
