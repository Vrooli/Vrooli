// Package eligibility implements the `test-genie eligibility` CLI subcommand
// tree. It is a thin client over the test-genie API's
// EligibilityService Connect-RPC.
package eligibility

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility/eligibility_v1connect"
)

// ExitCode is the documented exit-code contract for the `check` subcommand.
type ExitCode int

const (
	// ExitRouted means the scenario qualifies for the routed test-db path.
	ExitRouted ExitCode = 0
	// ExitNotRouted means the scenario does not qualify; reasons are
	// surfaced in human and JSON output.
	ExitNotRouted ExitCode = 1
	// ExitUnreachable means the test-genie API could not be reached or
	// returned an unexpected error.
	ExitUnreachable ExitCode = 2
)

// Run dispatches eligibility subcommands.
func Run(httpClient *cliutil.APIClient, args []string) error {
	if len(args) == 0 {
		return printUsage(os.Stdout)
	}
	switch args[0] {
	case "check":
		return runCheck(httpClient, args[1:], os.Stdout)
	case "help", "-h", "--help":
		return printUsage(os.Stdout)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'test-genie eligibility help' for usage", args[0])
	}
}

func printUsage(w io.Writer) error {
	fmt.Fprintln(w, `Usage: test-genie eligibility <command>

Commands:
  check <scenario>  Check whether a scenario qualifies for the routed test-db path

Run 'test-genie eligibility <command> -h' for command-specific options.`)
	return nil
}

// Register returns the manifest-backed eligibility command group.
func Register(manifest []byte, apiClient *cliutil.APIClient) (cliapp.SubcommandGroup, error) {
	return cliapp.LoadFromManifestPrimitives(manifest, "eligibility", map[string]cliapp.PrimitiveHandler{
		"EligibilityService.Check": cliapp.ProtoListOutcome(checkCall(apiClient), checkReport, checkExit),
	})
}

func checkCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (*eligpb.CheckResponse, error) {
	return func(ctx cliapp.OperationContext) (*eligpb.CheckResponse, error) {
		resp, err := callCheck(context.Background(), apiClient, strings.TrimSpace(ctx.Positional("scenario")))
		if err != nil {
			return nil, &exitErr{code: ExitUnreachable, err: err}
		}
		return resp.Msg, nil
	}
}

func checkReport(_ cliapp.OperationContext, msg *eligpb.CheckResponse) cliapp.ListReport {
	if msg.GetRouted() {
		return cliapp.ListReport{Summary: []string{"Routed: yes (scenario qualifies for the routed test-db path)"}}
	}
	results := make([]string, 0, len(msg.GetDisqualifyingReasons())+len(msg.GetViolations())+1)
	for _, reason := range msg.GetDisqualifyingReasons() {
		results = append(results, reason)
	}
	for _, v := range msg.GetViolations() {
		loc := v.GetFile()
		if v.GetLine() > 0 {
			loc = fmt.Sprintf("%s:%d", loc, v.GetLine())
		}
		if loc == "" {
			loc = "(no location)"
		}
		results = append(results, fmt.Sprintf("[%s] %s  %s", strings.ToUpper(v.GetSeverity()), v.GetRuleId(), loc))
	}
	if ra := msg.GetRuleAssertion(); ra != nil && len(ra.GetMissingRules()) > 0 {
		results = append(results, fmt.Sprintf("Missing auditor rules: %s", strings.Join(ra.GetMissingRules(), ", ")))
	}
	return cliapp.ListReport{Summary: []string{"Routed: no"}, ResultsHeading: "Reasons", Results: results}
}

func checkExit(msg *eligpb.CheckResponse) error {
	if msg.GetRouted() {
		return nil
	}
	return &exitErr{code: ExitNotRouted, err: errors.New("scenario not eligible for routed test-db path")}
}

func runCheck(httpClient *cliutil.APIClient, args []string, w io.Writer) error {
	fs := flag.NewFlagSet("eligibility check", flag.ContinueOnError)
	fs.SetOutput(w)
	jsonOut := fs.Bool("json", false, "Emit the raw CheckResponse as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fs.Usage()
		return errors.New("exactly one scenario name is required")
	}
	scenario := strings.TrimSpace(rest[0])
	if scenario == "" {
		return errors.New("scenario name must not be empty")
	}

	resp, err := callCheck(context.Background(), httpClient, scenario)
	if err != nil {
		fmt.Fprintf(w, "Failed to reach test-genie: %v\n", err)
		// cli-core converts a returned error into a non-zero exit; encode
		// the contract via a sentinel error wrapping the code.
		return &exitErr{code: ExitUnreachable, err: err}
	}

	if *jsonOut {
		if err := writeJSON(w, resp.Msg); err != nil {
			return err
		}
	} else {
		writeHuman(w, resp.Msg)
	}
	if !resp.Msg.GetRouted() {
		return &exitErr{code: ExitNotRouted, err: errors.New("scenario not eligible for routed test-db path")}
	}
	return nil
}

// callCheck is exported via package variable to make CLI tests trivial: tests
// override it with a stub that returns a canned response.
var callCheck = func(ctx context.Context, apiClient *cliutil.APIClient, scenario string) (*connect.Response[eligpb.CheckResponse], error) {
	baseURL := strings.TrimRight(apiClient.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("test-genie API base URL is not configured")
	}
	client := eligibility_v1connect.NewEligibilityServiceClient(http.DefaultClient, baseURL)
	return client.Check(ctx, connect.NewRequest(&eligpb.CheckRequest{Scenario: scenario}))
}

func writeJSON(w io.Writer, msg *eligpb.CheckResponse) error {
	payload := map[string]any{
		"routed":     msg.GetRouted(),
		"violations": violationsJSON(msg.GetViolations()),
	}
	if ra := msg.GetRuleAssertion(); ra != nil {
		payload["rule_assertion"] = map[string]any{
			"missing_rules": ra.GetMissingRules(),
		}
	}
	if reasons := msg.GetDisqualifyingReasons(); len(reasons) > 0 {
		payload["disqualifying_reasons"] = reasons
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(encoded))
	return nil
}

func violationsJSON(in []*eligpb.Violation) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, v := range in {
		out = append(out, map[string]any{
			"rule_id":  v.GetRuleId(),
			"severity": v.GetSeverity(),
			"file":     v.GetFile(),
			"line":     v.GetLine(),
			"excerpt":  v.GetExcerpt(),
		})
	}
	return out
}

func writeHuman(w io.Writer, msg *eligpb.CheckResponse) {
	if msg.GetRouted() {
		fmt.Fprintln(w, "Routed: yes (scenario qualifies for the routed test-db path)")
		return
	}
	fmt.Fprintln(w, "Routed: no")
	for _, reason := range msg.GetDisqualifyingReasons() {
		fmt.Fprintf(w, "  - %s\n", reason)
	}
	if violations := msg.GetViolations(); len(violations) > 0 {
		fmt.Fprintln(w, "Violations:")
		for _, v := range violations {
			loc := v.GetFile()
			if v.GetLine() > 0 {
				loc = fmt.Sprintf("%s:%d", loc, v.GetLine())
			}
			if loc == "" {
				loc = "(no location)"
			}
			fmt.Fprintf(w, "  [%s] %s  %s\n", strings.ToUpper(v.GetSeverity()), v.GetRuleId(), loc)
			if v.GetExcerpt() != "" {
				fmt.Fprintf(w, "      %s\n", v.GetExcerpt())
			}
		}
	}
	if ra := msg.GetRuleAssertion(); ra != nil && len(ra.GetMissingRules()) > 0 {
		fmt.Fprintf(w, "Missing auditor rules: %s\n", strings.Join(ra.GetMissingRules(), ", "))
	}
}

// exitErr wraps an error with a documented exit code. The CLI core inspects
// the returned error's ExitCode() if present.
type exitErr struct {
	code ExitCode
	err  error
}

func (e *exitErr) Error() string { return e.err.Error() }
func (e *exitErr) Unwrap() error { return e.err }

// ExitCode satisfies cli-core's interface for documented exit codes.
func (e *exitErr) ExitCode() int { return int(e.code) }
