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

func runResourceCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showResourceHelp(stdout)
		return nil
	}

	switch normalizeResourceSubcommand(args[0]) {
	case "blueprint":
		return runResourceBlueprintCommand(controller, globals, args[1:], stdout)
	case "template":
		return runResourceTemplateCommand(controller, globals, args[1:], stdout, stderr)
	case "deprecate":
		return runResourceDeprecateCommand(controller, globals, args[1:], stdout)
	case "list-deprecated":
		return runResourceListDeprecatedCommand(controller, globals, args[1:], stdout)
	case "archive-to-blueprint":
		return runResourceArchiveToBlueprintCommand(controller, globals, args[1:], stdout)
	case "list-blueprint-archived":
		return runResourceListBlueprintArchivedCommand(controller, globals, args[1:], stdout)
	case "restore":
		return runResourceRestoreCommand(controller, globals, args[1:], stdout)
	case "restore-blueprint":
		return runResourceRestoreBlueprintCommand(controller, globals, args[1:], stdout)
	case "archive":
		return runResourceArchiveCommand(controller, globals, args[1:], stdout)
	case "list":
		return runResourceListCommand(controller, globals, args[1:], stdout)
	case "status":
		return runResourceStatusCommand(controller, globals, args[1:], stdout)
	case "install":
		return runSingleResourceControlCommand(controller, "install", args[1:], stdout, stderr)
	case "start":
		return runSingleResourceControlCommand(controller, "start", args[1:], stdout, stderr)
	case "stop":
		return runSingleResourceControlCommand(controller, "stop", args[1:], stdout, stderr)
	case "start-all":
		return runResourceStartAllCommand(controller, globals, stdout, stderr)
	case "stop-all":
		return runResourceStopAllCommand(controller, globals, stdout, stderr)
	case "enable":
		return runResourceToggleCommand(controller, true, args[1:], stdout)
	case "disable":
		return runResourceToggleCommand(controller, false, args[1:], stdout)
	case "info":
		return runResourceInfoCommand(controller, globals, args[1:], stdout)
	default:
		return controller.Run(args[0], args[1:], stdout, stderr)
	}
}

func runResourceBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showResourceBlueprintHelp(stdout)
		return nil
	}

	switch normalizeResourceSubcommand(args[0]) {
	case "list":
		return executeResourceCommand(controller, globals, args[1:], stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resourceBlueprintsResponse]{
			parse:  parseResourceBlueprintListRequest,
			run:    runResourceBlueprintListRequest,
			render: renderResourceBlueprintListResponse,
		})
	case "info":
		return executeResourceCommand(controller, globals, args[1:], stdout, io.Discard, resourceCommandAction[resourceNameRequest, resourceBlueprintResponse]{
			parse:  parseResourceBlueprintInfoRequest,
			run:    runResourceBlueprintInfoRequest,
			render: renderResourceBlueprintInfoResponse,
		})
	case "search":
		return executeResourceCommand(controller, globals, args[1:], stdout, io.Discard, resourceCommandAction[resourceBlueprintSearchRequest, resourceBlueprintSearchResponse]{
			parse:  parseResourceBlueprintSearchRequest,
			run:    runResourceBlueprintSearchRequest,
			render: renderResourceBlueprintSearchResponse,
		})
	case "validate":
		return executeResourceCommand(controller, globals, args[1:], stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resources.BlueprintValidationReport]{
			parse:  parseResourceBlueprintValidateRequest,
			run:    runResourceBlueprintValidateRequest,
			render: renderResourceBlueprintValidateResponse,
		})
	default:
		return usageErrorf("resource blueprint", "unknown resource blueprint command: %s", args[0])
	}
}

func runResourceArchiveCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showResourceArchiveHelp(stdout)
		return nil
	}
	switch normalizeResourceSubcommand(args[0]) {
	case "gc":
		return executeResourceCommand(controller, globals, args[1:], stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resources.ArchiveGCReport]{
			parse:  parseResourceArchiveGCRequest,
			run:    runResourceArchiveGCRequest,
			render: renderResourceArchiveGCResponse,
		})
	case "gc-blueprints":
		return executeResourceCommand(controller, globals, args[1:], stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resources.ArchiveGCReport]{
			parse:  parseResourceArchiveBlueprintGCRequest,
			run:    runResourceArchiveBlueprintGCRequest,
			render: renderResourceArchiveBlueprintGCResponse,
		})
	default:
		return usageErrorf("resource archive", "unknown resource archive command: %s", args[0])
	}
}

func runResourceListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resourceListResponse]{
		parse:  parseResourceListRequest,
		run:    runResourceListRequest,
		render: renderResourceListResponse,
	})
}

func runResourceStatusCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceStatusRequest, any]{
		parse:  parseResourceStatusRequest,
		run:    runResourceStatusRequest,
		render: renderResourceStatusResponse,
	})
}

func runSingleResourceControlCommand(controller *resources.Controller, action string, args []string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource "+action, "resource %s requires exactly one resource name", action)
	}
	if err := controller.Run(args[0], []string{action}, stdout, stderr); err != nil {
		return err
	}
	return nil
}

func runResourceStartAllCommand(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer) error {
	return executeResourceCommand(controller, globals, nil, stdout, stderr, resourceCommandAction[resourceNoArgsRequest, any]{
		parse:  parseResourceStartAllRequest,
		run:    runResourceStartAllRequest,
		render: renderResourceControlReportResponse,
	})
}

func runResourceStopAllCommand(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer) error {
	return executeResourceCommand(controller, globals, nil, stdout, stderr, resourceCommandAction[resourceNoArgsRequest, any]{
		parse:  parseResourceStopAllRequest,
		run:    runResourceStopAllRequest,
		render: renderResourceControlReportResponse,
	})
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
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNameRequest, resources.BlueprintArchiveReport]{
		parse:  parseResourceArchiveToBlueprintRequest,
		run:    runResourceArchiveToBlueprintRequest,
		render: renderResourceArchiveToBlueprintResponse,
	})
}

func runResourceListBlueprintArchivedCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, []resources.BlueprintArchivedResource]{
		parse:  parseResourceListBlueprintArchivedRequest,
		run:    runResourceListBlueprintArchivedRequest,
		render: renderResourceListBlueprintArchivedResponse,
	})
}

func runResourceRestoreBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNameRequest, resources.BlueprintRestoreReport]{
		parse:  parseResourceRestoreBlueprintRequest,
		run:    runResourceRestoreBlueprintRequest,
		render: renderResourceRestoreBlueprintResponse,
	})
}

func runResourceArchiveGCCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resources.ArchiveGCReport]{
		parse:  parseResourceArchiveGCRequest,
		run:    runResourceArchiveGCRequest,
		render: renderResourceArchiveGCResponse,
	})
}

func runResourceArchiveBlueprintGCCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resources.ArchiveGCReport]{
		parse:  parseResourceArchiveBlueprintGCRequest,
		run:    runResourceArchiveBlueprintGCRequest,
		render: renderResourceArchiveBlueprintGCResponse,
	})
}

func runResourceBlueprintListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resourceBlueprintsResponse]{
		parse:  parseResourceBlueprintListRequest,
		run:    runResourceBlueprintListRequest,
		render: renderResourceBlueprintListResponse,
	})
}

func runResourceBlueprintInfoCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNameRequest, resourceBlueprintResponse]{
		parse:  parseResourceBlueprintInfoRequest,
		run:    runResourceBlueprintInfoRequest,
		render: renderResourceBlueprintInfoResponse,
	})
}

func runResourceBlueprintSearchCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceBlueprintSearchRequest, resourceBlueprintSearchResponse]{
		parse:  parseResourceBlueprintSearchRequest,
		run:    runResourceBlueprintSearchRequest,
		render: renderResourceBlueprintSearchResponse,
	})
}

func runResourceBlueprintValidateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	return executeResourceCommand(controller, globals, args, stdout, io.Discard, resourceCommandAction[resourceNoArgsRequest, resources.BlueprintValidationReport]{
		parse:  parseResourceBlueprintValidateRequest,
		run:    runResourceBlueprintValidateRequest,
		render: renderResourceBlueprintValidateResponse,
	})
}

func showResourceHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource <list|status|install|start|start-all|stop|stop-all|enable|disable|info|deprecate|list-deprecated|archive-to-blueprint|list-blueprint-archived|restore|restore-blueprint> [...]")
	_, _ = fmt.Fprintln(w, "       vrooli resource archive <gc|gc-blueprints> [...]")
	_, _ = fmt.Fprintln(w, "       vrooli resource blueprint <list|info|search|validate> [...]")
	_, _ = fmt.Fprintln(w, "       vrooli resource template <list|show|validate|generate> [...]")
	_, _ = fmt.Fprintln(w, "       vrooli resource <name> <command> [options]")
}

func showResourceBlueprintHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource blueprint <list|info|search|validate> [...]")
}

func showResourceArchiveHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource archive <gc|gc-blueprints> [...]")
}
