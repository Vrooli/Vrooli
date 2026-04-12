package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
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
		return runResourceBlueprintListCommand(controller, globals, args[1:], stdout)
	case "info":
		return runResourceBlueprintInfoCommand(controller, globals, args[1:], stdout)
	case "search":
		return runResourceBlueprintSearchCommand(controller, globals, args[1:], stdout)
	case "validate":
		return runResourceBlueprintValidateCommand(controller, globals, args[1:], stdout)
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
		return runResourceArchiveGCCommand(controller, globals, args[1:], stdout)
	case "gc-blueprints":
		return runResourceArchiveBlueprintGCCommand(controller, globals, args[1:], stdout)
	default:
		return usageErrorf("resource archive", "unknown resource archive command: %s", args[0])
	}
}

func runResourceListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource list", "resource list does not accept positional arguments")
	}
	items, err := controller.Discover()
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(globals)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return writeSuccessData(stdout, "resources", items)
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		decision := ""
		if item.ControlMode == "legacy-adapter" {
			decision = item.LegacyAdapter.FinalDisposition
		}
		rows = append(rows, []string{
			item.Name,
			boolLabel(item.Enabled),
			item.ControlMode,
			item.Driver,
			item.PortabilityTier,
			decision,
			boolLabel(item.Registered),
		})
	}
	return cliout.RenderTable(stdout, []string{"Name", "Enabled", "Control", "Driver", "Portability", "Decision", "Registered"}, rows)
}

func runResourceStatusCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	fast := true
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--fast":
			fast = true
		case "--no-fast":
			fast = false
		default:
			filtered = append(filtered, arg)
		}
	}

	format, err := parseOutputFormat(globals)
	if err != nil {
		return err
	}

	if len(filtered) == 0 {
		items, err := controller.ListStatuses(fast, false)
		if err != nil {
			return err
		}
		if format == cliout.FormatJSON {
			return writeSuccessData(stdout, "resources", items)
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			healthy := "n/a"
			if item.Healthy != nil {
				if *item.Healthy {
					healthy = "healthy"
				} else {
					healthy = "unhealthy"
				}
			}
			rows = append(rows, []string{
				item.Resource.Name,
				boolLabel(item.Resource.Enabled),
				item.Resource.ControlMode,
				boolLabel(item.Running),
				healthy,
				item.Message,
			})
		}
		return cliout.RenderTable(stdout, []string{"Name", "Enabled", "Control", "Running", "Health", "Status"}, rows)
	}

	if len(filtered) != 1 {
		return usageErrorf("resource status", "resource status accepts at most one resource name")
	}
	item, err := controller.Status(filtered[0], fast)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":   true,
			"name":      item.Resource.Name,
			"installed": item.Installed,
			"running":   item.Running,
			"healthy":   item.Healthy,
			"status":    item.Message,
			"resource":  item,
		})
	}
	rows := [][]string{
		{"Name", item.Resource.Name},
		{"Enabled", boolLabel(item.Resource.Enabled)},
		{"Control", item.Resource.ControlMode},
		{"Driver", item.Resource.Driver},
		{"Portability", item.Resource.PortabilityTier},
		{"Installed", boolLabel(item.Installed)},
		{"Running", boolLabel(item.Running)},
	}
	if item.Healthy != nil {
		rows = append(rows, []string{"Healthy", boolLabel(*item.Healthy)})
	}
	if item.StatusCode != "" {
		rows = append(rows, []string{"Status Code", item.StatusCode})
	}
	rows = append(rows, []string{"Status", item.Message})
	if item.Resource.ControlMode == "legacy-adapter" {
		rows = append(rows,
			[]string{"Adapter Owner", item.Resource.LegacyAdapter.Owner},
			[]string{"Decision Deadline", item.Resource.LegacyAdapter.DecisionDeadline},
			[]string{"Final Disposition", item.Resource.LegacyAdapter.FinalDisposition},
			[]string{"Legacy CLI", item.Resource.LegacyAdapter.LegacyCLIPath},
		)
		if item.Resource.LegacyAdapter.Notes != "" {
			rows = append(rows, []string{"Adapter Notes", item.Resource.LegacyAdapter.Notes})
		}
	}
	if item.ProbeError != "" {
		rows = append(rows, []string{"Probe Error", item.ProbeError})
	}
	return cliout.RenderTable(stdout, []string{"Field", "Value"}, rows)
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
	report, err := controller.StartAll(stdout, stderr)
	if err != nil {
		return err
	}
	if globals.json {
		return cliout.WriteJSON(stdout, map[string]any{
			"success": true,
			"started": report.Started,
			"failed":  report.Failed,
			"message": report.Message,
		})
	}
	for _, item := range report.Started {
		_, _ = fmt.Fprintf(stdout, "Started %s\n", item.Name)
	}
	for _, item := range report.Failed {
		_, _ = fmt.Fprintf(stdout, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
}

func runResourceStopAllCommand(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer) error {
	report, err := controller.StopAll(stdout, stderr)
	if err != nil {
		return err
	}
	if globals.json {
		return cliout.WriteJSON(stdout, map[string]any{
			"success": true,
			"stopped": report.Stopped,
			"failed":  report.Failed,
			"message": report.Message,
		})
	}
	for _, item := range report.Stopped {
		_, _ = fmt.Fprintf(stdout, "Stopped %s\n", item.Name)
	}
	for _, item := range report.Failed {
		_, _ = fmt.Fprintf(stdout, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
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
	if len(args) != 1 {
		return usageErrorf("resource info", "resource info requires exactly one resource name")
	}
	item, err := controller.Status(args[0], true)
	if err != nil {
		return err
	}
	return cliout.WriteJSON(stdout, map[string]any{
		"success":  true,
		"resource": item,
	})
}

func runResourceDeprecateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource deprecate", "resource deprecate requires exactly one resource name")
	}
	report, err := controller.DeprecateResource(args[0])
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
	_, _ = fmt.Fprintf(stdout, "Deprecated %s\n", report.Resource.Name)
	if report.ArchiveDir != "" {
		_, _ = fmt.Fprintf(stdout, "Archive: %s\n", report.ArchiveDir)
	}
	return nil
}

func runResourceListDeprecatedCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource list-deprecated", "resource list-deprecated does not accept positional arguments")
	}
	items, err := controller.ListDeprecatedResources()
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
			"resources": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "deprecated"
		if strings.TrimSpace(item.PurgedAt) != "" {
			state = "purged"
		}
		rows = append(rows, []string{
			item.Name,
			state,
			item.DeprecatedAt,
			item.PurgeAfter,
			item.Replacement,
		})
	}
	return cliout.RenderTable(stdout, []string{"Name", "State", "Deprecated", "Purge After", "Replacement"}, rows)
}

func runResourceRestoreCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource restore", "resource restore requires exactly one resource name")
	}
	report, err := controller.RestoreDeprecatedResource(args[0])
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
	_, _ = fmt.Fprintf(stdout, "Restored %s to %s\n", report.Resource.Name, report.RestoredPath)
	return nil
}

func runResourceArchiveToBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource archive-to-blueprint", "resource archive-to-blueprint requires exactly one resource name")
	}
	report, err := controller.ArchiveResourceToBlueprint(args[0])
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
	_, _ = fmt.Fprintf(stdout, "Archived %s to blueprint-only state\n", report.Resource.Name)
	if report.ArchiveDir != "" {
		_, _ = fmt.Fprintf(stdout, "Archive: %s\n", report.ArchiveDir)
	}
	return nil
}

func runResourceListBlueprintArchivedCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource list-blueprint-archived", "resource list-blueprint-archived does not accept positional arguments")
	}
	items, err := controller.ListBlueprintArchivedResources()
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
			"resources": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "blueprint-archived"
		if strings.TrimSpace(item.PurgedAt) != "" {
			state = "purged"
		}
		rows = append(rows, []string{
			item.Name,
			state,
			item.ArchivedAt,
			item.PurgeAfter,
			item.BlueprintName,
		})
	}
	return cliout.RenderTable(stdout, []string{"Name", "State", "Archived", "Purge After", "Blueprint"}, rows)
}

func runResourceRestoreBlueprintCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource restore-blueprint", "resource restore-blueprint requires exactly one resource name")
	}
	report, err := controller.RestoreBlueprintArchivedResource(args[0])
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
	_, _ = fmt.Fprintf(stdout, "Restored blueprint-archived %s to %s\n", report.Resource.Name, report.RestoredPath)
	return nil
}

func runResourceArchiveGCCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource archive", "resource archive gc does not accept positional arguments")
	}
	report, err := controller.GarbageCollectDeprecatedArchives(timeNowForResourceGC())
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
	_, _ = fmt.Fprintf(stdout, "Purged %d deprecated resource archives\n", len(report.Removed))
	return nil
}

func runResourceArchiveBlueprintGCCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource archive", "resource archive gc-blueprints does not accept positional arguments")
	}
	report, err := controller.GarbageCollectBlueprintArchives(timeNowForResourceGC())
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
	_, _ = fmt.Fprintf(stdout, "Purged %d blueprint resource archives\n", len(report.Removed))
	return nil
}

func runResourceBlueprintListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource blueprint", "resource blueprint list does not accept positional arguments")
	}
	items, err := controller.ListBlueprints()
	if err != nil {
		return err
	}
	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":    true,
			"blueprints": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			item.Category,
			item.Status,
			item.SuggestedTemplate,
			item.LastReviewed,
		})
	}
	return cliout.RenderTable(stdout, []string{"Name", "Category", "Status", "Template", "Reviewed"}, rows)
}

func runResourceBlueprintInfoCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource blueprint", "resource blueprint info requires exactly one blueprint name")
	}
	item, err := controller.Blueprint(args[0])
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
			"blueprint": item,
		})
	}
	rows := [][]string{
		{"Name", item.Name},
		{"Display Name", item.DisplayName},
		{"Category", item.Category},
		{"Status", item.Status},
		{"Integration Kind", item.IntegrationKind},
		{"Template", item.SuggestedTemplate},
		{"Reviewed", item.LastReviewed},
		{"Summary", item.Summary},
		{"Why It Matters", item.WhyItMatters},
	}
	return cliout.RenderTable(stdout, []string{"Field", "Value"}, rows)
}

func runResourceBlueprintSearchCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return usageErrorf("resource blueprint", "resource blueprint search requires exactly one query")
	}
	items, err := controller.SearchBlueprints(args[0])
	if err != nil {
		return err
	}
	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{
			"success":    true,
			"query":      args[0],
			"blueprints": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			item.Category,
			item.Status,
			item.Summary,
		})
	}
	return cliout.RenderTable(stdout, []string{"Name", "Category", "Status", "Summary"}, rows)
}

func runResourceBlueprintValidateCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return usageErrorf("resource blueprint", "resource blueprint validate does not accept positional arguments")
	}
	report, err := controller.ValidateBlueprints()
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
	_, _ = fmt.Fprintf(stdout, "Validated %d resource blueprints\n", report.Count)
	return nil
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
