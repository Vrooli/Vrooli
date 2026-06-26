package validation

import (
	"context"
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
	return deriveScope(p, refs), nil
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
	scope := deriveScope(p, refs)
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
	commandFindings, commandVerdict, commandDetail := s.validatePlanCommands(ctx, p, phaseID)
	res.CommandFindings = commandFindings
	res.Verdict = combineValidationVerdicts(res.Verdict, commandVerdict)
	res.Detail = joinDetails(res.Detail, commandDetail)
	// Persist for the cheap-read context path (status/next). Best-effort: a cache
	// write failure must not fail the live validation the agent asked for.
	if s.results != nil {
		_ = s.results.SaveResult(ctx, res)
	}
	return res, nil
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
		for _, phase := range p.Phases {
			addCommandRefs(&out, "phase."+phase.ID+".intent", phase.Intent)
			for i, item := range phase.RequiredReading {
				addCommandRefs(&out, fmt.Sprintf("phase.%s.required_reading[%d]", phase.ID, i), item)
			}
			for i, item := range phase.Reminders {
				addCommandRefs(&out, fmt.Sprintf("phase.%s.reminders[%d]", phase.ID, i), item)
			}
			addCommandRefs(&out, "phase."+phase.ID+".acceptance", phase.Acceptance)
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
		commands = deriveScope(p, p.References).Commands
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
	return strings.HasPrefix(strings.TrimSpace(cmd), "git-control-tower baseline diff")
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
// affected locations a plan/phase's references touch. Scenario-scoped code refs
// map to a git-control-tower scenario baseline diff only when the plan carries a
// baseline name; the verified GCT CLI requires both --scenario and --name. If no
// baseline name exists, those locations are still returned but no oracle command
// is fabricated. Non-scenario code refs map to a repo-level informational git
// diff. The plan's own regression-anchor commands are folded in. Output is
// deduped and stably ordered so the command set is deterministic.
func deriveScope(p planmodel.Plan, refs []planmodel.Reference) BaselineScope {
	scenarios := map[string]bool{}
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
	names := make([]string, 0, len(scenarios))
	for name := range scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		locations = append(locations, "scenarios/"+name)
		if cmd := baselineDiffCommand(name, p.RegressionAnchor); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	if repoLevel {
		locations = append(locations, "repo")
		// Repo-level changes have no scenario baseline; emit an INFORMATIONAL diff
		// (scoped to the anchor's HeadSha when one was captured) so the agent sees
		// what changed. This is not an oracle — see isOracleCommand — so it never
		// produces a false PASS on its own.
		if sha := strings.TrimSpace(p.RegressionAnchor.HeadSha); sha != "" {
			commands = append(commands, "git diff --stat "+sha)
		} else {
			commands = append(commands, "git diff --stat")
		}
	}
	for _, c := range p.RegressionAnchor.Commands {
		commands = appendUnique(commands, c)
	}
	return BaselineScope{Commands: commands, Locations: locations}
}

func baselineDiffCommand(scenario string, anchor planmodel.RegressionAnchor) string {
	name := strings.TrimSpace(anchor.BaselineName)
	if name == "" || strings.ContainsAny(name, " \t\r\n") {
		return ""
	}
	return fmt.Sprintf("git-control-tower baseline diff --scenario %s --name %s", scenario, name)
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
