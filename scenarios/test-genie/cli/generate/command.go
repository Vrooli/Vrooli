package generate

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

const UsageLine = "test-genie generate <scenario> [--types unit,integration] [--coverage 95] [--priority normal] [--notes text] [--notes-file path] [--json]"

// HelpText returns the framework-rendered help body for the generate command.
func HelpText() string {
	return `Queue suite generation for a scenario.

Examples:
  test-genie generate swarm-manager --types unit,integration
  test-genie generate prompt-manager --coverage 90 --priority high
  test-genie generate agent-inbox --notes-file notes.txt`
}

// Run executes the generate command.
func Run(client *Client, args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}

	payload := Request{
		ScenarioName:   parsed.Scenario,
		RequestedTypes: cliutil.ParseCSV(parsed.Types),
		Priority:       strings.ToLower(parsed.Priority),
		Notes:          parsed.Notes,
	}
	if parsed.Coverage > 0 {
		val := parsed.Coverage
		payload.CoverageTarget = &val
	}
	if parsed.NotesFile != "" {
		content, err := cliutil.ReadFileString(parsed.NotesFile)
		if err != nil {
			return fmt.Errorf("read notes file: %w", err)
		}
		payload.Notes = content
	}

	resp, raw, err := client.Create(payload)
	if err != nil {
		return err
	}
	if parsed.JSON {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Printf("Suite request queued for %s\n", resp.ScenarioName)
	if resp.ID != "" {
		fmt.Printf("  Request ID : %s\n", resp.ID)
	}
	if resp.Status != "" {
		fmt.Printf("  Status     : %s\n", resp.Status)
	}
	if len(resp.RequestedTypes) > 0 {
		fmt.Printf("  Types      : %s\n", strings.Join(resp.RequestedTypes, ", "))
	}
	if resp.CoverageTarget != nil {
		fmt.Printf("  Coverage   : %d%%\n", *resp.CoverageTarget)
	}
	if resp.Priority != "" {
		fmt.Printf("  Priority   : %s\n", resp.Priority)
	}
	if resp.EstimatedQueueSec > 0 {
		fmt.Printf("  ETA        : ~%ds\n", resp.EstimatedQueueSec)
	}
	return nil
}

// ParseArgs parses command line arguments for the generate command.
func ParseArgs(args []string) (Args, error) {
	if len(args) == 0 {
		return Args{}, usageError("usage: " + strings.TrimPrefix(UsageLine, "test-genie "))
	}
	out := Args{Scenario: args[0]}
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.StringVar(&out.Types, "types", "", "Comma-separated types to request")
	fs.IntVar(&out.Coverage, "coverage", 0, "Coverage target (1-100)")
	fs.StringVar(&out.Priority, "priority", "", "Priority (low|normal|high|urgent)")
	fs.StringVar(&out.Notes, "notes", "", "Notes for this request")
	fs.StringVar(&out.NotesFile, "notes-file", "", "Path to notes file")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return Args{}, err
	}
	out.JSON = *jsonOutput

	if out.Coverage < 0 || out.Coverage > 100 {
		return Args{}, usageError("coverage must be between 0 and 100")
	}
	if out.Priority != "" && !isAllowedPriority(out.Priority) {
		return Args{}, usageError("priority must be one of: low, normal, high, urgent")
	}
	return out, nil
}

func usageError(msg string) error {
	return errors.New(msg)
}

func isAllowedPriority(priority string) bool {
	switch strings.ToLower(priority) {
	case "low", "normal", "high", "urgent":
		return true
	default:
		return false
	}
}
