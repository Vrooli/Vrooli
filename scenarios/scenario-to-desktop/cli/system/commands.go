// Package system provides CLI commands for system operations (health, templates, records, download, wine).
package system

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

const appName = "scenario-to-desktop"

// Commands provides system CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new system Commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) apiPath(path string) string {
	return cmdutil.APIPath(path)
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.api.Get(c.apiPath(path), cmdutil.MapToValues(query))
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.api.Request("POST", c.apiPath(path), nil, body)
}

func (c *Commands) apiDelete(path string) ([]byte, error) {
	return c.api.Request("DELETE", c.apiPath(path), nil, nil)
}

// Status checks API health and system status.
func (c *Commands) Status(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Get health
	healthBody, err := c.apiGet("/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Get status
	statusBody, err := c.apiGet("/status", nil)
	if err != nil {
		return fmt.Errorf("status check failed: %w", err)
	}

	if *jsonOutput {
		fmt.Println("{")
		fmt.Printf("  \"health\": %s,\n", string(healthBody))
		fmt.Printf("  \"status\": %s\n", string(statusBody))
		fmt.Println("}")
		return nil
	}

	var health map[string]interface{}
	var status map[string]interface{}
	_ = json.Unmarshal(healthBody, &health)
	_ = json.Unmarshal(statusBody, &status)

	fmt.Printf("Health: %v\n", health["status"])
	if svc, ok := status["service"].(map[string]interface{}); ok {
		fmt.Printf("Service: %v v%v (%v)\n", svc["name"], svc["version"], svc["status"])
	}
	if stats, ok := status["statistics"].(map[string]interface{}); ok {
		fmt.Printf("Builds: %v total, %v active, %v completed, %v failed\n",
			stats["total_builds"], stats["active_builds"], stats["completed_builds"], stats["failed_builds"])
	}
	return nil
}

// TemplatesList lists available desktop templates.
func (c *Commands) TemplatesList(args []string) error {
	fs := flag.NewFlagSet("templates", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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

	fmt.Println("Available Templates:")
	for _, t := range resp.Templates {
		fmt.Printf("  %-15s %s [%s]\n", t.Type, t.Name, t.Complexity)
		fmt.Printf("                  %s\n", t.Description)
	}
	return nil
}

// TemplateGet gets template details.
func (c *Commands) TemplateGet(args []string) error {
	fs := flag.NewFlagSet("template", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
		fmt.Println("No desktop records found")
		return nil
	}

	fmt.Println("Desktop Records:")
	for _, r := range resp.Records {
		fmt.Printf("  %-36s %-20s %s\n", r.Record.ID, r.Record.ScenarioName, r.BuildState)
	}
	return nil
}

// RecordsMove moves desktop wrapper.
func (c *Commands) RecordsMove(args []string) error {
	fs := flag.NewFlagSet("records-move", flag.ContinueOnError)
	target := fs.String("target", "destination", "Move target: 'destination' or 'custom'")
	destPath := fs.String("path", "", "Custom destination path")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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

	fmt.Println("Record moved successfully")
	cliutil.PrintJSON(body)
	return nil
}

// RecordsDelete deletes desktop app.
func (c *Commands) RecordsDelete(args []string) error {
	fs := flag.NewFlagSet("records-delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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

	fmt.Printf("Desktop app deleted for %s\n", scenario)
	return nil
}

// Download downloads built package.
func (c *Commands) Download(args []string) error {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	output := fs.String("output", "", "Output file path (default: current directory)")
	if err := fs.Parse(args); err != nil {
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

// WineCheck checks Wine installation status.
func (c *Commands) WineCheck(args []string) error {
	fs := flag.NewFlagSet("wine-check", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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

	if resp.Installed {
		usable := "usable"
		if !resp.Usable {
			usable = "not usable"
		}
		fmt.Printf("Wine: installed (%s) via %s - %s\n", resp.Version, resp.InstallMethod, usable)
	} else {
		fmt.Println("Wine: not installed")
		if len(resp.Options) > 0 {
			fmt.Printf("Available install methods: %s\n", strings.Join(resp.Options, ", "))
		}
	}
	return nil
}

// WineInstall installs Wine.
func (c *Commands) WineInstall(args []string) error {
	fs := flag.NewFlagSet("wine-install", flag.ContinueOnError)
	method := fs.String("method", "", "Installation method: flatpak, flatpak-auto, appimage")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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

	fmt.Printf("Wine installation started: %s\n", resp.InstallID)
	fmt.Printf("Check status: %s wine-status %s\n", appName, resp.InstallID)
	return nil
}

// WineStatus gets Wine install status.
func (c *Commands) WineStatus(args []string) error {
	fs := flag.NewFlagSet("wine-status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
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
