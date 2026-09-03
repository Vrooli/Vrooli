package watchdog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/repo-contract-go/cliinvoke"
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
			"sudo vrooli setup",
			"vrooli setup status",
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
	if *system {
		return fmt.Errorf("root autoheal services are not supported; use `sudo vrooli setup` for the project-owned user service")
	}
	if *enableLinger {
		fmt.Fprintln(os.Stdout, "autoheal watchdog installation is owned by project setup; the dedicated boot policy enables lingering there")
	}
	return runVrooli(cliinvoke.Setup(*jsonOutput)...)
}

// runVrooli streams one control-plane command to the operator through the
// shared invoker; setup can prompt, so it gets a long deadline.
func runVrooli(args ...string) error {
	home, _ := os.UserHomeDir()
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{RuntimeHome: home})
	if err != nil {
		return err
	}
	return cliinvoke.Run(context.Background(), cliinvoke.Invocation{
		Binary:  binary,
		Args:    args,
		Timeout: time.Hour,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}).Error()
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
			"sudo vrooli setup",
		},
	})
}
