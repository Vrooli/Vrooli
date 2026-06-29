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
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	catalog "test-genie/internal/orchestrator/phases"
)

// DefaultPhaseMeta builds the phase→meta attribution map from the default phase
// catalog. The reliability ledger Builder needs it to attribute phases to
// providers; both the GetSelfHealth handler and the background snapshot sweeper
// derive it from this one source so they cannot drift.
func DefaultPhaseMeta() map[string]PhaseMeta {
	specs := catalog.DefaultCatalog().All()
	meta := make(map[string]PhaseMeta, len(specs))
	for _, spec := range specs {
		provider := ""
		if spec.Delegated != nil {
			provider = spec.Delegated.ProviderScenario
		}
		meta[spec.Name.String()] = PhaseMeta{
			Provider:  provider,
			Delegated: spec.Delegated != nil,
		}
	}
	return meta
}

// AuditorProviderScenario is the one delegated provider (the standards phase)
// with no Connect ScenarioValidationService. Its conformance scorecard is
// synthesized client-side rather than probed over Connect (A2-thin).
const AuditorProviderScenario = "scenario-auditor"

// resolveAuditorURL reports scenario-auditor's lifecycle reachability. It is a
// package var so tests can simulate a running/absent auditor without live
// discovery. A non-nil error means "not reachable" (environmental).
var resolveAuditorURL = func(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, AuditorProviderScenario)
}

// synthesizeAuditorScorecard builds scenario-auditor's ValidateScenarioResponse
// client-side: it confirms the auditor is reachable (lifecycle presence), loads
// its shipped maturity.json, and assembles a contract-valid assessment plus
// client-synthesized ExecutionMetrics (gauge client_synthesized=1). The metrics
// reflect the test-genie-side synthesis cost, not the auditor's internal work —
// the honest caveat documented for A2-thin.
func synthesizeAuditorScorecard(ctx context.Context, repoRoot, target string) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", AuditorProviderScenario))
	if err != nil {
		return nil, fmt.Errorf("load scenario-auditor maturity spec: %w", err)
	}
	if _, err := resolveAuditorURL(ctx); err != nil {
		return nil, fmt.Errorf("scenario-auditor not reachable: %w", err)
	}
	collector := metrics.Start()
	collector.Gauge("client_synthesized", 1)
	a, err := assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: target, Spec: *spec})
	if err != nil {
		collector.Stop()
		return nil, fmt.Errorf("build scenario-auditor assessment: %w", err)
	}
	execMetrics := collector.Stop()
	return assessment.BuildValidationResponse(target, a, nil, execMetrics)
}

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
	// Autofix is the spec-derived autofix declaration rollup (Stage 1). It is the
	// 6th, advisory conformance dimension: DeclarationComplete is the gated-later
	// signal (every finding classified, every manual justified); Pending is the
	// headline backlog. It is always computed from the shipped spec — available
	// even when the provider is unreachable — and never enters AdoptionScore or
	// the hard-violation set.
	Autofix assessment.AutofixCoverage `json:"autofix"`
}

// IsHardViolation is the single source of truth for whether a delegated
// provider's conformance scorecard is a contract violation (as opposed to an
// environmental/liveness signal). It is called by the API conformance method,
// the `provider-contract scan` CLI, and the `test-genie health` CLI so the
// rule never drifts across the three surfaces (utils-unification).
//
// spec_valid is a local, always-evaluable requirement. contract_valid,
// identity_ok, and metrics_adopted are judged only when the provider is
// reachable — an unreachable provider is an environmental signal (reported,
// not a mis-specification). Now that the delegated fleet has adopted
// ExecutionMetrics (Plan 3, Part A), a reachable provider that stops emitting
// metrics is a hard violation: metrics_adopted is no longer advisory.
func IsHardViolation(specValid, reachable, contractValid, identityOK, metricsAdopted bool) bool {
	if !specValid {
		return true
	}
	if reachable && (!contractValid || !identityOK || !metricsAdopted) {
		return true
	}
	return false
}

// HasHardViolation reports whether this provider is mis-specified or
// contract-breaking among reachable providers. It delegates to the
// IsHardViolation SSOT so the API and both CLIs share one rule.
func (r ProviderConformance) HasHardViolation() bool {
	return IsHardViolation(r.SpecValid, r.Reachable, r.ContractValid, r.IdentityOK, r.MetricsAdopted)
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
	// Subject optionally narrows the scan to one delegated phase or provider.
	// Empty scans every delegated provider.
	Subject string
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
	subject := catalog.NormalizeKey(s.Subject)
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
		phase := spec.Name.String()
		provider := spec.Delegated.ProviderScenario
		if subject != "" && subject != phase && subject != catalog.NormalizeKey(provider) {
			continue
		}
		timeout := s.Timeout
		if timeout <= 0 {
			timeout = defaultConformanceTimeout
		}
		if spec.Delegated.Timeout > timeout {
			timeout = spec.Delegated.Timeout
		}
		jobs = append(jobs, job{phase: phase, provider: provider, timeout: timeout})
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
	// Autofix coverage is a declaration property — compute it whenever the spec
	// parsed, even if its identity mismatched or the provider is unreachable.
	if spec != nil {
		pr.Autofix = assessment.ComputeAutofixCoverage(*spec)
	}

	// scenario-auditor (the standards phase) has no Connect ScenarioValidationService
	// — it is REST/Postgres. The generic Connect probe cannot reach it, so we
	// synthesize its scorecard client-side (A2-thin): reachability is its
	// lifecycle presence, and the response (assessment + client-synthesized
	// metrics) is built from its shipped maturity.json. This lets standards
	// participate in the metrics-required gate without a REST→Connect rewrite.
	var (
		resp     *scenariovalidationv1.ValidateScenarioResponse
		probeErr error
	)
	if provider == AuditorProviderScenario {
		resp, probeErr = synthesizeAuditorScorecard(ctx, repoRoot, target)
	} else {
		resp, probeErr = probe(ctx, provider, target, timeout)
	}
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

	// metrics_adopted is a hard requirement among reachable providers (Plan 3
	// Part B). For scenario-auditor it is client-synthesized; for the Connect
	// fleet it is provider-emitted.
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
