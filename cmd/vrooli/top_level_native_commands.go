package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
)

func runTopLevelBuildCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "build", args, stdout, stderr)
}

func runTopLevelDeployCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return runProjectLifecyclePhaseCommand(root, "deploy", args, stdout, stderr)
}

func runTopLevelBackupCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return fmt.Errorf("project lifecycle phase %q is not defined in %s/.vrooli/service.json", "backup", root)
}

func runTopLevelRestoreCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	return fmt.Errorf("project lifecycle phase %q is not defined in %s/.vrooli/service.json", "restore", root)
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
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	controller := project.New(root, home, stdout, stderr)
	return controller.RunProjectPhase(phase, args)
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
