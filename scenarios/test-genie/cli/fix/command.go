// Package fix implements the `test-genie fix` CLI command. It exposes two
// paths over the test-genie API:
//
//   - default (agent-based): spawns a Claude Code fix agent for the named phases
//     (POST /api/v1/scenarios/{name}/fix). This is the non-deterministic path.
//   - --deterministic: aggregates each delegated provider's shared Fix RPC and
//     reports the autofixable candidates (POST .../fix/deterministic). Dry-run by
//     default; pass --apply to write changes.
package fix

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// stringList collects repeatable flag values.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	v = strings.TrimSpace(v)
	if v != "" {
		*s = append(*s, v)
	}
	return nil
}

// Run executes the fix command.
func Run(apiClient *cliutil.APIClient, args []string) error {
	return run(apiClient, args, os.Stdout)
}

func run(apiClient *cliutil.APIClient, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("fix", flag.ContinueOnError)
	deterministic := fs.Bool("deterministic", false, "Use the deterministic provider-driven autofix aggregate instead of the fix agent")
	fleetMode := fs.Bool("fleet", false, "Remediate the whole priority-ordered fleet (implies --deterministic); no scenario argument")
	apply := fs.Bool("apply", false, "Apply changes to disk (deterministic path only; dry-run otherwise)")
	asJSON := fs.Bool("json", false, "Emit JSON output")
	maxScenarios := fs.Int("max-scenarios", 0, "Cap the number of fleet scenarios walked (0 = all; --fleet only)")
	concurrency := fs.Int("concurrency", 1, "Concurrent scenarios in a --fleet dry-run (forced to 1 with --apply)")
	var rules stringList
	fs.Var(&rules, "rule", "Restrict to a rule/finding code (repeatable)")
	var providers stringList
	fs.Var(&providers, "provider", "Restrict deterministic fix to a provider scenario (repeatable)")
	var phases stringList
	fs.Var(&phases, "phase", "Phase to hand the fix agent (repeatable; agent path only)")
	message := fs.String("message", "", "Extra instruction for the fix agent (agent path only)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if apiClient == nil {
		return errors.New("test-genie API client is not configured")
	}

	// Fleet remediation walks every scenario; it is deterministic-only and takes
	// no positional scenario argument (Stage 2 × Stage 3).
	if *fleetMode {
		return runFleet(apiClient, *apply, rules, providers, *asJSON, *maxScenarios, *concurrency, out)
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return errors.New("usage: test-genie fix <scenario> [--deterministic] [--apply] [--rule code] [--provider name] [--json]  |  test-genie fix --fleet [--apply] [--max-scenarios N]")
	}
	scenario := rest[0]

	if *deterministic {
		return runDeterministic(apiClient, scenario, *apply, rules, providers, *asJSON, out)
	}
	if *apply {
		return errors.New("--apply only applies to --deterministic; the agent path always writes")
	}
	return runAgent(apiClient, scenario, phases, *message, *asJSON, out)
}

func runDeterministic(apiClient *cliutil.APIClient, scenario string, apply bool, rules, providers stringList, asJSON bool, out io.Writer) error {
	body := map[string]any{"apply": apply}
	if len(rules) > 0 {
		body["ruleIds"] = []string(rules)
	}
	if len(providers) > 0 {
		body["providers"] = []string(providers)
	}
	path := fmt.Sprintf("/api/v1/scenarios/%s/fix/deterministic", url.PathEscape(scenario))
	raw, err := apiClient.Request(http.MethodPost, path, nil, body)
	if err != nil {
		return fmt.Errorf("deterministic fix request failed: %w", err)
	}
	if asJSON {
		_, _ = out.Write(raw)
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			_, _ = fmt.Fprintln(out)
		}
		return nil
	}
	var report deterministicReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("parse deterministic fix report: %w", err)
	}
	report.render(out)
	return nil
}

type deterministicReport struct {
	Scenario        string `json:"scenario"`
	Applied         bool   `json:"applied"`
	TotalCandidates int    `json:"totalCandidates"`
	Providers       []struct {
		Provider   string `json:"provider"`
		Status     string `json:"status"`
		Candidates []struct {
			RuleID      string `json:"ruleId"`
			FilePath    string `json:"filePath"`
			Description string `json:"description"`
			Applied     bool   `json:"applied"`
		} `json:"candidates"`
		Messages []string `json:"messages"`
		Error    string   `json:"error"`
	} `json:"providers"`
}

func (r deterministicReport) render(out io.Writer) {
	mode := "DRY-RUN (preview)"
	if r.Applied {
		mode = "APPLIED"
	}
	fmt.Fprintf(out, "Deterministic fix for %s — %s\n", r.Scenario, mode)
	fmt.Fprintf(out, "Total candidates: %d\n\n", r.TotalCandidates)
	for _, p := range r.Providers {
		fmt.Fprintf(out, "• %s [%s]\n", p.Provider, p.Status)
		for _, c := range p.Candidates {
			marker := "would fix"
			if c.Applied {
				marker = "fixed"
			}
			fmt.Fprintf(out, "    - [%s] %s (%s) — %s\n", marker, c.RuleID, c.FilePath, c.Description)
		}
		for _, m := range p.Messages {
			fmt.Fprintf(out, "    · %s\n", m)
		}
		if p.Error != "" {
			fmt.Fprintf(out, "    ! %s\n", p.Error)
		}
	}
	if r.TotalCandidates == 0 {
		fmt.Fprintln(out, "\nNo deterministic remediations available.")
	} else if !r.Applied {
		fmt.Fprintln(out, "\nRe-run with --apply to write these changes.")
	}
}

func runAgent(apiClient *cliutil.APIClient, scenario string, phases stringList, message string, asJSON bool, out io.Writer) error {
	if len(phases) == 0 {
		return errors.New("the agent fix path requires at least one --phase (or use --deterministic)")
	}
	phaseInfos := make([]map[string]string, 0, len(phases))
	for _, p := range phases {
		phaseInfos = append(phaseInfos, map[string]string{"name": p})
	}
	body := map[string]any{"phases": phaseInfos}
	if message != "" {
		body["message"] = message
	}
	path := fmt.Sprintf("/api/v1/scenarios/%s/fix", url.PathEscape(scenario))
	raw, err := apiClient.Request(http.MethodPost, path, nil, body)
	if err != nil {
		return fmt.Errorf("fix agent spawn failed: %w", err)
	}
	if asJSON {
		_, _ = out.Write(raw)
		return nil
	}
	fmt.Fprintf(out, "Fix agent spawned for %s.\n%s\n", scenario, strings.TrimSpace(string(raw)))
	return nil
}
