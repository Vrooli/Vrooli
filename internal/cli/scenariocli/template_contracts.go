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

type TemplateManifest struct {
	Name         string                 `json:"name,omitempty"`
	DisplayName  string                 `json:"displayName,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Stack        []string               `json:"stack,omitempty"`
	RequiredVars map[string]TemplateVar `json:"requiredVars,omitempty"`
	OptionalVars map[string]TemplateVar `json:"optionalVars,omitempty"`
	Docs         map[string]string      `json:"docs,omitempty"`
	PostHooks    []TemplateHook         `json:"postHooks,omitempty"`
}

type TemplateInfo struct {
	Name     string
	Path     string
	Manifest TemplateManifest
	Missing  bool
}

type GenerateOptions struct {
	Destination string
	Force       bool
	DryRun      bool
	RunHooks    bool
	Values      map[string]string
}

type (
	TemplateListRequest struct{}
	TemplateShowRequest struct{ Name string }
	GenerateRequest     struct {
		TemplateInfo TemplateInfo
		Options      GenerateOptions
	}
)

type GenerateResult struct {
	TemplateName string
	DisplayName  string
	Destination  string
	Values       map[string]string
	Manifest     TemplateManifest
	DryRun       bool
	RunHooks     bool
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
		return nil
	}
	_, _ = fmt.Fprintf(w, "Created %s at %s\n", result.DisplayName, result.Destination)
	WriteTemplateValues(w, result.Values)
	WriteTemplateNextSteps(w, result.Destination, result.Manifest)
	if !result.RunHooks {
		WriteTemplateHooks(w, result.Manifest)
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
	_, _ = fmt.Fprintln(w, "Next steps:")
	_, _ = fmt.Fprintf(w, "  1. Review files in %s\n", destination)
	_, _ = fmt.Fprintln(w, "  2. Run scenario setup and tests")
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
