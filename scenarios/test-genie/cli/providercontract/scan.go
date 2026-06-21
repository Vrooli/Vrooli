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
	"test-genie/internal/selfhealth"
)

const scanUsage = "usage: provider-contract scan [--json] [--target <fixture-scenario>] [--timeout <dur>]"

// scanResolveRepoRoot is a test seam for the repository root resolver.
var scanResolveRepoRoot = cliutil.ResolveRepoRoot

// ScanArgs holds parsed `provider-contract scan` flags.
type ScanArgs struct {
	Target  string
	JSON    bool
	Timeout time.Duration
}

// ProviderReport is one provider's adoption scorecard. The snake_case JSON shape
// is the stable CLI contract; it is mapped from the shared selfhealth core.
type ProviderReport struct {
	Provider       string        `json:"provider"`
	Phase          string        `json:"phase"`
	Reachable      bool          `json:"reachable"`
	ContractValid  bool          `json:"contract_valid"`
	IdentityOK     bool          `json:"identity_ok"`
	SpecValid      bool          `json:"spec_valid"`
	MetricsAdopted bool          `json:"metrics_adopted"`
	AdoptionScore  float64       `json:"adoption_score"`
	Autofix        AutofixReport `json:"autofix"`
	Violations     []string      `json:"violations,omitempty"`
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
	Target    string           `json:"target"`
	Providers []ProviderReport `json:"providers"`
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
		if selfhealth.IsHardViolation(pr.SpecValid, pr.Reachable, pr.ContractValid, pr.IdentityOK, pr.MetricsAdopted) {
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
	out := ScanArgs{Target: selfhealth.DefaultScanTarget, Timeout: 30 * time.Second}
	fs.BoolVar(&out.JSON, "json", false, "Output JSON")
	fs.StringVar(&out.Target, "target", out.Target, "Fixture scenario each provider validates")
	fs.DurationVar(&out.Timeout, "timeout", out.Timeout, "Default per-provider probe timeout")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return ScanArgs{}, err
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
	report := selfhealth.ConformanceScanner{
		RepoRoot: scanResolveRepoRoot(),
		Target:   args.Target,
		Timeout:  args.Timeout,
	}.Scan(ctx)

	out := ScanReport{Target: report.Target}
	for _, pr := range report.Providers {
		out.Providers = append(out.Providers, ProviderReport{
			Provider:       pr.Provider,
			Phase:          pr.Phase,
			Reachable:      pr.Reachable,
			ContractValid:  pr.ContractValid,
			IdentityOK:     pr.IdentityOK,
			SpecValid:      pr.SpecValid,
			MetricsAdopted: pr.MetricsAdopted,
			AdoptionScore:  pr.AdoptionScore,
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

func printScanReport(report ScanReport) {
	fmt.Printf("Provider contract scan (target=%s)\n", report.Target)
	for _, pr := range report.Providers {
		fmt.Printf("  %-14s → %-28s adoption=%.0f%% reach=%s contract=%s identity=%s spec=%s metrics=%s\n",
			pr.Phase, pr.Provider, pr.AdoptionScore*100,
			yesno(pr.Reachable), yesno(pr.ContractValid), yesno(pr.IdentityOK), yesno(pr.SpecValid), yesno(pr.MetricsAdopted))
		fmt.Printf("      autofix: implemented=%d pending=%d manual=%d declared=%d/%d complete=%s\n",
			pr.Autofix.Implemented, pr.Autofix.Pending, pr.Autofix.Manual,
			pr.Autofix.Declared, pr.Autofix.Total, yesno(pr.Autofix.DeclarationComplete))
		for _, v := range pr.Violations {
			fmt.Printf("      - %s\n", v)
		}
	}
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
