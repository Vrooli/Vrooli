package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

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
		return executeResourceTemplateCommand(controller, globals, args[1:], stdout, stderr, resourceTemplateCommandAction[resourceTemplateNoArgsRequest, []resources.ResourceTemplateInfo]{
			parse:  parseResourceTemplateListRequest,
			run:    runResourceTemplateListRequest,
			render: renderResourceTemplateListResponse,
		})
	case "show":
		return executeResourceTemplateCommand(controller, globals, args[1:], stdout, stderr, resourceTemplateCommandAction[resourceTemplateNameRequest, resources.ResourceTemplateInfo]{
			parse:  parseResourceTemplateShowRequest,
			run:    runResourceTemplateShowRequest,
			render: renderResourceTemplateShowResponse,
		})
	case "validate":
		return executeResourceTemplateCommand(controller, globals, args[1:], stdout, stderr, resourceTemplateCommandAction[resourceTemplateNoArgsRequest, resources.ResourceTemplateValidationReport]{
			parse:  parseResourceTemplateValidateRequest,
			run:    runResourceTemplateValidateRequest,
			render: renderResourceTemplateValidateResponse,
		})
	case "generate":
		return executeResourceTemplateCommand(controller, globals, args[1:], stdout, stderr, resourceTemplateCommandAction[resourceTemplateGenerateOptions, resources.ResourceTemplateGenerateReport]{
			parse: func(args []string) (resourceTemplateGenerateOptions, error) {
				return parseResourceTemplateGenerateRequest(controller, args, stderr)
			},
			run:    runResourceTemplateGenerateRequest,
			render: renderResourceTemplateGenerateResponse,
		})
	default:
		return usageErrorf("resource template", "unknown resource template command: %s", args[0])
	}
}

func runResourceTemplateListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceTemplateCommand(controller, globals, args, stdout, io.Discard, resourceTemplateCommandAction[resourceTemplateNoArgsRequest, []resources.ResourceTemplateInfo]{
		parse:  parseResourceTemplateListRequest,
		run:    runResourceTemplateListRequest,
		render: renderResourceTemplateListResponse,
	})
}

func runResourceTemplateShowCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceTemplateCommand(controller, globals, args, stdout, io.Discard, resourceTemplateCommandAction[resourceTemplateNameRequest, resources.ResourceTemplateInfo]{
		parse:  parseResourceTemplateShowRequest,
		run:    runResourceTemplateShowRequest,
		render: renderResourceTemplateShowResponse,
	})
}

func runResourceTemplateValidateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceTemplateCommand(controller, globals, args, stdout, io.Discard, resourceTemplateCommandAction[resourceTemplateNoArgsRequest, resources.ResourceTemplateValidationReport]{
		parse:  parseResourceTemplateValidateRequest,
		run:    runResourceTemplateValidateRequest,
		render: renderResourceTemplateValidateResponse,
	})
}

func runResourceTemplateGenerateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return executeResourceTemplateCommand(controller, globals, args, stdout, stderr, resourceTemplateCommandAction[resourceTemplateGenerateOptions, resources.ResourceTemplateGenerateReport]{
		parse: func(args []string) (resourceTemplateGenerateOptions, error) {
			return parseResourceTemplateGenerateRequest(controller, args, stderr)
		},
		run:    runResourceTemplateGenerateRequest,
		render: renderResourceTemplateGenerateResponse,
	})
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
