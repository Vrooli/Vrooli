// Package system provides CLI commands for system operations (health, templates, records, download, wine).
package system

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const appName = "scenario-to-desktop"

// Commands provides system CLI commands.
type Commands struct {
	deps support.Dependencies
}

// New creates a new system Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{deps: deps}
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.deps.Get(path, query)
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.deps.Request("POST", path, nil, body)
}

func (c *Commands) apiDelete(path string) ([]byte, error) {
	return c.deps.Request("DELETE", path, nil, nil)
}

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	cmds := New(deps)
	// Root /health is served by cli-core's built-in `status` command, so this
	// group only wraps scenario-specific operations.
	return []cliapp.CommandGroup{
		{
			Title: "Templates",
			Commands: []cliapp.Command{
				{Name: "templates", NeedsAPI: true, Description: "List available desktop templates", Run: cmds.TemplatesList},
				{Name: "template", NeedsAPI: true, Description: "Get template details: template <type>", Run: cmds.TemplateGet},
			},
		},
		{
			Title: "Desktop Records",
			Commands: []cliapp.Command{
				{Name: "records", NeedsAPI: true, Description: "List desktop generation records", Run: cmds.RecordsList},
				{Name: "records-move", NeedsAPI: true, Description: "Move desktop wrapper: records-move <id> [--target <path>]", Run: cmds.RecordsMove},
				{Name: "records-delete", NeedsAPI: true, Description: "Delete desktop app: records-delete <scenario>", Run: cmds.RecordsDelete},
			},
		},
		{
			Title: "Download",
			Commands: []cliapp.Command{
				{Name: "download", NeedsAPI: true, Description: "Download built package: download <scenario> <platform> [--output <path>]", Run: cmds.Download},
			},
		},
		{
			Title: "Scenarios",
			Commands: []cliapp.Command{
				{Name: "desktop-status", NeedsAPI: true, Description: "List desktop build status and artifacts", Run: cmds.DesktopStatus},
			},
		},
	}
}

func WineRegister(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "wine",
		Description: "Wine for Windows builds on Linux (run 'wine help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "check", Description: "Check Wine installation status", Run: cmds.WineCheck},
			{Name: "install", Description: "Install Wine: install --method <flatpak|appimage>", Run: cmds.WineInstall},
			{Name: "status", Description: "Get Wine install status: status <id>", Run: cmds.WineStatus},
		},
	}
}

// TemplatesList lists available desktop templates.
func (c *Commands) TemplatesList(args []string) error {
	fs := flag.NewFlagSet("templates", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := c.apiGet("/templates", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Templates []struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
			Complexity  string `json:"complexity"`
		} `json:"templates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Available desktop templates: %d", len(resp.Templates))},
		ResultsHeading: "Templates",
		RetrievalHints: []string{"Use `scenario-to-desktop template <type>` for full template details."},
	}
	for _, t := range resp.Templates {
		report.Results = append(report.Results, fmt.Sprintf("%s - %s [%s]: %s", t.Type, t.Name, t.Complexity, t.Description))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// TemplateGet gets template details.
func (c *Commands) TemplateGet(args []string) error {
	fs := flag.NewFlagSet("template", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: template <type>\nTypes: universal, advanced, multi_window, kiosk")
	}

	templateType := fs.Args()[0]
	body, err := c.apiGet("/templates/"+templateType, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Template: %s\n", templateType)
	cliutil.PrintJSON(body)
	return nil
}

// RecordsList lists desktop generation records.
func (c *Commands) RecordsList(args []string) error {
	fs := flag.NewFlagSet("records", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := c.apiGet("/desktop/records", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Records []struct {
			Record struct {
				ID           string `json:"id"`
				ScenarioName string `json:"scenario_name"`
				Status       string `json:"status"`
			} `json:"record"`
			BuildState string `json:"build_state"`
		} `json:"records"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Records) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"Desktop records: 0"},
			ResultsHeading: "Records",
			RetrievalHints: []string{"Run `scenario-to-desktop pipeline list` to inspect active desktop pipelines."},
		})
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Desktop records: %d", len(resp.Records))},
		ResultsHeading: "Records",
		RetrievalHints: []string{"Use `scenario-to-desktop records-move <id>` to relocate an existing wrapper."},
	}
	for _, r := range resp.Records {
		report.Results = append(report.Results, fmt.Sprintf("%s | %s | %s", r.Record.ID, r.Record.ScenarioName, r.BuildState))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// RecordsMove moves desktop wrapper.
func (c *Commands) RecordsMove(args []string) error {
	fs := flag.NewFlagSet("records-move", flag.ContinueOnError)
	target := fs.String("target", "destination", "Move target: 'destination' or 'custom'")
	destPath := fs.String("path", "", "Custom destination path")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: records-move <id> [--target destination|custom] [--path <path>]")
	}

	recordID := fs.Args()[0]
	req := map[string]interface{}{
		"target": *target,
	}
	if *destPath != "" {
		req["destination_path"] = *destPath
	}

	body, err := c.apiPost("/desktop/records/"+recordID+"/move", req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Desktop record %s moved successfully.", recordID)},
		Changes:     []string{fmt.Sprintf("Target: %s", *target)},
		NextCommand: []string{"scenario-to-desktop records"},
	})
}

// RecordsDelete deletes desktop app.
func (c *Commands) RecordsDelete(args []string) error {
	fs := flag.NewFlagSet("records-delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: records-delete <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiDelete("/desktop/delete/" + scenario)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Desktop app deleted for %s.", scenario)},
		Changes:     []string{"Desktop wrapper and generated artifacts were removed."},
		NextCommand: []string{"scenario-to-desktop desktop-status"},
	})
}

// Download downloads built package.
func (c *Commands) Download(args []string) error {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	output := fs.String("output", "", "Output file path (default: current directory)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 2 {
		return fmt.Errorf("usage: download <scenario> <platform> [--output <path>]\nPlatforms: win, mac, linux")
	}

	scenario := fs.Args()[0]
	platform := fs.Args()[1]

	body, err := c.apiGet("/desktop/download/"+scenario+"/"+platform, nil)
	if err != nil {
		return err
	}

	// Determine output filename
	outputPath := *output
	if outputPath == "" {
		// Try to determine filename from content
		ext := ".bin"
		switch platform {
		case "win":
			ext = ".exe"
		case "mac":
			ext = ".zip"
		case "linux":
			ext = ".AppImage"
		}
		outputPath = scenario + "-" + platform + ext
	}

	if err := os.WriteFile(outputPath, body, 0o755); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Downloaded to %s (%d bytes)\n", outputPath, len(body))
	return nil
}

type desktopBuildArtifact struct {
	Platform     string `json:"platform"`
	FileName     string `json:"file_name"`
	SizeBytes    int64  `json:"size_bytes"`
	RelativePath string `json:"relative_path"`
	AbsolutePath string `json:"absolute_path"`
}

type desktopScenarioStatus struct {
	Name           string                 `json:"name"`
	DisplayName    string                 `json:"display_name"`
	Version        string                 `json:"version"`
	Built          bool                   `json:"built"`
	Platforms      []string               `json:"platforms"`
	BuildArtifacts []desktopBuildArtifact `json:"build_artifacts"`
}

type desktopStatusResponse struct {
	Scenarios []desktopScenarioStatus `json:"scenarios"`
	Stats     struct {
		Total       int `json:"total"`
		WithDesktop int `json:"with_desktop"`
		Built       int `json:"built"`
		WebOnly     int `json:"web_only"`
	} `json:"stats"`
}

// DOC: docs/QUICKSTART.md#check-build-artifacts-cli
// DesktopStatus lists desktop build status and artifacts for scenarios.
func (c *Commands) DesktopStatus(args []string) error {
	fs := flag.NewFlagSet("desktop-status", flag.ContinueOnError)
	nameFilter := fs.String("name", "", "Filter by scenario name")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: desktop-status [--name <scenario>] [--json]")
	}

	body, err := c.apiGet("/scenarios/desktop-status", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp desktopStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if filter := strings.TrimSpace(*nameFilter); filter != "" {
		resp.Scenarios = filterScenariosByName(resp.Scenarios, filter)
	}

	if len(resp.Scenarios) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"Desktop scenarios: 0"},
			ResultsHeading: "Scenarios",
			RetrievalHints: []string{"Run `scenario-to-desktop pipeline run <scenario>` to start a desktop build."},
		})
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios: %d total", resp.Stats.Total),
			fmt.Sprintf("With desktop: %d", resp.Stats.WithDesktop),
			fmt.Sprintf("Built: %d", resp.Stats.Built),
			fmt.Sprintf("Web-only: %d", resp.Stats.WebOnly),
		},
		ResultsHeading: "Scenarios",
		RetrievalHints: []string{"Use `scenario-to-desktop download <scenario> <platform>` once a scenario is built."},
	}
	for _, scenario := range resp.Scenarios {
		report.Results = append(report.Results, desktopScenarioLine(scenario))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func filterScenariosByName(scenarios []desktopScenarioStatus, name string) []desktopScenarioStatus {
	filtered := make([]desktopScenarioStatus, 0)
	for _, s := range scenarios {
		if s.Name == name {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func desktopScenarioLine(scenario desktopScenarioStatus) string {
	name := scenario.Name
	if strings.TrimSpace(scenario.DisplayName) != "" {
		name = fmt.Sprintf("%s (%s)", name, scenario.DisplayName)
	}
	version := "unknown"
	if strings.TrimSpace(scenario.Version) != "" {
		version = scenario.Version
	}
	status := "not built"
	if scenario.Built {
		status = "built"
	}
	parts := []string{fmt.Sprintf("%s v%s [%s]", name, version, status)}
	if len(scenario.Platforms) > 0 {
		parts = append(parts, fmt.Sprintf("platforms=%s", strings.Join(scenario.Platforms, ", ")))
	}
	if len(scenario.BuildArtifacts) > 0 {
		artifactLines := make([]string, 0, len(scenario.BuildArtifacts))
		for _, artifact := range scenario.BuildArtifacts {
			fileName := artifact.FileName
			if fileName == "" {
				fileName = artifact.RelativePath
			}
			artifactLines = append(artifactLines, fmt.Sprintf("%s=%s (%d bytes)", artifact.Platform, fileName, artifact.SizeBytes))
		}
		parts = append(parts, "artifacts="+strings.Join(artifactLines, "; "))
	}
	return strings.Join(parts, " | ")
}

// WineCheck checks Wine installation status.
func (c *Commands) WineCheck(args []string) error {
	fs := flag.NewFlagSet("wine-check", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := c.apiGet("/system/wine/check", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Installed     bool     `json:"installed"`
		Version       string   `json:"version"`
		Usable        bool     `json:"usable"`
		InstallMethod string   `json:"install_method"`
		Options       []string `json:"available_install_methods"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.OperationalReport{
		NextSteps: []string{"scenario-to-desktop wine install --method <flatpak|flatpak-auto|appimage>"},
	}
	if resp.Installed {
		usable := "usable"
		if !resp.Usable {
			usable = "not usable"
		}
		report.Status = append(report.Status, fmt.Sprintf("Wine installed (%s) via %s: %s", resp.Version, resp.InstallMethod, usable))
	} else {
		report.Status = append(report.Status, "Wine not installed")
		if len(resp.Options) > 0 {
			report.Triage = append(report.Triage, cliapp.TriageGroup{
				Heading: "Install Methods",
				Items:   []string{strings.Join(resp.Options, ", ")},
			})
		}
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

// WineInstall installs Wine.
func (c *Commands) WineInstall(args []string) error {
	fs := flag.NewFlagSet("wine-install", flag.ContinueOnError)
	method := fs.String("method", "", "Installation method: flatpak, flatpak-auto, appimage")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *method == "" {
		return fmt.Errorf("usage: wine-install --method <flatpak|flatpak-auto|appimage>")
	}

	req := map[string]string{"method": *method}
	body, err := c.apiPost("/system/wine/install", req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		InstallID string `json:"install_id"`
		Status    string `json:"status"`
		StatusURL string `json:"status_url"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Wine installation started: %s", resp.InstallID)},
		Changes:     []string{fmt.Sprintf("Method: %s", *method)},
		NextCommand: []string{fmt.Sprintf("%s wine status %s", appName, resp.InstallID)},
	})
}

// WineStatus gets Wine install status.
func (c *Commands) WineStatus(args []string) error {
	fs := flag.NewFlagSet("wine-status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: wine-status <install_id>")
	}

	installID := fs.Args()[0]
	body, err := c.apiGet("/system/wine/install/status/"+installID, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
