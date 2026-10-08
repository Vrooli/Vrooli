package domain

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "lifestyle-dashboard"

type RegisterDomainRequest struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	HealthURL    string   `json:"health_url,omitempty"`
}

type DomainResponse struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Status       string   `json:"status"`
	HealthURL    string   `json:"health_url,omitempty"`
	LastHealthAt *string  `json:"last_health_at,omitempty"`
	RegisteredAt string   `json:"registered_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type DomainsListResponse struct {
	Domains []DomainResponse `json:"domains"`
	Count   int              `json:"count"`
}

type HealthCheckResponse struct {
	Domain    string `json:"domain"`
	Status    string `json:"status"`
	LastCheck string `json:"last_check"`
	Message   string `json:"message,omitempty"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "domain",
		Description: "Manage registered lifestyle domains",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "register", NeedsAPI: true, Description: "Register a new domain", Run: func(args []string) error { return runRegister(core, args) }},
			{Name: "list", NeedsAPI: true, Description: "List all registered domains", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get a domain by name", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update domain attributes", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "health", NeedsAPI: true, Description: "Check domain health status", Run: func(args []string) error { return runHealth(core, args) }},
		},
	}
}

func runRegister(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("domain register", flag.ContinueOnError)
	name := fs.String("name", "", "Domain name (required)")
	displayName := fs.String("display-name", "", "Display name (required)")
	description := fs.String("description", "", "Description (optional)")
	capabilities := fs.String("capabilities", "", "Comma-separated capabilities (optional)")
	healthURL := fs.String("health-url", "", "Health check URL (optional)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	if *displayName == "" {
		return fmt.Errorf("--display-name is required")
	}

	req := RegisterDomainRequest{
		Name:        *name,
		DisplayName: *displayName,
		Description: *description,
		HealthURL:   *healthURL,
	}
	if *capabilities != "" {
		req.Capabilities = cliutil.ParseCSV(*capabilities)
	}

	body, err := core.Request("POST", "/domains", nil, req)
	if err != nil {
		return err
	}
	var resp DomainResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Domain registered", "Domain: " + resp.Name},
		Changes: []string{
			"Display name: " + resp.DisplayName,
			"Status: " + resp.Status,
		},
		NextCommand: []string{cliName + " domain get " + resp.Name, cliName + " domain health " + resp.Name},
	}
	if len(resp.Capabilities) > 0 {
		report.Changes = append(report.Changes, "Capabilities: "+strings.Join(resp.Capabilities, ", "))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("domain list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/domains", nil)
	if err != nil {
		return err
	}
	var resp DomainsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Domains: %d", resp.Count)},
		Results:        renderRows(resp.Domains),
		RetrievalHints: []string{cliName + " domain get <name>", cliName + " domain register --name <name> --display-name \"...\""},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("domain get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: domain get <name> [--json]")
	}

	name := fs.Arg(0)
	body, err := core.Get("/domains/"+name, nil)
	if err != nil {
		return err
	}
	var resp DomainResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Domain: " + resp.Name, "Status: " + resp.Status},
		ResultsHeading: "Details",
		Results:        detailLines(resp),
		RetrievalHints: []string{cliName + " domain update " + resp.Name + " --display-name \"...\"", cliName + " domain health " + resp.Name},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("domain update", flag.ContinueOnError)
	displayName := fs.String("display-name", "", "New display name")
	description := fs.String("description", "", "New description")
	healthURL := fs.String("health-url", "", "New health check URL")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: domain update <name> [--display-name NAME] [--description DESC] [--health-url URL] [--json]")
	}

	name := fs.Arg(0)
	updates := map[string]interface{}{}
	if *displayName != "" {
		updates["display_name"] = *displayName
	}
	if *description != "" {
		updates["description"] = *description
	}
	if *healthURL != "" {
		updates["health_url"] = *healthURL
	}
	if len(updates) == 0 {
		return fmt.Errorf("no updates specified; use --display-name, --description, or --health-url")
	}

	body, err := core.Request("PATCH", "/domains/"+name, nil, updates)
	if err != nil {
		return err
	}
	var resp DomainResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Domain updated", "Domain: " + resp.Name},
		Changes:     updateChanges(resp, updates),
		NextCommand: []string{cliName + " domain get " + resp.Name, cliName + " domain health " + resp.Name},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runHealth(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("domain health", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: domain health <name> [--json]")
	}

	name := fs.Arg(0)
	body, err := core.Get("/domains/"+name+"/health", nil)
	if err != nil {
		return err
	}
	var resp HealthCheckResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.OperationalReport{
		Status: []string{
			"Domain: " + resp.Domain,
			"Status: " + resp.Status,
			"Last check: " + resp.LastCheck,
		},
		NextSteps: []string{cliName + " domain get " + resp.Domain},
	}
	if resp.Message != "" {
		report.Triage = []cliapp.TriageGroup{{Heading: "Health", Items: []string{resp.Message}}}
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func renderRows(domains []DomainResponse) []string {
	if len(domains) == 0 {
		return nil
	}
	rows := make([]string, 0, len(domains))
	for _, domain := range domains {
		rows = append(rows, fmt.Sprintf("%s | %s | %s", domain.Name, domain.DisplayName, domain.Status))
	}
	return rows
}

func detailLines(resp DomainResponse) []string {
	lines := []string{
		"Display name: " + resp.DisplayName,
		"Status: " + resp.Status,
		"Registered: " + resp.RegisteredAt,
		"Updated: " + resp.UpdatedAt,
	}
	if resp.Description != "" {
		lines = append(lines, "Description: "+resp.Description)
	}
	if len(resp.Capabilities) > 0 {
		lines = append(lines, "Capabilities: "+strings.Join(resp.Capabilities, ", "))
	}
	if resp.HealthURL != "" {
		lines = append(lines, "Health URL: "+resp.HealthURL)
	}
	if resp.LastHealthAt != nil {
		lines = append(lines, "Last health check: "+*resp.LastHealthAt)
	}
	return lines
}

func updateChanges(resp DomainResponse, updates map[string]interface{}) []string {
	lines := []string{}
	if _, ok := updates["display_name"]; ok {
		lines = append(lines, "Display name: "+resp.DisplayName)
	}
	if _, ok := updates["description"]; ok {
		lines = append(lines, "Description: "+resp.Description)
	}
	if _, ok := updates["health_url"]; ok {
		lines = append(lines, "Health URL: "+resp.HealthURL)
	}
	if len(lines) == 0 {
		lines = append(lines, "Status: "+resp.Status)
	}
	return lines
}
