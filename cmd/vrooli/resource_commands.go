package main

import (
	"fmt"
	"io"
	"time"

	resourceapp "github.com/vrooli/vrooli/internal/app/resource"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/resources"
)

var timeNowForResourceGC = func() time.Time {
	return time.Now().UTC()
}

var resourceCommandTable = buildResourceCommandTable()

var resourceBlueprintCommandTable = buildResourceBlueprintCommandTable()

var resourceArchiveCommandTable = buildResourceArchiveCommandTable()

var (
	resourceCommandHandlers          = commandtree.BuildHandlerMap(resourceCommandTable)
	resourceBlueprintCommandHandlers = commandtree.BuildHandlerMap(resourceBlueprintCommandTable)
	resourceArchiveCommandHandlers   = commandtree.BuildHandlerMap(resourceArchiveCommandTable)
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

	name := commandtree.NormalizeName(args[0])
	if handler, ok := resourceCommandHandlers[name]; ok {
		return handler(app, ctx, controller, args[1:])
	}
	return usageErrorf("resource", "unknown resource command: %s", args[0])
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
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resourceListResponse]{
		parse:  parseResourceListRequest,
		run:    runResourceListRequest,
		render: renderResourceListResponse,
	})
}

func runResourceStatusCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceStatusRequest, resourceStatusResponse]{
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
	return executeResourceCommand(controller, globals, nil, stdout, stderr, resourceCommandAction[resourceNoArgsRequest, resourceapp.ControlReportResponse]{
		parse:  parseResourceStartAllRequest,
		run:    runResourceStartAllRequest,
		render: renderResourceControlReportResponse,
	})
}

func runResourceStopAllCommand(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer) error {
	return executeResourceCommand(controller, globals, nil, stdout, stderr, resourceCommandAction[resourceNoArgsRequest, resourceapp.ControlReportResponse]{
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
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNameRequest, resourceStatusResponse]{
		parse:  parseResourceInfoRequest,
		run:    runResourceInfoRequest,
		render: renderResourceInfoResponse,
	})
}

func runResourceDeprecateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNameRequest, resources.DeprecationReport]{
		parse:  parseResourceDeprecateRequest,
		run:    runResourceDeprecateRequest,
		render: renderResourceDeprecateResponse,
	})
}

func runResourceListDeprecatedCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, []resources.DeprecatedResource]{
		parse:  parseResourceListDeprecatedRequest,
		run:    runResourceListDeprecatedRequest,
		render: renderResourceListDeprecatedResponse,
	})
}

func runResourceRestoreCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNameRequest, resources.RestoreReport]{
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
	resourcecli.RenderCommandHelp(w, "", "vrooli resource <subcommand> [options]", "Resource Management", resourceCommandTable)
}

func showResourceBlueprintHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource blueprint <subcommand> [options]", "Resource Blueprints", resourceBlueprintCommandTable)
}

func showResourceArchiveHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource archive <subcommand> [options]", "Resource Archive", resourceArchiveCommandTable)
}

func buildResourceCommandTable() []commandtree.Spec[resourceCommandHandler] {
	handlerMap := map[resourcecli.CommandID]resourceCommandHandler{
		resourcecli.CommandList:                  bindResourceCommand(parseResourceListRequest, runResourceListRequest, renderResourceListResponse),
		resourcecli.CommandStatus:                bindResourceCommand(parseResourceStatusRequest, runResourceStatusRequest, renderResourceStatusResponse),
		resourcecli.CommandValidate:              bindResourceCommand(parseResourceValidateRequest, runResourceValidateRequest, renderResourceValidateResponse),
		resourcecli.CommandInstall:               runResourceInstallCommandWithApp,
		resourcecli.CommandStart:                 runResourceStartCommandWithApp,
		resourcecli.CommandStartAll:              bindResourceCommand(parseResourceStartAllRequest, runResourceStartAllRequest, renderResourceControlReportResponse),
		resourcecli.CommandStop:                  runResourceStopCommandWithApp,
		resourcecli.CommandStopAll:               bindResourceCommand(parseResourceStopAllRequest, runResourceStopAllRequest, renderResourceControlReportResponse),
		resourcecli.CommandEnable:                runResourceEnableCommandWithApp,
		resourcecli.CommandDisable:               runResourceDisableCommandWithApp,
		resourcecli.CommandInfo:                  bindResourceCommand(parseResourceInfoRequest, runResourceInfoRequest, renderResourceInfoResponse),
		resourcecli.CommandDeprecate:             bindResourceCommand(parseResourceDeprecateRequest, runResourceDeprecateRequest, renderResourceDeprecateResponse),
		resourcecli.CommandListDeprecated:        bindResourceCommand(parseResourceListDeprecatedRequest, runResourceListDeprecatedRequest, renderResourceListDeprecatedResponse),
		resourcecli.CommandArchiveToBlueprint:    bindResourceCommand(parseResourceArchiveToBlueprintRequest, runResourceArchiveToBlueprintRequest, renderResourceArchiveToBlueprintResponse),
		resourcecli.CommandListBlueprintArchived: bindResourceCommand(parseResourceListBlueprintArchivedRequest, runResourceListBlueprintArchivedRequest, renderResourceListBlueprintArchivedResponse),
		resourcecli.CommandRestore:               bindResourceCommand(parseResourceRestoreRequest, runResourceRestoreRequest, renderResourceRestoreResponse),
		resourcecli.CommandRestoreBlueprint:      bindResourceCommand(parseResourceRestoreBlueprintRequest, runResourceRestoreBlueprintRequest, renderResourceRestoreBlueprintResponse),
		resourcecli.CommandArchive:               runResourceArchiveCommandWithApp,
		resourcecli.CommandBlueprint:             runResourceBlueprintCommandWithApp,
		resourcecli.CommandTemplate:              runResourceTemplateCommandWithApp,
	}

	source := resourcecli.CommandSpecs()
	specs := make([]commandtree.Spec[resourceCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[resourceCommandHandler]{
			Name:        spec.Name,
			Aliases:     append([]string(nil), spec.Aliases...),
			Group:       spec.Group,
			Summary:     spec.Summary,
			Hidden:      spec.Hidden,
			Suggestable: spec.Suggestable,
			RootPolicy:  spec.RootPolicy,
			Help:        spec.Help,
			Handler:     handler,
		})
	}
	return specs
}

func buildResourceBlueprintCommandTable() []commandtree.Spec[resourceCommandHandler] {
	handlerMap := map[resourcecli.BlueprintCommandID]resourceCommandHandler{
		resourcecli.BlueprintCommandList:     bindResourceCommand(parseResourceBlueprintListRequest, runResourceBlueprintListRequest, renderResourceBlueprintListResponse),
		resourcecli.BlueprintCommandInfo:     bindResourceCommand(parseResourceBlueprintInfoRequest, runResourceBlueprintInfoRequest, renderResourceBlueprintInfoResponse),
		resourcecli.BlueprintCommandSearch:   bindResourceCommand(parseResourceBlueprintSearchRequest, runResourceBlueprintSearchRequest, renderResourceBlueprintSearchResponse),
		resourcecli.BlueprintCommandValidate: bindResourceCommand(parseResourceBlueprintValidateRequest, runResourceBlueprintValidateRequest, renderResourceBlueprintValidateResponse),
	}
	source := resourcecli.BlueprintCommandSpecs()
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

func buildResourceArchiveCommandTable() []commandtree.Spec[resourceCommandHandler] {
	handlerMap := map[resourcecli.ArchiveCommandID]resourceCommandHandler{
		resourcecli.ArchiveCommandGC:           bindResourceCommand(parseResourceArchiveGCRequest, runResourceArchiveGCRequest, renderResourceArchiveGCResponse),
		resourcecli.ArchiveCommandGCBlueprints: bindResourceCommand(parseResourceArchiveBlueprintGCRequest, runResourceArchiveBlueprintGCRequest, renderResourceArchiveBlueprintGCResponse),
	}
	source := resourcecli.ArchiveCommandSpecs()
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
