package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
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
	class := strings.ToLower(strings.TrimSpace(extras.Class))
	if !aisearch.ValidProviderClass(class) {
		report.add(Finding{
			Code:     CodeProviderClassMissing,
			Severity: SeverityError,
			Title:    "Search provider class is missing or invalid",
			Message: fmt.Sprintf("class %q is not one of %s; an active provider must declare its operability class so Search Hub can enforce the right control/latency posture without scenario-name exceptions.",
				extras.Class, strings.Join(aisearch.KnownProviderClasses, ", ")),
			Location:    providerPath + ".class",
			Remediation: "Set class to local_index (tunable indexed), local_live (computed live), external (third-party), or async (async-indexed).",
		})
		// Without a class the endpoint posture is undecidable; still validate the
		// class-independent corpus and tuning below.
	} else {
		validateEndpointPosture(report, providerPath, provider, class)
	}
	validateEvalCorpus(report, providerPath, provider, extras)
	validateTuningBudget(report, providerPath, extras)
}

// validateEndpointPosture enforces status/control-endpoint expectations by
// provider class. It is deliberately class-driven, never scenario-name driven:
//   - external: no local status endpoint or control plane is expected.
//   - local_live: routable-only, computed live; a status endpoint is advisory
//     and no reindex/config control plane is expected.
//   - local_index / async: own a rebuildable corpus, so a reindex endpoint is
//     required (Search Hub must be able to reconcile and re-evaluate the corpus);
//     a config write-back endpoint is advisory (hybrid-by-construction corpora
//     legitimately pin their tuning and expose none).
func validateEndpointPosture(report *Report, providerPath string, provider *registryv1.ProviderDescriptor, class string) {
	if class == aisearch.ClassExternal {
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
	if !aisearch.IndexedClass(class) {
		return
	}
	if provider.GetReindexEndpoint() == nil {
		report.add(Finding{
			Code:     CodeReindexEndpointMissing,
			Severity: SeverityError,
			Title:    "Indexed search provider has no reindex endpoint",
			Message: fmt.Sprintf("A %s provider owns a rebuildable corpus but declares no reindex_endpoint, so Search Hub cannot reconcile or re-evaluate it.",
				class),
			Location:    providerPath + ".reindex_endpoint",
			Remediation: "Declare reindex_endpoint targeting the shared token-gated SearchControlService, or reclassify the provider as local_live if it computes results live.",
		})
	}
	if provider.GetConfigEndpoint() == nil {
		report.add(Finding{
			Code:        CodeControlEndpointMissing,
			Severity:    SeverityWarning,
			Title:       "Indexed search provider has no config write-back endpoint",
			Message:     "Search Hub can route and reindex the provider, but the sweep cannot write a winning tuning block back. Hybrid-by-construction providers legitimately pin their config.",
			Location:    providerPath + ".config_endpoint",
			Remediation: "Declare config_endpoint when the provider's tuning is writable; leave it absent for hybrid-by-construction corpora.",
		})
	}
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
	validateCorpusAdequacy(report, providerPath, extras.Tests)
}

// difficultyBands mirrors the eval package's difficulty labels: a corpus whose
// reviewed positives all share one band over-reports recall by never exercising
// the hard retrievals.
var difficultyBands = []string{"strong", "weak", "weak-real", "hard"}

// overfitPositiveFloor is the point below which a reviewed-positive corpus is
// flagged (advisory) as overfittable. It only gates the thin-difficulty advisory
// so tiny clean corpora are not spammed; the certification requirement is simply
// that at least one reviewed positive and one negative exist.
const overfitPositiveFloor = 12

// validateCorpusAdequacy enforces certification-grade corpus adequacy on the
// declared corpus (a static check, no provider execution):
//
//   - REQUIRED: at least one REVIEWED positive case. Zero reviewed positives means
//     the corpus is empty of acceptance signal or entirely generated/candidate,
//     so retrieval quality cannot be certified.
//   - REQUIRED: at least one junk NEGATIVE (expect_no_strong_hit) so junk-rejection
//     is measured.
//   - ADVISORY: duplicate queries (count inflation) and thin difficulty spread.
//
// Candidate/generated cases are preserved as expansion inputs but excluded from
// the reviewed-positive count (they are also excluded from acceptance recall by
// the grader), so a generated-only corpus cannot certify.
func validateCorpusAdequacy(report *Report, providerPath string, suite *aisearch.TestSuite) {
	reviewedPositives := 0
	candidatePositives := 0
	negatives := 0
	difficulty := map[string]bool{}
	for _, c := range suite.Cases {
		switch {
		case c.ExpectNoStrongHit || c.HasTag("gibberish"):
			negatives++
		case len(c.ExpectIDs) > 0:
			if c.IsCandidate() {
				candidatePositives++
				continue
			}
			reviewedPositives++
			for _, band := range difficultyBands {
				if c.HasTag(band) {
					difficulty[band] = true
				}
			}
		}
	}

	if reviewedPositives == 0 {
		msg := "corpus has no reviewed positive cases; certification needs at least one reviewed positive label."
		if candidatePositives > 0 {
			msg = fmt.Sprintf("corpus has %d candidate/generated positive case(s) but no reviewed positive; promote reviewed positives before certification.", candidatePositives)
		}
		report.add(Finding{
			Code:        CodeEvalCorpusInadequate,
			Severity:    SeverityError,
			Title:       "Search eval corpus has no reviewed positives",
			Message:     msg,
			Location:    providerPath + ".tests.cases",
			Remediation: "Add or promote reviewed positive cases (status reviewed) with expected ids.",
		})
	}
	if negatives == 0 {
		report.add(Finding{
			Code:        CodeEvalCorpusInadequate,
			Severity:    SeverityError,
			Title:       "Search eval corpus has no junk negative",
			Message:     "corpus has no negative (expect_no_strong_hit) case, so junk-rejection quality is unmeasured.",
			Location:    providerPath + ".tests.cases",
			Remediation: "Add at least one negative case with expect_no_strong_hit and expect_max_score.",
		})
	}
	for _, q := range duplicateCorpusQueries(suite.Cases) {
		report.add(Finding{
			Code:        CodeEvalCorpusThin,
			Severity:    SeverityWarning,
			Title:       "Search eval corpus has duplicate queries",
			Message:     fmt.Sprintf("query %q appears on more than one case (same scope); duplicates inflate corpus counts.", q),
			Location:    providerPath + ".tests.cases",
			Remediation: "Remove or re-scope duplicate queries so each case measures a distinct retrieval.",
		})
	}
	if reviewedPositives >= overfitPositiveFloor && len(difficulty) <= 1 {
		band := "untagged"
		if len(difficulty) == 1 {
			for b := range difficulty {
				band = b
			}
		}
		report.add(Finding{
			Code:        CodeEvalCorpusThin,
			Severity:    SeverityWarning,
			Title:       "Search eval corpus difficulty is thin",
			Message:     fmt.Sprintf("all reviewed positives share one difficulty band (%s); recall is over-reported without hard cases.", band),
			Location:    providerPath + ".tests.cases",
			Remediation: "Tag reviewed positives across difficulty bands (strong/weak/weak-real/hard).",
		})
	}
}

// duplicateCorpusQueries returns queries that appear on more than one case within
// the same scope (two cases with the same query but different scopes are
// legitimately distinct and are not flagged).
func duplicateCorpusQueries(cases []aisearch.TestCase) []string {
	seen := map[string]int{}
	label := map[string]string{}
	for _, c := range cases {
		norm := normalizeCorpusQuery(c.Query)
		if norm == "" {
			continue
		}
		key := norm + "\x00" + strings.TrimSpace(c.Scope)
		seen[key]++
		label[key] = c.Query
	}
	var dups []string
	for key, n := range seen {
		if n > 1 {
			dups = append(dups, label[key])
		}
	}
	sort.Strings(dups)
	return dups
}

func normalizeCorpusQuery(q string) string {
	return strings.ToLower(strings.Join(strings.Fields(q), " "))
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
	if runLeakedJunk(lastRun) {
		report.add(Finding{
			Code:        CodeEvalAssertFailed,
			Severity:    SeverityError,
			Title:       "Search eval run leaked a junk hit",
			Message:     fmt.Sprintf("Eval run %q has a negative case whose top result exceeded its junk ceiling (unexpected_hit).", lastRun.GetRunId()),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Repair the provider's junk-rejection (relevance floor / ranking) or correct the negative case ceiling, then rerun the suite.",
		})
	}
	runAt, err := time.Parse(time.RFC3339Nano, lastRun.GetCreatedAt())
	if err != nil {
		evidence.Freshness = "unknown"
		report.add(Finding{
			Code:        CodeEvalRunStale,
			Severity:    SeverityError,
			Title:       "Search eval run freshness is unknown",
			Message:     fmt.Sprintf("Eval run %q has an unparsable created_at timestamp.", lastRun.GetRunId()),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Re-run the eval suite so Search Hub records fresh timestamped evidence.",
		})
	} else if s.now().Sub(runAt) > opts.EvalFreshnessWindow {
		evidence.Freshness = "stale"
		report.add(Finding{
			Code:        CodeEvalRunStale,
			Severity:    SeverityError,
			Title:       "Search eval run evidence is stale",
			Message:     fmt.Sprintf("Latest eval run %q is older than %s; certification requires fresh evidence.", lastRun.GetRunId(), opts.EvalFreshnessWindow),
			Location:    providerPath + ".tests.suite_id",
			Remediation: "Run the eval suite again to refresh search-quality evidence.",
		})
	} else {
		evidence.Freshness = "fresh"
	}

	// Aggregate scoring policy: the latest run's recall over reviewed positives
	// must meet the declared (or default) recall target. This is the aggregate
	// gate — per-case outcomes are covered by CodeEvalAssertFailed above.
	policy := aisearch.DefaultScoringPolicy
	if extras.Scoring != nil {
		policy = extras.Scoring.WithDefaults()
	}
	if recall, gradeable, ok := runRecall(suite, lastRun); ok {
		evidence.Recall = recall
		evidence.RecallTarget = policy.RecallTarget
		evidence.GradeablePositives = gradeable
		if recall+1e-9 < policy.RecallTarget {
			report.add(Finding{
				Code:     CodeEvalRecallBelowTarget,
				Severity: SeverityError,
				Title:    "Search eval recall is below target",
				Message: fmt.Sprintf("latest run %q recall %.2f over %d reviewed positives is below the target %.2f.",
					lastRun.GetRunId(), recall, gradeable, policy.RecallTarget),
				Location:    providerPath + ".scoring.recall_target",
				Remediation: "Repair retrieval quality or correct corpus expectations, then re-run the eval suite.",
			})
		}
	}
	if agg := lastRun.GetAggregate(); agg != nil {
		evidence.MetCases = int(agg.GetMet())
		evidence.BelowCases = int(agg.GetBelow())
		evidence.LatencyP95Ms = agg.GetLatencyP95Ms()
	}

	// Fingerprint freshness: a populated run config that no longer matches the
	// declared index-time/query-time tuning means the run predates the current
	// descriptor state, so wall-clock freshness alone is not enough.
	if drift := tuningDrift(extras.Tuning, lastRun.GetConfig()); drift != "" {
		evidence.Freshness = "outdated"
		report.add(Finding{
			Code:     CodeEvalRunOutdated,
			Severity: SeverityError,
			Title:    "Search eval run predates the current tuning",
			Message: fmt.Sprintf("latest run %q was produced under a different %s; its evidence no longer reflects the declared tuning.",
				lastRun.GetRunId(), drift),
			Location:    providerPath + ".tuning",
			Remediation: "Re-run the eval suite under the current tuning so certification evidence matches the declared descriptor.",
		})
	}

	validatePerformanceBudget(report, providerPath, strings.ToLower(strings.TrimSpace(extras.Class)), extras, &evidence, suite, lastRun)

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

// runLeakedJunk reports whether any negative case's top result exceeded its junk
// ceiling. A junk leak is always a hard certification failure; positive misses
// (below_expectation) are governed by the aggregate recall target instead, so a
// provider may miss some positives yet still certify if it meets recall_target.
func runLeakedJunk(run *evalv1.EvalRun) bool {
	for _, result := range run.GetResults() {
		if strings.TrimSpace(result.GetOutcome()) == "unexpected_hit" {
			return true
		}
	}
	return false
}

// runRecall computes recall over the suite's reviewed positive cases from a
// stored run's per-case outcomes. ok is false when there are no gradeable
// reviewed positives (recall is undefined and left to the corpus/live checks).
func runRecall(suite *evalv1.EvalSuite, run *evalv1.EvalRun) (recall float64, gradeable int, ok bool) {
	outcome := make(map[string]string, len(run.GetResults()))
	for _, r := range run.GetResults() {
		outcome[r.GetCaseId()] = strings.TrimSpace(r.GetOutcome())
	}
	met := 0
	for _, c := range suite.GetCases() {
		if c.GetExpectNoStrongHit() || len(c.GetExpectIds()) == 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(c.GetStatus()), "candidate") {
			continue
		}
		switch outcome[c.GetCaseId()] {
		case "met":
			gradeable++
			met++
		case "below_expectation":
			gradeable++
		}
	}
	if gradeable == 0 {
		return 0, 0, false
	}
	return float64(met) / float64(gradeable), gradeable, true
}

// validatePerformanceBudget checks the provider's latency/reliability budget
// against the latest run's evidence. The provider class supplies a conservative
// default p95 budget (external/async looser) so an undeclared budget still has a
// bar, modeled by class rather than scenario name:
//
//   - SEARCH_PERF_BUDGET_BREACH (advisory): the run's p95 latency exceeds the
//     effective budget. Advisory because eval-run latency is a small, noisy sample.
//   - SEARCH_PERF_BUDGET_UNPROVEN (required): the provider declared
//     telemetry_required but no latency evidence exists — it opted in and must
//     deliver measurable latency.
//   - SEARCH_PERF_DEGRADED (advisory): only when degraded_rate_max is declared and
//     the run's empty-result rate exceeds it.
func validatePerformanceBudget(report *Report, providerPath, class string, extras providerExtras, evidence *EvalEvidence, suite *evalv1.EvalSuite, lastRun *evalv1.EvalRun) {
	perf := aisearch.PerformanceConfig{}
	if extras.Performance != nil {
		perf = *extras.Performance
	}
	budget := perf.P95Ms
	if budget <= 0 {
		budget = aisearch.DefaultP95BudgetMs(class)
	}
	if evidence.LatencyP95Ms > 0 {
		if int(evidence.LatencyP95Ms) > budget {
			report.add(Finding{
				Code:     CodePerfBudgetBreach,
				Severity: SeverityWarning,
				Title:    "Search provider exceeds its latency budget",
				Message: fmt.Sprintf("latest run p95 latency %dms exceeds the %s budget of %dms.",
					evidence.LatencyP95Ms, class, budget),
				Location:    providerPath + ".performance.p95_ms",
				Remediation: "Tune retrieval/rerank cost or raise the declared p95_ms budget if the class default is too strict for this provider.",
			})
		}
	} else if perf.TelemetryRequired {
		report.add(Finding{
			Code:        CodePerfBudgetUnproven,
			Severity:    SeverityError,
			Title:       "Search provider latency is unproven",
			Message:     "performance.telemetry_required is set but no eval-run latency evidence exists to prove the budget.",
			Location:    providerPath + ".performance.telemetry_required",
			Remediation: "Run the eval suite so Search Hub records p95 latency, or clear telemetry_required if this provider cannot be measured.",
		})
	}
	if extras.Performance != nil && extras.Performance.DegradedRateMax > 0 {
		if rate, ok := degradedRate(suite, lastRun); ok && rate > extras.Performance.DegradedRateMax {
			report.add(Finding{
				Code:     CodePerfDegraded,
				Severity: SeverityWarning,
				Title:    "Search provider degradation rate is high",
				Message: fmt.Sprintf("latest run returned no result for %.0f%% of cases, above the declared max of %.0f%%.",
					rate*100, extras.Performance.DegradedRateMax*100),
				Location:    providerPath + ".performance.degraded_rate_max",
				Remediation: "Investigate empty-result queries (index coverage, provider errors) or adjust the declared degraded_rate_max.",
			})
		}
	}
}

// degradedRate is the fraction of the run's non-negative cases that returned no
// result at all (a coverage/availability signal). Negative (junk) cases are
// excluded — an empty result for a junk query is correct rejection, not
// degradation.
func degradedRate(suite *evalv1.EvalSuite, run *evalv1.EvalRun) (float64, bool) {
	negative := make(map[string]bool)
	for _, c := range suite.GetCases() {
		if c.GetExpectNoStrongHit() {
			negative[c.GetCaseId()] = true
		}
	}
	total := 0
	degraded := 0
	for _, r := range run.GetResults() {
		if negative[r.GetCaseId()] {
			continue
		}
		total++
		if len(r.GetTop()) == 0 {
			degraded++
		}
	}
	if total == 0 {
		return 0, false
	}
	return float64(degraded) / float64(total), true
}

// tuningDrift returns a human description of the first index/query tuning factor
// that diverges between the declared tuning and a run's captured config. It
// returns "" when there is no drift, or when the run captured no config snapshot
// (embed_model empty ⇒ the provider was unreachable at run time, so drift is
// undecidable and must not be reported as a failure).
func tuningDrift(declared *aisearch.TuningConfig, snap *evalv1.ConfigSnapshot) string {
	if snap == nil || strings.TrimSpace(snap.GetEmbedModel()) == "" {
		return ""
	}
	t := aisearch.TuningConfig{}
	if declared != nil {
		t = *declared
	}
	res := t.WithDefaults()
	if snap.GetEngine() != "" && !strings.EqualFold(res.Engine, snap.GetEngine()) {
		return fmt.Sprintf("engine (declared %q, run %q)", res.Engine, snap.GetEngine())
	}
	if !strings.EqualFold(res.EmbedModel, snap.GetEmbedModel()) {
		return fmt.Sprintf("embed model (declared %q, run %q)", res.EmbedModel, snap.GetEmbedModel())
	}
	if res.EmbedTaskPrefix != snap.GetEmbedTaskPrefix() {
		return "embed task prefix"
	}
	if res.RerankEnabled != snap.GetRerankEnabled() {
		return "rerank enabled flag"
	}
	if res.RerankBlend != snap.GetRerankBlend() {
		return "rerank blend flag"
	}
	return ""
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
	Class       string                      `json:"class"`
	Tests       *aisearch.TestSuite         `json:"tests"`
	Tuning      *aisearch.TuningConfig      `json:"tuning"`
	Scoring     *aisearch.ScoringConfig     `json:"scoring"`
	Performance *aisearch.PerformanceConfig `json:"performance"`
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
