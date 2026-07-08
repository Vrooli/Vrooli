package generate

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const UsageLine = "test-genie generate <scenario> [--types unit,integration] [--coverage 95] [--priority normal] [--notes text] [--notes-file path] [--json]"

// ArgsSchema is the cli-core declarative argument contract for the primitive
// command path. The built-in --json flag is supplied by cli-core.
var ArgsSchema = cliapp.ArgSchema{
	Positionals: []cliapp.Positional{{
		Name:        "scenario",
		Required:    true,
		Description: "Scenario id",
	}},
	Flags: []cliapp.Flag{
		{Name: "types", Description: "Comma-separated types to request"},
		{Name: "coverage", Description: "Coverage target (1-100)"},
		{Name: "priority", Description: "Priority (low|normal|high|urgent)"},
		{Name: "notes", Description: "Notes for this request"},
		{Name: "notes-file", Description: "Path to notes file"},
	},
}

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

// Primitive returns the renderer-separated action primitive for suite
// generation. The operation callback receives only OperationContext, so output
// mode cannot change which request is sent.
func Primitive(client *Client) cliapp.PrimitiveHandler {
	return cliapp.Action(
		func(ctx cliapp.OperationContext) (Response, error) {
			payload, err := RequestFromContext(ctx)
			if err != nil {
				return Response{}, err
			}
			resp, _, err := client.Create(payload)
			return resp, err
		},
		func(ctx cliapp.OperationContext, resp Response) cliapp.MutationReport {
			return Report(resp)
		},
	)
}

// RequestFromContext maps parsed cli-core inputs into the API request.
func RequestFromContext(ctx cliapp.OperationContext) (Request, error) {
	coverage, err := parseCoverage(ctx.Flag("coverage"))
	if err != nil {
		return Request{}, err
	}
	priority := ctx.Flag("priority")
	if priority != "" && !isAllowedPriority(priority) {
		return Request{}, usageError("priority must be one of: low, normal, high, urgent")
	}

	payload := Request{
		ScenarioName:   ctx.Positional("scenario"),
		RequestedTypes: cliutil.ParseCSV(ctx.Flag("types")),
		Priority:       strings.ToLower(priority),
		Notes:          ctx.Flag("notes"),
	}
	if coverage > 0 {
		payload.CoverageTarget = &coverage
	}
	if notesFile := ctx.Flag("notes-file"); notesFile != "" {
		content, err := cliutil.ReadFileString(notesFile)
		if err != nil {
			return Request{}, fmt.Errorf("read notes file: %w", err)
		}
		payload.Notes = content
	}
	return payload, nil
}

// Report renders the human mutation report for a queued suite request.
func Report(resp Response) cliapp.MutationReport {
	result := []string{fmt.Sprintf("Suite request queued for %s", resp.ScenarioName)}
	changes := make([]string, 0, 6)
	if resp.ID != "" {
		changes = append(changes, fmt.Sprintf("Request ID: %s", resp.ID))
	}
	if resp.Status != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", resp.Status))
	}
	if len(resp.RequestedTypes) > 0 {
		changes = append(changes, fmt.Sprintf("Types: %s", strings.Join(resp.RequestedTypes, ", ")))
	}
	if resp.CoverageTarget != nil {
		changes = append(changes, fmt.Sprintf("Coverage: %d%%", *resp.CoverageTarget))
	}
	if resp.Priority != "" {
		changes = append(changes, fmt.Sprintf("Priority: %s", resp.Priority))
	}
	if resp.EstimatedQueueSec > 0 {
		changes = append(changes, fmt.Sprintf("ETA: ~%ds", resp.EstimatedQueueSec))
	}
	return cliapp.MutationReport{
		Result:  result,
		Changes: changes,
	}
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

func parseCoverage(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, usageError("coverage must be between 0 and 100")
	}
	if value < 0 || value > 100 {
		return 0, usageError("coverage must be between 0 and 100")
	}
	return value, nil
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
