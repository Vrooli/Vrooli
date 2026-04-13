package main

import (
	"fmt"
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/resources"
)

var timeNowForResourceGC = func() time.Time {
	return time.Now().UTC()
}

type resourceCommandHandler func(app *App, ctx *commandContext, controller *resources.Controller, args []string) error

var resourceCommandTable = []resourceSubcommandDescriptor{
	{Name: "list", Summary: "List discovered resources", Handler: bindResourceCommand(parseResourceListRequest, runResourceListRequest, renderResourceListResponse)},
	{Name: "status", Summary: "Show resource status", Handler: bindResourceCommand(parseResourceStatusRequest, runResourceStatusRequest, renderResourceStatusResponse)},
	{Name: "install", Summary: "Install a resource", Handler: runResourceInstallCommandWithApp},
	{Name: "start", Summary: "Start a resource", Handler: runResourceStartCommandWithApp},
	{Name: "start-all", Summary: "Start all enabled resources", Handler: runResourceStartAllCommandWithApp},
	{Name: "stop", Summary: "Stop a resource", Handler: runResourceStopCommandWithApp},
	{Name: "stop-all", Summary: "Stop all running resources", Handler: runResourceStopAllCommandWithApp},
	{Name: "enable", Summary: "Enable a resource in configuration", Handler: runResourceEnableCommandWithApp},
	{Name: "disable", Summary: "Disable a resource in configuration", Handler: runResourceDisableCommandWithApp},
	{Name: "info", Summary: "Show resource metadata", Handler: bindResourceCommand(parseResourceInfoRequest, runResourceInfoRequest, renderResourceInfoResponse)},
	{Name: "deprecate", Summary: "Deprecate a resource", Handler: bindResourceCommand(parseResourceDeprecateRequest, runResourceDeprecateRequest, renderResourceDeprecateResponse)},
	{Name: "list-deprecated", Summary: "List deprecated resources", Handler: bindResourceCommand(parseResourceListDeprecatedRequest, runResourceListDeprecatedRequest, renderResourceListDeprecatedResponse)},
	{Name: "archive-to-blueprint", Summary: "Archive a resource into blueprint-only state", Handler: bindResourceCommand(parseResourceArchiveToBlueprintRequest, runResourceArchiveToBlueprintRequest, renderResourceArchiveToBlueprintResponse)},
	{Name: "list-blueprint-archived", Summary: "List blueprint-archived resources", Handler: bindResourceCommand(parseResourceListBlueprintArchivedRequest, runResourceListBlueprintArchivedRequest, renderResourceListBlueprintArchivedResponse)},
	{Name: "restore", Summary: "Restore a deprecated resource", Handler: bindResourceCommand(parseResourceRestoreRequest, runResourceRestoreRequest, renderResourceRestoreResponse)},
	{Name: "restore-blueprint", Summary: "Restore a blueprint-archived resource", Handler: bindResourceCommand(parseResourceRestoreBlueprintRequest, runResourceRestoreBlueprintRequest, renderResourceRestoreBlueprintResponse)},
	{Name: "archive", Summary: "Manage resource archive maintenance", Handler: runResourceArchiveCommandWithApp},
	{Name: "blueprint", Summary: "Inspect resource blueprints", Handler: runResourceBlueprintCommandWithApp},
	{Name: "template", Summary: "Manage resource templates", Handler: runResourceTemplateCommandWithApp},
}

var resourceBlueprintCommandTable = []resourceSubcommandDescriptor{
	{Name: "list", Summary: "List resource blueprints", Handler: bindResourceCommand(parseResourceBlueprintListRequest, runResourceBlueprintListRequest, renderResourceBlueprintListResponse)},
	{Name: "info", Summary: "Show a resource blueprint", Handler: bindResourceCommand(parseResourceBlueprintInfoRequest, runResourceBlueprintInfoRequest, renderResourceBlueprintInfoResponse)},
	{Name: "search", Summary: "Search resource blueprints", Handler: bindResourceCommand(parseResourceBlueprintSearchRequest, runResourceBlueprintSearchRequest, renderResourceBlueprintSearchResponse)},
	{Name: "validate", Summary: "Validate blueprint metadata", Handler: bindResourceCommand(parseResourceBlueprintValidateRequest, runResourceBlueprintValidateRequest, renderResourceBlueprintValidateResponse)},
}

var resourceArchiveCommandTable = []resourceSubcommandDescriptor{
	{Name: "gc", Summary: "Purge expired deprecated-resource archives", Handler: bindResourceCommand(parseResourceArchiveGCRequest, runResourceArchiveGCRequest, renderResourceArchiveGCResponse)},
	{Name: "gc-blueprints", Summary: "Purge expired blueprint-resource archives", Handler: bindResourceCommand(parseResourceArchiveBlueprintGCRequest, runResourceArchiveBlueprintGCRequest, renderResourceArchiveBlueprintGCResponse)},
}

var (
	resourceCommandHandlers          = buildResourceSubcommandMap(resourceCommandTable)
	resourceBlueprintCommandHandlers = buildResourceSubcommandMap(resourceBlueprintCommandTable)
	resourceArchiveCommandHandlers   = buildResourceSubcommandMap(resourceArchiveCommandTable)
)

func runResourceCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	app, ctx := newConfiguredCommandContext("", globals, stdout, stderr)
	if controller != nil {
		ctx.Root = controller.Root
	}
	return runResourceCommandWithApp(app, ctx, controller, args)
}

func runResourceCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		showResourceHelp(ctx.Stdout)
		return nil
	}

	name := normalizeSubcommand(args[0])
	if handler, ok := resourceCommandHandlers[name]; ok {
		return handler(app, ctx, controller, args[1:])
	}
	return runLegacyResourceInvocationWithApp(ctx, controller, args)
}

func runLegacyResourceInvocationWithApp(ctx *commandContext, controller *resources.Controller, args []string) error {
	if controller == nil {
		return usageErrorf("resource", "unknown resource command: %s", args[0])
	}
	return controller.Run(args[0], args[1:], ctx.Stdout, ctx.Stderr)
}

func runResourceBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	app, ctx := newConfiguredCommandContext("", globals, stdout, io.Discard)
	if controller != nil {
		ctx.Root = controller.Root
	}
	return runResourceBlueprintCommandWithApp(app, ctx, controller, args)
}

func runResourceBlueprintCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runResourceSubcommandSet(app, ctx, controller, args, showResourceBlueprintHelp, "resource blueprint", resourceBlueprintCommandHandlers)
}

func runResourceArchiveCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	app, ctx := newConfiguredCommandContext("", globals, stdout, io.Discard)
	if controller != nil {
		ctx.Root = controller.Root
	}
	return runResourceArchiveCommandWithApp(app, ctx, controller, args)
}

func runResourceArchiveCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runResourceSubcommandSet(app, ctx, controller, args, showResourceArchiveHelp, "resource archive", resourceArchiveCommandHandlers)
}

func runResourceListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBoundCommand(controller, globals, args, stdout, io.Discard, boundCommandAction[*resources.Controller, resourceNoArgsRequest, resourceListResponse]{
		parse:  parseResourceListRequest,
		run:    runResourceListRequest,
		render: renderResourceListResponse,
	})
}

func runResourceStatusCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBoundCommand(controller, globals, args, stdout, io.Discard, boundCommandAction[*resources.Controller, resourceStatusRequest, resourceStatusResponse]{
		parse:  parseResourceStatusRequest,
		run:    runResourceStatusRequest,
		render: renderResourceStatusResponse,
	})
}

func runSingleResourceControlCommand(controller *resources.Controller, action string, args []string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource "+action, "resource %s requires exactly one resource name", action)
	}
	return controller.Run(args[0], []string{action}, stdout, stderr)
}

func runResourceInstallCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runSingleResourceControlCommand(controller, "install", args, ctx.Stdout, ctx.Stderr)
}

func runResourceStartCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runSingleResourceControlCommand(controller, "start", args, ctx.Stdout, ctx.Stderr)
}

func runResourceStopCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runSingleResourceControlCommand(controller, "stop", args, ctx.Stdout, ctx.Stderr)
}

func runResourceStartAllCommand(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer) error {
	return runResourceBoundCommand(controller, globals, nil, stdout, stderr, boundCommandAction[*resources.Controller, resourceNoArgsRequest, resourceControlReportResponse]{
		parse:  parseResourceStartAllRequest,
		run:    runResourceStartAllRequest,
		render: renderResourceControlReportResponse,
	})
}

func runResourceStopAllCommand(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer) error {
	return runResourceBoundCommand(controller, globals, nil, stdout, stderr, boundCommandAction[*resources.Controller, resourceNoArgsRequest, resourceControlReportResponse]{
		parse:  parseResourceStopAllRequest,
		run:    runResourceStopAllRequest,
		render: renderResourceControlReportResponse,
	})
}

func runResourceStartAllCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return bindResourceCommand(parseResourceStartAllRequest, runResourceStartAllRequest, renderResourceControlReportResponse)(app, ctx, controller, args)
}

func runResourceStopAllCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return bindResourceCommand(parseResourceStopAllRequest, runResourceStopAllRequest, renderResourceControlReportResponse)(app, ctx, controller, args)
}

func runResourceToggleCommand(controller *resources.Controller, enabled bool, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		action := "enable"
		if !enabled {
			action = "disable"
		}
		return usageErrorf("resource "+action, "resource %s requires exactly one resource name", action)
	}
	if err := controller.SetEnabled(args[0], enabled); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Updated %s: enabled=%t\n", args[0], enabled)
	return nil
}

func runResourceEnableCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runResourceToggleCommand(controller, true, args, ctx.Stdout)
}

func runResourceDisableCommandWithApp(app *App, ctx *commandContext, controller *resources.Controller, args []string) error {
	return runResourceToggleCommand(controller, false, args, ctx.Stdout)
}

func runResourceInfoCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBoundCommand(controller, globals, args, stdout, io.Discard, boundCommandAction[*resources.Controller, resourceNameRequest, resourceStatusResponse]{
		parse:  parseResourceInfoRequest,
		run:    runResourceInfoRequest,
		render: renderResourceInfoResponse,
	})
}

func runResourceDeprecateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBoundCommand(controller, globals, args, stdout, io.Discard, boundCommandAction[*resources.Controller, resourceNameRequest, resources.DeprecationReport]{
		parse:  parseResourceDeprecateRequest,
		run:    runResourceDeprecateRequest,
		render: renderResourceDeprecateResponse,
	})
}

func runResourceListDeprecatedCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBoundCommand(controller, globals, args, stdout, io.Discard, boundCommandAction[*resources.Controller, resourceNoArgsRequest, []resources.DeprecatedResource]{
		parse:  parseResourceListDeprecatedRequest,
		run:    runResourceListDeprecatedRequest,
		render: renderResourceListDeprecatedResponse,
	})
}

func runResourceRestoreCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBoundCommand(controller, globals, args, stdout, io.Discard, boundCommandAction[*resources.Controller, resourceNameRequest, resources.RestoreReport]{
		parse:  parseResourceRestoreRequest,
		run:    runResourceRestoreRequest,
		render: renderResourceRestoreResponse,
	})
}

func runResourceArchiveToBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceCommand(controller, globals, append([]string{"archive-to-blueprint"}, args...), stdout, io.Discard)
}

func runResourceListBlueprintArchivedCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceCommand(controller, globals, append([]string{"list-blueprint-archived"}, args...), stdout, io.Discard)
}

func runResourceRestoreBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceCommand(controller, globals, append([]string{"restore-blueprint"}, args...), stdout, io.Discard)
}

func runResourceArchiveGCCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceArchiveCommand(controller, globals, append([]string{"gc"}, args...), stdout)
}

func runResourceArchiveBlueprintGCCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceArchiveCommand(controller, globals, append([]string{"gc-blueprints"}, args...), stdout)
}

func runResourceBlueprintListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBlueprintCommand(controller, globals, append([]string{"list"}, args...), stdout)
}

func runResourceBlueprintInfoCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBlueprintCommand(controller, globals, append([]string{"info"}, args...), stdout)
}

func runResourceBlueprintSearchCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBlueprintCommand(controller, globals, append([]string{"search"}, args...), stdout)
}

func runResourceBlueprintValidateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return runResourceBlueprintCommand(controller, globals, append([]string{"validate"}, args...), stdout)
}

func showResourceHelp(w io.Writer) {
	renderResourceSubcommandHelp(w, "", "vrooli resource <subcommand> [options]", "Resource Management", resourceCommandTable)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Resource names may still be invoked directly via `vrooli resource <name> <command> [options]`.")
}

func showResourceBlueprintHelp(w io.Writer) {
	renderResourceSubcommandHelp(w, "", "vrooli resource blueprint <subcommand> [options]", "Resource Blueprints", resourceBlueprintCommandTable)
}

func showResourceArchiveHelp(w io.Writer) {
	renderResourceSubcommandHelp(w, "", "vrooli resource archive <subcommand> [options]", "Resource Archive", resourceArchiveCommandTable)
}

func writeResourceCommandHelp(w io.Writer, descriptors []resourceSubcommandDescriptor) {
	for _, descriptor := range descriptors {
		_, _ = fmt.Fprintf(w, "  %-22s %s\n", descriptor.Name, descriptor.Summary)
	}
}
