package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/api-core/markedrefs"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// Service is the validation application surface.
type Service interface {
	ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error)
	// CaptureBaseline captures the regression-anchor's baseline snapshot for a plan
	// from its typed anchor intent (scenario + baseline name), shelling
	// git-control-tower through the command-runner seam, then validates the
	// finalized snapshot status before calling it captured. This is the
	// execution-start "capture the before when before is true" action; it degrades
	// honestly (Captured=false + Detail) when the anchor intent is incomplete,
	// git-control-tower is unavailable, or the finalized manifest captured zero
	// surfaces — never a fabricated capture.
	CaptureBaseline(ctx context.Context, planID string) (BaselineCapture, error)
	RunValidation(ctx context.Context, planID, phaseID string) (Result, error)
	// LastValidation returns the most recent STORED validation result for a
	// plan/phase (the cheap read path the execution context server uses). ok=false
	// when none has been recorded yet, or when no result store is wired.
	LastValidation(ctx context.Context, planID, phaseID string) (Result, bool, error)
	VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error)
}

type service struct {
	plans     PlanSource
	resolver  ReferenceResolver
	staleness StalenessComputer
	runner    CommandRunner
	results   ResultStore
	clock     clock.Clock
	commands  CommandReferenceValidator
}

// Deps wires the validation Service. plans is required; resolver/staleness/runner/
// results are optional (nil => that capability degrades to a marked gap, never a
// false positive). A nil Results store means RunValidation still returns its live
// result but caches nothing — LastValidation then reports "no result yet".
type Deps struct {
	Plans     PlanSource
	Resolver  ReferenceResolver
	Staleness StalenessComputer
	Runner    CommandRunner
	Results   ResultStore
	Clock     clock.Clock
	Commands  CommandReferenceValidator
}

// NewService constructs the validation Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	return &service{
		plans:     d.Plans,
		resolver:  d.Resolver,
		staleness: d.Staleness,
		runner:    d.Runner,
		results:   d.Results,
		clock:     clk,
		commands:  d.Commands,
	}
}

var _ Service = (*service)(nil)

// scopedReferences returns the references in scope: a phase's references when
// phaseID is set, else the plan-level references.
func (s *service) scopedReferences(p planmodel.Plan, phaseID string) ([]planmodel.Reference, error) {
	if phaseID == "" {
		return p.References, nil
	}
	for _, ph := range p.Phases {
		if ph.ID == phaseID {
			return ph.References, nil
		}
	}
	return nil, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
}

func (s *service) ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return ReferenceReport{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return ReferenceReport{}, err
	}
	resolved, degraded := s.resolveAll(ctx, refs)
	return ReferenceReport{References: resolved, Degraded: degraded}, nil
}

// resolveAll resolves every reference, degrading honestly. A nil resolver or a
// per-reference error marks that reference UNRESOLVED (or FUTURE preserved) and
// flags the report degraded.
func (s *service) resolveAll(ctx context.Context, refs []planmodel.Reference) ([]planmodel.Reference, bool) {
	out := make([]planmodel.Reference, 0, len(refs))
	degraded := false
	for _, ref := range refs {
		if ref.Future {
			ref.Resolution = planmodel.ResolutionFuture
			out = append(out, ref)
			continue
		}
		if s.resolver == nil {
			ref.Resolution = planmodel.ResolutionUnresolved
			ref.Note = "code-facts unavailable"
			degraded = true
			out = append(out, ref)
			continue
		}
		got, err := s.resolver.Resolve(ctx, ref)
		if err != nil {
			ref.Resolution = planmodel.ResolutionUnresolved
			ref.Note = "resolve failed: " + err.Error()
			degraded = true
			out = append(out, ref)
			continue
		}
		out = append(out, got)
	}
	return out, degraded
}

func (s *service) ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error) {
	report, err := s.ResolveReferences(ctx, planID, phaseID)
	if err != nil {
		return ReferenceReport{}, err
	}
	// The regression anchor's HeadSha is the "before" point against which a
	// still-present reference is graded fresh vs lightly-stale. The platform
	// freshness engine is scenario-artifact scoped today, so per-reference
	// change magnitude remains git-sourced until that substrate exposes a
	// reference-level contract.
	headSha := ""
	if p, gerr := s.plans.GetPlan(ctx, planID); gerr == nil {
		headSha = p.RegressionAnchor.HeadSha
	}
	overall := planmodel.StalenessFresh
	anyKnown := false
	for i := range report.References {
		ref := report.References[i]
		if ref.Future {
			continue // proposed code is never "stale"
		}
		if s.staleness == nil {
			ref.Staleness = planmodel.StalenessUnknown
			report.Degraded = true
			report.References[i] = ref
			continue
		}
		tier, factor, err := s.staleness.Compute(ctx, ref)
		if err != nil {
			ref.Staleness = planmodel.StalenessUnknown
			report.Degraded = true
			report.References[i] = ref
			continue
		}
		// Refine the existence floor: a still-present reference (FRESH) whose code
		// changed since the anchor is LIGHTLY_STALE ("small diffs in referenced
		// code"). DEFINITELY_STALE (moved/deleted) is never downgraded. Absent a
		// HeadSha or a git runner, the floor's FRESH stands — honest, never guessed.
		if tier == planmodel.StalenessFresh {
			if t2, f2, refined := s.gitChangeTier(ctx, headSha, ref); refined {
				tier, factor = t2, f2
			}
		}
		ref.Staleness = tier
		ref.ChangeFactor = factor
		report.References[i] = ref
		anyKnown = true
		if stalenessRank(tier) > stalenessRank(overall) {
			overall = tier
		}
	}
	if !anyKnown {
		overall = planmodel.StalenessUnknown
	}
	report.Overall = overall
	return report, nil
}

// gitChangeTier upgrades a still-present (FRESH) reference to LIGHTLY_STALE when
// its location has changed since the anchor's HeadSha, with a change factor from
// the diff magnitude. Returns refined=false (keep FRESH) when there is no anchor
// sha, no runner, the tool is absent, the ref is not file-backed, or nothing
// changed. Uses `git diff --numstat <sha> -- <target>` because the live
// freshness engine only reports scenario-artifact staleness, not per-reference
// code drift. Empty output (exit 0) means unchanged; non-empty means changed.
func (s *service) gitChangeTier(ctx context.Context, headSha string, ref planmodel.Reference) (planmodel.StalenessTier, float64, bool) {
	headSha = strings.TrimSpace(headSha)
	if headSha == "" || s.runner == nil {
		return "", 0, false
	}
	if ref.Kind != planmodel.ReferenceCode && ref.Kind != planmodel.ReferenceDoc {
		return "", 0, false
	}
	out, err := s.runner(ctx, "git", "diff", "--numstat", headSha, "--", ref.Target)
	if err != nil {
		return "", 0, false // tool absent / bad sha → keep the existence floor
	}
	added, deleted, changed := parseNumstat(string(out))
	if !changed {
		return "", 0, false
	}
	factor := float64(added+deleted) / 200.0
	switch {
	case factor > 1:
		factor = 1
	case factor <= 0:
		factor = 0.05 // changed but tiny — still a non-zero signal
	}
	return planmodel.StalenessLightlyStale, factor, true
}

// parseNumstat sums the added/deleted columns of `git diff --numstat` output.
// Binary files report "-" for both counts and are treated as changed with zero
// line magnitude. changed=false only when there are no data rows at all.
func parseNumstat(out string) (added, deleted int, changed bool) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		changed = true
		if a, err := strconv.Atoi(fields[0]); err == nil {
			added += a
		}
		if d, err := strconv.Atoi(fields[1]); err == nil {
			deleted += d
		}
	}
	return added, deleted, changed
}

func (s *service) DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineScope{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return BaselineScope{}, err
	}
	return deriveScope(p, refs, effectiveBoundary(p, phaseID)), nil
}

func (s *service) CaptureBaseline(ctx context.Context, planID string) (BaselineCapture, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return BaselineCapture{}, err
	}
	anchor := p.RegressionAnchor
	scenario := strings.TrimSpace(anchor.Scenario)
	name := strings.TrimSpace(anchor.BaselineName)
	// An unconfirmed authoring placeholder (e.g. "<scenario>") is not a real
	// target — degrade honestly so the agent confirms the anchor intent.
	if scenario == "" || strings.ContainsAny(scenario, "<>") {
		return BaselineCapture{BaselineName: name, Detail: "regression-anchor scenario is unset or still a placeholder; confirm the anchor intent before execution"}, nil
	}
	if name == "" {
		name = scenario + "-baseline"
	}
	if s.runner == nil {
		return BaselineCapture{Scenario: scenario, BaselineName: name, Detail: "no command runner configured (git-control-tower unavailable)"}, nil
	}
	reason := "execution-start baseline for plan " + planIdentifier(p)
	if _, err := s.runner(ctx, "git-control-tower", "baseline", "snapshot", "--scenario", scenario, "--name", name, "--reason", reason); err != nil {
		return BaselineCapture{Scenario: scenario, BaselineName: name, Detail: "baseline snapshot failed: " + err.Error()}, nil
	}
	health := s.validateCapturedBaseline(ctx, scenario, name)
	if !health.usable {
		return BaselineCapture{
			Scenario:             scenario,
			BaselineName:         name,
			CapturedSurfaceCount: health.capturedSurfaceCount,
			SkippedSurfaces:      health.skipped,
			Detail:               health.detail,
		}, nil
	}
	return BaselineCapture{
		Captured:             true,
		Scenario:             scenario,
		BaselineName:         name,
		CapturedSurfaceCount: health.capturedSurfaceCount,
		SkippedSurfaces:      health.skipped,
		Detail:               fmt.Sprintf("captured baseline %s for %s with %d surface(s)", name, scenario, health.capturedSurfaceCount),
	}, nil
}

type baselineCaptureHealth struct {
	usable               bool
	capturedSurfaceCount int
	skipped              map[string]string
	detail               string
}

func (s *service) validateCapturedBaseline(ctx context.Context, scenario, name string) baselineCaptureHealth {
	out, err := s.runner(ctx, "git-control-tower", "baseline", "snapshot", "status", "--scenario", scenario, "--name", name, "--wait", "--json")
	if err != nil {
		return baselineCaptureHealth{
			detail: fmt.Sprintf("baseline snapshot started but finalized validation failed: %v; treat %q as unusable and fall back to HEAD sha + allowlist diff until git-control-tower reports captured surfaces", err, name),
		}
	}
	count, skipped, parseErr := parseBaselineStatusHealth(out)
	if parseErr != nil {
		return baselineCaptureHealth{
			detail: fmt.Sprintf("baseline snapshot completed but validation output was unreadable: %v; treat %q as unusable and fall back to HEAD sha + allowlist diff", parseErr, name),
		}
	}
	if count == 0 {
		return baselineCaptureHealth{
			capturedSurfaceCount: count,
			skipped:              skipped,
			detail:               fmt.Sprintf("baseline %q is unusable: captured 0 surfaces; %s; fall back to HEAD sha + allowlist diff until the skipped surfaces are fixed", name, summarizeSkippedSurfaces(skipped)),
		}
	}
	return baselineCaptureHealth{usable: true, capturedSurfaceCount: count, skipped: skipped}
}

func parseBaselineStatusHealth(out []byte) (int, map[string]string, error) {
	var resp struct {
		Status   string `json:"status"`
		Error    string `json:"error"`
		Baseline struct {
			Surfaces map[string]json.RawMessage `json:"surfaces"`
			Skipped  map[string]string          `json:"skipped"`
		} `json:"baseline"`
	}
	if err := json.Unmarshal(out, &resp); err == nil {
		if resp.Status != "" && resp.Status != "ready" {
			detail := strings.TrimSpace(resp.Error)
			if detail == "" {
				detail = resp.Status
			}
			return 0, resp.Baseline.Skipped, fmt.Errorf("snapshot status %s", detail)
		}
		return len(resp.Baseline.Surfaces), resp.Baseline.Skipped, nil
	}
	text := string(out)
	count := 0
	inSurfaces := false
	inSkipped := false
	skipped := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "surfaces:":
			inSurfaces = true
			inSkipped = false
		case strings.HasPrefix(trimmed, "not captured"):
			inSurfaces = false
			inSkipped = true
		case inSurfaces && trimmed != "":
			count++
		case inSkipped && trimmed != "":
			trimmed = strings.TrimLeft(trimmed, "!* \t")
			fields := strings.Fields(trimmed)
			if len(fields) > 1 && len([]rune(fields[0])) == 1 && !strings.Contains(fields[0], ":") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, fields[0]))
				fields = strings.Fields(trimmed)
			}
			if len(fields) > 0 {
				reason := strings.TrimSpace(strings.TrimPrefix(trimmed, fields[0]))
				skipped[fields[0]] = reason
			}
		}
	}
	if count == 0 && len(skipped) == 0 && strings.TrimSpace(text) == "" {
		return 0, nil, errors.New("empty baseline status output")
	}
	if len(skipped) == 0 {
		skipped = nil
	}
	return count, skipped, nil
}

func summarizeSkippedSurfaces(skipped map[string]string) string {
	if len(skipped) == 0 {
		return "no skipped-surface reason was reported"
	}
	keys := make([]string, 0, len(skipped))
	for k := range skipped {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		reason := strings.TrimSpace(skipped[k])
		if reason == "" {
			parts = append(parts, k)
			continue
		}
		parts = append(parts, k+": "+reason)
	}
	return "skipped surfaces: " + strings.Join(parts, "; ")
}

// planIdentifier prefers the human slug for log/reason text, falling back to id.
func planIdentifier(p planmodel.Plan) string {
	if s := strings.TrimSpace(p.Slug); s != "" {
		return s
	}
	return strings.TrimSpace(p.ID)
}

func (s *service) RunValidation(ctx context.Context, planID, phaseID string) (Result, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Result{}, err
	}
	refs, err := s.scopedReferences(p, phaseID)
	if err != nil {
		return Result{}, err
	}
	scope := deriveScope(p, refs, effectiveBoundary(p, phaseID))
	staleReport, _ := s.ComputeStaleness(ctx, planID, phaseID)
	res := Result{
		ID:          uuid.NewString(),
		PlanID:      p.ID, // canonical id so the execution context server reads the same key
		PhaseID:     phaseID,
		CommandsRun: scope.Commands,
		Staleness:   staleReport.Overall,
		RanAt:       s.now(),
	}
	res.Verdict, res.Detail = s.runCommands(ctx, scope.Commands)
	structureFindings, structureVerdict, structureDetail := validateRelevantContextStructure(p, phaseID)
	commandFindings, commandVerdict, commandDetail := s.validatePlanCommands(ctx, p, phaseID)
	contextRefFindings, contextRefVerdict, contextRefDetail := s.validateContextReferences(ctx, p, phaseID)
	res.CommandFindings = append(structureFindings, commandFindings...)
	res.CommandFindings = append(res.CommandFindings, contextRefFindings...)
	res.Verdict = combineValidationVerdicts(res.Verdict, structureVerdict)
	res.Verdict = combineValidationVerdicts(res.Verdict, commandVerdict)
	res.Verdict = combineValidationVerdicts(res.Verdict, contextRefVerdict)
	res.Detail = joinDetails(res.Detail, structureDetail, commandDetail, contextRefDetail)
	// Persist for the cheap-read context path (status/next). Best-effort: a cache
	// write failure must not fail the live validation the agent asked for.
	if s.results != nil {
		_ = s.results.SaveResult(ctx, res)
	}
	return res, nil
}

func validateRelevantContextStructure(p planmodel.Plan, phaseID string) ([]CommandFinding, Verdict, string) {
	var findings []CommandFinding
	if phaseID == "" {
		findings = append(findings, validateContextItems("plan.relevant_context", p.RelevantContext)...)
		for _, phase := range p.Phases {
			findings = append(findings, validatePhaseContextStructure(phase)...)
		}
	} else {
		found := false
		for _, phase := range p.Phases {
			if phase.ID != phaseID {
				continue
			}
			found = true
			findings = append(findings, validatePhaseContextStructure(phase)...)
			break
		}
		if !found {
			findings = append(findings, CommandFinding{
				Verdict:  string(VerdictUnknown),
				Message:  fmt.Sprintf("phase %q was not found for relevant context validation", phaseID),
				Location: "phase." + phaseID,
			})
		}
	}
	if len(findings) == 0 {
		return nil, VerdictPass, ""
	}
	verdict := VerdictPass
	for _, finding := range findings {
		switch finding.Verdict {
		case string(VerdictFail):
			verdict = VerdictFail
		case string(VerdictUnknown):
			verdict = combineValidationVerdicts(verdict, VerdictUnknown)
		}
	}
	return findings, verdict, relevantContextFindingsDetail(findings)
}

func validatePhaseContextStructure(phase planmodel.Phase) []CommandFinding {
	location := "phase." + phase.ID
	if strings.TrimSpace(phase.ID) == "" {
		location = fmt.Sprintf("phase.%d", phase.Order)
	}
	findings := validateContextItems(location+".relevant_context", phase.RelevantContext)
	if !phaseHasContextOrNoContextReason(phase) {
		findings = append(findings, CommandFinding{
			Verdict:  string(VerdictFail),
			Message:  "phase has no relevant context and no explicit NO_CONTEXT reason",
			Location: location + ".relevant_context",
			IssueCodes: []string{
				"missing_phase_context",
			},
			Guidance: []string{"Add phase relevant context or an operator note starting with NO_CONTEXT: when no setup is useful."},
		})
	}
	return findings
}

func phaseHasContextOrNoContextReason(phase planmodel.Phase) bool {
	if len(phase.RelevantContext) > 0 || len(phase.RequiredReading) > 0 {
		return true
	}
	for _, reminder := range phase.Reminders {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(reminder)), "NO_CONTEXT:") {
			return true
		}
	}
	for _, item := range phase.RelevantContext {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(firstNonEmpty(item.Instruction, item.Label, item.Reason))), "NO_CONTEXT:") {
			return true
		}
	}
	return false
}

func validateContextItems(location string, items []planmodel.RelevantContextItem) []CommandFinding {
	var findings []CommandFinding
	for i, item := range items {
		itemLocation := fmt.Sprintf("%s[%d]", location, i)
		if item.Required && item.RepeatPolicy == "" {
			findings = append(findings, contextStructureFinding(itemLocation, "missing_repeat_policy", "required context item has no repeat policy"))
		}
		if item.Required && !contextItemHasPayload(item) {
			findings = append(findings, contextStructureFinding(itemLocation, "missing_context_payload", "required context item has no command, argv, target, instruction, or note payload"))
		}
		if item.Kind == planmodel.RelevantContextCommand || item.Kind == planmodel.RelevantContextSearch {
			if strings.TrimSpace(item.Reason) == "" {
				findings = append(findings, contextStructureFinding(itemLocation+".reason", "missing_context_reason", "command/search context item has no reason"))
			}
			if strings.TrimSpace(item.Instruction) == "" {
				findings = append(findings, contextStructureFinding(itemLocation+".instruction", "missing_context_instruction", "command/search context item has no instruction"))
			}
			if strings.TrimSpace(item.Command) == "" && len(item.Argv) == 0 && strings.TrimSpace(item.Target) == "" {
				findings = append(findings, contextStructureFinding(itemLocation+".command", "missing_context_command", "command/search context item has no runnable command, argv, or target"))
			}
		}
	}
	return findings
}

func contextItemHasPayload(item planmodel.RelevantContextItem) bool {
	if strings.TrimSpace(item.Command) != "" || len(item.Argv) > 0 || strings.TrimSpace(item.Target) != "" {
		return true
	}
	return item.Kind == planmodel.RelevantContextNote && strings.TrimSpace(firstNonEmpty(item.Instruction, item.Label, item.Reason)) != ""
}

func contextStructureFinding(location, code, message string) CommandFinding {
	return CommandFinding{
		Verdict:    string(VerdictFail),
		Message:    message,
		Location:   location,
		IssueCodes: []string{code},
	}
}

func relevantContextFindingsDetail(findings []CommandFinding) string {
	if len(findings) == 0 {
		return ""
	}
	lines := []string{"relevant context structure validation:"}
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", finding.Verdict, finding.Location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

func (s *service) validatePlanCommands(ctx context.Context, p planmodel.Plan, phaseID string) ([]CommandFinding, Verdict, string) {
	refs, err := commandRefsForScope(p, phaseID)
	if err != nil {
		return []CommandFinding{{Verdict: string(VerdictUnknown), Message: err.Error()}}, VerdictUnknown, "command reference validation unknown: " + err.Error()
	}
	if len(refs) == 0 {
		return nil, VerdictPass, ""
	}
	if s.commands == nil {
		return []CommandFinding{{Verdict: string(VerdictUnknown), Message: "CLI Health command validator unavailable"}}, VerdictUnknown, "command reference validation unknown: CLI Health command validator unavailable"
	}
	var findings []CommandFinding
	verdict := VerdictPass
	for _, ref := range refs {
		if !markedrefs.RequiresExistence(ref.ref) {
			continue
		}
		result, err := s.commands.ValidateCommandReference(ctx, CommandReferenceRequest{
			CommandText: ref.ref.Value,
			Qualifiers:  append([]string(nil), ref.ref.Qualifiers...),
		})
		if err != nil {
			findings = append(findings, CommandFinding{
				CommandText: ref.ref.Value,
				Verdict:     string(VerdictUnknown),
				Message:     "CLI Health unavailable: " + err.Error(),
				Location:    ref.location,
			})
			verdict = combineValidationVerdicts(verdict, VerdictUnknown)
			continue
		}
		finding := CommandFinding{
			CommandText: ref.ref.Value,
			Verdict:     result.Verdict,
			Level:       result.ValidationLevel,
			Message:     commandResultMessage(result),
			Location:    ref.location,
			IssueCodes:  commandIssueCodes(result.Issues),
			Suggestions: append([]string(nil), result.Suggestions...),
			Guidance:    append([]string(nil), result.Guidance...),
		}
		switch strings.ToLower(result.Verdict) {
		case "valid", "skipped":
		case "partial":
			findings = append(findings, finding)
		case "invalid", "unsupported":
			findings = append(findings, finding)
			verdict = VerdictFail
		default:
			findings = append(findings, finding)
			verdict = combineValidationVerdicts(verdict, VerdictUnknown)
		}
	}
	return findings, verdict, commandFindingsDetail(findings)
}

type scopedCommandRef struct {
	ref      markedrefs.Reference
	location string
}

func commandRefsForScope(p planmodel.Plan, phaseID string) ([]scopedCommandRef, error) {
	var out []scopedCommandRef
	if phaseID == "" {
		addCommandRefs(&out, "plan.purpose", p.Purpose)
		addCommandRefs(&out, "plan.scope", p.Scope)
		addCommandRefs(&out, "plan.constraints", p.Constraints)
		addCommandRefs(&out, "plan.definition_of_done", p.DefinitionOfDone)
		addContextCommandRefs(&out, "plan.relevant_context", p.RelevantContext)
		for _, phase := range p.Phases {
			addCommandRefs(&out, "phase."+phase.ID+".intent", phase.Intent)
			for i, item := range phase.RequiredReading {
				addCommandRefs(&out, fmt.Sprintf("phase.%s.required_reading[%d]", phase.ID, i), item)
			}
			for i, item := range phase.Reminders {
				addCommandRefs(&out, fmt.Sprintf("phase.%s.reminders[%d]", phase.ID, i), item)
			}
			addCommandRefs(&out, "phase."+phase.ID+".acceptance", phase.Acceptance)
			addContextCommandRefs(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		}
		return out, nil
	}
	for _, phase := range p.Phases {
		if phase.ID != phaseID {
			continue
		}
		addCommandRefs(&out, "phase."+phase.ID+".intent", phase.Intent)
		for i, item := range phase.RequiredReading {
			addCommandRefs(&out, fmt.Sprintf("phase.%s.required_reading[%d]", phase.ID, i), item)
		}
		for i, item := range phase.Reminders {
			addCommandRefs(&out, fmt.Sprintf("phase.%s.reminders[%d]", phase.ID, i), item)
		}
		addCommandRefs(&out, "phase."+phase.ID+".acceptance", phase.Acceptance)
		addContextCommandRefs(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		return out, nil
	}
	return nil, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
}

func addCommandRefs(out *[]scopedCommandRef, location, text string) {
	for lineNumber, line := range strings.Split(text, "\n") {
		for _, ref := range markedrefs.ParseInlineCode(line, lineNumber+1) {
			if ref.Marker == markedrefs.MarkerCLI {
				*out = append(*out, scopedCommandRef{ref: ref, location: location})
			}
		}
	}
}

func addContextCommandRefs(out *[]scopedCommandRef, location string, items []planmodel.RelevantContextItem) {
	for i, item := range items {
		itemLocation := fmt.Sprintf("%s[%d]", location, i)
		addCommandRefs(out, itemLocation+".instruction", item.Instruction)
		addCommandRefs(out, itemLocation+".reason", item.Reason)
		addCommandRefs(out, itemLocation+".command", item.Command)
		if item.Kind != planmodel.RelevantContextCommand && item.Kind != planmodel.RelevantContextSearch {
			continue
		}
		command := strings.TrimSpace(item.Command)
		if command == "" && len(item.Argv) > 0 {
			command = strings.Join(item.Argv, " ")
		}
		if command == "" {
			continue
		}
		*out = append(*out, scopedCommandRef{
			ref: markedrefs.Reference{
				Marker: markedrefs.MarkerCLI,
				Value:  command,
			},
			location: itemLocation + ".command",
		})
	}
}

func commandResultMessage(result CommandReferenceResult) string {
	var parts []string
	for _, issue := range result.Issues {
		if issue.Code != "" && issue.Message != "" {
			parts = append(parts, issue.Code+": "+issue.Message)
		} else if issue.Message != "" {
			parts = append(parts, issue.Message)
		}
	}
	for _, suggestion := range result.Suggestions {
		if suggestion != "" {
			parts = append(parts, "suggestion: "+suggestion)
		}
	}
	parts = append(parts, result.Guidance...)
	if len(parts) == 0 {
		return strings.TrimSpace(result.Verdict + " " + result.ValidationLevel)
	}
	return strings.Join(parts, "; ")
}

func (s *service) validateContextReferences(ctx context.Context, p planmodel.Plan, phaseID string) ([]CommandFinding, Verdict, string) {
	refs, err := contextReferencesForScope(p, phaseID)
	if err != nil {
		return []CommandFinding{{Verdict: string(VerdictUnknown), Message: err.Error()}}, VerdictUnknown, "relevant context reference validation unknown: " + err.Error()
	}
	if len(refs) == 0 {
		return nil, VerdictPass, ""
	}
	if s.resolver == nil {
		return []CommandFinding{{
			Verdict:  string(VerdictUnknown),
			Message:  "reference resolver unavailable",
			Location: "relevant_context",
		}}, VerdictUnknown, "relevant context reference validation unknown: reference resolver unavailable"
	}
	var findings []CommandFinding
	verdict := VerdictPass
	for _, ref := range refs {
		resolved, err := s.resolver.Resolve(ctx, ref.ref)
		if err != nil {
			findings = append(findings, CommandFinding{
				CommandText: ref.ref.Target,
				Verdict:     string(VerdictUnknown),
				Message:     "reference resolver unavailable: " + err.Error(),
				Location:    ref.location,
				IssueCodes:  []string{"context_reference_resolver_error"},
			})
			verdict = combineValidationVerdicts(verdict, VerdictUnknown)
			continue
		}
		switch resolved.Resolution {
		case planmodel.ResolutionResolved, planmodel.ResolutionFuture:
			continue
		case planmodel.ResolutionMissing, planmodel.ResolutionUnresolved:
			findings = append(findings, CommandFinding{
				CommandText: ref.ref.Target,
				Verdict:     string(VerdictFail),
				Message:     contextReferenceMessage(resolved),
				Location:    ref.location,
				IssueCodes:  []string{"context_reference_unresolved"},
			})
			verdict = VerdictFail
		default:
			findings = append(findings, CommandFinding{
				CommandText: ref.ref.Target,
				Verdict:     string(VerdictUnknown),
				Message:     contextReferenceMessage(resolved),
				Location:    ref.location,
				IssueCodes:  []string{"context_reference_unknown"},
			})
			verdict = combineValidationVerdicts(verdict, VerdictUnknown)
		}
	}
	return findings, verdict, contextReferenceFindingsDetail(findings)
}

type scopedContextReference struct {
	ref      planmodel.Reference
	location string
}

func contextReferencesForScope(p planmodel.Plan, phaseID string) ([]scopedContextReference, error) {
	var out []scopedContextReference
	if phaseID == "" {
		addContextReferences(&out, "plan.relevant_context", p.RelevantContext)
		for _, phase := range p.Phases {
			addContextReferences(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		}
		return out, nil
	}
	for _, phase := range p.Phases {
		if phase.ID != phaseID {
			continue
		}
		addContextReferences(&out, "phase."+phase.ID+".relevant_context", phase.RelevantContext)
		return out, nil
	}
	return nil, ErrPhaseNotFound{PlanID: p.ID, PhaseID: phaseID}
}

func addContextReferences(out *[]scopedContextReference, location string, items []planmodel.RelevantContextItem) {
	for i, item := range items {
		kind, ok := contextReferenceKind(item.Kind)
		if !ok {
			continue
		}
		target := strings.TrimSpace(item.Target)
		if target == "" {
			continue
		}
		*out = append(*out, scopedContextReference{
			ref: planmodel.Reference{
				ID:     item.ID,
				Kind:   kind,
				Target: target,
			},
			location: fmt.Sprintf("%s[%d].target", location, i),
		})
	}
}

func contextReferenceKind(kind planmodel.RelevantContextKind) (planmodel.ReferenceKind, bool) {
	switch kind {
	case planmodel.RelevantContextCodeRef:
		return planmodel.ReferenceCode, true
	case planmodel.RelevantContextDoc:
		return planmodel.ReferenceDoc, true
	case planmodel.RelevantContextReqRef:
		return planmodel.ReferenceReq, true
	default:
		return "", false
	}
}

func contextReferenceMessage(ref planmodel.Reference) string {
	if strings.TrimSpace(ref.Note) != "" {
		return ref.Note
	}
	if ref.Resolution != "" {
		return "relevant context reference " + string(ref.Resolution)
	}
	return "relevant context reference resolution unknown"
}

func contextReferenceFindingsDetail(findings []CommandFinding) string {
	if len(findings) == 0 {
		return ""
	}
	lines := []string{"relevant context reference validation:"}
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", finding.Verdict, finding.Location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

func commandIssueCodes(issues []CommandIssue) []string {
	var out []string
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) != "" {
			out = append(out, issue.Code)
		}
	}
	return out
}

func commandFindingsDetail(findings []CommandFinding) string {
	if len(findings) == 0 {
		return ""
	}
	lines := []string{"command reference validation:"}
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s %s (%s): %s", finding.Verdict, finding.CommandText, finding.Location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

func combineValidationVerdicts(a, b Verdict) Verdict {
	if a == VerdictFail || b == VerdictFail {
		return VerdictFail
	}
	if a == VerdictUnknown || b == VerdictUnknown || a == VerdictUnspecified || b == VerdictUnspecified {
		return VerdictUnknown
	}
	return VerdictPass
}

func joinDetails(parts ...string) string {
	var nonEmpty []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// LastValidation returns the most recent STORED validation result for a
// plan/phase — the cheap read the execution context server uses for status/next
// so those verbs never shell a subprocess. ok=false when nothing has been run yet
// or no store is wired.
func (s *service) LastValidation(ctx context.Context, planID, phaseID string) (Result, bool, error) {
	if s.results == nil {
		return Result{}, false, nil
	}
	if p, err := s.plans.GetPlan(ctx, planID); err == nil {
		planID = p.ID
	}
	return s.results.LastResult(ctx, planID, phaseID)
}

func (s *service) VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Result{}, false, err
	}
	commands := p.RegressionAnchor.Commands
	if len(commands) == 0 && !p.RegressionAnchor.Unavailable {
		// Wizard-authored plans carry the anchor as captured prose with no explicit
		// commands. Derive the oracle command set from the plan's connected code so
		// DoD verifies against a real diff oracle instead of always degrading to
		// UNKNOWN (the authoring→DoD gap).
		commands = deriveScope(p, p.References, p.ChangeBoundary).Commands
	}
	res := Result{PlanID: planID, CommandsRun: commands, RanAt: s.now()}
	if p.RegressionAnchor.Unavailable || len(commands) == 0 {
		res.Verdict = VerdictUnknown
		res.Detail = "regression anchor unavailable; DoD cannot be verified against an oracle"
		return res, false, nil
	}
	res.Verdict, res.Detail = s.runCommands(ctx, commands)
	return res, res.Verdict == VerdictPass, nil
}

// isOracleCommand reports whether a derived command has trustworthy pass/fail
// exit semantics. Only a git-control-tower baseline diff is an oracle (exit 0
// safe, 1 regression, 2 not-comparable). A bare `git diff`/`git diff --stat`
// exits 0 essentially always, so it is INFORMATIONAL — run for its output and
// surfaced to the agent, but it never determines the verdict (treating it as an
// oracle is how "validation passed" used to mean only "git ran").
func isOracleCommand(cmd string) bool {
	return strings.HasPrefix(strings.TrimSpace(cmd), "git-control-tower baseline diff ")
}

// runCommands runs the derived command set and computes a verdict from the
// ORACLE commands only. A tool that is not installed yields UNKNOWN for that
// command (not FAIL — absence of git-control-tower must not look like a
// regression); a baseline diff exit 2 ("not comparable") is UNKNOWN; any other
// non-zero oracle exit is FAIL. PASS requires at least one oracle to have run
// cleanly with no oracle failing or going unknown. With no oracle command at all
// (e.g. only an informational repo-level diff), the verdict is UNKNOWN — honest,
// never a fabricated pass.
func (s *service) runCommands(ctx context.Context, commands []string) (Verdict, string) {
	if len(commands) == 0 {
		return VerdictUnknown, "no baseline commands derived"
	}
	if s.runner == nil {
		return VerdictUnknown, "no command runner configured (git-control-tower unavailable)"
	}
	var (
		details       []string
		oraclePassed  int
		oracleFailed  bool
		oracleUnknown bool
	)
	for _, cmd := range commands {
		result, ok := s.runOneCommand(ctx, cmd)
		if !ok {
			continue
		}
		details = append(details, result.detail)
		if !result.oracle {
			continue
		}
		switch result.verdict {
		case VerdictPass:
			oraclePassed++
		case VerdictFail:
			oracleFailed = true
		default:
			oracleUnknown = true
		}
	}
	detail := strings.Join(details, "\n")
	switch {
	case oracleFailed:
		return VerdictFail, detail
	case oracleUnknown:
		return VerdictUnknown, detail
	case oraclePassed > 0:
		return VerdictPass, detail
	default:
		return VerdictUnknown, detail
	}
}

type commandRunResult struct {
	oracle  bool
	verdict Verdict
	detail  string
}

func (s *service) runOneCommand(ctx context.Context, cmd string) (commandRunResult, bool) {
	name, args := splitCommand(cmd)
	if name == "" {
		return commandRunResult{}, false
	}
	oracle := isOracleCommand(cmd)
	_, err := s.runner(ctx, name, args...)
	return classifyCommandRun(cmd, oracle, err), true
}

func classifyCommandRun(cmd string, oracle bool, err error) commandRunResult {
	result := commandRunResult{oracle: oracle, verdict: VerdictPass, detail: fmt.Sprintf("ok %s", cmd)}
	if err == nil {
		return result
	}
	if errors.Is(err, ErrToolNotFound) {
		result.verdict = VerdictUnknown
		result.detail = fmt.Sprintf("unknown %s: tool not found", cmd)
		return result
	}
	var exitErr CommandExitError
	if errors.As(err, &exitErr) && exitErr.Code == 2 {
		result.verdict = VerdictUnknown
		result.detail = fmt.Sprintf("unknown %s: not comparable (exit 2)", cmd)
		return result
	}
	result.verdict = VerdictFail
	result.detail = fmt.Sprintf("FAIL %s: %v", cmd, err)
	return result
}

func (s *service) now() string {
	return s.clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
}

// deriveScope computes the exact baseline/validation command set across all
// affected locations for a plan/phase. The CHANGE BOUNDARY is the source of
// truth: affected scenarios come from acceptance_allow first, supplemented by
// scenario-scoped code references (a reference to a scenario outside the boundary
// is still included, so validation never under-covers). Non-scenario allow globs
// and non-scenario references are repo-level paths with no scenario baseline
// oracle today.
//
// Commands are emitted in tiers:
//   - one git-control-tower snapshot-status + baseline diff ORACLE pair per
//     affected scenario, only when a baseline name exists (the verified GCT CLI
//     requires --scenario and --name); no oracle is fabricated otherwise.
//   - one INFORMATIONAL `git diff --stat [<sha>] -- <repo paths>` for the
//     non-scenario allowed paths — never an oracle (see isOracleCommand).
//
// The plan's own regression-anchor commands are folded in. Output is deduped and
// stably ordered so the command set is deterministic.
func deriveScope(p planmodel.Plan, refs []planmodel.Reference, boundary planmodel.ChangeBoundary) BaselineScope {
	scenarios := map[string]bool{}
	for _, name := range boundary.AffectedScenarios() {
		scenarios[name] = true
	}
	repoLevel := false
	for _, ref := range refs {
		if ref.Kind != planmodel.ReferenceCode || ref.Future {
			continue
		}
		if name := scenarioFromTarget(ref.Target); name != "" {
			scenarios[name] = true
		} else {
			repoLevel = true
		}
	}

	locations := make([]string, 0, len(scenarios)+1)
	commands := make([]string, 0, len(scenarios)+2)
	for _, c := range planmodel.RegressionAnchorCommands(p.RegressionAnchor) {
		commands = appendUnique(commands, c)
	}
	switch p.RegressionAnchor.Strategy {
	case planmodel.AnchorStrategyScenarioBaseline:
		if scenario := strings.TrimSpace(p.RegressionAnchor.Scenario); scenario != "" {
			locations = appendUnique(locations, "scenarios/"+scenario)
		}
	case planmodel.AnchorStrategyHeadShaAllowlist:
		locations = appendUnique(locations, "repo")
	}
	names := make([]string, 0, len(scenarios))
	for name := range scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		locations = appendUnique(locations, "scenarios/"+name)
		commands = append(commands, baselineCommands(name, p.RegressionAnchor)...)
	}
	// Repo-level paths: boundary non-scenario allow globs (preferred, path-scoped)
	// plus any non-scenario reference. These have no scenario baseline oracle, so
	// emit a single INFORMATIONAL diff scoped to the boundary's repo paths.
	repoPaths := boundary.RepoPaths()
	if len(repoPaths) > 0 || repoLevel {
		locations = appendUnique(locations, "repo")
		cmd := "git diff --stat"
		if sha := strings.TrimSpace(p.RegressionAnchor.HeadSha); sha != "" && !planmodel.ContainsUnresolvedPlaceholder(sha) {
			cmd += " " + sha
		}
		if len(repoPaths) > 0 {
			cmd += " -- " + strings.Join(repoPaths, " ")
		}
		commands = appendUnique(commands, cmd)
	}
	for _, c := range p.RegressionAnchor.Commands {
		commands = appendUnique(commands, c)
	}
	return BaselineScope{Commands: commands, Locations: locations}
}

// effectiveBoundary returns the boundary that scopes validation for a phase: the
// phase's own boundary when it declares one (a narrowing refinement), otherwise
// the plan-level boundary.
func effectiveBoundary(p planmodel.Plan, phaseID string) planmodel.ChangeBoundary {
	if strings.TrimSpace(phaseID) != "" {
		for _, ph := range p.Phases {
			if ph.ID == phaseID && !ph.ChangeBoundary.IsZero() {
				return ph.ChangeBoundary
			}
		}
	}
	return p.ChangeBoundary
}

func baselineCommands(scenario string, anchor planmodel.RegressionAnchor) []string {
	name := strings.TrimSpace(anchor.BaselineName)
	if name == "" || strings.ContainsAny(name, " \t\r\n") {
		return nil
	}
	return []string{
		fmt.Sprintf("git-control-tower baseline snapshot status --scenario %s --name %s --wait --json", scenario, name),
		fmt.Sprintf("git-control-tower baseline diff --scenario %s --name %s --wait", scenario, name),
	}
}

func scenarioFromTarget(target string) string {
	target = strings.TrimPrefix(target, "./")
	const prefix = "scenarios/"
	if !strings.HasPrefix(target, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(target, prefix)
	if i := strings.IndexByte(rest, '/'); i > 0 {
		return rest[:i]
	}
	return rest
}

// splitCommand splits a shell-ish command string into a name + args by
// whitespace. Sufficient for the derived baseline commands (no quoting); the
// LookPath guard in execRunner contains the exec.
func splitCommand(cmd string) (string, []string) {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

func stalenessRank(t planmodel.StalenessTier) int {
	switch t {
	case planmodel.StalenessFresh:
		return 1
	case planmodel.StalenessLightlyStale:
		return 2
	case planmodel.StalenessDefinitelyStale:
		return 3
	default:
		return 0
	}
}

func appendUnique(list []string, v string) []string {
	for _, e := range list {
		if e == v {
			return list
		}
	}
	return append(list, v)
}
