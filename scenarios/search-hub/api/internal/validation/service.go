package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"google.golang.org/protobuf/encoding/protojson"

	internaleval "search-hub/internal/eval"
	internalregistry "search-hub/internal/registry"
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

const defaultEvalFreshnessWindow = 30 * 24 * time.Hour

type Service struct {
	RepoRoot      string
	EvalStore     EvalStore
	EvalValidator EvalValidator
	Now           func() time.Time
}

type Options struct {
	IncludeEvals        bool
	EvalFreshnessWindow time.Duration
}

type EvalStore interface {
	GetSuite(ctx context.Context, id string) (*evalv1.EvalSuite, error)
	ListRuns(ctx context.Context, filter internaleval.ListRunsFilter) ([]*evalv1.EvalRun, error)
}

type EvalValidator interface {
	ValidateCorpus(ctx context.Context, suite *evalv1.EvalSuite, deepK int32) (*evalv1.ValidateCorpusResponse, error)
}

func New(repoRoot string) *Service {
	return &Service{RepoRoot: strings.TrimSpace(repoRoot)}
}

func (s *Service) ValidateScenario(scenario, path string) (Report, error) {
	return s.ValidateScenarioWithOptions(context.Background(), scenario, path, Options{})
}

func (s *Service) ValidateScenarioWithOptions(ctx context.Context, scenario, path string, opts Options) (Report, error) {
	scenario, scenarioPath, err := s.resolveTarget(scenario, path)
	if err != nil {
		return Report{}, err
	}
	report := Report{Scenario: scenario, Path: scenarioPath}
	configPath := filepath.Join(scenarioPath, ".vrooli", "search.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			report.add(Finding{
				Code:        CodeConfigMissing,
				Severity:    SeverityError,
				Title:       "Search descriptor missing",
				Message:     "The target declares search applicability but has no .vrooli/search.json file.",
				Location:    ".vrooli/search.json",
				Remediation: "Add .vrooli/search.json with at least one provider descriptor, or remove the search capability declaration.",
			})
			report.finish()
			return report, nil
		}
		return Report{}, fmt.Errorf("read %s: %w", configPath, err)
	}
	var cfg searchConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		report.add(Finding{
			Code:        CodeConfigInvalid,
			Severity:    SeverityError,
			Title:       "Search descriptor is not valid JSON",
			Message:     err.Error(),
			Location:    ".vrooli/search.json",
			Remediation: "Fix .vrooli/search.json so Search Hub can parse the provider descriptors.",
		})
		report.finish()
		return report, nil
	}
	s.validateConfig(ctx, &report, cfg, opts.withDefaults())
	report.finish()
	return report, nil
}

func (o Options) withDefaults() Options {
	if o.EvalFreshnessWindow <= 0 {
		o.EvalFreshnessWindow = defaultEvalFreshnessWindow
	}
	return o
}

func (s *Service) resolveTarget(scenario, path string) (string, string, error) {
	scenario = normalizeScenario(scenario)
	path = strings.TrimSpace(path)
	if path != "" {
		if scenario == "" {
			scenario = filepath.Base(filepath.Clean(path))
		}
		return scenario, path, nil
	}
	if scenario == "" {
		return "", "", fmt.Errorf("scenario or path is required")
	}
	if s.RepoRoot == "" {
		return "", "", fmt.Errorf("repo root is required when path is omitted")
	}
	return scenario, filepath.Join(s.RepoRoot, "scenarios", scenario), nil
}

func (s *Service) validateConfig(ctx context.Context, report *Report, cfg searchConfig, opts Options) {
	if strings.TrimSpace(cfg.Version) == "" || !semverPattern.MatchString(strings.TrimSpace(cfg.Version)) {
		report.add(Finding{
			Code:        CodeConfigInvalid,
			Severity:    SeverityError,
			Title:       "Search descriptor version is invalid",
			Message:     "version must be a semantic version string.",
			Location:    ".vrooli/search.json:version",
			Remediation: "Set version to a semantic version such as 1.0.0.",
		})
	}
	if len(cfg.Providers) == 0 {
		report.add(Finding{
			Code:        CodeConfigInvalid,
			Severity:    SeverityError,
			Title:       "Search descriptor has no providers",
			Message:     "providers must contain at least one search provider descriptor.",
			Location:    ".vrooli/search.json:providers",
			Remediation: "Declare at least one provider, or remove the search capability declaration.",
		})
		return
	}
	for i, rawProvider := range cfg.Providers {
		report.Summary.Providers++
		providerPath := fmt.Sprintf(".vrooli/search.json:providers[%d]", i)
		provider, extras, err := decodeProvider(rawProvider)
		if err != nil {
			report.add(Finding{
				Code:        CodeProviderInvalid,
				Severity:    SeverityError,
				Title:       "Search provider descriptor is invalid",
				Message:     err.Error(),
				Location:    providerPath,
				Remediation: "Fix the provider descriptor so it matches the search-hub registry contract.",
			})
			continue
		}
		internalregistry.Normalize(provider)
		if err := internalregistry.Validate(provider); err != nil {
			report.add(Finding{
				Code:        CodeProviderInvalid,
				Severity:    SeverityError,
				Title:       "Search provider descriptor violates registry contract",
				Message:     err.Error(),
				Location:    providerPath,
				Remediation: "Provide bucket/type/description plus a callable endpoint and result mapping, or mark a true stub as capability_gap.",
			})
		}
		if provider.GetProviderGroup() != report.Scenario {
			report.add(Finding{
				Code:        CodeProviderGroupMismatch,
				Severity:    SeverityError,
				Title:       "Search provider group does not match scenario",
				Message:     fmt.Sprintf("provider_group is %q, want %q.", provider.GetProviderGroup(), report.Scenario),
				Location:    providerPath + ".provider_group",
				Remediation: "Set provider_group to the owning scenario directory name.",
			})
		}
		validateProviderOperationalPosture(report, providerPath, provider, extras)
		if opts.IncludeEvals {
			s.validateEvalEvidence(ctx, report, providerPath, provider, extras, opts)
		}
	}
}

func validateProviderOperationalPosture(report *Report, providerPath string, provider *registryv1.ProviderDescriptor, extras providerExtras) {
	if provider.GetState() == registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP {
		return
	}
	if provider.GetStatusEndpoint() == nil {
		report.add(Finding{
			Code:        CodeStatusEndpointMissing,
			Severity:    SeverityWarning,
			Title:       "Search provider has no status endpoint",
			Message:     "Search Hub can route the provider but cannot cheaply inspect provider readiness.",
			Location:    providerPath + ".status_endpoint",
			Remediation: "Expose a lightweight status endpoint and declare it in .vrooli/search.json.",
		})
	}
	if provider.GetReindexEndpoint() == nil || provider.GetConfigEndpoint() == nil {
		report.add(Finding{
			Code:        CodeControlEndpointMissing,
			Severity:    SeverityWarning,
			Title:       "Search provider has no complete search control endpoints",
			Message:     "Search Hub can route the provider, but async reindex/config write-back is not fully declared.",
			Location:    providerPath,
			Remediation: "Declare reindex_endpoint and config_endpoint when the provider owns a tunable indexed corpus.",
		})
	}
	validateEvalCorpus(report, providerPath, provider, extras)
	validateTuningBudget(report, providerPath, extras)
}

func validateEvalCorpus(report *Report, providerPath string, provider *registryv1.ProviderDescriptor, extras providerExtras) {
	if extras.Tests == nil {
		report.add(Finding{
			Code:        CodeEvalCorpusMissing,
			Severity:    SeverityError,
			Title:       "Search provider has no eval corpus",
			Message:     "A search-enabled provider needs labelled cases so Search Hub can detect degraded routing quality.",
			Location:    providerPath + ".tests",
			Remediation: "Add a tests block with reviewed positive or negative cases, or mark the provider as a capability gap until a corpus exists.",
		})
		return
	}
	if strings.TrimSpace(extras.Tests.Description) == "" {
		report.add(Finding{
			Code:        CodeEvalCorpusInvalid,
			Severity:    SeverityWarning,
			Title:       "Search eval corpus lacks rationale",
			Message:     "tests.description should explain what the corpus proves.",
			Location:    providerPath + ".tests.description",
			Remediation: "Describe the corpus intent and acceptance posture.",
		})
	}
	if len(extras.Tests.Cases) == 0 {
		report.add(Finding{
			Code:        CodeEvalCorpusInvalid,
			Severity:    SeverityError,
			Title:       "Search eval corpus has no cases",
			Message:     "tests.cases must include at least one labelled query.",
			Location:    providerPath + ".tests.cases",
			Remediation: "Add reviewed positive cases or negative junk-rejection cases.",
		})
		return
	}
	for i, c := range extras.Tests.Cases {
		casePath := fmt.Sprintf("%s.tests.cases[%d]", providerPath, i)
		if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Query) == "" {
			report.add(Finding{
				Code:        CodeEvalCorpusInvalid,
				Severity:    SeverityError,
				Title:       "Search eval case is missing identity or query",
				Message:     "Each eval case needs a stable id and a non-empty query.",
				Location:    casePath,
				Remediation: "Set id and query on every eval case.",
			})
		}
		if !c.ExpectNoStrongHit && len(c.ExpectIDs) == 0 {
			report.add(Finding{
				Code:        CodeEvalCorpusInvalid,
				Severity:    SeverityError,
				Title:       "Search eval positive case has no expected ids",
				Message:     "Positive eval cases must name at least one expected result id.",
				Location:    casePath + ".expect_ids",
				Remediation: "Add expect_ids or mark the case as expect_no_strong_hit with an expect_max_score.",
			})
		}
	}
	if provider.GetProviderId() != "" && strings.TrimSpace(extras.Tests.SuiteID) != "" && !strings.HasPrefix(extras.Tests.SuiteID, provider.GetProviderId()) {
		report.add(Finding{
			Code:        CodeEvalCorpusInvalid,
			Severity:    SeverityWarning,
			Title:       "Search eval suite id is not provider-scoped",
			Message:     "suite_id should normally start with the provider_id to keep registry mirrors traceable.",
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Use the default suite id or prefix custom suite_id with the provider_id.",
		})
	}
}

func validateTuningBudget(report *Report, providerPath string, extras providerExtras) {
	if extras.Tuning == nil {
		return
	}
	if extras.Tuning.RerankEnabled && extras.Tuning.RerankShortlist > 250 {
		report.add(Finding{
			Code:        CodeTuningBudgetInvalid,
			Severity:    SeverityWarning,
			Title:       "Search rerank shortlist is expensive",
			Message:     "rerank_shortlist above 250 can consume the query timeout budget.",
			Location:    providerPath + ".tuning.rerank_shortlist",
			Remediation: "Lower rerank_shortlist or document the measured latency budget in the eval corpus.",
		})
	}
}

func (s *Service) validateEvalEvidence(ctx context.Context, report *Report, providerPath string, provider *registryv1.ProviderDescriptor, extras providerExtras, opts Options) {
	if provider.GetState() == registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP || extras.Tests == nil || len(extras.Tests.Cases) == 0 {
		return
	}
	suiteID := strings.TrimSpace(extras.Tests.SuiteID)
	if suiteID == "" && strings.TrimSpace(provider.GetProviderId()) != "" {
		suiteID = strings.TrimSpace(provider.GetProviderId()) + ".primary"
	}
	if suiteID == "" {
		return
	}
	evidence := EvalEvidence{SuiteID: suiteID, Freshness: "not_checked", CorpusStatus: "not_checked"}
	defer func() {
		report.EvalEvidence = append(report.EvalEvidence, evidence)
	}()
	if s.EvalStore == nil {
		evidence.Freshness = "unavailable"
		evidence.CorpusStatus = "unavailable"
		evidence.FailureReason = "eval store is not configured"
		report.add(Finding{
			Code:        CodeEvalProviderUnavailable,
			Severity:    SeverityError,
			Title:       "Search eval evidence unavailable",
			Message:     "Search Hub cannot inspect eval run history because the eval store is not configured.",
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Run Search Hub with the eval store configured, then rerun validation with include_execution.",
		})
		return
	}
	suite, err := s.EvalStore.GetSuite(ctx, suiteID)
	if err != nil {
		evidence.Freshness = "missing_suite"
		evidence.CorpusStatus = "missing_suite"
		evidence.FailureReason = err.Error()
		report.add(Finding{
			Code:        CodeEvalRunMissing,
			Severity:    SeverityError,
			Title:       "Search eval suite has no registered run history",
			Message:     fmt.Sprintf("Eval suite %q is not registered in Search Hub's eval store.", suiteID),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Register the eval suite through search-hub evals register, then run it to create freshness evidence.",
		})
		return
	}
	if err := internaleval.Validate(suite); err != nil {
		evidence.CorpusStatus = "invalid"
		evidence.FailureReason = err.Error()
		report.add(Finding{
			Code:        CodeEvalCorpusInvalid,
			Severity:    SeverityError,
			Title:       "Registered search eval suite is invalid",
			Message:     err.Error(),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Fix and re-register the eval suite so Search Hub can trust run-history evidence.",
		})
		return
	}
	evidence.CorpusStatus = "registered"
	runs, err := s.EvalStore.ListRuns(ctx, internaleval.ListRunsFilter{SuiteID: suiteID, Limit: 1})
	if err != nil {
		evidence.Freshness = "unavailable"
		evidence.FailureReason = err.Error()
		report.add(Finding{
			Code:        CodeEvalProviderUnavailable,
			Severity:    SeverityError,
			Title:       "Search eval run history unavailable",
			Message:     err.Error(),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Restore Search Hub eval storage, then rerun validation with include_execution.",
		})
		return
	}
	if len(runs) == 0 {
		evidence.Freshness = "missing_run"
		report.add(Finding{
			Code:        CodeEvalRunMissing,
			Severity:    SeverityError,
			Title:       "Search eval run history is missing",
			Message:     fmt.Sprintf("Eval suite %q has no stored runs.", suiteID),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Run search-hub evals run for the suite and rerun validation with include_execution.",
		})
		return
	}
	lastRun := runs[0]
	evidence.LastRunID = lastRun.GetRunId()
	evidence.LastRunAt = lastRun.GetCreatedAt()
	if runFailed(lastRun) {
		report.add(Finding{
			Code:        CodeEvalAssertFailed,
			Severity:    SeverityError,
			Title:       "Search eval run failed expectations",
			Message:     fmt.Sprintf("Eval run %q contains cases below expectation or unexpected hits.", lastRun.GetRunId()),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Inspect the latest eval run, repair retrieval quality or corpus expectations, then rerun the suite.",
		})
	}
	runAt, err := time.Parse(time.RFC3339Nano, lastRun.GetCreatedAt())
	if err != nil {
		evidence.Freshness = "unknown"
		report.add(Finding{
			Code:        CodeEvalRunStale,
			Severity:    SeverityWarning,
			Title:       "Search eval run freshness is unknown",
			Message:     fmt.Sprintf("Eval run %q has an unparsable created_at timestamp.", lastRun.GetRunId()),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Re-run the eval suite so Search Hub records fresh timestamped evidence.",
		})
	} else if s.now().Sub(runAt) > opts.EvalFreshnessWindow {
		evidence.Freshness = "stale"
		report.add(Finding{
			Code:        CodeEvalRunStale,
			Severity:    SeverityWarning,
			Title:       "Search eval run evidence is stale",
			Message:     fmt.Sprintf("Latest eval run %q is older than %s.", lastRun.GetRunId(), opts.EvalFreshnessWindow),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Run the eval suite again to refresh search-quality evidence.",
		})
	} else {
		evidence.Freshness = "fresh"
	}
	if s.EvalValidator == nil {
		report.add(Finding{
			Code:        CodeEvalProviderUnavailable,
			Severity:    SeverityError,
			Title:       "Search eval live corpus validation unavailable",
			Message:     "Search Hub cannot probe live eval labels because the eval validator is not configured.",
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Run Search Hub with live eval validation configured, then rerun full search validation.",
		})
		return
	}
	corpus, err := s.EvalValidator.ValidateCorpus(ctx, suite, 0)
	if err != nil {
		evidence.CorpusStatus = "unavailable"
		evidence.FailureReason = err.Error()
		report.add(Finding{
			Code:        CodeEvalProviderUnavailable,
			Severity:    SeverityError,
			Title:       "Search eval live label validation unavailable",
			Message:     err.Error(),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Ensure the provider is reachable, then rerun validation with include_execution.",
		})
		return
	}
	if rollup := corpus.GetRollup(); rollup == nil || rollup.GetPositives() == 0 {
		evidence.CorpusStatus = "unproven"
		report.add(Finding{
			Code:        CodeEvalLabelsStale,
			Severity:    SeverityError,
			Title:       "Search eval corpus has no live positive labels",
			Message:     "Live corpus validation found no reviewed positive labels to prove retrieval quality.",
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Add reviewed positive eval cases with expected ids, register the suite, and rerun full search validation.",
		})
		return
	} else if rollup.GetStale() > 0 || rollup.GetInconclusive() > 0 {
		evidence.CorpusStatus = "stale"
		report.add(Finding{
			Code:        CodeEvalLabelsStale,
			Severity:    SeverityError,
			Title:       "Search eval labels are stale or inconclusive",
			Message:     fmt.Sprintf("Corpus validation found %d stale and %d inconclusive positive labels.", rollup.GetStale(), rollup.GetInconclusive()),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Refresh stale expected ids or repair provider reachability before treating eval evidence as live proof.",
		})
		return
	}
	evidence.CorpusStatus = "live"
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func runFailed(run *evalv1.EvalRun) bool {
	for _, result := range run.GetResults() {
		switch strings.TrimSpace(result.GetOutcome()) {
		case "below_expectation", "unexpected_hit":
			return true
		}
	}
	return false
}

func (r *Report) add(f Finding) {
	r.Findings = append(r.Findings, f)
}

func (r *Report) finish() {
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			r.Summary.Errors++
		case SeverityWarning:
			r.Summary.Warnings++
		}
	}
}

func BuildMaturityAssessment(scenario string, findings []Finding, spec assessment.Spec) (*commonv1.MaturityAssessment, error) {
	assessed := make([]assessment.Finding, 0, len(findings))
	for _, f := range findings {
		assessed = append(assessed, assessment.Finding{
			Code:        f.Code,
			Severity:    severityToAssessment(f.Severity),
			Title:       f.Title,
			Message:     f.Message,
			Location:    filepath.ToSlash(f.Location),
			Remediation: f.Remediation,
			Source:      architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: scenario,
		Spec:     spec,
		Findings: assessed,
	})
}

type searchConfig struct {
	Version   string            `json:"version"`
	Providers []json.RawMessage `json:"providers"`
}

type providerExtras struct {
	Tests  *testsConfig  `json:"tests"`
	Tuning *tuningConfig `json:"tuning"`
}

type testsConfig struct {
	SuiteID     string     `json:"suite_id"`
	Description string     `json:"description"`
	Cases       []testCase `json:"cases"`
}

type testCase struct {
	ID                string   `json:"id"`
	Query             string   `json:"query"`
	ExpectIDs         []string `json:"expect_ids"`
	ExpectNoStrongHit bool     `json:"expect_no_strong_hit"`
}

type tuningConfig struct {
	RerankEnabled   bool  `json:"rerank_enabled"`
	RerankShortlist int32 `json:"rerank_shortlist"`
}

func decodeProvider(raw json.RawMessage) (*registryv1.ProviderDescriptor, providerExtras, error) {
	var provider registryv1.ProviderDescriptor
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &provider); err != nil {
		return nil, providerExtras{}, err
	}
	var extras providerExtras
	if err := json.Unmarshal(raw, &extras); err != nil {
		return nil, providerExtras{}, err
	}
	return &provider, extras, nil
}
