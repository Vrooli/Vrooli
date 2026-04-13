package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/resources"
)

var resourceTemplateCommandTable = buildResourceTemplateCommandTable()

var resourceTemplateCommandHandlers = commandtree.BuildHandlerMap(resourceTemplateCommandTable)

func runResourceTemplateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	app, ctx := newConfiguredCommandContext("", globals, stdout, stderr)
	if controller != nil {
		ctx.Root = controller.Root
	}
	return runResourceTemplateCommandWithApp(app, ctx, controller, args)
}

func runResourceTemplateCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runResourceSubcommandSet(app, ctx, controller, args, showResourceTemplateHelp, "resource template", resourceTemplateCommandHandlers)
}

func runResourceTemplateGenerateCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return executeResourceCommandWithApp(app, ctx, controller, args, resourceCommandAction[resourcecli.TemplateGenerateOptions, resources.ResourceTemplateGenerateReport]{
		parse: func(globals globalOptions, args []string) (resourcecli.TemplateGenerateOptions, error) {
			return parseResourceTemplateGenerateRequest(controller, globals, args, ctx.Stderr)
		},
		run:    runResourceTemplateGenerateRequest,
		render: renderResourceTemplateGenerateResponse,
	})
}

func showResourceTemplateHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource template <subcommand> [options]", "Resource Templates", resourceTemplateCommandTable)
}

func showResourceTemplateGenerateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, resourcecli.RenderTemplateGenerateHelpText())
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

func buildResourceTemplateCommandTable() []commandtree.Spec[resourceCommandHandler] {
	handlerMap := map[resourcecli.TemplateCommandID]resourceCommandHandler{
		resourcecli.TemplateCommandList:     bindResourceCommand(parseResourceTemplateListRequest, runResourceTemplateListRequest, renderResourceTemplateListResponse),
		resourcecli.TemplateCommandShow:     bindResourceCommand(parseResourceTemplateShowRequest, runResourceTemplateShowRequest, renderResourceTemplateShowResponse),
		resourcecli.TemplateCommandValidate: bindResourceCommand(parseResourceTemplateValidateRequest, runResourceTemplateValidateRequest, renderResourceTemplateValidateResponse),
		resourcecli.TemplateCommandGenerate: runResourceTemplateGenerateCommandWithApp,
	}
	source := resourcecli.TemplateCommandSpecs()
	specs := make([]commandtree.Spec[resourceCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[resourceCommandHandler]{Name: spec.Name, Summary: spec.Summary, Handler: handler})
	}
	return specs
}
