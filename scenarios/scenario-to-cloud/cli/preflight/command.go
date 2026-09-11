package preflight

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes preflight subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "run":
		return runPreflight(client, args[1:])
	case "requirements":
		return runRequirements(client, args[1:])
	case "fix-firewall":
		return runFixFirewall(client, args[1:])
	case "fix-processes":
		return runFixProcesses(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud preflight help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud preflight <command> [arguments]

Commands:
  run <manifest.json>    Run VPS preflight checks for a cloud manifest
  requirements           Show canonical VPS requirements/policy
  fix-firewall           Open required firewall ports
  fix-processes          Stop stale scenario processes on target VPS

Selector flags for target-dependent commands:
  --host <host> | --domain <domain> | --target <domain-or-host>
  [--scenario <id>] [--user <ssh-user>] [--ssh-port <n>]

Run 'scenario-to-cloud preflight <command> -h' for command-specific options.`)
	return nil
}

func runPreflight(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud preflight run <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.Run(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runRequirements(client *Client, args []string) error {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud preflight requirements [flags]

Show canonical VPS requirements used by preflight/runtime checks.

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag: %s", arg)
			}
			return fmt.Errorf("usage: scenario-to-cloud preflight requirements")
		}
	}

	body, resp, err := client.Requirements()
	if err != nil {
		return err
	}
	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("VPS Requirements")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("OS: %s %s (compatible: %s)\n",
		resp.VPS.OS.RequiredID,
		resp.VPS.OS.RecommendedVersion,
		strings.Join(resp.VPS.OS.CompatibleVersions, ", "),
	)
	fmt.Printf("RAM: min %s, recommended %s\n",
		formatSize(resp.VPS.Resources.MinRAMBytes),
		formatSize(resp.VPS.Resources.RecommendedRAMBytes),
	)
	fmt.Printf("Disk: min free %s\n", formatSize(resp.VPS.Resources.MinDiskFreeBytes))
	fmt.Printf("Inbound ports: %s\n", joinPorts(resp.VPS.Network.RequiredInboundPorts))
	fmt.Printf("Auth: %s (bootstrap: %s)\n",
		resp.VPS.Authentication.RequiredMethod,
		resp.VPS.Authentication.BootstrapFlow,
	)
	return nil
}

func runFixFirewall(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight fix-firewall", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	var ports intListFlag
	fs.Var(&ports, "port", "Open specific port (repeatable; default 80,443)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	target, err := targetFlags.resolve(client, fs.Args())
	if err != nil {
		return err
	}

	req := FixFirewallRequest{
		Host:  target.Host,
		Port:  target.Port,
		User:  target.User,
		Ports: ports.Values,
	}

	body, resp, err := client.FixFirewall(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Println("Firewall rules updated successfully.")
		fmt.Printf("Ports: %s\n", joinPorts(resp.Ports))
		if strings.TrimSpace(resp.Status) != "" {
			fmt.Printf("Status:\n%s\n", strings.TrimSpace(resp.Status))
		}
		return nil
	}

	fmt.Printf("Failed to update firewall: %s\n", resp.Message)
	return nil
}

func runFixProcesses(client *Client, args []string) error {
	fs := flag.NewFlagSet("preflight fix-processes", flag.ContinueOnError)
	targetFlags := registerPreflightTargetFlags(fs)
	scenarioID := fs.String("scenario-id", "", "Override scenario ID for targeted stop")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	target, err := targetFlags.resolve(client, fs.Args())
	if err != nil {
		return err
	}

	req := FixProcessesRequest{
		Host:       target.Host,
		Port:       target.Port,
		User:       target.User,
		Workdir:    target.Workdir,
		ScenarioID: target.ScenarioID,
	}
	if v := strings.TrimSpace(*scenarioID); v != "" {
		req.ScenarioID = v
	}

	body, resp, err := client.FixProcesses(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Printf("Process cleanup succeeded (%s).\n", resp.Action)
		if strings.TrimSpace(resp.Output) != "" {
			fmt.Printf("%s\n", strings.TrimSpace(resp.Output))
		}
		return nil
	}

	fmt.Printf("Process cleanup failed (%s): %s\n", resp.Action, resp.Message)
	if strings.TrimSpace(resp.Output) != "" {
		fmt.Printf("%s\n", strings.TrimSpace(resp.Output))
	}
	return nil
}

func joinPorts(ports []int) string {
	if len(ports) == 0 {
		return "-"
	}
	values := make([]string, 0, len(ports))
	for _, p := range ports {
		values = append(values, strconv.Itoa(p))
	}
	return strings.Join(values, ", ")
}

// formatSize formats bytes into a human-readable string.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

type intListFlag struct {
	Values []int
}

func (f *intListFlag) String() string {
	if len(f.Values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f.Values))
	for _, v := range f.Values {
		parts = append(parts, strconv.Itoa(v))
	}
	return strings.Join(parts, ",")
}

func (f *intListFlag) Set(value string) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("invalid integer %q", value)
	}
	f.Values = append(f.Values, n)
	return nil
}
