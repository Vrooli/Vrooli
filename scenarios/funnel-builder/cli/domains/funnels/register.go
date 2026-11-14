package funnels

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"funnel-builder/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "funnel-builder"

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "funnels",
		Description: "Manage funnels, analytics, and lead exports",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List funnels", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a funnel", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one funnel", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Description: "Update a funnel", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Description: "Delete a funnel", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "analytics", Description: "Show funnel analytics", Run: func(args []string) error { return runAnalytics(core, args) }},
			{Name: "leads", Description: "List captured leads", Run: func(args []string) error { return runLeads(core, args) }},
			{Name: "export-leads", Description: "Export leads as JSON or CSV", Run: func(args []string) error { return runExportLeads(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels list")
	tenantID := fs.String("tenant", "", "Tenant ID filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/funnels", support.BuildQuery(map[string]string{
		"tenant_id": *tenantID,
	}))
	if err != nil {
		return err
	}

	var funnels []support.Funnel
	if err := support.Decode(body, &funnels); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Funnels: %d", len(funnels)),
		},
		ResultsHeading: "Funnels",
		Results:        funnelRows(funnels),
		RetrievalHints: []string{
			fmt.Sprintf("%s funnels get <funnel-id>", cliName),
			fmt.Sprintf("%s funnels analytics <funnel-id>", cliName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels create")
	name := fs.String("name", "", "Funnel name")
	projectID := fs.String("project", "", "Project ID")
	templateSlug := fs.String("template", "", "Template slug")
	slug := fs.String("slug", "", "Custom slug")
	description := fs.String("description", "", "Funnel description")
	tenantID := fs.String("tenant", "", "Tenant ID")
	stepsFile := fs.String("steps-file", "", "JSON file containing funnel steps")
	settingsFile := fs.String("settings-file", "", "JSON file containing funnel settings")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*projectID) == "" {
		return fmt.Errorf("usage: funnels create --name <name> --project <project-id> [--template <slug>] [--slug <slug>] [--description <text>] [--steps-file <path>] [--settings-file <path>]")
	}

	payload := map[string]interface{}{
		"name":        strings.TrimSpace(*name),
		"description": strings.TrimSpace(*description),
		"project_id":  strings.TrimSpace(*projectID),
	}
	if value := strings.TrimSpace(*tenantID); value != "" {
		payload["tenant_id"] = value
	}
	if value := strings.TrimSpace(*slug); value != "" {
		payload["slug"] = value
	}

	if value := strings.TrimSpace(*templateSlug); value != "" {
		template, err := fetchTemplate(core, value)
		if err != nil {
			return err
		}
		if err := mergeTemplateData(payload, template.TemplateData); err != nil {
			return err
		}
		payload["name"] = strings.TrimSpace(*name)
		payload["description"] = strings.TrimSpace(*description)
		payload["project_id"] = strings.TrimSpace(*projectID)
		if value := strings.TrimSpace(*tenantID); value != "" {
			payload["tenant_id"] = value
		}
		if value := strings.TrimSpace(*slug); value != "" {
			payload["slug"] = value
		}
	}

	if raw, err := support.ReadJSONFile(*stepsFile, false); err != nil {
		return err
	} else if raw != nil {
		payload["steps"] = json.RawMessage(raw)
	}
	if raw, err := support.ReadJSONFile(*settingsFile, false); err != nil {
		return err
	} else if raw != nil {
		payload["settings"] = json.RawMessage(raw)
	}

	body, err := core.Request("POST", "/funnels", nil, payload)
	if err != nil {
		return err
	}
	var created support.CreateFunnelResponse
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	previewURL := created.PreviewURL
	if resolved := support.APIRootPreviewURL(core.APIRootBase(), created.ID); resolved != "" {
		previewURL = resolved
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Funnel %s created", created.ID),
			fmt.Sprintf("Slug: %s", created.Slug),
		},
		Changes: []string{
			fmt.Sprintf("Project: %s", strings.TrimSpace(*projectID)),
			fmt.Sprintf("Preview URL: %s", previewURL),
		},
		NextCommand: []string{
			fmt.Sprintf("%s funnels get %s", cliName, created.ID),
			fmt.Sprintf("%s funnels analytics %s", cliName, created.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: funnels get <funnel-id>")
	}
	funnelID := fs.Arg(0)

	funnel, err := fetchFunnel(core, funnelID)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Funnel: %s", funnel.Name),
			fmt.Sprintf("ID: %s", funnel.ID),
			fmt.Sprintf("Status: %s", funnel.Status),
		},
		ResultsHeading: "Details",
		Results: []string{
			fmt.Sprintf("Slug: %s", funnel.Slug),
			fmt.Sprintf("Project ID: %s", blankFallback(support.PtrValue(funnel.ProjectID), "(none)")),
			fmt.Sprintf("Tenant ID: %s", blankFallback(support.PtrValue(funnel.TenantID), "(none)")),
			fmt.Sprintf("Description: %s", blankFallback(funnel.Description, "(none)")),
			fmt.Sprintf("Steps: %d", len(funnel.Steps)),
			fmt.Sprintf("Updated: %s", support.FormatTimeValue(funnel.UpdatedAt)),
			stepSummary(funnel.Steps),
		},
		RetrievalHints: []string{
			fmt.Sprintf("%s funnels analytics %s", cliName, funnel.ID),
			fmt.Sprintf("%s funnels leads %s", cliName, funnel.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels update")
	name := fs.String("name", "", "Funnel name")
	description := fs.String("description", "", "Funnel description")
	status := fs.String("status", "", "Funnel status")
	projectID := fs.String("project", "", "Project ID")
	clearProject := fs.Bool("clear-project", false, "Detach funnel from its project")
	stepsFile := fs.String("steps-file", "", "JSON file containing funnel steps")
	settingsFile := fs.String("settings-file", "", "JSON file containing funnel settings")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: funnels update <funnel-id> [--name <name>] [--description <text>] [--status <status>] [--project <project-id>|--clear-project] [--steps-file <path>] [--settings-file <path>]")
	}
	funnelID := fs.Arg(0)

	current, err := fetchFunnel(core, funnelID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"name":        current.Name,
		"description": current.Description,
		"settings":    current.Settings,
		"status":      current.Status,
	}
	if current.ProjectID != nil {
		payload["project_id"] = *current.ProjectID
	}
	if value := strings.TrimSpace(*name); value != "" {
		payload["name"] = value
	}
	if value := strings.TrimSpace(*description); value != "" {
		payload["description"] = value
	}
	if value := strings.TrimSpace(*status); value != "" {
		payload["status"] = value
	}
	if *clearProject {
		payload["project_id"] = ""
	} else if value := strings.TrimSpace(*projectID); value != "" {
		payload["project_id"] = value
	}
	if raw, err := support.ReadJSONFile(*stepsFile, false); err != nil {
		return err
	} else if raw != nil {
		payload["steps"] = json.RawMessage(raw)
	}
	if raw, err := support.ReadJSONFile(*settingsFile, false); err != nil {
		return err
	} else if raw != nil {
		payload["settings"] = json.RawMessage(raw)
	}

	body, err := core.Request("PUT", "/funnels/"+funnelID, nil, payload)
	if err != nil {
		return err
	}
	var response support.MessageResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	updated, err := fetchFunnel(core, funnelID)
	if err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Funnel %s updated", updated.ID),
			fmt.Sprintf("API message: %s", response.Message),
		},
		Changes: []string{
			fmt.Sprintf("Name: %s", updated.Name),
			fmt.Sprintf("Status: %s", updated.Status),
			fmt.Sprintf("Project ID: %s", blankFallback(support.PtrValue(updated.ProjectID), "(none)")),
			fmt.Sprintf("Steps: %d", len(updated.Steps)),
		},
		NextCommand: []string{
			fmt.Sprintf("%s funnels get %s", cliName, updated.ID),
			fmt.Sprintf("%s funnels analytics %s", cliName, updated.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels delete")
	force := fs.Bool("force", false, "Delete without confirmation")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: funnels delete <funnel-id> [--force]")
	}
	funnelID := fs.Arg(0)

	if !*force {
		ok, err := support.Confirm("Delete funnel " + funnelID + "?")
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("delete cancelled")
		}
	}

	body, err := core.Request("DELETE", "/funnels/"+funnelID, nil, nil)
	if err != nil {
		return err
	}
	var response support.MessageResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Funnel %s deleted", funnelID),
		},
		Changes: []string{
			fmt.Sprintf("API message: %s", response.Message),
		},
		NextCommand: []string{
			fmt.Sprintf("%s funnels list", cliName),
			fmt.Sprintf("%s templates list", cliName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runAnalytics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels analytics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: funnels analytics <funnel-id>")
	}
	funnelID := fs.Arg(0)

	body, err := core.Get("/funnels/"+funnelID+"/analytics", nil)
	if err != nil {
		return err
	}
	var analytics support.Analytics
	if err := support.Decode(body, &analytics); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Funnel ID: %s", analytics.FunnelID),
			fmt.Sprintf("Views: %d", analytics.TotalViews),
			fmt.Sprintf("Leads: %d captured / %d total", analytics.CapturedLeads, analytics.TotalLeads),
			fmt.Sprintf("Conversion rate: %.1f%%", analytics.ConversionRate),
		},
		ResultsHeading: "Analytics",
		Results:        analyticsRows(analytics),
		RetrievalHints: []string{
			fmt.Sprintf("%s funnels leads %s", cliName, funnelID),
			fmt.Sprintf("%s funnels export-leads %s --format csv --output leads.csv", cliName, funnelID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runLeads(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels leads")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: funnels leads <funnel-id>")
	}
	funnelID := fs.Arg(0)

	body, err := core.Get("/funnels/"+funnelID+"/leads", nil)
	if err != nil {
		return err
	}
	var leads []support.Lead
	if err := support.Decode(body, &leads); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Leads: %d", len(leads)),
			fmt.Sprintf("Completed leads: %d", completedLeadCount(leads)),
		},
		ResultsHeading: "Lead Details",
		Results:        leadRows(leads),
		RetrievalHints: []string{
			fmt.Sprintf("%s funnels export-leads %s --format csv --output leads.csv", cliName, funnelID),
			fmt.Sprintf("%s funnels analytics %s", cliName, funnelID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runExportLeads(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("funnels export-leads")
	format := fs.String("format", "json", "Export format: json or csv")
	output := fs.String("output", "", "Write export to a file instead of stdout")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: funnels export-leads <funnel-id> [--format json|csv] [--output <path>]")
	}
	funnelID := fs.Arg(0)

	body, err := core.Get("/funnels/"+funnelID+"/leads", support.BuildQuery(map[string]string{
		"format": *format,
	}))
	if err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		_, err := os.Stdout.Write(body)
		if err == nil && !strings.HasSuffix(string(body), "\n") {
			_, err = fmt.Fprintln(os.Stdout)
		}
		return err
	}
	if err := support.WriteOutput(*output, body); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Exported funnel leads for %s", funnelID),
		},
		Changes: []string{
			fmt.Sprintf("Format: %s", strings.ToLower(strings.TrimSpace(*format))),
			fmt.Sprintf("Output file: %s", strings.TrimSpace(*output)),
		},
		NextCommand: []string{
			fmt.Sprintf("%s funnels analytics %s", cliName, funnelID),
			fmt.Sprintf("%s funnels leads %s", cliName, funnelID),
		},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func fetchFunnel(core *cliapp.ScenarioApp, funnelID string) (support.Funnel, error) {
	body, err := core.Get("/funnels/"+funnelID, nil)
	if err != nil {
		return support.Funnel{}, err
	}
	var funnel support.Funnel
	if err := support.Decode(body, &funnel); err != nil {
		return support.Funnel{}, err
	}
	return funnel, nil
}

func fetchTemplate(core *cliapp.ScenarioApp, slug string) (support.Template, error) {
	body, err := core.Get("/templates/"+slug, nil)
	if err != nil {
		return support.Template{}, err
	}
	var tpl support.Template
	if err := support.Decode(body, &tpl); err != nil {
		return support.Template{}, err
	}
	return tpl, nil
}

func mergeTemplateData(payload map[string]interface{}, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse templateData: %w", err)
	}
	for key, value := range data {
		payload[key] = value
	}
	return nil
}

func funnelRows(funnels []support.Funnel) []string {
	if len(funnels) == 0 {
		return []string{"No funnels found"}
	}
	rows := make([]string, 0, len(funnels))
	for _, funnel := range funnels {
		rows = append(rows, fmt.Sprintf("%s | %s | status=%s | steps=%d | project=%s", funnel.ID, funnel.Name, funnel.Status, len(funnel.Steps), blankFallback(support.PtrValue(funnel.ProjectID), "(none)")))
	}
	return rows
}

func stepSummary(steps []support.FunnelStep) string {
	if len(steps) == 0 {
		return "Steps: none"
	}
	stepLines := make([]string, 0, len(steps))
	for _, step := range steps {
		stepLines = append(stepLines, fmt.Sprintf("%d:%s (%s)", step.Position, step.Title, step.Type))
	}
	sort.Strings(stepLines)
	return "Steps: " + strings.Join(stepLines, ", ")
}

func analyticsRows(analytics support.Analytics) []string {
	rows := []string{
		fmt.Sprintf("Capture rate: %.1f%%", analytics.CaptureRate),
		fmt.Sprintf("Completed leads: %d", analytics.CompletedLeads),
		fmt.Sprintf("Average completion time: %.1fs", analytics.AverageTime),
	}
	if len(analytics.DropOffPoints) == 0 {
		rows = append(rows, "Drop-off points: none")
	} else {
		for _, point := range analytics.DropOffPoints {
			rows = append(rows, fmt.Sprintf("Drop-off | step %d %s | %.1f%% | visitors=%d | responses=%d", point.Position, point.StepTitle, point.DropOffRate, point.Visitors, point.Responses))
		}
	}
	if len(analytics.TrafficSources) > 0 {
		for _, source := range analytics.TrafficSources {
			rows = append(rows, fmt.Sprintf("Traffic | %s | count=%d | %.1f%%", source.Source, source.Count, source.Percentage))
		}
	}
	if len(analytics.DailyStats) > 0 {
		last := analytics.DailyStats[len(analytics.DailyStats)-1]
		rows = append(rows, fmt.Sprintf("Latest day | %s | views=%d | leads=%d | conversions=%d", last.Date, last.Views, last.Leads, last.Conversions))
	}
	return rows
}

func leadRows(leads []support.Lead) []string {
	if len(leads) == 0 {
		return []string{"No leads found"}
	}
	rows := make([]string, 0, len(leads))
	for _, lead := range leads {
		identity := firstNonBlank(lead.Email, lead.Name, lead.Phone, "(anonymous lead)")
		rows = append(rows, fmt.Sprintf("%s | %s | completed=%t | source=%s | created=%s", lead.ID, identity, lead.Completed, blankFallback(lead.Source, "Direct"), support.FormatTimeValue(lead.CreatedAt)))
	}
	return rows
}

func completedLeadCount(leads []support.Lead) int {
	count := 0
	for _, lead := range leads {
		if lead.Completed {
			count++
		}
	}
	return count
}

func blankFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
