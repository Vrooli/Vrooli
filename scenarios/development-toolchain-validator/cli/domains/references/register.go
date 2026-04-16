package references

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"development-toolchain-validator/cli/internal/textutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "development-toolchain-validator"

type ReferenceResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Template    string `json:"template"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type ReferenceListResponse struct {
	References []ReferenceResponse `json:"references"`
	Count      int                 `json:"count"`
}

type ReferenceCreateRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Template    string `json:"template"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

type ReferenceUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Template    *string `json:"template,omitempty"`
	Path        *string `json:"path,omitempty"`
	Description *string `json:"description,omitempty"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "reference",
		Description: "Manage reference scenarios",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, NeedsAPI: true, Description: "List reference scenarios", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, NeedsAPI: true, Description: "Get a reference by ID or slug", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Aliases: []string{"add"}, NeedsAPI: true, Description: "Create a reference", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update a reference", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, NeedsAPI: true, Description: "Delete a reference", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func Run(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "list", "ls":
		return runList(core, args[1:])
	case "get", "show":
		return runGet(core, args[1:])
	case "create", "add":
		return runCreate(core, args[1:])
	case "update":
		return runUpdate(core, args[1:])
	case "delete", "rm":
		return runDelete(core, args[1:])
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", args[0], usageText())
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("reference list", flag.ContinueOnError)
	template := fs.String("template", "", "Filter by template")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*template) != "" {
		query.Set("template", strings.TrimSpace(*template))
	}

	var resp ReferenceListResponse
	if err := getWithQuery(core, "/references", query, &resp); err != nil {
		return fmt.Errorf("failed to list references: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("References found: %d", resp.Count),
		},
		Results:        renderListResults(resp.References),
		RetrievalHints: []string{cliName + " reference get <id-or-slug>", cliName + " reference list --template react-vite"},
	}
	if strings.TrimSpace(*template) != "" {
		report.Summary = append(report.Summary, "Template filter: "+strings.TrimSpace(*template))
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("reference get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reference get <id|slug> [--json]")
	}

	identifier := fs.Arg(0)
	var ref ReferenceResponse
	path := fmt.Sprintf("/references/%s", identifier)
	if err := get(core, path, &ref); err != nil {
		path = fmt.Sprintf("/references/by-slug/%s", identifier)
		if err2 := get(core, path, &ref); err2 != nil {
			return fmt.Errorf("failed to get reference: %w", err)
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{"Reference: " + ref.Slug, "Reference ID: " + ref.ID},
		ResultsHeading: "Details",
		Results:        detailLines(ref),
		RetrievalHints: []string{cliName + " reference update " + ref.ID + " --name \"...\"", cliName + " connection list --reference " + ref.ID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("reference create", flag.ContinueOnError)
	slug := fs.String("slug", "", "Reference slug (required)")
	name := fs.String("name", "", "Display name (required)")
	template := fs.String("template", "", "Template type (required)")
	path := fs.String("path", "", "Scenario path (required)")
	description := fs.String("description", "", "Description")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *slug == "" || *name == "" || *template == "" || *path == "" {
		return fmt.Errorf("usage: reference create --slug S --name N --template T --path P [--description D] [--json]\n\nAll of --slug, --name, --template, and --path are required")
	}

	req := ReferenceCreateRequest{
		Slug:        *slug,
		Name:        *name,
		Template:    *template,
		Path:        *path,
		Description: *description,
	}

	var ref ReferenceResponse
	if err := request(core, "POST", "/references", req, &ref); err != nil {
		return fmt.Errorf("failed to create reference: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Reference created", "Reference ID: " + ref.ID},
		Changes: []string{
			"Slug: " + ref.Slug,
			"Name: " + ref.Name,
			"Template: " + ref.Template,
			"Path: " + ref.Path,
		},
		NextCommand: []string{cliName + " reference get " + ref.ID, cliName + " connection connect --reference " + ref.ID + " --skill <skill-id>"},
	}
	if ref.Description != "" {
		report.Changes = append(report.Changes, "Description: "+ref.Description)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("reference update", flag.ContinueOnError)
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
	if *name == "" && *template == "" && *path == "" && *description == "" {
		return fmt.Errorf("must specify at least one field to update: --name, --template, --path, or --description")
	}

	req := ReferenceUpdateRequest{}
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

	var ref ReferenceResponse
	if err := request(core, "PATCH", fmt.Sprintf("/references/%s", refID), req, &ref); err != nil {
		return fmt.Errorf("failed to update reference: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Reference updated", "Reference ID: " + ref.ID},
		Changes:     changesForUpdate(ref, req),
		NextCommand: []string{cliName + " reference get " + ref.ID, cliName + " reference list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("reference delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reference delete <id> [--json]")
	}

	refID := fs.Arg(0)
	if _, err := core.Request("DELETE", fmt.Sprintf("/references/%s", refID), nil, nil); err != nil {
		return fmt.Errorf("failed to delete reference: %w", err)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Reference deleted", "Reference ID: " + refID},
		Changes:     []string{"Removed reference from the development-toolchain-validator catalog"},
		NextCommand: []string{cliName + " reference list"},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func get(core *cliapp.ScenarioApp, path string, result interface{}) error {
	return getWithQuery(core, path, nil, result)
}

func getWithQuery(core *cliapp.ScenarioApp, path string, query url.Values, result interface{}) error {
	body, err := core.Get(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func request(core *cliapp.ScenarioApp, method, path string, payload interface{}, result interface{}) error {
	body, err := core.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}

func renderListResults(items []ReferenceResponse) []string {
	if len(items) == 0 {
		return nil
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		line := fmt.Sprintf("%s | %s | %s | %s", item.ID, item.Slug, item.Template, item.Name)
		if item.Description != "" {
			line += " | " + textutil.Truncate(item.Description, 40)
		}
		lines = append(lines, line)
	}
	return lines
}

func detailLines(ref ReferenceResponse) []string {
	lines := []string{
		"Slug: " + ref.Slug,
		"Name: " + ref.Name,
		"Template: " + ref.Template,
		"Path: " + ref.Path,
	}
	if ref.Description != "" {
		lines = append(lines, "Description: "+ref.Description)
	}
	if ref.CreatedAt != "" {
		lines = append(lines, "Created: "+ref.CreatedAt)
	}
	if ref.UpdatedAt != "" {
		lines = append(lines, "Updated: "+ref.UpdatedAt)
	}
	return lines
}

func changesForUpdate(ref ReferenceResponse, req ReferenceUpdateRequest) []string {
	var changes []string
	if req.Name != nil {
		changes = append(changes, "Name: "+ref.Name)
	}
	if req.Template != nil {
		changes = append(changes, "Template: "+ref.Template)
	}
	if req.Path != nil {
		changes = append(changes, "Path: "+ref.Path)
	}
	if req.Description != nil {
		changes = append(changes, "Description: "+ref.Description)
	}
	return changes
}

func printUsage() {
	fmt.Println(usageText())
}

func usageText() string {
	return `Usage: development-toolchain-validator reference <subcommand> [args]

Subcommands:
  list, ls                              List all references
  get, show <id|slug>                   Get reference by ID or slug
  create, add --slug S --name N --template T --path P  Create a reference
  update <id> [--name N] [--template T] [--path P] [--description D]  Update a reference
  delete, rm <id>                       Delete a reference

Flags:
  --json        Output as structured report JSON
  --template    Filter by template (list) or set template (create/update)

Examples:
  reference list --template react-vite
  reference get reference-react-vite
  reference create --slug my-ref --name "My Reference" --template react-vite --path /path/to/scenario
  reference update abc123 --name "Updated Name"
  reference delete abc123`
}
