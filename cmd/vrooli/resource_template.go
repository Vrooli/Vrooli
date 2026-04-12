package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

type resourceTemplateGenerateOptions struct {
	TemplateName  string
	BlueprintName string
	Destination   string
	Force         bool
	DryRun        bool
	Values        map[string]string
}

func runResourceTemplateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showResourceTemplateHelp(stdout)
		return nil
	}

	switch normalizeResourceSubcommand(args[0]) {
	case "list":
		return runResourceTemplateListCommand(controller, globals, args[1:], stdout)
	case "show":
		return runResourceTemplateShowCommand(controller, globals, args[1:], stdout)
	case "validate":
		return runResourceTemplateValidateCommand(controller, globals, args[1:], stdout)
	case "generate":
		return runResourceTemplateGenerateCommand(controller, globals, args[1:], stdout, stderr)
	default:
		return usageErrorf("resource template", "unknown resource template command: %s", args[0])
	}
}

func runResourceTemplateListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource template list", "resource template list does not accept positional arguments")
	}
	items, err := controller.ListResourceTemplates()
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":   true,
			"templates": items,
		})
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			item.Manifest.DisplayName,
			item.Manifest.Driver,
			formatResourceTemplateRequiredVars(item.Manifest.RequiredVars),
		})
	}
	if err := cliout.RenderTable(stdout, []string{"Name", "Display Name", "Driver", "Required Vars"}, rows); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Tip: vrooli resource template show <name>")
	return nil
}

func runResourceTemplateShowCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource template show", "resource template show requires exactly one template name")
	}
	info, err := controller.ResourceTemplate(args[0])
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":  true,
			"template": info,
		})
	}

	manifest := info.Manifest
	rows := [][]string{
		{"Name", info.Name},
		{"Display Name", manifest.DisplayName},
		{"Driver", manifest.Driver},
		{"Transitional", boolLabel(manifest.Transitional)},
		{"Description", manifest.Description},
	}
	if err := cliout.RenderTable(stdout, []string{"Field", "Value"}, rows); err != nil {
		return err
	}
	writeResourceTemplateVarTable(stdout, "Required Variables", manifest.RequiredVars)
	writeResourceTemplateVarTable(stdout, "Optional Variables", manifest.OptionalVars)
	if len(manifest.PlatformExpectations) > 0 {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Platform Expectations:")
		for _, line := range manifest.PlatformExpectations {
			_, _ = fmt.Fprintf(stdout, "  - %s\n", line)
		}
	}
	if len(manifest.Docs) > 0 {
		keys := make([]string, 0, len(manifest.Docs))
		for key := range manifest.Docs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Docs:")
		for _, key := range keys {
			_, _ = fmt.Fprintf(stdout, "  - %s: %s\n", key, manifest.Docs[key])
		}
	}
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintf(stdout, "Tip: vrooli resource template generate %s%s\n", info.Name, formatResourceTemplateRequiredFlags(manifest.RequiredVars))
	return nil
}

func runResourceTemplateValidateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource template validate", "resource template validate does not accept positional arguments")
	}
	report, err := controller.ValidateResourceTemplates()
	if err != nil {
		return err
	}
	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(stdout, "Validated %d resource templates\n", report.Count)
	return nil
}

func runResourceTemplateGenerateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			showResourceTemplateGenerateHelp(stdout)
			return nil
		}
	}
	opts, err := parseResourceTemplateGenerateArgs(controller, args, stderr)
	if err != nil {
		showResourceTemplateGenerateHelp(stdout)
		return err
	}
	report, err := controller.GenerateResourceTemplate(resources.ResourceTemplateGenerateRequest{
		TemplateName:  opts.TemplateName,
		BlueprintName: opts.BlueprintName,
		Destination:   opts.Destination,
		Force:         opts.Force,
		DryRun:        opts.DryRun,
		Values:        opts.Values,
	})
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success": true,
			"report":  report,
		})
	}

	if report.DryRun {
		_, _ = fmt.Fprintf(stdout, "[DRY-RUN] Would generate resource template %s at %s\n", report.Template.Name, report.Destination)
	} else {
		_, _ = fmt.Fprintf(stdout, "Generated resource template %s at %s\n", report.Template.Name, report.Destination)
	}
	if strings.TrimSpace(report.BlueprintName) != "" {
		_, _ = fmt.Fprintf(stdout, "Blueprint: %s\n", report.BlueprintName)
	}
	writeResourceTemplateValues(stdout, report.Values)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Files:")
	for _, path := range report.Files {
		_, _ = fmt.Fprintf(stdout, "  - %s\n", path)
	}
	return nil
}

func parseResourceTemplateGenerateArgs(controller *resources.Controller, args []string, stderr io.Writer) (resourceTemplateGenerateOptions, error) {
	opts := resourceTemplateGenerateOptions{Values: map[string]string{}}

	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		opts.TemplateName = args[0]
		args = args[1:]
	}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--from-blueprint":
			if index+1 >= len(args) {
				return resourceTemplateGenerateOptions{}, usageErrorf("resource template generate", "resource template generate --from-blueprint requires a value")
			}
			index++
			opts.BlueprintName = args[index]
		case strings.HasPrefix(arg, "--from-blueprint="):
			opts.BlueprintName = strings.TrimPrefix(arg, "--from-blueprint=")
		}
	}

	var manifest resources.ResourceTemplateManifest
	if opts.TemplateName != "" || opts.BlueprintName != "" {
		info, err := controller.ResolveTemplateGenerationRequest(resources.ResourceTemplateGenerateRequest{
			TemplateName:  opts.TemplateName,
			BlueprintName: opts.BlueprintName,
		})
		if err != nil {
			return resourceTemplateGenerateOptions{}, err
		}
		manifest = info.Manifest
		opts.TemplateName = info.Name
	}

	flagMap := make(map[string]string, len(manifest.RequiredVars)+len(manifest.OptionalVars))
	for key, variable := range manifest.RequiredVars {
		if variable.Flag != "" {
			flagMap[variable.Flag] = key
		}
	}
	for key, variable := range manifest.OptionalVars {
		if variable.Flag != "" {
			flagMap[variable.Flag] = key
		}
	}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--from-blueprint":
			index++
		case strings.HasPrefix(arg, "--from-blueprint="):
			continue
		case arg == "--dest" || arg == "--destination":
			if index+1 >= len(args) {
				return resourceTemplateGenerateOptions{}, usageErrorf("resource template generate", "%s requires a value", arg)
			}
			index++
			opts.Destination = args[index]
		case strings.HasPrefix(arg, "--dest="):
			opts.Destination = strings.TrimPrefix(arg, "--dest=")
		case strings.HasPrefix(arg, "--destination="):
			opts.Destination = strings.TrimPrefix(arg, "--destination=")
		case arg == "--force":
			opts.Force = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--var":
			if index+1 >= len(args) {
				return resourceTemplateGenerateOptions{}, usageErrorf("resource template generate", "resource template generate --var requires KEY=VALUE")
			}
			index++
			key, value, err := parseScenarioTemplateKeyValue(args[index])
			if err != nil {
				return resourceTemplateGenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--var="):
			key, value, err := parseScenarioTemplateKeyValue(strings.TrimPrefix(arg, "--var="))
			if err != nil {
				return resourceTemplateGenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--"):
			flagName, flagValue, consumesNext, err := parseScenarioTemplateFlag(arg, args, index)
			if err != nil {
				return resourceTemplateGenerateOptions{}, err
			}
			if consumesNext {
				index++
			}
			key, ok := flagMap[flagName]
			if !ok {
				_, _ = fmt.Fprintf(stderr, "Warning: unknown flag --%s; use --var KEY=VALUE for arbitrary placeholders\n", flagName)
				continue
			}
			opts.Values[key] = flagValue
		default:
			return resourceTemplateGenerateOptions{}, usageErrorf("resource template generate", "unexpected argument: %s", arg)
		}
	}

	return opts, nil
}

func showResourceTemplateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource template <list|show|validate|generate> [...]")
}

func showResourceTemplateGenerateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource template generate <template> [options]")
	_, _ = fmt.Fprintln(w, "       vrooli resource template generate --from-blueprint <name> [options]")
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --from-blueprint <name>  Seed values from an existing blueprint")
	_, _ = fmt.Fprintln(w, "  --dest <path>            Destination directory (defaults to resources/<name>)")
	_, _ = fmt.Fprintln(w, "  --var KEY=VALUE          Additional placeholder override (repeatable)")
	_, _ = fmt.Fprintln(w, "  --force                  Overwrite destination if it already exists")
	_, _ = fmt.Fprintln(w, "  --dry-run                Print the planned actions without writing files")
}

func writeResourceTemplateVarTable(w io.Writer, title string, vars map[string]resources.ResourceTemplateVar) {
	if len(vars) == 0 {
		return
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s:\n", title)
	for _, key := range keys {
		item := vars[key]
		line := fmt.Sprintf("  - %s (--%s)", key, item.Flag)
		if item.Description != "" {
			line += ": " + item.Description
		}
		if item.Default != "" {
			line += " [default: " + item.Default + "]"
		}
		_, _ = fmt.Fprintln(w, line)
	}
}

func formatResourceTemplateRequiredVars(vars map[string]resources.ResourceTemplateVar) string {
	if len(vars) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s (--%s)", key, vars[key].Flag))
	}
	return strings.Join(parts, ", ")
}

func formatResourceTemplateRequiredFlags(vars map[string]resources.ResourceTemplateVar) string {
	if len(vars) == 0 {
		return ""
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(" --%s <%s>", vars[key].Flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}

func writeResourceTemplateValues(w io.Writer, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w, "Applied variables:")
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "  - %s=%s\n", key, values[key])
	}
}
