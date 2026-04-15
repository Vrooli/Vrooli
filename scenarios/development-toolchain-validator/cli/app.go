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

func boolPtr(v bool) *bool { return &v }

// App wraps the cli-core ScenarioApp with scenario-specific commands.
type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                    appName,
		Version:                 appVersion,
		Description:             "Development Toolchain Validator CLI",
		DefaultAPIBase:          defaultAPIBase,
		ExtraAPIEnvVars:         []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:        buildFingerprint,
		BuildTimestamp:          buildTimestamp,
		BuildSourceRoot:         buildSourceRoot,
		AllowAnonymous:          true,
		IncludeConfigureCommand: boolPtr(false),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return app.registerCommands()
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
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

	// Skill connection commands - maps to /api/v1/connections endpoints
	// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
	connections := cliapp.CommandGroup{
		Title: "Skill Connections",
		Commands: []cliapp.Command{
			{
				Name:        "connection",
				Aliases:     []string{"conn", "connections"},
				NeedsAPI:    true,
				Description: "Manage skill connections (list|get|connect|disconnect|drift)",
				Run:         a.cmdConnection,
			},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{references, connections, config}
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
	body, err := a.core.Get(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// post performs a POST request with the given payload.
func (a *App) post(path string, payload interface{}, result interface{}) error {
	body, err := a.core.Request("POST", path, nil, payload)
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
	body, err := a.core.Request("PATCH", path, nil, payload)
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
	_, err := a.core.Request("DELETE", path, nil, nil)
	return err
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
		return writeJSONOutput(resp)
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
		return writeJSONOutput(ref)
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
		return writeJSONOutput(ref)
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
		return writeJSONOutput(ref)
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
		return writeJSONOutput(map[string]interface{}{
			"success": true,
			"deleted": refID,
		})
	}

	fmt.Printf("Deleted reference: %s\n", refID)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Skill Connection Commands
// Maps to: GET/POST /api/v1/connections, GET/PATCH/DELETE /api/v1/connections/{id}
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
// ─────────────────────────────────────────────────────────────────────────────

// connectionResponse represents a skill connection from the API.
type connectionResponse struct {
	ID               string `json:"id"`
	ReferenceID      string `json:"reference_id"`
	SkillID          string `json:"skill_id"`
	SkillVersion     string `json:"skill_version,omitempty"`
	SkillContentHash string `json:"skill_content_hash,omitempty"`
	ConnectedAt      string `json:"connected_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// connectionListResponse represents a list of connections.
type connectionListResponse struct {
	Connections []connectionResponse `json:"connections"`
	Count       int                  `json:"count"`
}

// connectionConnectRequest represents the request body for creating a connection.
type connectionConnectRequest struct {
	ReferenceID      string `json:"reference_id"`
	SkillID          string `json:"skill_id"`
	SkillVersion     string `json:"skill_version,omitempty"`
	SkillContentHash string `json:"skill_content_hash,omitempty"`
}

// driftCheckRequest represents the request body for checking drift.
type driftCheckRequest struct {
	CurrentVersion string `json:"current_version"`
	CurrentHash    string `json:"current_hash"`
}

// driftStatusResponse represents the drift check response.
type driftStatusResponse struct {
	ConnectionID   string `json:"connection_id"`
	SkillID        string `json:"skill_id"`
	StoredVersion  string `json:"stored_version"`
	StoredHash     string `json:"stored_hash"`
	CurrentVersion string `json:"current_version"`
	CurrentHash    string `json:"current_hash"`
	HasDrifted     bool   `json:"has_drifted"`
	VersionChanged bool   `json:"version_changed"`
	ContentChanged bool   `json:"content_changed"`
}

// cmdConnection routes to connection subcommands.
func (a *App) cmdConnection(args []string) error {
	if len(args) == 0 {
		return a.printConnUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "help":
		return a.printConnUsage()
	case "list", "ls":
		return a.cmdConnList(subArgs)
	case "get", "show":
		return a.cmdConnGet(subArgs)
	case "connect", "add":
		return a.cmdConnConnect(subArgs)
	case "disconnect", "rm", "delete":
		return a.cmdConnDisconnect(subArgs)
	case "drift", "check":
		return a.cmdConnDrift(subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, a.connUsageText())
	}
}

func (a *App) printConnUsage() error {
	fmt.Println(a.connUsageText())
	return nil
}

func (a *App) connUsageText() string {
	return `Usage: development-toolchain-validator connection <subcommand> [args]

Subcommands:
  list, ls                              List all skill connections
  get, show <id>                        Get connection by ID
  connect, add --reference R --skill S  Connect a skill to a reference
  disconnect, rm <id>                   Disconnect a skill
  drift, check <id> --version V --hash H  Check if skill has drifted

Flags:
  --json        Output as JSON
  --reference   Filter by reference ID (list) or set reference ID (connect)
  --skill       Filter by skill ID (list) or set skill ID (connect)
  --version     Skill version (connect/drift)
  --hash        Skill content hash (connect/drift)

Examples:
  connection list --reference ref-123
  connection get conn-abc123
  connection connect --reference ref-123 --skill api-steer --version v1.0
  connection disconnect conn-abc123
  connection drift conn-abc123 --version v2.0 --hash newHash123`
}

// cmdConnList lists all connections with optional filtering.
// Maps to: GET /api/v1/connections
func (a *App) cmdConnList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	referenceID := fs.String("reference", "", "Filter by reference ID")
	skillID := fs.String("skill", "", "Filter by skill ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *referenceID != "" {
		query.Set("reference_id", *referenceID)
	}
	if *skillID != "" {
		query.Set("skill_id", *skillID)
	}

	var resp connectionListResponse
	if err := a.getWithQuery("/connections", query, &resp); err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	if *jsonOut {
		return writeJSONOutput(resp)
	}

	if len(resp.Connections) == 0 {
		fmt.Println("No connections found")
		return nil
	}

	fmt.Printf("Connections (%d):\n", resp.Count)
	for _, c := range resp.Connections {
		version := c.SkillVersion
		if version == "" {
			version = "(no version)"
		}
		fmt.Printf("  %-36s  %-20s  %s\n", c.ID, c.SkillID, version)
	}
	return nil
}

// cmdConnGet retrieves a connection by ID.
// Maps to: GET /api/v1/connections/{id}
func (a *App) cmdConnGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: connection get <id> [--json]")
	}
	connID := fs.Arg(0)

	var conn connectionResponse
	if err := a.get(fmt.Sprintf("/connections/%s", connID), &conn); err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	if *jsonOut {
		return writeJSONOutput(conn)
	}

	fmt.Printf("ID: %s\n", conn.ID)
	fmt.Printf("Reference ID: %s\n", conn.ReferenceID)
	fmt.Printf("Skill ID: %s\n", conn.SkillID)
	if conn.SkillVersion != "" {
		fmt.Printf("Version: %s\n", conn.SkillVersion)
	}
	if conn.SkillContentHash != "" {
		fmt.Printf("Content Hash: %s\n", conn.SkillContentHash)
	}
	if conn.ConnectedAt != "" {
		fmt.Printf("Connected: %s\n", conn.ConnectedAt)
	}
	if conn.UpdatedAt != "" {
		fmt.Printf("Updated: %s\n", conn.UpdatedAt)
	}
	return nil
}

// cmdConnConnect creates a new skill connection.
// Maps to: POST /api/v1/connections
func (a *App) cmdConnConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	referenceID := fs.String("reference", "", "Reference ID (required)")
	skillID := fs.String("skill", "", "Skill ID (required)")
	version := fs.String("version", "", "Skill version")
	hash := fs.String("hash", "", "Skill content hash")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	// Validate required fields
	if *referenceID == "" || *skillID == "" {
		return fmt.Errorf("usage: connection connect --reference R --skill S [--version V] [--hash H] [--json]\n\nBoth --reference and --skill are required")
	}

	req := connectionConnectRequest{
		ReferenceID:      *referenceID,
		SkillID:          *skillID,
		SkillVersion:     *version,
		SkillContentHash: *hash,
	}

	var conn connectionResponse
	if err := a.post("/connections", req, &conn); err != nil {
		return fmt.Errorf("failed to connect skill: %w", err)
	}

	if *jsonOut {
		return writeJSONOutput(conn)
	}

	fmt.Printf("Connected skill: %s to %s [%s]\n", conn.SkillID, conn.ReferenceID, conn.ID)
	fmt.Printf("  View: development-toolchain-validator connection get %s\n", conn.ID)
	return nil
}

// cmdConnDisconnect removes a skill connection.
// Maps to: DELETE /api/v1/connections/{id}
func (a *App) cmdConnDisconnect(args []string) error {
	fs := flag.NewFlagSet("disconnect", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: connection disconnect <id> [--json]")
	}
	connID := fs.Arg(0)

	if err := a.delete(fmt.Sprintf("/connections/%s", connID)); err != nil {
		return fmt.Errorf("failed to disconnect skill: %w", err)
	}

	if *jsonOut {
		return writeJSONOutput(map[string]interface{}{
			"success":      true,
			"disconnected": connID,
		})
	}

	fmt.Printf("Disconnected: %s\n", connID)
	return nil
}

// cmdConnDrift checks if a skill has drifted from its stored version.
// Maps to: POST /api/v1/connections/{id}/drift
func (a *App) cmdConnDrift(args []string) error {
	fs := flag.NewFlagSet("drift", flag.ContinueOnError)
	version := fs.String("version", "", "Current skill version (required)")
	hash := fs.String("hash", "", "Current skill content hash (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: connection drift <id> --version V --hash H [--json]")
	}
	connID := fs.Arg(0)

	if *version == "" || *hash == "" {
		return fmt.Errorf("both --version and --hash are required for drift check")
	}

	req := driftCheckRequest{
		CurrentVersion: *version,
		CurrentHash:    *hash,
	}

	var status driftStatusResponse
	if err := a.post(fmt.Sprintf("/connections/%s/drift", connID), req, &status); err != nil {
		return fmt.Errorf("failed to check drift: %w", err)
	}

	if *jsonOut {
		return writeJSONOutput(status)
	}

	fmt.Printf("Skill: %s\n", status.SkillID)
	fmt.Printf("Connection: %s\n", status.ConnectionID)
	fmt.Printf("Stored Version: %s → Current: %s\n", status.StoredVersion, status.CurrentVersion)
	fmt.Printf("Stored Hash: %s → Current: %s\n", status.StoredHash, status.CurrentHash)

	if status.HasDrifted {
		fmt.Println("\n⚠️  DRIFT DETECTED")
		if status.VersionChanged {
			fmt.Println("  - Version has changed")
		}
		if status.ContentChanged {
			fmt.Println("  - Content has changed")
		}
		fmt.Println("\nConsider updating the connection configuration.")
	} else {
		fmt.Println("\n✓ No drift detected - skill matches stored version")
	}

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

// writeJSONOutput writes data as indented JSON to stdout.
// This consolidates the repeated JSON output pattern across commands.
func writeJSONOutput(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
