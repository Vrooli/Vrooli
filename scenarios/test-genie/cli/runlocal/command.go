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
	return nil
}

// ParseArgs parses command line arguments for the run-tests command.
func ParseArgs(args []string) (Args, error) {
	if len(args) == 0 {
		return Args{}, usageError("usage: run-tests <scenario> [--type phased] [--json]")
	}
	out := Args{Scenario: args[0]}
	fs := flag.NewFlagSet("run-tests", flag.ContinueOnError)
	fs.StringVar(&out.Type, "type", "", "Test runner type to request")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return Args{}, err
	}
	out.JSON = *jsonOutput
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
