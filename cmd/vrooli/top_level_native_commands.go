package main

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
)

func runTopLevelBuildCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "build", args, stdout, stderr)
}

func runTopLevelDeployCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "deploy", args, stdout, stderr)
}

func runTopLevelCleanCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "clean", args, stdout, stderr)
}

func runTopLevelBackupCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "backup", args, stdout, stderr)
}

func runTopLevelRestoreCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "restore", args, stdout, stderr)
}

func runTopLevelStatusCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	opts, err := parseTopLevelStatusArgs(args)
	if err != nil {
		return err
	}
	if opts.Help {
		showStatusHelp(stdout)
		return nil
	}

	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := project.New(root, home, stdout, stderr)
	report, err := controller.Status(project.StatusOptions{
		ResourcesOnly: opts.ResourcesOnly,
		ScenariosOnly: opts.ScenariosOnly,
		Fast:          opts.Fast,
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
			"status":  report,
		})
	}

	if !opts.ScenariosOnly {
		_, _ = fmt.Fprintln(stdout, "Resources")
		rows := make([][]string, 0, len(report.Resources))
		for _, item := range report.Resources {
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
		_ = cliout.RenderTable(stdout, []string{"Name", "Enabled", "Running", "Health", "Status"}, rows)
		_, _ = fmt.Fprintln(stdout)
	}
	if !opts.ResourcesOnly {
		_, _ = fmt.Fprintln(stdout, "Scenarios")
		rows := make([][]string, 0, len(report.Scenarios))
		for _, item := range report.Scenarios {
			health := ""
			if item.Health != nil {
				health = fmt.Sprint(item.Health)
			}
			rows = append(rows, []string{
				item.Name,
				item.Status,
				fmt.Sprintf("%d", item.Processes),
				health,
				item.Runtime,
			})
		}
		_ = cliout.RenderTable(stdout, []string{"Name", "Status", "Processes", "Health", "Runtime"}, rows)
	}

	return nil
}

func runTopLevelDoctorCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showDoctorHelp(stdout)
			return nil
		default:
			return fmt.Errorf("unknown option for doctor: %s", arg)
		}
	}

	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := project.New(root, home, stdout, stderr)
	report, err := controller.Doctor()
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
			"checks":  report.Checks,
		})
	}

	rows := make([][]string, 0, len(report.Checks))
	for _, item := range report.Checks {
		rows = append(rows, []string{item.Name, item.Status})
	}
	return cliout.RenderTable(stdout, []string{"Check", "Status"}, rows)
}

func runTopLevelStopCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			showStopHelp(stdout)
			return nil
		}
	}

	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := project.New(root, home, stdout, stderr)
	result, err := controller.Stop(project.StopOptions{Args: args})
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
			"data":    result,
		})
	}

	for _, item := range result.Stopped {
		_, _ = fmt.Fprintf(stdout, "Stopped %s\n", item.Name)
	}
	for _, item := range result.Failed {
		_, _ = fmt.Fprintf(stdout, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
}

func runTopLevelResourceCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := resources.NewController(root, home)
	return runResourceCommand(controller, globals, args, stdout, stderr)
}

func runTopLevelOrphansCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := maintenance.NewController(root, home)

	mode := "list"
	for _, arg := range args {
		switch arg {
		case "kill":
			mode = "kill"
		case "--help", "-h", "help":
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli orphans [kill] [--json]")
			return nil
		default:
			return fmt.Errorf("unknown option for orphans: %s", arg)
		}
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if mode == "kill" {
		report, err := controller.KillOrphans()
		if err != nil {
			return err
		}
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(stdout, map[string]any{"success": true, "data": report})
		}
		for _, item := range report.Stopped {
			_, _ = fmt.Fprintf(stdout, "Stopped orphan PID %s (%s)\n", item.Name, item.Message)
		}
		for _, item := range report.Failed {
			_, _ = fmt.Fprintf(stdout, "Failed orphan PID %s: %s\n", item.Name, item.Error)
		}
		if len(report.Stopped) == 0 && len(report.Failed) == 0 {
			_, _ = fmt.Fprintln(stdout, "No orphaned Vrooli processes found.")
		}
		return nil
	}

	orphans, err := controller.ListOrphans()
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{"success": true, "orphans": orphans})
	}
	if len(orphans) == 0 {
		_, _ = fmt.Fprintln(stdout, "No orphaned Vrooli processes found.")
		return nil
	}
	rows := make([][]string, 0, len(orphans))
	for _, item := range orphans {
		rows = append(rows, []string{strconv.Itoa(item.PID), strconv.Itoa(item.PPID), item.Command})
	}
	return cliout.RenderTable(stdout, []string{"PID", "PPID", "Command"}, rows)
}

func runTopLevelLocksCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := maintenance.NewController(root, home)

	mode := "list"
	for _, arg := range args {
		switch arg {
		case "clean":
			mode = "clean"
		case "--help", "-h", "help":
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli locks [clean] [--json]")
			return nil
		default:
			return fmt.Errorf("unknown option for locks: %s", arg)
		}
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if mode == "clean" {
		report, err := controller.CleanStaleLocks()
		if err != nil {
			return err
		}
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(stdout, map[string]any{"success": true, "data": report})
		}
		for _, item := range report.Stopped {
			_, _ = fmt.Fprintf(stdout, "Removed stale lock for port %s\n", item.Name)
		}
		for _, item := range report.Failed {
			_, _ = fmt.Fprintf(stdout, "Failed to remove lock for port %s: %s\n", item.Name, item.Error)
		}
		if len(report.Stopped) == 0 && len(report.Failed) == 0 {
			_, _ = fmt.Fprintln(stdout, "No stale port locks found.")
		}
		return nil
	}

	locks, err := controller.ListLocks()
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{"success": true, "locks": locks})
	}
	if len(locks) == 0 {
		_, _ = fmt.Fprintln(stdout, "No port locks found.")
		return nil
	}
	rows := make([][]string, 0, len(locks))
	for _, item := range locks {
		status := "active"
		if item.Stale {
			status = "stale"
		}
		rows = append(rows, []string{
			strconv.Itoa(item.Port),
			item.Scenario,
			strconv.Itoa(item.PID),
			status,
		})
	}
	return cliout.RenderTable(stdout, []string{"Port", "Scenario", "PID", "Status"}, rows)
}

func runTopLevelDiagnosePortCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: vrooli diagnose-port <port> [scenario] [--json]")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		_, _ = fmt.Fprintln(stdout, "Usage: vrooli diagnose-port <port> [scenario] [--json]")
		return nil
	}
	port, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil || port <= 0 {
		return fmt.Errorf("invalid port: %s", args[0])
	}
	scenarioName := ""
	if len(args) > 1 {
		scenarioName = args[1]
	}

	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := maintenance.NewController(root, home)
	diagnostic, err := controller.DiagnosePort(port, scenarioName)
	if err != nil {
		return err
	}

	format, err := cliout.ParseFormat("", globals.json)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(stdout, map[string]any{"success": true, "diagnostic": diagnostic})
	}

	_, _ = fmt.Fprintf(stdout, "Port %d\n", diagnostic.Port)
	if diagnostic.Scenario != "" {
		_, _ = fmt.Fprintf(stdout, "Scenario: %s\n", diagnostic.Scenario)
	}
	if diagnostic.InUse {
		_, _ = fmt.Fprintln(stdout, "Listeners:")
		for _, listener := range diagnostic.Listeners {
			_, _ = fmt.Fprintf(stdout, "  PID %d  zombie=%t  %s\n", listener.PID, listener.Zombie, listener.Command)
		}
	} else {
		_, _ = fmt.Fprintln(stdout, "Listeners: none")
	}
	if diagnostic.Lock != nil {
		_, _ = fmt.Fprintf(stdout, "Lock: %s (scenario=%s pid=%d stale=%t)\n", diagnostic.Lock.Path, diagnostic.Lock.Scenario, diagnostic.Lock.PID, diagnostic.Lock.Stale)
	} else {
		_, _ = fmt.Fprintln(stdout, "Lock: none")
	}
	_, _ = fmt.Fprintf(stdout, "Orphans detected: %d\n", diagnostic.OrphanCount)
	_, _ = fmt.Fprintln(stdout, "Recommended actions:")
	for _, recommendation := range diagnostic.Recommendations {
		_, _ = fmt.Fprintf(stdout, "  - %s\n", recommendation)
	}
	return nil
}

type topLevelStatusOptions struct {
	Help          bool
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

func parseTopLevelStatusArgs(args []string) (topLevelStatusOptions, error) {
	opts := topLevelStatusOptions{Fast: true}
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			opts.Help = true
		case "--resources":
			opts.ResourcesOnly = true
		case "--scenarios":
			opts.ScenariosOnly = true
		case "--fast":
			opts.Fast = true
		case "--no-fast":
			opts.Fast = false
		default:
			return topLevelStatusOptions{}, fmt.Errorf("unknown option for status: %s", arg)
		}
	}
	if opts.ResourcesOnly && opts.ScenariosOnly {
		return topLevelStatusOptions{}, fmt.Errorf("status accepts only one of --resources or --scenarios")
	}
	return opts, nil
}

func runProjectLifecyclePhaseCommand(root, phase string, args []string, stdout, stderr io.Writer) error {
	if wantsCommandHelp(args) {
		showProjectLifecycleHelp(stdout, phase)
		return nil
	}

	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := project.New(root, home, stdout, stderr)
	return controller.RunProjectPhase(phase, args)
}

func wantsCommandHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "help":
			return true
		}
	}
	return false
}

func showProjectLifecycleHelp(w io.Writer, phase string) {
	_, _ = fmt.Fprintf(w, "Usage: vrooli %s\n", phase)
}

func showStatusHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli status [--resources|--scenarios] [--fast|--no-fast] [--json]")
}

func showDoctorHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli doctor [--json]")
}

func showStopHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli stop [all|scenarios|resources|scenario:<name>|resource:<name>|<name>...] [--json]")
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func installedCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func normalizeResourceSubcommand(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
