package apidefs

import (
	"fmt"
	"os"
	"strings"

	"network-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `api` subcommand group for managing API definitions and
// running OpenAPI-driven test suites. Endpoints live under /api/v1/api/...
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "api",
		Description: "Manage API definitions and run API test suites",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List registered API definitions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one API definition", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Aliases: []string{"add"}, Description: "Create an API definition", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update an API definition", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete an API definition", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "discover", Description: "Discover endpoints from an OpenAPI spec", Run: func(args []string) error { return runDiscover(core, args) }},
			{Name: "test", Description: "Run an API test suite", Run: func(args []string) error { return runTest(core, args) }},
		},
	}
}

// ---------- list ----------

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/api/definitions", nil)
	if err != nil {
		return err
	}
	var defs []support.APIDefinition
	if err := support.Decode(body, &defs); err != nil {
		return err
	}

	results := make([]string, 0, len(defs))
	if len(defs) == 0 {
		results = append(results, "(no API definitions)")
	}
	for _, d := range defs {
		line := fmt.Sprintf("%s (%s) | %s", d.Name, support.ShortID(d.ID), d.BaseURL)
		if d.Version != "" {
			line += " v" + d.Version
		}
		if d.EndpointsCount > 0 {
			line += fmt.Sprintf(" | %d endpoints", d.EndpointsCount)
		}
		if d.ValidationStatus != "" {
			line += " | " + d.ValidationStatus
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("API definitions: %d", len(defs))},
		ResultsHeading: "Definitions",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s api get <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- get ----------

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: api get <id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/api/definitions/"+id, nil)
	if err != nil {
		return err
	}
	var def support.APIDefinition
	if err := support.Decode(body, &def); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", def.ID),
		fmt.Sprintf("Name: %s", def.Name),
		fmt.Sprintf("Base URL: %s", def.BaseURL),
	}
	if def.Version != "" {
		results = append(results, "Version: "+def.Version)
	}
	if def.Specification != "" {
		results = append(results, "Specification: "+def.Specification)
	}
	if def.EndpointsCount > 0 {
		results = append(results, fmt.Sprintf("Endpoints: %d", def.EndpointsCount))
	}
	if def.ValidationStatus != "" {
		results = append(results, "Validation: "+def.ValidationStatus)
	}
	if def.LastValidated != nil {
		results = append(results, "Last validated: "+support.FormatTimeValue(*def.LastValidated))
	}
	if def.DocumentationURL != "" {
		results = append(results, "Docs: "+def.DocumentationURL)
	}
	if len(def.AuthenticationMethods) > 0 {
		results = append(results, "Auth: "+strings.Join(def.AuthenticationMethods, ", "))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("API: %s", def.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s api test %s --body-file suite.json", support.CLIName, def.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- create ----------

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api create")
	name := fs.String("name", "", "Definition name (required unless --body-file)")
	baseURL := fs.String("base-url", "", "API base URL (required unless --body-file)")
	version := fs.String("version", "", "Optional API version")
	specFile := fs.String("spec", "", "Path to an OpenAPI JSON spec to attach")
	docURL := fs.String("docs", "", "Optional documentation URL")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*name) == "" || strings.TrimSpace(*baseURL) == "" {
			return fmt.Errorf("usage: api create --name NAME --base-url URL [--version V] [--spec PATH] [--docs URL] [--body-file PATH]")
		}
		req := map[string]interface{}{
			"name":     strings.TrimSpace(*name),
			"base_url": strings.TrimSpace(*baseURL),
		}
		if strings.TrimSpace(*version) != "" {
			req["version"] = strings.TrimSpace(*version)
		}
		if strings.TrimSpace(*docURL) != "" {
			req["documentation_url"] = strings.TrimSpace(*docURL)
		}
		if strings.TrimSpace(*specFile) != "" {
			spec, err := support.ReadJSONFile(*specFile, true)
			if err != nil {
				return err
			}
			req["spec_document"] = spec
			req["specification"] = "openapi"
		}
		payload = req
	}

	body, err := core.Request("POST", "/api/definitions", nil, payload)
	if err != nil {
		return err
	}
	var created map[string]interface{}
	if err := support.Decode(body, &created); err != nil {
		return err
	}
	id := ""
	if v, ok := created["id"].(string); ok {
		id = v
	}

	report := cliapp.MutationReport{
		Result:  []string{"API definition created"},
		Changes: []string{fmt.Sprintf("New definition ID: %s", id)},
		NextCommand: []string{
			fmt.Sprintf("%s api get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// ---------- update ----------

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api update")
	name := fs.String("name", "", "New name")
	baseURL := fs.String("base-url", "", "New base URL")
	version := fs.String("version", "", "New version")
	specFile := fs.String("spec", "", "Path to a new OpenAPI JSON spec")
	docURL := fs.String("docs", "", "New documentation URL")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full update payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: api update <id> [--name N] [--base-url URL] [--version V] [--spec PATH] [--docs URL] [--body-file PATH]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		req := map[string]interface{}{}
		if strings.TrimSpace(*name) != "" {
			req["name"] = strings.TrimSpace(*name)
		}
		if strings.TrimSpace(*baseURL) != "" {
			req["base_url"] = strings.TrimSpace(*baseURL)
		}
		if strings.TrimSpace(*version) != "" {
			req["version"] = strings.TrimSpace(*version)
		}
		if strings.TrimSpace(*docURL) != "" {
			req["documentation_url"] = strings.TrimSpace(*docURL)
		}
		if strings.TrimSpace(*specFile) != "" {
			spec, err := support.ReadJSONFile(*specFile, true)
			if err != nil {
				return err
			}
			req["spec_document"] = spec
		}
		if len(req) == 0 {
			return fmt.Errorf("no updates specified; provide at least one flag or --body-file")
		}
		payload = req
	}

	body, err := core.Request("PUT", "/api/definitions/"+id, nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "API definition updated"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Definition %s updated", id)},
		NextCommand: []string{
			fmt.Sprintf("%s api get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// ---------- delete ----------

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: api delete <id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/api/definitions/"+id, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "API definition deleted"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Definition %s removed", id)},
		NextCommand: []string{
			fmt.Sprintf("%s api list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// ---------- discover ----------

func runDiscover(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api discover")
	specURL := fs.String("spec-url", "", "URL of an OpenAPI JSON spec")
	specFile := fs.String("spec-file", "", "Path to a local OpenAPI JSON spec")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request payload (overrides other flags)")
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
	case strings.TrimSpace(*specFile) != "":
		spec, err := support.ReadJSONFile(*specFile, true)
		if err != nil {
			return err
		}
		payload = map[string]interface{}{"spec_doc": spec}
	case strings.TrimSpace(*specURL) != "":
		payload = map[string]interface{}{"spec_url": strings.TrimSpace(*specURL)}
	default:
		return fmt.Errorf("usage: api discover [--spec-url URL | --spec-file PATH | --body-file PATH]")
	}

	body, err := core.Request("POST", "/api/discover", nil, payload)
	if err != nil {
		return err
	}
	var data struct {
		Info      map[string]interface{}   `json:"info"`
		Endpoints []map[string]interface{} `json:"endpoints"`
	}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Endpoints discovered: %d", len(data.Endpoints))}
	if title, ok := data.Info["title"].(string); ok && title != "" {
		summary = append(summary, "Title: "+title)
	}
	if version, ok := data.Info["version"].(string); ok && version != "" {
		summary = append(summary, "Version: "+version)
	}
	if baseURL, ok := data.Info["base_url"].(string); ok && baseURL != "" {
		summary = append(summary, "Base URL: "+baseURL)
	}

	results := make([]string, 0, len(data.Endpoints))
	if len(data.Endpoints) == 0 {
		results = append(results, "(no endpoints found)")
	}
	for _, ep := range data.Endpoints {
		method, _ := ep["method"].(string)
		path, _ := ep["path"].(string)
		summaryText, _ := ep["summary"].(string)
		line := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
		if summaryText != "" {
			line += " - " + summaryText
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Endpoints",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s api create --name NAME --base-url URL --spec spec.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- test ----------

func runTest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("api test")
	apiID := fs.String("api-id", "", "Run against a stored API definition ID")
	baseURL := fs.String("base-url", "", "Run against this base URL")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*apiID) == "" && strings.TrimSpace(*baseURL) == "" {
			return fmt.Errorf("usage: api test (--api-id ID | --base-url URL) --body-file PATH")
		}
		// Without a body-file, we can still issue a minimal payload the API
		// will reject for missing test_suite. Prefer the explicit error.
		return fmt.Errorf("api test requires --body-file PATH with {base_url|api_definition_id, test_suite: [...]}")
	}

	body, err := core.Request("POST", "/network/api/test", nil, payload)
	if err != nil {
		return err
	}
	var data struct {
		OverallSuccessRate float64                  `json:"overall_success_rate"`
		TestResults        []map[string]interface{} `json:"test_results"`
	}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	results := make([]string, 0, len(data.TestResults))
	if len(data.TestResults) == 0 {
		results = append(results, "(no endpoint results)")
	}
	for _, r := range data.TestResults {
		endpoint, _ := r["endpoint"].(string)
		passed := floatField(r, "passed_tests")
		total := floatField(r, "total_tests")
		failed := floatField(r, "failed_tests")
		execMs := floatField(r, "execution_time_ms")
		line := fmt.Sprintf("%s | passed=%.0f/%.0f | failed=%.0f | %.0fms", endpoint, passed, total, failed, execMs)
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Overall success rate: %.1f%%", data.OverallSuccessRate)},
		ResultsHeading: "Endpoint results",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s api test --body-file suite.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func floatField(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
