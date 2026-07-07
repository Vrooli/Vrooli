package maturity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const defaultScanTimeout = 30 * time.Second

type validationClient interface {
	ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error)
	PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error)
	ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error)
}

var newValidationClient = func(core *cliapp.ScenarioApp, timeout time.Duration) validationClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, timeout)
	return scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
}

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

type scanReport struct {
	RepoRoot         string           `json:"repo_root"`
	ValidationMode   string           `json:"validation_mode"`
	ModeMeaning      string           `json:"mode_meaning"`
	ProviderLiveness providerLiveness `json:"provider_liveness"`
	Summary          scanSummary      `json:"summary"`
	Results          []scenarioResult `json:"results"`
	Groups           scanGroups       `json:"groups"`
}

type providerLiveness struct {
	State   string `json:"state"`
	Source  string `json:"source,omitempty"`
	Message string `json:"message,omitempty"`
}

type scanSummary struct {
	Total       int `json:"total"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Degraded    int `json:"degraded"`
	Unavailable int `json:"unavailable"`
	Findings    int `json:"findings"`
	Blocking    int `json:"blocking"`
	Advisory    int `json:"advisory"`
}

type scanGroups struct {
	ByStatus     map[string][]string `json:"by_status"`
	ByCapability map[string][]string `json:"by_capability"`
	ByFinding    map[string][]string `json:"by_finding"`
}

type scenarioResult struct {
	Scenario              string               `json:"scenario"`
	Path                  string               `json:"path"`
	Status                string               `json:"status"`
	ApplicabilityReason   string               `json:"applicability_reason,omitempty"`
	ProviderLiveness      providerLiveness     `json:"provider_liveness"`
	CurrentLevel          string               `json:"current_level,omitempty"`
	NextLevel             string               `json:"next_level,omitempty"`
	HighestPriority       priorityFocus        `json:"highest_priority,omitempty"`
	Capabilities          []capabilityReport   `json:"capabilities,omitempty"`
	Findings              []findingReport      `json:"findings,omitempty"`
	EvalEvidence          []evalEvidenceReport `json:"eval_evidence,omitempty"`
	BlockingFindingCodes  []string             `json:"blocking_finding_codes,omitempty"`
	AdvisoryFindingCodes  []string             `json:"advisory_finding_codes,omitempty"`
	RecommendedNextAction string               `json:"recommended_next_action,omitempty"`
	RepairCommands        []string             `json:"repair_commands,omitempty"`
	Error                 string               `json:"error,omitempty"`
}

// evalEvidenceReport surfaces the certification-grade run/corpus/performance
// evidence from the validation RPC's native detail so operators can repair a
// failing suite without scraping logs.
type evalEvidenceReport struct {
	SuiteID       string  `json:"suite_id,omitempty"`
	Freshness     string  `json:"freshness,omitempty"`
	CorpusStatus  string  `json:"corpus_status,omitempty"`
	LastRunID     string  `json:"last_run_id,omitempty"`
	LastRunAt     string  `json:"last_run_at,omitempty"`
	Recall        float64 `json:"recall,omitempty"`
	RecallTarget  float64 `json:"recall_target,omitempty"`
	MetCases      int     `json:"met_cases,omitempty"`
	BelowCases    int     `json:"below_cases,omitempty"`
	LatencyP95Ms  int     `json:"latency_p95_ms,omitempty"`
	FailureReason string  `json:"failure_reason,omitempty"`
}

type priorityFocus struct {
	CapabilityID    string `json:"capability_id,omitempty"`
	CapabilityLabel string `json:"capability_label,omitempty"`
	CurrentLevel    string `json:"current_level,omitempty"`
	NextLevel       string `json:"next_level,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

type capabilityReport struct {
	ID                   string         `json:"id"`
	Label                string         `json:"label"`
	CurrentLevel         string         `json:"current_level"`
	NextLevel            string         `json:"next_level,omitempty"`
	Clean                bool           `json:"clean"`
	PriorityRank         int32          `json:"priority_rank,omitempty"`
	PriorityReason       string         `json:"priority_reason,omitempty"`
	BlockingFindingCodes []string       `json:"blocking_finding_codes,omitempty"`
	FindingsBySeverity   map[string]int `json:"findings_by_severity,omitempty"`
}

type findingReport struct {
	Code             string `json:"code"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Message          string `json:"message,omitempty"`
	CapabilityID     string `json:"capability_id,omitempty"`
	LocalLevel       string `json:"local_level,omitempty"`
	Location         string `json:"location,omitempty"`
	Remediation      string `json:"remediation,omitempty"`
	FixClass         string `json:"fix_class,omitempty"`
	FindingClass     string `json:"finding_class,omitempty"`
	AutofixAvailable bool   `json:"autofix_available,omitempty"`
	Advisory         bool   `json:"advisory"`
	Gating           bool   `json:"gating"`
}

type target struct {
	Scenario string
	Path     string
	// Reason records why the scenario was discovered as search-applicable
	// (descriptor, service capability, or both) so scan output can be compared
	// against Test Genie's applicability decision.
	Reason string
}

func (h *handlers) scan(ctx cliapp.RunContext) error {
	root, err := resolveRepoRoot(ctx.Flag("root"))
	if err != nil {
		return err
	}
	timeout := defaultScanTimeout
	if raw := strings.TrimSpace(ctx.Flag("timeout")); raw != "" {
		timeout, err = time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid --timeout %q: %w", raw, err)
		}
	}
	targets, err := discoverTargets(root)
	if err != nil {
		return err
	}
	report := scanReport{
		RepoRoot: root,
		ProviderLiveness: providerLiveness{
			State:  "not_checked",
			Source: "search-hub ScenarioValidationService",
		},
		Results: []scenarioResult{},
		Groups: scanGroups{
			ByStatus:     map[string][]string{},
			ByCapability: map[string][]string{},
			ByFinding:    map[string][]string{},
		},
	}
	client := newValidationClient(h.core, timeout)
	includeEvals := !ctx.BoolFlag("fast") || ctx.BoolFlag("include-evals")
	if includeEvals {
		report.ValidationMode = "full"
		report.ModeMeaning = "certification: descriptor + corpus adequacy + live eval/latency evidence"
	} else {
		report.ValidationMode = "fast"
		report.ModeMeaning = "inventory only: descriptor/corpus shape; live eval and latency proof are skipped"
	}
	for _, target := range targets {
		report.add(h.validateTarget(client, timeout, target, includeEvals))
	}
	report.finish()
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	printScanReport(ctx, report)
	return nil
}

func (h *handlers) fix(ctx cliapp.RunContext) error {
	timeout := defaultScanTimeout
	var err error
	if raw := strings.TrimSpace(ctx.Flag("timeout")); raw != "" {
		timeout, err = time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid --timeout %q: %w", raw, err)
		}
	}
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: strings.TrimSpace(ctx.Positional("scenario")),
		Path:     strings.TrimSpace(ctx.Flag("path")),
		RuleIds:  splitCSV(ctx.Flag("rule")),
	})
	client := newValidationClient(h.core, timeout)
	runCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var resp *connect.Response[scenariovalidationv1.FixResponse]
	if ctx.BoolFlag("apply") {
		resp, err = client.ApplyFix(runCtx, req)
	} else {
		resp, err = client.PreviewFix(runCtx, req)
	}
	if err != nil {
		return err
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	printFixReport(ctx, resp.Msg)
	return nil
}

func (h *handlers) validateTarget(client validationClient, timeout time.Duration, target target, includeEvals bool) scenarioResult {
	runCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := client.ValidateScenario(runCtx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         target.Scenario,
		Path:             target.Path,
		IncludeExecution: includeEvals,
	}))
	if err != nil {
		return scenarioResult{
			Scenario:            target.Scenario,
			Path:                target.Path,
			Status:              "unavailable",
			ApplicabilityReason: target.Reason,
			ProviderLiveness: providerLiveness{
				State:   "unavailable",
				Source:  "search-hub ScenarioValidationService",
				Message: err.Error(),
			},
			RecommendedNextAction: "Start search-hub through lifecycle and rerun maturity scan.",
			Error:                 err.Error(),
		}
	}
	out := scenarioResult{
		Scenario:            target.Scenario,
		Path:                target.Path,
		Status:              statusLabel(resp.Msg.GetStatus()),
		ApplicabilityReason: target.Reason,
		ProviderLiveness:    providerLiveness{State: "available", Source: "search-hub ScenarioValidationService"},
	}
	if a := resp.Msg.GetAssessment(); a != nil {
		applyAssessment(&out, a)
	}
	out.EvalEvidence = decodeEvalEvidence(resp.Msg.GetNativeDetail())
	out.RecommendedNextAction = recommendedNextAction(out)
	out.RepairCommands = repairCommands(out)
	return out
}

// decodeEvalEvidence reads the eval_evidence array the validation service packs
// into the response's native detail (a structpb.Struct), so scan output can show
// run id, freshness, recall vs target, and latency without a second RPC.
func decodeEvalEvidence(detail *anypb.Any) []evalEvidenceReport {
	if detail == nil {
		return nil
	}
	var native structpb.Struct
	if err := detail.UnmarshalTo(&native); err != nil {
		return nil
	}
	raw, ok := native.AsMap()["eval_evidence"].([]any)
	if !ok {
		return nil
	}
	out := make([]evalEvidenceReport, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, evalEvidenceReport{
			SuiteID:       stringField(m, "suite_id"),
			Freshness:     stringField(m, "freshness"),
			CorpusStatus:  stringField(m, "corpus_status"),
			LastRunID:     stringField(m, "last_run_id"),
			LastRunAt:     stringField(m, "last_run_at"),
			Recall:        floatField(m, "recall"),
			RecallTarget:  floatField(m, "recall_target"),
			MetCases:      intField(m, "met_cases"),
			BelowCases:    intField(m, "below_cases"),
			LatencyP95Ms:  intField(m, "latency_p95_ms"),
			FailureReason: stringField(m, "failure_reason"),
		})
	}
	return out
}

func stringField(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func floatField(m map[string]any, key string) float64 {
	f, _ := m[key].(float64)
	return f
}

func intField(m map[string]any, key string) int {
	f, _ := m[key].(float64)
	return int(f)
}

func applyAssessment(out *scenarioResult, a *commonv1.MaturityAssessment) {
	if out.Scenario == "" {
		out.Scenario = a.GetScenario()
	}
	if local := a.GetLocal(); local != nil {
		out.CurrentLevel = local.GetCurrentLevel()
		out.NextLevel = local.GetNextLevel()
		out.BlockingFindingCodes = append(out.BlockingFindingCodes, local.GetBlockingFindingCodes()...)
	}
	if focus := a.GetHighestPriorityCapability(); focus != nil {
		out.HighestPriority = priorityFocus{
			CapabilityID:    focus.GetCapabilityId(),
			CapabilityLabel: focus.GetCapabilityLabel(),
			CurrentLevel:    focus.GetCurrentLevel(),
			NextLevel:       focus.GetNextLevel(),
			Reason:          focus.GetReason(),
		}
	}
	for _, c := range a.GetCapabilities() {
		out.Capabilities = append(out.Capabilities, capabilityReport{
			ID:                   c.GetId(),
			Label:                c.GetLabel(),
			CurrentLevel:         c.GetCurrentLevel(),
			NextLevel:            c.GetNextLevel(),
			Clean:                c.GetClean(),
			PriorityRank:         c.GetPriorityRank(),
			PriorityReason:       c.GetPriorityReason(),
			BlockingFindingCodes: append([]string(nil), c.GetBlockingFindingCodes()...),
			FindingsBySeverity:   copyStringInt32Map(c.GetFindingsBySeverity()),
		})
	}
	for _, f := range a.GetFindings() {
		cleanRequirement := f.GetMaturity().GetCleanRequirement()
		advisory := cleanRequirement == commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY
		gating := !advisory
		if f.GetSeverity() == "SEVERITY_WARNING" && cleanRequirement != commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED {
			advisory = true
			gating = false
		}
		item := findingReport{
			Code:             f.GetCode(),
			Severity:         f.GetSeverity(),
			Title:            f.GetTitle(),
			Message:          f.GetMessage(),
			CapabilityID:     f.GetMaturity().GetCapabilityId(),
			LocalLevel:       f.GetMaturity().GetLocalLevel(),
			Location:         f.GetLocation(),
			Remediation:      f.GetRemediation(),
			FixClass:         f.GetFixClass(),
			FindingClass:     findingClass(f.GetCode()),
			AutofixAvailable: f.GetAutofixAvailable(),
			Advisory:         advisory,
			Gating:           gating,
		}
		out.Findings = append(out.Findings, item)
		if item.Advisory {
			out.AdvisoryFindingCodes = append(out.AdvisoryFindingCodes, item.Code)
		}
	}
	out.BlockingFindingCodes = uniqueSorted(out.BlockingFindingCodes)
	out.AdvisoryFindingCodes = uniqueSorted(out.AdvisoryFindingCodes)
}

func (r *scanReport) add(result scenarioResult) {
	r.Results = append(r.Results, result)
	r.Summary.Total++
	switch result.Status {
	case "passed":
		r.Summary.Passed++
	case "failed":
		r.Summary.Failed++
	case "degraded":
		r.Summary.Degraded++
	case "unavailable":
		r.Summary.Unavailable++
	}
	r.Summary.Findings += len(result.Findings)
	r.Summary.Blocking += len(result.BlockingFindingCodes)
	r.Summary.Advisory += len(result.AdvisoryFindingCodes)
	r.Groups.ByStatus[result.Status] = append(r.Groups.ByStatus[result.Status], result.Scenario)
	if result.ProviderLiveness.State == "unavailable" && r.ProviderLiveness.State != "available" {
		r.ProviderLiveness = result.ProviderLiveness
	} else if result.ProviderLiveness.State == "available" {
		r.ProviderLiveness = result.ProviderLiveness
	}
	for _, finding := range result.Findings {
		r.Groups.ByFinding[finding.Code] = appendUnique(r.Groups.ByFinding[finding.Code], result.Scenario)
		if finding.CapabilityID != "" {
			r.Groups.ByCapability[finding.CapabilityID] = appendUnique(r.Groups.ByCapability[finding.CapabilityID], result.Scenario)
		}
	}
}

func (r *scanReport) finish() {
	sort.Slice(r.Results, func(i, j int) bool { return r.Results[i].Scenario < r.Results[j].Scenario })
	sortGroups(r.Groups.ByStatus)
	sortGroups(r.Groups.ByCapability)
	sortGroups(r.Groups.ByFinding)
	if r.Summary.Total == 0 {
		r.ProviderLiveness.State = "not_checked"
	}
}

// searchCapabilityTokens mirrors the Test Genie search phase applicability
// contract (scenarios/test-genie/docs/phases/search/README.md): a scenario is
// search-applicable when it owns .vrooli/search.json OR declares one of these
// service capabilities. Keeping this list and the resolution below aligned with
// Test Genie's serviceCapabilities() is what makes fleet scan and Test Genie
// applicability report the same target set.
var searchCapabilityTokens = []string{"search", "ai-search"}

func discoverTargets(root string) ([]target, error) {
	scenariosDir := filepath.Join(root, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("read scenarios dir %s: %w", scenariosDir, err)
	}
	var targets []target
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		scenarioPath := filepath.Join(scenariosDir, scenario)
		reason, applicable, err := searchApplicability(scenarioPath)
		if err != nil {
			return nil, fmt.Errorf("stat %s search applicability: %w", scenario, err)
		}
		if applicable {
			targets = append(targets, target{Scenario: scenario, Path: scenarioPath, Reason: reason})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Scenario < targets[j].Scenario })
	return targets, nil
}

// searchApplicability decides whether a scenario is a search maturity target and
// why. A descriptor (.vrooli/search.json) or a declared search/ai-search service
// capability both qualify. A capability-only target is deliberately included: it
// then flows through validation and fails with SEARCH_CONFIG_MISSING instead of
// silently disappearing from the fleet, matching the Test Genie search phase.
func searchApplicability(scenarioPath string) (reason string, applicable bool, err error) {
	descriptor := false
	if _, statErr := os.Stat(filepath.Join(scenarioPath, ".vrooli", "search.json")); statErr == nil {
		descriptor = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", false, statErr
	}
	caps := serviceCapabilities(scenarioPath)
	var matched []string
	for _, token := range searchCapabilityTokens {
		if caps[token] {
			matched = append(matched, token)
		}
	}
	switch {
	case descriptor && len(matched) > 0:
		return "descriptor+capability:" + strings.Join(matched, ","), true, nil
	case descriptor:
		return "descriptor", true, nil
	case len(matched) > 0:
		return "capability:" + strings.Join(matched, ","), true, nil
	default:
		return "", false, nil
	}
}

// serviceCapabilities reads a scenario's declared service capabilities from
// .vrooli/service.json, replicating Test Genie's resolver: the union of
// service.tags, service.capabilities, and the top-level capabilities string
// array, lowercased and trimmed. A missing or malformed manifest declares no
// capabilities (never an error) so one bad file cannot abort a fleet scan —
// again matching Test Genie's applicability behavior.
func serviceCapabilities(scenarioPath string) map[string]bool {
	caps := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join(scenarioPath, ".vrooli", "service.json"))
	if err != nil {
		return caps
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return caps
	}
	addServiceCapabilities(caps, nestedStringSlice(doc, "service", "tags"))
	addServiceCapabilities(caps, nestedStringSlice(doc, "service", "capabilities"))
	addServiceCapabilities(caps, nestedStringSlice(doc, "capabilities"))
	return caps
}

func nestedStringSlice(doc map[string]any, path ...string) []string {
	var current any = doc
	for _, segment := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = obj[segment]
	}
	values, ok := current.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func addServiceCapabilities(out map[string]bool, values []string) {
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key != "" {
			out[key] = true
		}
	}
}

func resolveRepoRoot(raw string) (string, error) {
	start := strings.TrimSpace(raw)
	if start == "" {
		start = "."
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(abs, "scenarios")); err == nil && info.IsDir() {
			return abs, nil
		}
		next := filepath.Dir(abs)
		if next == abs {
			return "", fmt.Errorf("could not find repo root with scenarios/ from %s", start)
		}
		abs = next
	}
}

// findingClass labels a finding as static (descriptor/corpus shape, visible in
// fast inventory) or execution (needs live eval/latency evidence, only proven in
// full certification).
func findingClass(code string) string {
	switch code {
	case "SEARCH_EVAL_RUN_MISSING", "SEARCH_EVAL_RUN_STALE", "SEARCH_EVAL_RUN_OUTDATED",
		"SEARCH_EVAL_RECALL_BELOW_TARGET", "SEARCH_EVAL_ASSERT_FAILED", "SEARCH_EVAL_LABELS_STALE",
		"SEARCH_EVAL_PROVIDER_UNAVAILABLE", "SEARCH_PERF_BUDGET_BREACH", "SEARCH_PERF_BUDGET_UNPROVEN",
		"SEARCH_PERF_DEGRADED":
		return "execution"
	default:
		return "static"
	}
}

// repairCommands turns a scenario's gating findings into the concrete next
// commands an operator runs, so a repair campaign can be planned from scan JSON.
func repairCommands(result scenarioResult) []string {
	if result.Status == "unavailable" {
		return []string{"vrooli scenario start search-hub"}
	}
	var static, execution bool
	for _, f := range result.Findings {
		if !f.Gating {
			continue
		}
		if f.FindingClass == "execution" {
			execution = true
		} else {
			static = true
		}
	}
	var cmds []string
	if static {
		cmds = append(cmds,
			"search-hub maturity fix "+result.Scenario+" --apply",
			"edit scenarios/"+result.Scenario+"/.vrooli/search.json")
	}
	if execution {
		suite := "<suite>"
		if len(result.EvalEvidence) > 0 && result.EvalEvidence[0].SuiteID != "" {
			suite = result.EvalEvidence[0].SuiteID
		}
		cmds = append(cmds, "search-hub evals run "+suite)
	}
	return cmds
}

func recommendedNextAction(result scenarioResult) string {
	if result.Status == "unavailable" {
		return "Start search-hub through lifecycle and rerun maturity scan."
	}
	if len(result.BlockingFindingCodes) > 0 {
		return "Resolve blocking search maturity findings: " + strings.Join(result.BlockingFindingCodes, ", ")
	}
	if len(result.AdvisoryFindingCodes) > 0 {
		return "Review advisory search maturity findings: " + strings.Join(result.AdvisoryFindingCodes, ", ")
	}
	return "No search maturity remediation needed."
}

func printFixReport(ctx cliapp.RunContext, report *scenariovalidationv1.FixResponse) {
	mode := "preview"
	if report.GetApplied() {
		mode = "apply"
	}
	fmt.Fprintf(ctx.Stdout(), "Search maturity fix %s for %s\n", mode, report.GetScenario())
	if len(report.GetCandidates()) == 0 {
		for _, message := range report.GetMessages() {
			fmt.Fprintf(ctx.Stdout(), "  %s\n", message)
		}
		if len(report.GetMessages()) == 0 {
			fmt.Fprintln(ctx.Stdout(), "  No changes.")
		}
		return
	}
	for _, candidate := range report.GetCandidates() {
		state := "would update"
		if candidate.GetApplied() {
			state = "updated"
		}
		fmt.Fprintf(ctx.Stdout(), "  %s %s [%s]\n", state, candidate.GetFilePath(), candidate.GetRuleId())
		if candidate.GetDescription() != "" {
			fmt.Fprintf(ctx.Stdout(), "    %s\n", candidate.GetDescription())
		}
	}
	if !report.GetApplied() {
		fmt.Fprintln(ctx.Stdout(), "  Re-run with --apply to write these mechanical descriptor fixes.")
	}
}

func printScanReport(ctx cliapp.RunContext, report scanReport) {
	mode := report.ValidationMode
	if mode == "" {
		mode = "full"
	}
	fmt.Fprintf(ctx.Stdout(), "Search maturity scan [%s] (%d scenario(s), provider %s)\n", mode, report.Summary.Total, report.ProviderLiveness.State)
	if report.ModeMeaning != "" {
		fmt.Fprintf(ctx.Stdout(), "  mode: %s\n", report.ModeMeaning)
	}
	fmt.Fprintf(ctx.Stdout(), "  passed=%d failed=%d degraded=%d unavailable=%d findings=%d blocking=%d advisory=%d\n",
		report.Summary.Passed, report.Summary.Failed, report.Summary.Degraded, report.Summary.Unavailable,
		report.Summary.Findings, report.Summary.Blocking, report.Summary.Advisory)
	if len(report.Results) == 0 {
		fmt.Fprintln(ctx.Stdout(), "  No search-applicable scenarios found.")
		return
	}
	for _, result := range report.Results {
		level := result.CurrentLevel
		if level == "" {
			level = "unknown"
		}
		reason := result.ApplicabilityReason
		if reason == "" {
			reason = "descriptor"
		}
		fmt.Fprintf(ctx.Stdout(), "  %-32s %-12s applies=%s level=%s findings=%d next=%s\n",
			result.Scenario, result.Status, reason, level, len(result.Findings), result.RecommendedNextAction)
		for _, finding := range result.Findings {
			scope := "blocking"
			if finding.Advisory {
				scope = "advisory"
			}
			fmt.Fprintf(ctx.Stdout(), "      %s [%s/%s/%s] %s", finding.Code, finding.CapabilityID, finding.FindingClass, scope, finding.Title)
			if finding.Location != "" {
				fmt.Fprintf(ctx.Stdout(), " (%s)", finding.Location)
			}
			fmt.Fprintln(ctx.Stdout())
		}
		for _, ev := range result.EvalEvidence {
			fmt.Fprintf(ctx.Stdout(), "      evidence %s: freshness=%s corpus=%s", ev.SuiteID, ev.Freshness, ev.CorpusStatus)
			if ev.LastRunID != "" {
				fmt.Fprintf(ctx.Stdout(), " run=%s", ev.LastRunID)
			}
			if ev.RecallTarget > 0 {
				fmt.Fprintf(ctx.Stdout(), " recall=%.2f/%.2f", ev.Recall, ev.RecallTarget)
			}
			if ev.LatencyP95Ms > 0 {
				fmt.Fprintf(ctx.Stdout(), " p95=%dms", ev.LatencyP95Ms)
			}
			fmt.Fprintln(ctx.Stdout())
		}
		for _, cmd := range result.RepairCommands {
			fmt.Fprintf(ctx.Stdout(), "      repair: %s\n", cmd)
		}
		if result.Error != "" {
			fmt.Fprintf(ctx.Stdout(), "      provider unavailable: %s\n", result.Error)
		}
	}
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	switch status {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED:
		return "passed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return "failed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED:
		return "degraded"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return "error"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED:
		return "skipped"
	default:
		return "unknown"
	}
}

func copyStringInt32Map(in map[string]int32) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = int(v)
	}
	return out
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = appendUnique(out, value)
	}
	sort.Strings(out)
	return out
}

func sortGroups(groups map[string][]string) {
	for key := range groups {
		sort.Strings(groups[key])
	}
}
