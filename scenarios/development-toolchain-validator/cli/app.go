// DOC: docs/internal/CLI_AUDIT.md
// DOC: docs/internal/SEAMS.md#cli-api-parity
//
// Package main implements the development-toolchain-validator CLI.
//
// The CLI is a thin wrapper over the API, following the cli-core patterns
// documented in the CLI Steer skill. All commands map to API endpoints.
//
// [REQ:REQ-P0-011] Core CLI Operations Interface
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "development-toolchain-validator"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wraps the cli-core ScenarioApp with scenario-specific commands.
type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Development Toolchain Validator CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
		},
	}

	// Reference commands - maps to /api/v1/references endpoints
	references := cliapp.CommandGroup{
		Title: "References",
		Commands: []cliapp.Command{
			{
				Name:        "reference",
				Aliases:     []string{"ref", "references"},
				NeedsAPI:    true,
				Description: "Manage reference scenarios (list|get|create|update|delete)",
				Run:         a.cmdReference,
			},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, references, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP Helper Methods
// ─────────────────────────────────────────────────────────────────────────────

// get performs a GET request and unmarshals the response.
func (a *App) get(path string, result interface{}) error {
	return a.getWithQuery(path, nil, result)
}

// getWithQuery performs a GET request with query parameters.
func (a *App) getWithQuery(path string, query url.Values, result interface{}) error {
	body, err := a.core.APIClient.Request("GET", a.apiPath(path), query, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// post performs a POST request with the given payload.
func (a *App) post(path string, payload interface{}, result interface{}) error {
	body, err := a.core.APIClient.Request("POST", a.apiPath(path), nil, payload)
	if err != nil {
		return err
	}
	if result != nil && len(body) > 0 {
		return json.Unmarshal(body, result)
	}
	return nil
}

// patch performs a PATCH request with the given payload.
func (a *App) patch(path string, payload interface{}, result interface{}) error {
	body, err := a.core.APIClient.Request("PATCH", a.apiPath(path), nil, payload)
	if err != nil {
		return err
	}
	if result != nil && len(body) > 0 {
		return json.Unmarshal(body, result)
	}
	return nil
}

// delete performs a DELETE request.
func (a *App) delete(path string) error {
	_, err := a.core.APIClient.Request("DELETE", a.apiPath(path), nil, nil)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Health Command
// ─────────────────────────────────────────────────────────────────────────────

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

func (a *App) cmdStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		fmt.Printf("Ready: %v\n", parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reference Commands
// Maps to: GET/POST /api/v1/references, GET/PATCH/DELETE /api/v1/references/{id}
// ─────────────────────────────────────────────────────────────────────────────

// referenceResponse represents a reference from the API.
type referenceResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Template    string `json:"template"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// referenceListResponse represents a list of references.
type referenceListResponse struct {
	References []referenceResponse `json:"references"`
	Count      int                 `json:"count"`
}

// referenceCreateRequest represents the request body for creating a reference.
type referenceCreateRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Template    string `json:"template"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// referenceUpdateRequest represents the request body for updating a reference.
type referenceUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Template    *string `json:"template,omitempty"`
	Path        *string `json:"path,omitempty"`
	Description *string `json:"description,omitempty"`
}

// cmdReference routes to reference subcommands.
func (a *App) cmdReference(args []string) error {
	if len(args) == 0 {
		return a.printRefUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "help":
		return a.printRefUsage()
	case "list", "ls":
		return a.cmdRefList(subArgs)
	case "get", "show":
		return a.cmdRefGet(subArgs)
	case "create", "add":
		return a.cmdRefCreate(subArgs)
	case "update":
		return a.cmdRefUpdate(subArgs)
	case "delete", "rm":
		return a.cmdRefDelete(subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, a.refUsageText())
	}
}

func (a *App) printRefUsage() error {
	fmt.Println(a.refUsageText())
	return nil
}

func (a *App) refUsageText() string {
	return `Usage: development-toolchain-validator reference <subcommand> [args]

Subcommands:
  list, ls                              List all references
  get, show <id|slug>                   Get reference by ID or slug
  create, add --slug S --name N --template T --path P  Create a reference
  update <id> [--name N] [--template T] [--path P] [--description D]  Update a reference
  delete, rm <id>                       Delete a reference

Flags:
  --json        Output as JSON
  --template    Filter by template (list) or set template (create/update)

Examples:
  reference list --template react-vite
  reference get reference-react-vite
  reference create --slug my-ref --name "My Reference" --template react-vite --path /path/to/scenario
  reference update abc123 --name "Updated Name"
  reference delete abc123`
}

// cmdRefList lists all references with optional filtering.
// Maps to: GET /api/v1/references
func (a *App) cmdRefList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	template := fs.String("template", "", "Filter by template")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *template != "" {
		query.Set("template", *template)
	}

	var resp referenceListResponse
	if err := a.getWithQuery("/references", query, &resp); err != nil {
		return fmt.Errorf("failed to list references: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.References) == 0 {
		fmt.Println("No references found")
		return nil
	}

	fmt.Printf("References (%d):\n", resp.Count)
	for _, r := range resp.References {
		desc := ""
		if r.Description != "" {
			desc = fmt.Sprintf(" - %s", truncate(r.Description, 40))
		}
		fmt.Printf("  %-30s %-15s %s%s\n", r.Slug, r.Template, r.Name, desc)
	}
	return nil
}

// cmdRefGet retrieves a reference by ID or slug.
// Maps to: GET /api/v1/references/{id} or GET /api/v1/references/by-slug/{slug}
func (a *App) cmdRefGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reference get <id|slug> [--json]")
	}
	identifier := fs.Arg(0)

	var ref referenceResponse
	// Try by ID first, fall back to slug
	path := fmt.Sprintf("/references/%s", identifier)
	if err := a.get(path, &ref); err != nil {
		// If not found, try by slug
		path = fmt.Sprintf("/references/by-slug/%s", identifier)
		if err2 := a.get(path, &ref); err2 != nil {
			return fmt.Errorf("failed to get reference: %w", err)
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ref)
	}

	fmt.Printf("ID: %s\n", ref.ID)
	fmt.Printf("Slug: %s\n", ref.Slug)
	fmt.Printf("Name: %s\n", ref.Name)
	fmt.Printf("Template: %s\n", ref.Template)
	fmt.Printf("Path: %s\n", ref.Path)
	if ref.Description != "" {
		fmt.Printf("Description: %s\n", ref.Description)
	}
	if ref.CreatedAt != "" {
		fmt.Printf("Created: %s\n", ref.CreatedAt)
	}
	if ref.UpdatedAt != "" {
		fmt.Printf("Updated: %s\n", ref.UpdatedAt)
	}
	return nil
}

// cmdRefCreate creates a new reference.
// Maps to: POST /api/v1/references
func (a *App) cmdRefCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	slug := fs.String("slug", "", "Reference slug (required)")
	name := fs.String("name", "", "Display name (required)")
	template := fs.String("template", "", "Template type (required)")
	path := fs.String("path", "", "Scenario path (required)")
	description := fs.String("description", "", "Description")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	// Validate required fields
	if *slug == "" || *name == "" || *template == "" || *path == "" {
		return fmt.Errorf("usage: reference create --slug S --name N --template T --path P [--description D] [--json]\n\nAll of --slug, --name, --template, and --path are required")
	}

	req := referenceCreateRequest{
		Slug:        *slug,
		Name:        *name,
		Template:    *template,
		Path:        *path,
		Description: *description,
	}

	var ref referenceResponse
	if err := a.post("/references", req, &ref); err != nil {
		return fmt.Errorf("failed to create reference: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ref)
	}

	fmt.Printf("Created reference: %s [%s]\n", ref.Slug, ref.ID)
	fmt.Printf("  View: development-toolchain-validator reference get %s\n", ref.ID)
	return nil
}

// cmdRefUpdate updates an existing reference.
// Maps to: PATCH /api/v1/references/{id}
func (a *App) cmdRefUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "Update display name")
	template := fs.String("template", "", "Update template type")
	path := fs.String("path", "", "Update scenario path")
	description := fs.String("description", "", "Update description")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reference update <id> [--name N] [--template T] [--path P] [--description D] [--json]")
	}
	refID := fs.Arg(0)

	// Check that at least one field is being updated
	if *name == "" && *template == "" && *path == "" && *description == "" {
		return fmt.Errorf("must specify at least one field to update: --name, --template, --path, or --description")
	}

	req := referenceUpdateRequest{}
	if *name != "" {
		req.Name = name
	}
	if *template != "" {
		req.Template = template
	}
	if *path != "" {
		req.Path = path
	}
	if *description != "" {
		req.Description = description
	}

	var ref referenceResponse
	if err := a.patch(fmt.Sprintf("/references/%s", refID), req, &ref); err != nil {
		return fmt.Errorf("failed to update reference: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ref)
	}

	fmt.Printf("Updated reference: %s [%s]\n", ref.Slug, ref.ID)
	return nil
}

// cmdRefDelete deletes a reference.
// Maps to: DELETE /api/v1/references/{id}
func (a *App) cmdRefDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reference delete <id> [--json]")
	}
	refID := fs.Arg(0)

	if err := a.delete(fmt.Sprintf("/references/%s", refID)); err != nil {
		return fmt.Errorf("failed to delete reference: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"success": true,
			"deleted": refID,
		})
	}

	fmt.Printf("Deleted reference: %s\n", refID)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────────────────────────

// truncate shortens a string to maxLen characters, adding "..." if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
