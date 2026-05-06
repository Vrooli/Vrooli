package scenariocli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
)

type TemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type TemplateHook struct {
	Description string `json:"description,omitempty"`
	Cmd         string `json:"cmd,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
}

// TemplateRelocation declares an out-of-tree placement performed by the
// generator after the in-tree copy. The directory at From (template-relative)
// is rendered (with placeholder substitution applied to both file content
// and path components) into To (repo-root-relative; may contain placeholders).
//
// Post commands run from the repo root after every relocation in the manifest
// has been applied — useful for codegen steps that depend on the relocated
// content (e.g., regenerating proto artifacts in packages/proto/).
//
// The From directory is automatically excluded from the in-tree copy that
// writes into the scenario destination, so the same source folder doesn't
// end up in two places.
type TemplateRelocation struct {
	Description string         `json:"description,omitempty"`
	From        string         `json:"from"`
	To          string         `json:"to"`
	Post        []TemplateHook `json:"post,omitempty"`
}

type TemplateDesign struct {
	Adapter  string `json:"adapter,omitempty"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type TemplateManifest struct {
	Name          string                 `json:"name,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Stack         []string               `json:"stack,omitempty"`
	StartDocument string                 `json:"startDocument,omitempty"`
	Design        TemplateDesign         `json:"design,omitempty"`
	RequiredVars  map[string]TemplateVar `json:"requiredVars,omitempty"`
	OptionalVars  map[string]TemplateVar `json:"optionalVars,omitempty"`
	Docs          map[string]string      `json:"docs,omitempty"`
	CopyExcludes  []string               `json:"copyExcludes,omitempty"`
	PostHooks     []TemplateHook         `json:"postHooks,omitempty"`
	Relocations   []TemplateRelocation   `json:"relocations,omitempty"`
}

type TemplateInfo struct {
	Name     string
	Path     string
	Manifest TemplateManifest
	Missing  bool
}

type GenerateOptions struct {
	Destination string
	Design      string
	Force       bool
	DryRun      bool
	RunHooks    bool
	Values      map[string]string
}

type (
	TemplateListRequest     struct{}
	TemplateShowRequest     struct{ Name string }
	TemplateValidateRequest struct{}
	GenerateRequest         struct {
		TemplateInfo TemplateInfo
		Options      GenerateOptions
	}
)

// ResolvedRelocation captures a relocation after placeholder substitution,
// so callers (and dry-run output) can show exactly where each From folder
// would land before any disk writes happen.
type ResolvedRelocation struct {
	Description string         `json:"description,omitempty"`
	From        string         `json:"from"` // template-relative source dir
	To          string         `json:"to"`   // absolute path under repo root after substitution
	Post        []TemplateHook `json:"post,omitempty"`
}

type ResolvedDesignCopy struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ResolvedDesign struct {
	KitID     string               `json:"kitId,omitempty"`
	KitName   string               `json:"kitName,omitempty"`
	Version   string               `json:"version,omitempty"`
	AdapterID string               `json:"adapterId,omitempty"`
	Copies    []ResolvedDesignCopy `json:"copies,omitempty"`
}

type GenerateResult struct {
	TemplateName string
	DisplayName  string
	Destination  string
	Values       map[string]string
	Manifest     TemplateManifest
	Design       ResolvedDesign
	Relocations  []ResolvedRelocation
	DryRun       bool
	RunHooks     bool
}

type TemplateValidationIssue struct {
	Template string `json:"template"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type TemplateValidationReport struct {
	Count  int                       `json:"count"`
	Issues []TemplateValidationIssue `json:"issues,omitempty"`
}

func RenderTemplateListResponse(w io.Writer, format cliout.Format, templates []TemplateInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "templates", templates)
	}
	rows := make([][]string, 0, len(templates))
	for _, item := range templates {
		required := formatTemplateRequiredVars(item.Manifest)
		if item.Missing {
			required = "?"
		}
		display := item.Manifest.DisplayName
		if display == "" {
			display = "(template.json missing)"
		}
		rows = append(rows, []string{item.Name, display, required})
	}
	_ = cliout.RenderTable(w, []string{"Name", "Display Name", "Required Vars"}, rows)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Tip: vrooli scenario template show <name>")
	return nil
}

func RenderTemplateShowResponse(w io.Writer, format cliout.Format, info TemplateInfo) error {
	_ = format
	manifest := info.Manifest
	title := manifest.DisplayName
	if title == "" {
		title = info.Name
	}
	_, _ = fmt.Fprintf(w, "%s (%s)\n", title, info.Name)
	if manifest.Description != "" {
		_, _ = fmt.Fprintln(w, manifest.Description)
	}
	if len(manifest.Stack) > 0 {
		_, _ = fmt.Fprintf(w, "Stack: %s\n", strings.Join(manifest.Stack, ", "))
	}
	if strings.TrimSpace(manifest.StartDocument) != "" {
		_, _ = fmt.Fprintf(w, "Start document: %s\n", manifest.StartDocument)
	}
	if strings.TrimSpace(manifest.Design.Default) != "" || strings.TrimSpace(manifest.Design.Adapter) != "" || manifest.Design.Required {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Design:")
		if manifest.Design.Default != "" {
			_, _ = fmt.Fprintf(w, "  default: %s\n", manifest.Design.Default)
		}
		if manifest.Design.Adapter != "" {
			_, _ = fmt.Fprintf(w, "  adapter: %s\n", manifest.Design.Adapter)
		}
		if manifest.Design.Required {
			_, _ = fmt.Fprintln(w, "  required: yes")
		}
	}
	writeTemplateVarTable(w, "Required Variables", manifest.RequiredVars)
	writeTemplateVarTable(w, "Optional Variables", manifest.OptionalVars)
	if len(manifest.PostHooks) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Post Hooks:")
		for _, hook := range manifest.PostHooks {
			line := hook.Description
			if line == "" {
				line = hook.Cmd
			}
			_, _ = fmt.Fprintf(w, "  - %s\n", line)
		}
	}
	if len(manifest.Relocations) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Relocations:")
		for _, reloc := range manifest.Relocations {
			label := reloc.Description
			if label == "" {
				label = reloc.From + " -> " + reloc.To
			}
			_, _ = fmt.Fprintf(w, "  - %s\n", label)
			_, _ = fmt.Fprintf(w, "      from: %s\n", reloc.From)
			_, _ = fmt.Fprintf(w, "      to:   %s\n", reloc.To)
			for _, hook := range reloc.Post {
				cmd := hook.Description
				if cmd == "" {
					cmd = hook.Cmd
				}
				_, _ = fmt.Fprintf(w, "      post: %s\n", cmd)
			}
		}
	}
	if len(manifest.Docs) > 0 {
		docKeys := make([]string, 0, len(manifest.Docs))
		for key := range manifest.Docs {
			docKeys = append(docKeys, key)
		}
		sort.Strings(docKeys)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Docs:")
		for _, key := range docKeys {
			_, _ = fmt.Fprintf(w, "  - %s: %s\n", key, manifest.Docs[key])
		}
	}
	if entries, err := os.ReadDir(info.Path); err == nil {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Files:")
		for _, name := range names {
			_, _ = fmt.Fprintf(w, "  - %s\n", name)
		}
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Tip: vrooli scenario generate %s%s\n", info.Name, FormatScenarioTemplateRequiredFlags(manifest.RequiredVars))
	return nil
}

func RenderGenerateResponse(w io.Writer, format cliout.Format, result GenerateResult) error {
	_ = format
	if result.DryRun {
		_, _ = fmt.Fprintf(w, "[DRY-RUN] Would generate template %s at %s\n", result.TemplateName, result.Destination)
		WriteTemplateValues(w, result.Values)
		WriteTemplateDesign(w, result.Design)
		WriteTemplateRelocations(w, result.Relocations)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Created %s at %s\n", result.DisplayName, result.Destination)
	WriteTemplateValues(w, result.Values)
	WriteTemplateDesign(w, result.Design)
	WriteTemplateRelocations(w, result.Relocations)
	WriteTemplateNextSteps(w, result.Destination, result.Manifest)
	if !result.RunHooks {
		WriteTemplateHooks(w, result.Manifest)
	}
	return nil
}

func WriteTemplateDesign(w io.Writer, design ResolvedDesign) {
	if strings.TrimSpace(design.KitID) == "" {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Design:")
	_, _ = fmt.Fprintf(w, "  kit: %s", design.KitID)
	if design.Version != "" {
		_, _ = fmt.Fprintf(w, " (%s)", design.Version)
	}
	_, _ = fmt.Fprintln(w)
	if design.AdapterID != "" {
		_, _ = fmt.Fprintf(w, "  adapter: %s\n", design.AdapterID)
	}
	if len(design.Copies) > 0 {
		_, _ = fmt.Fprintln(w, "  copy:")
		for _, copy := range design.Copies {
			_, _ = fmt.Fprintf(w, "    - %s -> %s\n", copy.From, copy.To)
		}
	}
}

// WriteTemplateRelocations renders the resolved relocations (if any). Used
// by both the dry-run path (so authors can see exactly where each folder
// would land) and the success path (so the destination summary explains
// what happened outside the scenario directory).
func WriteTemplateRelocations(w io.Writer, relocations []ResolvedRelocation) {
	if len(relocations) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Relocations:")
	for _, reloc := range relocations {
		label := reloc.Description
		if label == "" {
			label = reloc.From + " -> " + reloc.To
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", label)
		_, _ = fmt.Fprintf(w, "      from: %s\n", reloc.From)
		_, _ = fmt.Fprintf(w, "      to:   %s\n", reloc.To)
		for _, hook := range reloc.Post {
			cmd := hook.Description
			if cmd == "" {
				cmd = hook.Cmd
			}
			_, _ = fmt.Fprintf(w, "      post: %s\n", cmd)
		}
	}
}

func RenderTemplateValidateResponse(w io.Writer, format cliout.Format, report TemplateValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "report", report)
	}
	if len(report.Issues) == 0 {
		_, _ = fmt.Fprintf(w, "Validated %d scenario templates\n", report.Count)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Scenario template validation failed (%d templates checked)\n", report.Count)
	for _, issue := range report.Issues {
		line := issue.Template
		if strings.TrimSpace(issue.Path) != "" {
			line += " [" + issue.Path + "]"
		}
		_, _ = fmt.Fprintf(w, "  - %s: %s\n", line, issue.Message)
	}
	return nil
}

func WriteTemplateValues(w io.Writer, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w, "Resolved values:")
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "  %s=%s\n", key, values[key])
	}
}

func WriteTemplateNextSteps(w io.Writer, destination string, manifest TemplateManifest) {
	_, _ = fmt.Fprintln(w)
	if startDocument := strings.TrimSpace(manifest.StartDocument); startDocument != "" {
		_, _ = fmt.Fprintln(w, "Start here:")
		_, _ = fmt.Fprintf(w, "  %s\n", startDocument)
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "Next steps:")
	if strings.TrimSpace(manifest.StartDocument) != "" {
		_, _ = fmt.Fprintln(w, "  1. Read the start document")
		_, _ = fmt.Fprintln(w, "  2. Run scenario setup and tests")
	} else {
		_, _ = fmt.Fprintf(w, "  1. Review files in %s\n", destination)
		_, _ = fmt.Fprintln(w, "  2. Run scenario setup and tests")
	}
	if len(manifest.PostHooks) > 0 {
		_, _ = fmt.Fprintln(w, "  3. Consider re-running with --run-hooks")
	}
}

func WriteTemplateHooks(w io.Writer, manifest TemplateManifest) {
	if len(manifest.PostHooks) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Available post hooks:")
	for _, hook := range manifest.PostHooks {
		line := hook.Description
		if line == "" {
			line = hook.Cmd
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", line)
	}
}

func formatTemplateRequiredVars(manifest TemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func writeTemplateVarTable(w io.Writer, title string, variables map[string]TemplateVar) {
	if len(variables) == 0 {
		return
	}
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, title+":")
	for _, key := range keys {
		variable := variables[key]
		line := key
		if variable.Flag != "" {
			line = fmt.Sprintf("%s (--%s)", key, variable.Flag)
		}
		if variable.Description != "" {
			line += " - " + variable.Description
		}
		if variable.Default != "" {
			line += " [default: " + variable.Default + "]"
		}
		_, _ = fmt.Fprintf(w, "  - %s\n", line)
	}
}
