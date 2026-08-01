package watchdog

import (
	"fmt"
	"os"
	"strings"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Watchdog",
		Commands: []cliapp.Command{
			{
				Name:        "watchdog-status",
				Aliases:     []string{"watchdog-info"},
				NeedsAPI:    true,
				Description: "Show detailed watchdog installation status",
				Run: func(args []string) error {
					return runStatus(core, args)
				},
			},
		},
	}
}

func RenderStatus(core *cliapp.ScenarioApp, args []string) error {
	return runStatus(core, args)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("watchdog")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/watchdog/status", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var resp support.WatchdogInstallStatus
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Protection level: %s", strings.ToUpper(resp.ProtectionLevel)),
			fmt.Sprintf("Installed: %s, enabled: %s, running: %s", support.BoolWord(resp.Installed), support.BoolWord(resp.Enabled), support.BoolWord(resp.Running)),
			fmt.Sprintf("Watchdog type: %s", resp.WatchdogType),
		},
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Boot Protection",
				Items: []string{
					fmt.Sprintf("Boot protected: %s", support.BoolWord(resp.BootProtected)),
					fmt.Sprintf("Recommended setup: %s", resp.RecommendedSetup),
					fmt.Sprintf("Needs linger: %s", support.BoolWord(resp.NeedsLinger)),
					strings.TrimSpace(resp.LingerCommand),
				},
			},
		},
		NextSteps: []string{
			"vrooli-autoheal install",
			"vrooli-autoheal uninstall --yes",
			"vrooli-autoheal loop --interval-seconds=60",
		},
	})
}

func Install(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("install")
	system := fs.Bool("system", false, "Install as a system service")
	enableLinger := fs.Bool("enable-linger", false, "Enable systemd linger for user services")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/watchdog/install", nil, map[string]interface{}{
		"useSystemService": *system,
		"enableLingering":  *enableLinger,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var result support.WatchdogMutationResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			result.Message,
			fmt.Sprintf("Success: %s", support.BoolWord(result.Success)),
		},
		Changes: []string{
			strings.TrimSpace(result.ServicePath),
			strings.TrimSpace(result.Error),
			strings.TrimSpace(result.LingerCommand),
		},
		NextCommand: []string{
			"vrooli-autoheal watchdog",
			"vrooli-autoheal status",
		},
	})
}

func Uninstall(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("uninstall")
	yes := fs.Bool("yes", false, "Skip confirmation")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if !*yes {
		confirmed, err := support.Confirm("Remove the OS watchdog service?")
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("aborted")
		}
	}

	body, err := core.Request("POST", "/watchdog/uninstall", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var result support.WatchdogMutationResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{result.Message},
		Changes: []string{
			strings.TrimSpace(result.ServicePath),
			strings.TrimSpace(result.Error),
		},
		NextCommand: []string{
			"vrooli-autoheal watchdog",
			"vrooli-autoheal install",
		},
	})
}
