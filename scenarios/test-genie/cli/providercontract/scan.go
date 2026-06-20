package providercontract

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	catalog "test-genie/internal/orchestrator/phases"
)

// defaultScanTarget is the fixture scenario every provider is asked to validate
// during a conformance scan. It is always present (the orchestrator's own
// scenario) and, combined with include_execution=false, bounds the probe cost to
// each provider's static analysis path.
const defaultScanTarget = "test-genie"

const scanUsage = "usage: provider-contract scan [--json] [--target <fixture-scenario>] [--timeout <dur>]"

// Test seams: overridden by unit tests to avoid live RPC and the real repo tree.
var (
	scanResolveRepoRoot = cliutil.ResolveRepoRoot
	scanProbe           = defaultScanProbe
)

// ScanArgs holds parsed `provider-contract scan` flags.
type ScanArgs struct {
	Target  string
	JSON    bool
	Timeout time.Duration
}

// ProviderReport is one provider's adoption scorecard.
type ProviderReport struct {
	Provider       string   `json:"provider"`
	Phase          string   `json:"phase"`
	Reachable      bool     `json:"reachable"`
	ContractValid  bool     `json:"contract_valid"`
	IdentityOK     bool     `json:"identity_ok"`
	SpecValid      bool     `json:"spec_valid"`
	MetricsAdopted bool     `json:"metrics_adopted"`
	AdoptionScore  float64  `json:"adoption_score"`
	Violations     []string `json:"violations,omitempty"`
}

// hasHardViolation reports whether this provider is mis-specified or
// contract-breaking among reachable providers. metrics_adopted is advisory and
// never counts; unreachability is environmental (a liveness signal) and is
// reported but not treated as a mis-specification.
func (r ProviderReport) hasHardViolation() bool {
	if !r.SpecValid {
		return true
	}
	if r.Reachable && (!r.ContractValid || !r.IdentityOK) {
		return true
	}
	return false
}

// ScanReport is the fleet-wide conformance report.
type ScanReport struct {
	Target    string           `json:"target"`
	Providers []ProviderReport `json:"providers"`
}

// RunScan parses args, runs the scan, prints the report, and returns a non-zero
// error when any provider is mis-specified or breaks the contract while
// reachable (metrics adoption and unreachability never fail the gate).
func RunScan(args []string) error {
	parsed, err := ParseScanArgs(args)
	if err != nil {
		return err
	}
	report, err := Scan(context.Background(), parsed)
	if err != nil {
		return err
	}
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
		if pr.hasHardViolation() {
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
	out := ScanArgs{Target: defaultScanTarget, Timeout: 30 * time.Second}
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

// Scan probes every delegated provider phase from the in-process catalog and
// scores its adoption of the shared validation contract.
func Scan(ctx context.Context, args ScanArgs) (ScanReport, error) {
	repoRoot := scanResolveRepoRoot()
	report := ScanReport{Target: args.Target}
	for _, spec := range catalog.DefaultCatalog().All() {
		if spec.Delegated == nil {
			continue
		}
		timeout := args.Timeout
		if spec.Delegated.Timeout > 0 {
			timeout = spec.Delegated.Timeout
		}
		report.Providers = append(report.Providers, scanProvider(
			ctx, repoRoot, args.Target, spec.Name.String(), spec.Delegated.ProviderScenario, timeout,
		))
	}
	sort.Slice(report.Providers, func(i, j int) bool {
		return report.Providers[i].Phase < report.Providers[j].Phase
	})
	return report, nil
}

func scanProvider(ctx context.Context, repoRoot, target, phase, provider string, timeout time.Duration) ProviderReport {
	pr := ProviderReport{Provider: provider, Phase: phase}

	// spec_valid is a local check, independent of whether the provider is live.
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", provider))
	switch {
	case err != nil:
		pr.Violations = append(pr.Violations, fmt.Sprintf("maturity spec invalid: %v", err))
	case spec.Provider != provider || spec.Phase != phase:
		pr.Violations = append(pr.Violations, fmt.Sprintf(
			"maturity spec identity mismatch: provider=%q phase=%q, want provider=%q phase=%q",
			spec.Provider, spec.Phase, provider, phase))
	default:
		pr.SpecValid = true
	}

	// reachability + contract + identity + metrics require a live probe.
	resp, probeErr := scanProbe(ctx, provider, target, timeout)
	if probeErr != nil {
		pr.Violations = append(pr.Violations, fmt.Sprintf("unreachable: %v", probeErr))
		pr.AdoptionScore = adoptionScore(pr)
		return pr
	}
	pr.Reachable = true

	a := resp.GetAssessment()
	contractErr := assessment.ValidateAssessment(a)
	if resp.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		contractErr = errors.New("validation status is unspecified")
	}
	pr.ContractValid = contractErr == nil
	if contractErr != nil {
		pr.Violations = append(pr.Violations, fmt.Sprintf("contract invalid: %v", contractErr))
	}

	identErr := assessment.RequireIdentity(provider, phase, a)
	pr.IdentityOK = identErr == nil
	if identErr != nil && contractErr == nil {
		pr.Violations = append(pr.Violations, fmt.Sprintf("identity: %v", identErr))
	}

	// metrics_adopted is advisory during the partial rollout: reported but never
	// a violation.
	pr.MetricsAdopted = resp.GetMetrics() != nil

	pr.AdoptionScore = adoptionScore(pr)
	return pr
}

// adoptionScore is the fraction of the five adoption dimensions satisfied.
func adoptionScore(r ProviderReport) float64 {
	satisfied := 0
	for _, ok := range []bool{r.Reachable, r.ContractValid, r.IdentityOK, r.SpecValid, r.MetricsAdopted} {
		if ok {
			satisfied++
		}
	}
	return float64(satisfied) / 5.0
}

func defaultScanProbe(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("resolve %s URL: %w", provider, err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("%s base URL is empty", provider)
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	resp, err := client.ValidateScenario(runCtx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         target,
		IncludeExecution: false,
	}))
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil {
		return nil, errors.New("empty validation response")
	}
	return resp.Msg, nil
}

func printScanReport(report ScanReport) {
	fmt.Printf("Provider contract scan (target=%s)\n", report.Target)
	for _, pr := range report.Providers {
		fmt.Printf("  %-14s → %-28s adoption=%.0f%% reach=%s contract=%s identity=%s spec=%s metrics=%s\n",
			pr.Phase, pr.Provider, pr.AdoptionScore*100,
			yesno(pr.Reachable), yesno(pr.ContractValid), yesno(pr.IdentityOK), yesno(pr.SpecValid), yesno(pr.MetricsAdopted))
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
