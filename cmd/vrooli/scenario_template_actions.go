package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
)

type (
	scenarioTemplateListRequest struct{}
	scenarioTemplateShowRequest struct {
		Name string
	}
)

type scenarioGenerateRequest struct {
	TemplateInfo scenarioTemplateInfo
	Options      scenarioGenerateOptions
}

type scenarioGenerateResult struct {
	TemplateName string
	DisplayName  string
	Destination  string
	Values       map[string]string
	Manifest     scenarioTemplateManifest
	DryRun       bool
	RunHooks     bool
}

func parseScenarioTemplateListRequest(ctx *commandContext, args []string) (scenarioTemplateListRequest, error) {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		return scenarioTemplateListRequest{}, commandHelpOnly("Scenario Template Commands:\n  vrooli scenario template list\n  vrooli scenario template show <template>\n  vrooli scenario generate <template> [options]")
	}
	if len(args) > 0 {
		return scenarioTemplateListRequest{}, unknownOptionError("scenario template", args[0])
	}
	return scenarioTemplateListRequest{}, nil
}

func runScenarioTemplateListRequest(app *App, ctx *commandContext, req scenarioTemplateListRequest) (cliout.Format, []scenarioTemplateInfo, error) {
	templates, err := loadScenarioTemplates(ctx.Root)
	if err != nil {
		return "", nil, err
	}
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return "", nil, err
	}
	return format, templates, nil
}

func renderScenarioTemplateListResponse(w io.Writer, format cliout.Format, templates []scenarioTemplateInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"templates": templates,
		})
	}

	rows := make([][]string, 0, len(templates))
	for _, item := range templates {
		required := formatScenarioTemplateRequiredVars(item.Manifest)
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

func parseScenarioTemplateShowRequest(ctx *commandContext, args []string) (scenarioTemplateShowRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioTemplateShowRequest{}, commandHelpOnly("Scenario Template Commands:\n  vrooli scenario template list\n  vrooli scenario template show <template>\n  vrooli scenario generate <template> [options]")
		}
	}
	if len(args) == 0 {
		return scenarioTemplateShowRequest{}, usageErrorf("scenario template show", "scenario template show requires a template name")
	}
	if len(args) > 1 {
		return scenarioTemplateShowRequest{}, usageErrorf("scenario template show", "scenario template show accepts exactly one template name")
	}
	return scenarioTemplateShowRequest{Name: args[0]}, nil
}

func runScenarioTemplateShowRequest(app *App, ctx *commandContext, req scenarioTemplateShowRequest) (cliout.Format, scenarioTemplateInfo, error) {
	info, err := loadScenarioTemplate(ctx.Root, req.Name)
	if err != nil {
		return "", scenarioTemplateInfo{}, err
	}
	return cliout.FormatHuman, info, nil
}

func renderScenarioTemplateShowResponse(w io.Writer, format cliout.Format, info scenarioTemplateInfo) error {
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

	writeScenarioTemplateVarTable(w, "Required Variables", manifest.RequiredVars)
	writeScenarioTemplateVarTable(w, "Optional Variables", manifest.OptionalVars)

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

	entries, err := os.ReadDir(info.Path)
	if err == nil {
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
	_, _ = fmt.Fprintf(w, "Tip: vrooli scenario generate %s%s\n", info.Name, formatScenarioTemplateRequiredFlags(manifest))
	return nil
}

func parseScenarioGenerateRequest(ctx *commandContext, args []string) (scenarioGenerateRequest, error) {
	if len(args) == 0 {
		return scenarioGenerateRequest{}, usageErrorf("scenario generate", "scenario generate requires a template name")
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioGenerateRequest{}, commandHelpOnly("Usage: vrooli scenario generate <template> --id <slug> --display-name <name> --description <text> [options]\nOptions:\n  --dest <path>         Destination directory (defaults to scenarios/<id>)\n  --var KEY=VALUE       Additional placeholder override (repeatable)\n  --force               Overwrite destination if it already exists\n  --dry-run             Print the planned actions without writing files\n  --run-hooks           Execute template post hooks after generation")
		}
	}

	templateName := args[0]
	info, err := loadScenarioTemplate(ctx.Root, templateName)
	if err != nil {
		return scenarioGenerateRequest{}, err
	}

	opts, err := parseScenarioGenerateArgs(args[1:], info.Manifest, ctx.Stderr)
	if err != nil {
		return scenarioGenerateRequest{}, err
	}

	if opts.Values["SCENARIO_ID"] == "" {
		return scenarioGenerateRequest{}, usageErrorf("scenario generate", "missing required value: --id")
	}

	missing := make([]string, 0)
	for key, variable := range info.Manifest.RequiredVars {
		if strings.TrimSpace(opts.Values[key]) == "" {
			name := "--" + variable.Flag
			if variable.Flag == "" {
				name = key
			}
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return scenarioGenerateRequest{}, usageErrorf("scenario generate", "missing required values: %s", strings.Join(missing, ", "))
	}

	return scenarioGenerateRequest{TemplateInfo: info, Options: opts}, nil
}

func runScenarioGenerateRequest(app *App, ctx *commandContext, req scenarioGenerateRequest) (cliout.Format, scenarioGenerateResult, error) {
	info := req.TemplateInfo
	opts := req.Options

	currentDate := currentDateUTC()
	randomToken, err := randomTemplateToken()
	if err != nil {
		return "", scenarioGenerateResult{}, err
	}
	values := copyStringMap(opts.Values)
	values["CURRENT_DATE"] = currentDate
	values["RANDOM_TOKEN"] = randomToken

	optionalKeys := make([]string, 0, len(info.Manifest.OptionalVars))
	for key := range info.Manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) != "" {
			continue
		}
		values[key] = renderScenarioTemplateString(info.Manifest.OptionalVars[key].Default, values)
	}

	destination := opts.Destination
	if destination == "" {
		destination = filepath.Join(ctx.Root, "scenarios", values["SCENARIO_ID"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(ctx.Root, filepath.FromSlash(destination))
	}
	destination = filepath.Clean(destination)

	if opts.DryRun {
		return cliout.FormatHuman, scenarioGenerateResult{
			TemplateName: info.Name,
			DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
			Destination:  destination,
			Values:       values,
			Manifest:     info.Manifest,
			DryRun:       true,
			RunHooks:     false,
		}, nil
	}

	if stat, err := os.Stat(destination); err == nil && stat != nil {
		if !opts.Force {
			return "", scenarioGenerateResult{}, fmt.Errorf("destination already exists: %s (use --force to overwrite)", destination)
		}
		if err := os.RemoveAll(destination); err != nil {
			return "", scenarioGenerateResult{}, err
		}
	}

	if err := copyScenarioTemplate(info.Path, destination, values); err != nil {
		return "", scenarioGenerateResult{}, err
	}
	if err := verifyScenarioTemplate(destination); err != nil {
		return "", scenarioGenerateResult{}, err
	}

	result := scenarioGenerateResult{
		TemplateName: info.Name,
		DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
		Destination:  destination,
		Values:       values,
		Manifest:     info.Manifest,
		DryRun:       false,
		RunHooks:     opts.RunHooks,
	}
	if opts.RunHooks {
		if err := runScenarioTemplateHooksWithApp(app, ctx.Root, ctx.Globals, destination, info.Manifest, ctx.Stdout, ctx.Stderr); err != nil {
			return "", scenarioGenerateResult{}, err
		}
	}
	return cliout.FormatHuman, result, nil
}

func renderScenarioGenerateResponse(w io.Writer, format cliout.Format, result scenarioGenerateResult) error {
	if result.DryRun {
		_, _ = fmt.Fprintf(w, "[DRY-RUN] Would generate template %s at %s\n", result.TemplateName, result.Destination)
		writeScenarioTemplateValues(w, result.Values)
		return nil
	}

	_, _ = fmt.Fprintf(w, "Created %s at %s\n", result.DisplayName, result.Destination)
	writeScenarioTemplateValues(w, result.Values)
	writeScenarioTemplateNextSteps(w, result.Destination, result.Manifest)
	if !result.RunHooks {
		writeScenarioTemplateHooks(w, result.Manifest)
	}
	return nil
}
