package runlocal

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes the run-tests command.
func Run(client *Client, args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}

	req := Request{}
	if parsed.Type != "" {
		req.Type = parsed.Type
	}
	if len(parsed.Paths) > 0 {
		req.Paths = parsed.Paths
	}
	if len(parsed.Playbooks) > 0 {
		req.Playbooks = parsed.Playbooks
	}
	if parsed.Filter != "" {
		req.Filter = parsed.Filter
	}

	resp, raw, err := client.Run(parsed.Scenario, req)
	if err != nil {
		return err
	}
	if parsed.JSON {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Printf("Scenario tests completed via %s\n", defaultValue(resp.Type, "unknown"))
	if len(resp.Command.Command) > 0 {
		fmt.Printf("  Command : %s\n", strings.Join(resp.Command.Command, " "))
	}
	if resp.Command.WorkingDir != "" {
		fmt.Printf("  Workdir : %s\n", resp.Command.WorkingDir)
	}
	if resp.LogPath != "" {
		fmt.Printf("  Logs    : %s\n", resp.LogPath)
	}
	if resp.Status != "" {
		fmt.Printf("  Status  : %s\n", resp.Status)
	}
	if len(req.Paths) > 0 {
		fmt.Printf("  Paths   : %s\n", strings.Join(req.Paths, ", "))
	}
	if len(req.Playbooks) > 0 {
		fmt.Printf("  Playbook: %s\n", strings.Join(req.Playbooks, ", "))
	}
	if req.Filter != "" {
		fmt.Printf("  Filter  : %s\n", req.Filter)
	}
	return nil
}

// ParseArgs parses command line arguments for the run-tests command.
func ParseArgs(args []string) (Args, error) {
	if len(args) == 0 {
		return Args{}, usageError("usage: run-tests <scenario> [--type phased] [--path <p>] [--playbook <pb>] [--filter text] [--json]")
	}
	out := Args{Scenario: args[0]}
	fs := flag.NewFlagSet("run-tests", flag.ContinueOnError)
	fs.StringVar(&out.Type, "type", "", "Test runner type to request")
	var pathArgs multiFlag
	var playbookArgs multiFlag
	fs.Var(&pathArgs, "path", "Limit to one or more files/dirs (repeatable)")
	fs.Var(&playbookArgs, "playbook", "Limit to one or more playbooks (repeatable)")
	fs.StringVar(&out.Filter, "filter", "", "Pass through filter string to runner (e.g., test name)")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return Args{}, err
	}
	out.JSON = *jsonOutput
	out.Paths = pathArgs
	out.Playbooks = playbookArgs
	return out, nil
}

func usageError(msg string) error {
	return errors.New(msg)
}

func defaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

// multiFlag captures repeatable string flags.
type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(val string) error {
	trimmed := strings.TrimSpace(val)
	if trimmed != "" {
		*m = append(*m, trimmed)
	}
	return nil
}
