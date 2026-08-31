package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Runner Command Dispatcher
// =============================================================================

func (a *App) cmdRunner(args []string) error {
	return dispatchSubcommand(args, "runner", map[string]subcommandHandler{
		"list":  a.runnerList,
		"probe": a.runnerProbe,
		"tools": a.runnerTools,
	})
}

func (a *App) runnerTools(args []string) error {
	fs := flag.NewFlagSet("runner tools", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, runners, err := a.services.Runners.GetStatus()
	if err != nil {
		return err
	}
	if *jsonOutput || runners == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	tools := make(map[string]struct{})
	for _, runner := range runners {
		if runner.Capabilities == nil {
			continue
		}
		for tool := range runner.Capabilities.ToolRestrictionMappings {
			tools[tool] = struct{}{}
		}
	}
	names := make([]string, 0, len(tools))
	for tool := range tools {
		names = append(names, tool)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("no runner reported canonical tool mappings")
	}
	fmt.Printf("%-12s", "CANONICAL")
	for _, runner := range runners {
		fmt.Printf("  %-22s", formatEnumValue(runner.RunnerType, "RUNNER_TYPE_", "-"))
	}
	fmt.Println()
	for _, tool := range names {
		fmt.Printf("%-12s", tool)
		for _, runner := range runners {
			value := "unsupported"
			if runner.Capabilities != nil && runner.Capabilities.SupportsToolRestriction {
				value = runner.Capabilities.ToolRestrictionMappings[tool]
				if value == "" {
					value = "unmapped"
				}
			}
			fmt.Printf("  %-22s", value)
		}
		fmt.Println()
	}
	return nil
}

// =============================================================================
// Runner List
// =============================================================================

func (a *App) runnerList(args []string) error {
	fs := flag.NewFlagSet("runner list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, runners, err := a.services.Runners.GetStatus()
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if runners == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(runners) == 0 {
		fmt.Println("No runners found")
		return nil
	}

	fmt.Printf("%-12s  %-9s  %-40s\n", "TYPE", "AVAILABLE", "MESSAGE")
	fmt.Printf("%-12s  %-9s  %-40s\n", strings.Repeat("-", 12), strings.Repeat("-", 9), strings.Repeat("-", 40))
	for _, r := range runners {
		runnerType := formatEnumValue(r.RunnerType, "RUNNER_TYPE_", "-")
		available := "no"
		if r.Available {
			available = "yes"
		}
		message := r.Message
		if len(message) > 40 {
			message = message[:37] + "..."
		}
		fmt.Printf("%-12s  %-9s  %-40s\n", runnerType, available, message)
	}

	return nil
}

// =============================================================================
// Runner Probe
// =============================================================================

func (a *App) runnerProbe(args []string) error {
	fs := flag.NewFlagSet("runner probe", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional runner type first
	var runnerType string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		runnerType = args[0]
		args = args[1:]
	}

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if runnerType == "" {
		return fmt.Errorf("usage: agent-manager runner probe <type>\n\nValid types: claude-code, codex, opencode")
	}

	body, result, err := a.services.Runners.Probe(runnerType)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	status := "FAILED"
	if result.Success {
		status = "SUCCESS"
	}
	fmt.Printf("Runner:   %s\n", runnerType)
	fmt.Printf("Status:   %s\n", status)
	fmt.Printf("Latency:  %dms\n", result.LatencyMs)
	if result.Error != "" {
		fmt.Printf("Error:    %s\n", result.Error)
	}
	if len(result.Details) > 0 {
		fmt.Println("Details:")
		for k, v := range result.Details {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}
