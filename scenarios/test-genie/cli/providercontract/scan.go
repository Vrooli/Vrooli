package providercontract

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	catalog "test-genie/internal/orchestrator/phases"
	"test-genie/internal/selfhealth"
)

const scanUsage = "usage: provider-contract scan [<phase-or-provider>] [--json] [--target <fixture-scenario>] [--timeout <dur>] [--restart]"

// scanResolveRepoRoot is a test seam for the repository root resolver.
var scanResolveRepoRoot = cliutil.ResolveRepoRoot

// ScanArgs holds parsed `provider-contract scan` flags.
type ScanArgs struct {
	Target  string
	Subject string
	JSON    bool
	Timeout time.Duration
	Restart bool
}

// ProviderReport is one provider's adoption scorecard. The snake_case JSON shape
// is the stable CLI contract; it is mapped from the shared selfhealth core.
type ProviderReport struct {
	Provider       string   `json:"provider"`
	Phase          string   `json:"phase"`
	Classification string   `json:"classification"`
	ReasonCodes    []string `json:"reason_codes,omitempty"`
	Reachable      bool     `json:"reachable"`
	ContractValid  bool     `json:"contract_valid"`
	IdentityOK     bool     `json:"identity_ok"`
	// SpecValid means the provider-owned Test Genie descriptor loads, its
	// embedded maturity block validates, and descriptor/provider/phase identity
	// is coherent. The JSON key remains `spec_valid` for CLI compatibility.
	SpecValid           bool          `json:"spec_valid"`
	MetricsAdopted      bool          `json:"metrics_adopted"`
	MetricsReachable    bool          `json:"metrics_reachable"`
	ConcurrencyDeclared bool          `json:"concurrency_declared"`
	FixContractRequired bool          `json:"fix_contract_required"`
	FixContractValid    bool          `json:"fix_contract_valid"`
	AdoptionScore       float64       `json:"adoption_score"`
	Autofix             AutofixReport `json:"autofix"`
	Violations          []string      `json:"violations,omitempty"`
}

// AutofixReport is the per-provider autofix declaration rollup (advisory). The
// headline is `pending` — the actionable fixer backlog.
type AutofixReport struct {
	Total               int     `json:"total"`
	FixableUniverse     int     `json:"fixable_universe"`
	Implemented         int     `json:"implemented"`
	Pending             int     `json:"pending"`
	Manual              int     `json:"manual"`
	Declared            int     `json:"declared"`
	DeclarationComplete bool    `json:"declaration_complete"`
	ImplementationRate  float64 `json:"implementation_rate"`
}

// ScanReport is the fleet-wide conformance report.
type ScanReport struct {
	Target   string          `json:"target"`
	Restarts []RestartReport `json:"restarts,omitempty"`

	Providers []ProviderReport `json:"providers"`
}

// RestartReport records the optional lifecycle restart pass that can precede a
// provider contract scan. It is advisory because the scan still reports the
// authoritative live contract state after restart attempts finish.
type RestartReport struct {
	Provider string  `json:"provider"`
	Phase    string  `json:"phase"`
	Duration float64 `json:"duration_seconds"`
	OK       bool    `json:"ok"`
	Error    string  `json:"error,omitempty"`
}

// RunScan parses args, runs the scan via the shared selfhealth conformance core,
// prints the report, and returns a non-zero error when any provider is
// mis-specified, or — while reachable — breaks the contract, mismatches
// identity, or has dropped ExecutionMetrics. Unreachability is environmental
// and never fails the gate.
func RunScan(args []string) error {
	parsed, err := ParseScanArgs(args)
	if err != nil {
		return err
	}
	report := Scan(context.Background(), parsed)
	if parsed.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		printScanReport(report)
	}

	var offenders []string
	for _, pr := range report.Providers {
		if hardViolation(pr) {
			offenders = append(offenders, pr.Phase+"→"+pr.Provider)
		}
	}
	if len(offenders) > 0 {
		return fmt.Errorf("provider contract violations: %s", strings.Join(offenders, ", "))
	}
	return nil
}

// ParseScanArgs parses the `scan` subcommand flags.
func ParseScanArgs(args []string) (ScanArgs, error) {
	if len(args) == 0 || args[0] != "scan" {
		return ScanArgs{}, errors.New(scanUsage)
	}
	fs := flag.NewFlagSet("provider-contract scan", flag.ContinueOnError)
	fs.SetOutput(flag.CommandLine.Output())
	out := ScanArgs{Target: selfhealth.DefaultScanTarget, Timeout: time.Minute}
	fs.BoolVar(&out.JSON, "json", false, "Output JSON")
	fs.StringVar(&out.Target, "target", out.Target, "Fixture scenario each provider validates")
	fs.DurationVar(&out.Timeout, "timeout", out.Timeout, "Default per-provider probe timeout")
	fs.BoolVar(&out.Restart, "restart", false, "Restart delegated provider scenarios before scanning")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return ScanArgs{}, err
	}
	remaining := fs.Args()
	switch len(remaining) {
	case 0:
	case 1:
		out.Subject = catalog.NormalizeKey(remaining[0])
		if _, err := ResolveProbe(out.Subject); err != nil {
			return ScanArgs{}, err
		}
	default:
		return ScanArgs{}, errors.New(scanUsage)
	}
	out.Target = strings.TrimSpace(out.Target)
	if out.Target == "" {
		return ScanArgs{}, errors.New(scanUsage)
	}
	return out, nil
}

// Scan runs the shared conformance core and maps the result to the CLI report
// shape.
func Scan(ctx context.Context, args ScanArgs) ScanReport {
	var restarts []RestartReport
	if args.Restart {
		restarts = restartScanProviders(ctx, args.Timeout, args.Subject)
	}
	report := selfhealth.ConformanceScanner{
		RepoRoot: scanResolveRepoRoot(),
		Target:   args.Target,
		Subject:  args.Subject,
		Timeout:  args.Timeout,
	}.Scan(ctx)

	out := ScanReport{Target: report.Target, Restarts: restarts}
	for _, pr := range report.Providers {
		out.Providers = append(out.Providers, ProviderReport{
			Provider:            pr.Provider,
			Phase:               pr.Phase,
			Classification:      string(pr.Classification),
			ReasonCodes:         append([]string(nil), pr.ReasonCodes...),
			Reachable:           pr.Reachable,
			ContractValid:       pr.ContractValid,
			IdentityOK:          pr.IdentityOK,
			SpecValid:           pr.SpecValid,
			MetricsAdopted:      pr.MetricsAdopted,
			MetricsReachable:    pr.MetricsReachable,
			ConcurrencyDeclared: pr.ConcurrencyDeclared,
			FixContractRequired: pr.FixContractRequired,
			FixContractValid:    pr.FixContractValid,
			AdoptionScore:       pr.AdoptionScore,
			Autofix: AutofixReport{
				Total:               pr.Autofix.Total,
				FixableUniverse:     pr.Autofix.FixableUniverse,
				Implemented:         pr.Autofix.Implemented,
				Pending:             pr.Autofix.Pending,
				Manual:              pr.Autofix.Manual,
				Declared:            pr.Autofix.Declared,
				DeclarationComplete: pr.Autofix.DeclarationComplete,
				ImplementationRate:  pr.Autofix.ImplementationRate(),
			},
			Violations: pr.Violations,
		})
	}
	return out
}

func restartScanProviders(ctx context.Context, timeout time.Duration, subject string) []RestartReport {
	subject = catalog.NormalizeKey(subject)
	seen := map[string]bool{}
	var out []RestartReport
	for _, probe := range Probes() {
		if subject != "" && subject != probe.Phase && subject != catalog.NormalizeKey(probe.Provider) {
			continue
		}
		if probe.Provider == "" || seen[probe.Provider] {
			continue
		}
		seen[probe.Provider] = true
		fmt.Fprintf(os.Stderr, "restarting provider %s for phase %s\n", probe.Provider, probe.Phase)
		started := time.Now()
		_, err := commandRunner(ctx, timeout, "", "vrooli", "scenario", "restart", probe.Provider)
		report := RestartReport{
			Provider: probe.Provider,
			Phase:    probe.Phase,
			Duration: time.Since(started).Seconds(),
			OK:       err == nil,
		}
		if err != nil {
			report.Error = err.Error()
			fmt.Fprintf(os.Stderr, "restart provider %s failed: %v\n", probe.Provider, err)
		}
		out = append(out, report)
	}
	return out
}

func printScanReport(report ScanReport) {
	fmt.Printf("Provider contract scan (target=%s)\n", report.Target)
	for _, pr := range report.Providers {
		fmt.Printf("  %-14s → %-28s state=%-11s adoption=%.0f%% reach=%s contract=%s identity=%s spec=%s metrics=%s concurrency=%s fixes=%s\n",
			pr.Phase, pr.Provider, pr.Classification, pr.AdoptionScore*100,
			yesno(pr.Reachable), yesno(pr.ContractValid), yesno(pr.IdentityOK), yesno(pr.SpecValid), yesno(pr.MetricsAdopted), yesno(pr.ConcurrencyDeclared), fixContractStatus(pr))
		if len(pr.ReasonCodes) > 0 {
			fmt.Printf("      reasons: %s\n", strings.Join(pr.ReasonCodes, ", "))
		}
		fmt.Printf("      autofix: implemented=%d pending=%d manual=%d declared=%d/%d complete=%s\n",
			pr.Autofix.Implemented, pr.Autofix.Pending, pr.Autofix.Manual,
			pr.Autofix.Declared, pr.Autofix.Total, yesno(pr.Autofix.DeclarationComplete))
		for _, v := range pr.Violations {
			fmt.Printf("      - %s\n", v)
		}
	}
}

func hardViolation(pr ProviderReport) bool {
	return selfhealth.IsHardViolation(pr.SpecValid, pr.Reachable, pr.ContractValid, pr.IdentityOK, pr.MetricsAdopted) || (pr.FixContractRequired && !pr.FixContractValid)
}

func fixContractStatus(pr ProviderReport) string {
	if !pr.FixContractRequired {
		return "n/a"
	}
	return yesno(pr.FixContractValid)
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
