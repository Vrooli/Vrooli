package selfhealth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	catalog "test-genie/internal/orchestrator/phases"
)

// DefaultScanTarget is the fixture scenario every provider is asked to validate
// during a conformance scan. It is always present (the orchestrator's own
// scenario) and, combined with include_execution=false, bounds the probe cost to
// each provider's static analysis path.
const DefaultScanTarget = "test-genie"

// defaultConformanceTimeout bounds each provider probe when the caller does not
// supply one.
const defaultConformanceTimeout = 30 * time.Second

// maxConformanceConcurrency bounds how many providers are probed in parallel.
const maxConformanceConcurrency = 6

// ConformanceProbe validates the fixture target against one provider and returns
// its ValidateScenario response. It is the seam tests inject to avoid live RPC.
type ConformanceProbe func(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error)

// ProviderConformance is one provider's adoption scorecard against the shared
// ScenarioValidationService contract.
type ProviderConformance struct {
	Provider       string   `json:"provider"`
	Phase          string   `json:"phase"`
	Reachable      bool     `json:"reachable"`
	ContractValid  bool     `json:"contractValid"`
	IdentityOK     bool     `json:"identityOk"`
	SpecValid      bool     `json:"specValid"`
	MetricsAdopted bool     `json:"metricsAdopted"`
	AdoptionScore  float64  `json:"adoptionScore"`
	Violations     []string `json:"violations,omitempty"`
}

// HasHardViolation reports whether this provider is mis-specified or
// contract-breaking among reachable providers. metrics_adopted is advisory and
// never counts; unreachability is environmental (a liveness signal) and is
// reported but not treated as a mis-specification.
func (r ProviderConformance) HasHardViolation() bool {
	if !r.SpecValid {
		return true
	}
	if r.Reachable && (!r.ContractValid || !r.IdentityOK) {
		return true
	}
	return false
}

// ConformanceReport is the fleet-wide conformance report.
type ConformanceReport struct {
	Target    string                `json:"target"`
	Providers []ProviderConformance `json:"providers"`
}

// HardViolations returns the "phase→provider" labels of every provider that is
// mis-specified or breaks the contract while reachable.
func (r ConformanceReport) HardViolations() []string {
	var offenders []string
	for _, pr := range r.Providers {
		if pr.HasHardViolation() {
			offenders = append(offenders, pr.Phase+"→"+pr.Provider)
		}
	}
	return offenders
}

// ConformanceScanner probes every delegated provider phase from the in-process
// catalog and scores its adoption. It is the single core used by both the CLI
// `provider-contract scan` verb and the API GetSelfHealth endpoint.
type ConformanceScanner struct {
	// RepoRoot is the repository root used to load each provider's shipped
	// maturity.json (scenarios/<provider>/.vrooli/maturity.json).
	RepoRoot string
	// Target is the fixture scenario each provider validates (DefaultScanTarget
	// when empty).
	Target string
	// Timeout is the default per-provider probe timeout (a delegated phase's own
	// timeout overrides it when larger).
	Timeout time.Duration
	// Probe is the validation seam; defaultConformanceProbe when nil.
	Probe ConformanceProbe
}

// Scan probes every delegated provider in bounded parallel and returns the
// report sorted by phase. The scan is live + time-boxed (fixture target,
// include_execution=false); freshness is therefore "live" at the call site.
func (s ConformanceScanner) Scan(ctx context.Context) ConformanceReport {
	target := strings.TrimSpace(s.Target)
	if target == "" {
		target = DefaultScanTarget
	}
	probe := s.Probe
	if probe == nil {
		probe = defaultConformanceProbe
	}

	type job struct {
		phase    string
		provider string
		timeout  time.Duration
	}
	var jobs []job
	for _, spec := range catalog.DefaultCatalog().All() {
		if spec.Delegated == nil {
			continue
		}
		timeout := s.Timeout
		if timeout <= 0 {
			timeout = defaultConformanceTimeout
		}
		if spec.Delegated.Timeout > timeout {
			timeout = spec.Delegated.Timeout
		}
		jobs = append(jobs, job{phase: spec.Name.String(), provider: spec.Delegated.ProviderScenario, timeout: timeout})
	}

	results := make([]ProviderConformance, len(jobs))
	sem := make(chan struct{}, maxConformanceConcurrency)
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, j job) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = scanProvider(ctx, probe, s.RepoRoot, target, j.phase, j.provider, j.timeout)
		}(i, j)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].Phase < results[j].Phase })
	return ConformanceReport{Target: target, Providers: results}
}

// scanProvider scores one provider: spec validity (local), then reachability +
// contract + identity + metrics adoption (live probe).
func scanProvider(ctx context.Context, probe ConformanceProbe, repoRoot, target, phase, provider string, timeout time.Duration) ProviderConformance {
	pr := ProviderConformance{Provider: provider, Phase: phase}

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

	resp, probeErr := probe(ctx, provider, target, timeout)
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

	// metrics_adopted is advisory during the partial rollout: reported, never a
	// violation.
	pr.MetricsAdopted = resp.GetMetrics() != nil

	pr.AdoptionScore = adoptionScore(pr)
	return pr
}

// adoptionScore is the fraction of the five adoption dimensions satisfied.
func adoptionScore(r ProviderConformance) float64 {
	satisfied := 0
	for _, ok := range []bool{r.Reachable, r.ContractValid, r.IdentityOK, r.SpecValid, r.MetricsAdopted} {
		if ok {
			satisfied++
		}
	}
	return float64(satisfied) / 5.0
}

func defaultConformanceProbe(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	if timeout <= 0 {
		timeout = defaultConformanceTimeout
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
