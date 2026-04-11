package main

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

func runResourceCommand(controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		showResourceHelp(stdout)
		return nil
	}

	switch normalizeResourceSubcommand(args[0]) {
	case "blueprint":
		return runResourceBlueprintCommand(controller, globals, args[1:], stdout)
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
		return fmt.Errorf("unknown resource blueprint command: %s", args[0])
	}
}

func runResourceListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("resource list does not accept positional arguments")
	}
	items, err := controller.Discover()
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{"success": true, "resources": items})
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			boolLabel(item.Enabled),
			boolLabel(item.Registered),
			boolLabel(item.HasCLI || item.HasScript),
		})
	}
	return cliout.RenderTable(stdout, []string{"Name", "Enabled", "Registered", "Controllable"}, rows)
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

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}

	if len(filtered) == 0 {
		items, err := controller.ListStatuses(fast, false)
		if err != nil {
			return err
		}
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(stdout, map[string]any{"success": true, "resources": items})
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
				boolLabel(item.Running),
				healthy,
				item.Message,
			})
		}
		return cliout.RenderTable(stdout, []string{"Name", "Enabled", "Running", "Health", "Status"}, rows)
	}

	if len(filtered) != 1 {
		return fmt.Errorf("resource status accepts at most one resource name")
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
	if item.ProbeError != "" {
		rows = append(rows, []string{"Probe Error", item.ProbeError})
	}
	return cliout.RenderTable(stdout, []string{"Field", "Value"}, rows)
}

func runSingleResourceControlCommand(controller *resources.Controller, action string, args []string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("resource %s requires exactly one resource name", action)
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
		return fmt.Errorf("resource %s requires exactly one resource name", action)
	}
	if err := controller.SetEnabled(args[0], enabled); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Updated %s: enabled=%t\n", args[0], enabled)
	return nil
}

func runResourceInfoCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("resource info requires exactly one resource name")
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

func runResourceBlueprintListCommand(controller *resources.Controller, globals globalOptions, args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("resource blueprint list does not accept positional arguments")
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
		return fmt.Errorf("resource blueprint info requires exactly one blueprint name")
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
		return fmt.Errorf("resource blueprint search requires exactly one query")
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
		return fmt.Errorf("resource blueprint validate does not accept positional arguments")
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
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource <list|status|install|start|start-all|stop|stop-all|enable|disable|info> [...]")
	_, _ = fmt.Fprintln(w, "       vrooli resource blueprint <list|info|search|validate> [...]")
	_, _ = fmt.Fprintln(w, "       vrooli resource <name> <command> [options]")
}

func showResourceBlueprintHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli resource blueprint <list|info|search|validate> [...]")
}
