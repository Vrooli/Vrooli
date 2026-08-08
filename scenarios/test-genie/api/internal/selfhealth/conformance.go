package selfhealth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	"test-genie/internal/orchestrator/providerdescriptor"
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

// DefaultScanTarget is the fixture scenario most providers are asked to
// validate during a conformance scan. ProviderDefaultTarget supplies an
// applicability-valid fixture for the exceptional providers.
const DefaultScanTarget = "test-genie"

// ProviderDefaultTarget returns an applicability-valid fixture for providers
// that cannot inspect Test Genie's own scenario. An explicit scanner Target
// always takes precedence over this table.
func ProviderDefaultTarget(provider string) string {
	switch strings.TrimSpace(provider) {
	case "experience-manager":
		return "experience-manager"
	default:
		return DefaultScanTarget
	}
}

// defaultConformanceTimeout bounds each provider probe when the caller does not
// supply one.
const defaultConformanceTimeout = 30 * time.Second

// maxConformanceConcurrency bounds how many providers are probed in parallel.
const maxConformanceConcurrency = 6

// ConformanceProbe validates the fixture target against one provider and returns
// its ValidateScenario response. It is the seam tests inject to avoid live RPC.
type ConformanceProbe func(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error)

// FixConformanceProbe validates a provider's deterministic-fix lifecycle
// against an isolated fixture path. It is intentionally separate from the
// validation probe: PreviewFix/ApplyFix have different safety semantics.
type FixConformanceProbe func(ctx context.Context, provider, target, path string, ruleIDs []string, timeout time.Duration) (*scenariovalidationv1.FixResponse, *scenariovalidationv1.FixResponse, error)

// ConformanceClassification is the operator-facing, mutually exclusive fleet
// posture for one catalog phase. It keeps an unavailable provider distinct from
// a deterministic descriptor or response violation, and makes native phases
// explicit exemptions instead of silently omitting them from the inventory.
type ConformanceClassification string

const (
	ConformanceCompliant   ConformanceClassification = "compliant"
	ConformanceUnavailable ConformanceClassification = "unavailable"
	ConformanceExempt      ConformanceClassification = "exempt"
	ConformanceViolation   ConformanceClassification = "violation"
)

const (
	ReasonNativePhase         = "native_phase"
	ReasonDescriptorInvalid   = "descriptor_invalid"
	ReasonProviderUnreachable = "provider_unreachable"
	ReasonPresentationInvalid = "presentation_invalid"
	ReasonIdentityMismatch    = "identity_mismatch"
	ReasonMetricsMissing      = "metrics_missing"
	ReasonFixContractInvalid  = "fix_contract_invalid"
	ReasonDescribeNotAdopted  = "describe_provider_not_adopted"
	ReasonConcurrencyMissing  = "concurrency_declaration_missing"
)

// ProviderConformance is one provider's adoption scorecard against the shared
// ScenarioValidationService contract.
type ProviderConformance struct {
	Provider            string                    `json:"provider"`
	Phase               string                    `json:"phase"`
	Classification      ConformanceClassification `json:"classification"`
	ReasonCodes         []string                  `json:"reason_codes,omitempty"`
	Reachable           bool                      `json:"reachable"`
	ContractValid       bool                      `json:"contractValid"`
	IdentityOK          bool                      `json:"identityOk"`
	SpecValid           bool                      `json:"specValid"`
	MetricsAdopted      bool                      `json:"metricsAdopted"`
	MetricsReachable    bool                      `json:"metricsReachable"`
	ConcurrencyDeclared bool                      `json:"concurrencyDeclared"`
	// DescribeAdopted reports whether the provider answers DescribeProvider.
	// A provider that does not forces readiness onto the legacy ValidateScenario
	// probe, which for an inspection-only provider costs a full target analysis
	// on every suite run — the duplicate work DescribeProvider exists to remove.
	// Tracking it here keeps that cost visible instead of silently returning.
	DescribeAdopted     bool     `json:"describeAdopted"`
	FixContractRequired bool     `json:"fixContractRequired"`
	FixContractValid    bool     `json:"fixContractValid"`
	AdoptionScore       float64  `json:"adoptionScore"`
	Violations          []string `json:"violations,omitempty"`
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
	return IsHardViolation(r.SpecValid, r.Reachable, r.ContractValid, r.IdentityOK, r.MetricsAdopted) || (r.FixContractRequired && !r.FixContractValid)
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
	// descriptor (scenarios/<provider>/.vrooli/test-genie.json).
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
	// FixProbe runs PreviewFix then ApplyFix only against an isolated fixture
	// directory. Nil selects DefaultFixConformanceProbe.
	FixProbe FixConformanceProbe
	// StoredMetrics verifies that a reachable provider's metrics were actually
	// written to the execution history. Live response presence alone is not
	// durable adoption evidence.
	StoredMetrics func(context.Context, string, string) bool
}

// Scan probes every delegated provider in bounded parallel and returns the
// report sorted by phase. The scan is live + time-boxed (fixture target,
// include_execution=false); freshness is therefore "live" at the call site.
func (s ConformanceScanner) Scan(ctx context.Context) ConformanceReport {
	target := strings.TrimSpace(s.Target)
	usesDefaultTarget := target == ""
	if target == "" {
		target = DefaultScanTarget
	}
	subject := catalog.NormalizeKey(s.Subject)
	probe := s.Probe
	if probe == nil {
		probe = defaultConformanceProbe
	}
	fixProbe := s.FixProbe
	if fixProbe == nil {
		fixProbe = DefaultFixConformanceProbe
	}

	type job struct {
		phase    string
		provider string
		target   string
		timeout  time.Duration
	}
	var (
		jobs    []job
		results []ProviderConformance
	)
	for _, spec := range catalog.DefaultCatalog().All() {
		phase := spec.Name.String()
		if subject != "" && subject != phase {
			if spec.Delegated == nil || subject != catalog.NormalizeKey(spec.Delegated.ProviderScenario) {
				continue
			}
		}
		if spec.Delegated == nil {
			results = append(results, nativePhaseExemption(phase))
			continue
		}
		provider := spec.Delegated.ProviderScenario
		timeout := s.Timeout
		if timeout <= 0 {
			timeout = defaultConformanceTimeout
		}
		if spec.Delegated.Timeout > timeout {
			timeout = spec.Delegated.Timeout
		}
		jobTarget := target
		if usesDefaultTarget {
			jobTarget = ProviderDefaultTarget(provider)
		}
		jobs = append(jobs, job{phase: phase, provider: provider, target: jobTarget, timeout: timeout})
	}

	firstProbeResult := len(results)
	results = append(results, make([]ProviderConformance, len(jobs))...)
	sem := make(chan struct{}, maxConformanceConcurrency)
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, j job) {
			defer wg.Done()
			defer func() { <-sem }()
			pr, _ := CheckProvider(ctx, probe, fixProbe, s.RepoRoot, j.target, j.phase, j.provider, j.timeout)
			if pr.Reachable && s.StoredMetrics != nil {
				pr.MetricsAdopted = s.StoredMetrics(ctx, j.provider, j.phase)
				finishConformance(&pr)
			}
			results[firstProbeResult+i] = pr
		}(i, j)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].Phase < results[j].Phase })
	return ConformanceReport{Target: target, Providers: results}
}

func nativePhaseExemption(phase string) ProviderConformance {
	return ProviderConformance{
		Phase:          phase,
		Classification: ConformanceExempt,
		ReasonCodes:    []string{ReasonNativePhase},
	}
}

// ScanProvider scores one provider: spec validity (local), then reachability +
// contract + identity + metrics adoption (live probe). It is exported so the
// provider-conformance validation phase reuses the exact same adoption rules
// the fleet scan applies (no second implementation to drift).
func ScanProvider(ctx context.Context, probe ConformanceProbe, repoRoot, target, phase, provider string, timeout time.Duration) ProviderConformance {
	return ScanProviderWithFixProbe(ctx, probe, nil, repoRoot, target, phase, provider, timeout)
}

// ScanProviderWithFixProbe scores one provider and, when the descriptor
// declares implemented auto-fixes, proves the PreviewFix/ApplyFix lifecycle
// through the supplied isolated-fixture probe.
func ScanProviderWithFixProbe(ctx context.Context, probe ConformanceProbe, fixProbe FixConformanceProbe, repoRoot, target, phase, provider string, timeout time.Duration) ProviderConformance {
	pr, _ := CheckProvider(ctx, probe, fixProbe, repoRoot, target, phase, provider, timeout)
	return pr
}

// CheckProvider scores one provider with the exact rules the fleet scan
// applies and additionally returns the live ValidateScenario response (nil
// when the provider was unreachable). Single-provider diagnostics such as
// `provider-contract check` use it to render assessment detail from the same
// probed response the score was computed from, so the displayed evidence can
// never diverge from the verdict. The contract clauses evaluated here are the
// granular pieces of assessment.RequireProviderContract plus the scan-only
// dimensions (descriptor spec, metrics adoption, fix-contract proof).
func CheckProvider(ctx context.Context, probe ConformanceProbe, fixProbe FixConformanceProbe, repoRoot, target, phase, provider string, timeout time.Duration) (ProviderConformance, *scenariovalidationv1.ValidateScenarioResponse) {
	pr := ProviderConformance{Provider: provider, Phase: phase}

	// spec_valid is a local check, independent of whether the provider is live.
	spec := loadProviderDescriptorSpec(&pr, repoRoot, provider)
	switch {
	case spec == nil:
	case spec.Provider != provider || spec.Phase != phase:
		pr.Violations = append(pr.Violations, fmt.Sprintf(
			"descriptor maturity identity mismatch: provider=%q phase=%q, want provider=%q phase=%q",
			spec.Provider, spec.Phase, provider, phase))
	default:
		pr.SpecValid = true
	}
	// Autofix coverage is a declaration property — compute it whenever the spec
	// parsed, even if its identity mismatched or the provider is unreachable.
	if spec != nil {
		pr.Autofix = assessment.ComputeAutofixCoverage(*spec)
		pr.FixContractRequired = len(implementedFixRuleIDs(*spec)) > 0
	}
	pr.ConcurrencyDeclared = providerConcurrencyDeclared(repoRoot, provider)

	resp, probeErr := probe(ctx, provider, target, timeout)
	if probeErr != nil {
		pr.Violations = append(pr.Violations, fmt.Sprintf("unreachable: %v", probeErr))
		finishConformance(&pr)
		return pr, nil
	}
	pr.Reachable = true

	a := resp.GetAssessment()
	contractErr := assessment.ValidateAssessment(a)
	if contractErr == nil {
		contractErr = assessment.ValidatePhasePresentation(a)
	}
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
	// Part B). The provider fleet emits it through the shared
	// ScenarioValidationService contract.
	pr.MetricsReachable = resp.GetMetrics() != nil
	pr.MetricsAdopted = pr.MetricsReachable

	// DescribeProvider adoption is advisory for now: it lowers the adoption
	// score and names the cost, but does not classify the provider as a
	// violation while the fleet is still migrating. Promote it to a reason code
	// (and into IsHardViolation) once every provider reports it, exactly as
	// metrics_adopted was promoted.
	if probeFn := DescribeProbe; probeFn != nil {
		switch err := probeFn(ctx, provider, timeout); {
		case err == nil:
			pr.DescribeAdopted = true
		case connect.CodeOf(err) == connect.CodeUnimplemented:
			pr.Violations = append(pr.Violations,
				"DescribeProvider not adopted: readiness falls back to a full ValidateScenario probe, which re-runs this provider's entire analysis on every suite run")
		default:
			pr.Violations = append(pr.Violations, fmt.Sprintf("DescribeProvider probe failed: %v", err))
		}
	}
	if pr.FixContractRequired {
		if fixProbe == nil {
			pr.Violations = append(pr.Violations, "implemented auto-fixes were not contract-probed")
		} else if err := probeFixContract(ctx, fixProbe, provider, target, implementedFixRuleIDs(*spec), timeout); err != nil {
			pr.Violations = append(pr.Violations, fmt.Sprintf("fix contract invalid: %v", err))
		} else {
			pr.FixContractValid = true
		}
	}

	finishConformance(&pr)
	return pr, resp
}

func finishConformance(pr *ProviderConformance) {
	if pr == nil {
		return
	}
	pr.AdoptionScore = adoptionScore(*pr)
	pr.ReasonCodes = pr.ReasonCodes[:0]
	switch {
	case !pr.SpecValid:
		pr.Classification = ConformanceViolation
		pr.ReasonCodes = append(pr.ReasonCodes, ReasonDescriptorInvalid)
	case !pr.Reachable:
		pr.Classification = ConformanceUnavailable
		pr.ReasonCodes = append(pr.ReasonCodes, ReasonProviderUnreachable)
	default:
		if !pr.ContractValid {
			pr.ReasonCodes = append(pr.ReasonCodes, ReasonPresentationInvalid)
		}
		if !pr.IdentityOK {
			pr.ReasonCodes = append(pr.ReasonCodes, ReasonIdentityMismatch)
		}
		if !pr.MetricsAdopted {
			pr.ReasonCodes = append(pr.ReasonCodes, ReasonMetricsMissing)
		}
		if !pr.ConcurrencyDeclared {
			pr.ReasonCodes = append(pr.ReasonCodes, ReasonConcurrencyMissing)
		}
		if pr.FixContractRequired && !pr.FixContractValid {
			pr.ReasonCodes = append(pr.ReasonCodes, ReasonFixContractInvalid)
		}
		if len(pr.ReasonCodes) == 0 {
			pr.Classification = ConformanceCompliant
		} else {
			pr.Classification = ConformanceViolation
		}
	}
}

func providerConcurrencyDeclared(repoRoot, provider string) bool {
	if strings.TrimSpace(repoRoot) == "" || strings.TrimSpace(provider) == "" {
		return true
	}
	raw, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", provider, providerdescriptor.RelPath))
	if err != nil {
		return true
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	_, ok := payload["concurrency"]
	return ok
}

func implementedFixRuleIDs(spec assessment.Spec) []string {
	ids := make([]string, 0, len(spec.Findings))
	for code, mapping := range spec.Findings {
		if mapping.FixClass == assessment.FixClassAuto && mapping.FixerStatus == assessment.FixerStatusImplemented {
			ids = append(ids, code)
		}
	}
	sort.Strings(ids)
	return ids
}

func probeFixContract(ctx context.Context, probe FixConformanceProbe, provider, target string, ruleIDs []string, timeout time.Duration) error {
	root, err := os.MkdirTemp("", "test-genie-fix-contract-")
	if err != nil {
		return fmt.Errorf("create isolated fixture: %w", err)
	}
	defer os.RemoveAll(root)
	preview, applied, err := probe(ctx, provider, target, root, ruleIDs, timeout)
	if err != nil {
		return err
	}
	return validateFixContract(root, target, preview, applied)
}

func validateFixContract(root, target string, preview, applied *scenariovalidationv1.FixResponse) error {
	if preview == nil || applied == nil {
		return errors.New("empty PreviewFix or ApplyFix response")
	}
	if preview.GetScenario() != target || applied.GetScenario() != target {
		return fmt.Errorf("response scenario mismatch: preview=%q apply=%q want=%q", preview.GetScenario(), applied.GetScenario(), target)
	}
	if preview.GetApplied() {
		return errors.New("PreviewFix response is marked applied")
	}
	if len(preview.GetCandidates()) != len(applied.GetCandidates()) {
		return fmt.Errorf("preview/apply candidate count mismatch: %d != %d", len(preview.GetCandidates()), len(applied.GetCandidates()))
	}
	if applied.GetApplied() != (len(applied.GetCandidates()) > 0) {
		return fmt.Errorf("ApplyFix applied=%t with %d candidate(s)", applied.GetApplied(), len(applied.GetCandidates()))
	}
	for i, candidate := range preview.GetCandidates() {
		if candidate.GetApplied() {
			return fmt.Errorf("preview candidate %q is marked applied", candidate.GetRuleId())
		}
		if !candidatePathWithin(root, candidate.GetFilePath()) {
			return fmt.Errorf("preview candidate %q escapes isolated fixture: %q", candidate.GetRuleId(), candidate.GetFilePath())
		}
		other := applied.GetCandidates()[i]
		if !other.GetApplied() {
			return fmt.Errorf("applied candidate %q is not marked applied", other.GetRuleId())
		}
		if candidate.GetRuleId() != other.GetRuleId() || candidate.GetFilePath() != other.GetFilePath() || candidate.GetBefore() != other.GetBefore() || candidate.GetAfter() != other.GetAfter() || candidate.GetDescription() != other.GetDescription() {
			return fmt.Errorf("preview/apply candidate mismatch at index %d", i)
		}
		if !candidatePathWithin(root, other.GetFilePath()) {
			return fmt.Errorf("applied candidate %q escapes isolated fixture: %q", other.GetRuleId(), other.GetFilePath())
		}
	}
	return nil
}

func candidatePathWithin(root, candidatePath string) bool {
	if strings.TrimSpace(candidatePath) == "" {
		return false
	}
	path := candidatePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func loadProviderDescriptorSpec(pr *ProviderConformance, repoRoot, provider string) *assessment.Spec {
	path := filepath.Join(repoRoot, "scenarios", provider, filepath.FromSlash(providerdescriptor.RelPath))
	load := providerdescriptor.Load(providerdescriptor.LoadOptions{Paths: []string{path}})
	for _, diagnostic := range load.Diagnostics {
		pr.Violations = append(pr.Violations, fmt.Sprintf(
			"descriptor invalid: code=%s path=%s detail=%s",
			diagnostic.Code, diagnostic.Path, diagnostic.Message))
	}
	if len(load.Diagnostics) > 0 || len(load.Descriptors) != 1 {
		return nil
	}
	return load.Descriptors[0].MaturitySpec
}

// adoptionScore is the fraction of the five adoption dimensions satisfied.
// DescribeConformanceProbe reports whether a provider answers DescribeProvider.
type DescribeConformanceProbe func(ctx context.Context, provider string, timeout time.Duration) error

// DescribeProbe is the seam the conformance scan uses to test DescribeProvider
// adoption. It is package-level so adopting the check did not require threading
// another parameter through every scan entry point; tests override it.
var DescribeProbe DescribeConformanceProbe = DefaultDescribeProbe

// DefaultDescribeProbe calls DescribeProvider and reports the outcome. A
// CodeUnimplemented error means the provider has not adopted the RPC, which is
// the case worth surfacing: readiness then falls back to ValidateScenario, and
// for a provider with no cheap inspection mode that re-runs its entire analysis
// on every suite run.
func DefaultDescribeProbe(ctx context.Context, provider string, timeout time.Duration) error {
	_, err := DefaultDescribeProvider(ctx, provider, timeout)
	return err
}

// DefaultDescribeProvider returns the typed provider declaration used by
// target conformance checks. Keeping this beside the legacy error-only probe
// prevents the two transports from resolving providers differently.
func DefaultDescribeProvider(ctx context.Context, provider string, timeout time.Duration) (*scenariovalidationv1.DescribeProviderResponse, error) {
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
	response, err := scenariovalidationconnect.NewScenarioValidationServiceClient(
		&http.Client{Timeout: timeout}, baseURL,
	).DescribeProvider(ctx, connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func adoptionScore(r ProviderConformance) float64 {
	satisfied := 0
	for _, ok := range []bool{r.Reachable, r.ContractValid, r.IdentityOK, r.SpecValid, r.MetricsAdopted, r.DescribeAdopted} {
		if ok {
			satisfied++
		}
	}
	return float64(satisfied) / 6.0
}

// DefaultConformanceProbe is the live Connect probe used when no seam is
// injected: resolve the provider's URL through discovery and call
// ValidateScenario with include_execution=false. Exported for the
// provider-conformance validation phase, which shares the probe path.
func DefaultConformanceProbe(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	return defaultConformanceProbe(ctx, provider, target, timeout)
}

// DefaultFixConformanceProbe resolves the provider then invokes both fix RPCs
// with the same isolated target path. The caller owns fixture creation and
// cleanup; this transport helper never chooses a real scenario path.
func DefaultFixConformanceProbe(ctx context.Context, provider, target, path string, ruleIDs []string, timeout time.Duration) (*scenariovalidationv1.FixResponse, *scenariovalidationv1.FixResponse, error) {
	if timeout <= 0 {
		timeout = defaultConformanceTimeout
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve %s URL: %w", provider, err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, nil, fmt.Errorf("%s base URL is empty", provider)
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	req := &scenariovalidationv1.FixRequest{Scenario: target, Path: path, RuleIds: ruleIDs}
	preview, err := client.PreviewFix(runCtx, connect.NewRequest(req))
	if err != nil {
		return nil, nil, err
	}
	applied, err := client.ApplyFix(runCtx, connect.NewRequest(req))
	if err != nil {
		return nil, nil, err
	}
	if preview == nil || applied == nil {
		return nil, nil, errors.New("empty fix response")
	}
	return preview.Msg, applied.Msg, nil
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
