package validation

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"plan-manager/internal/clock"
	internalplans "plan-manager/internal/plans"

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
	}
}

var _ Service = (*service)(nil)

// scopedReferences returns the references in scope: a phase's references when
// phaseID is set, else the plan-level references.
func (s *service) scopedReferences(p internalplans.Plan, phaseID string) ([]internalplans.Reference, error) {
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
func (s *service) resolveAll(ctx context.Context, refs []internalplans.Reference) ([]internalplans.Reference, bool) {
	out := make([]internalplans.Reference, 0, len(refs))
	degraded := false
	for _, ref := range refs {
		if ref.Future {
			ref.Resolution = internalplans.ResolutionFuture
			out = append(out, ref)
			continue
		}
		if s.resolver == nil {
			ref.Resolution = internalplans.ResolutionUnresolved
			ref.Note = "code-facts unavailable"
			degraded = true
			out = append(out, ref)
			continue
		}
		got, err := s.resolver.Resolve(ctx, ref)
		if err != nil {
			ref.Resolution = internalplans.ResolutionUnresolved
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
	// still-present reference is graded fresh vs lightly-stale.
	headSha := ""
	if p, gerr := s.plans.GetPlan(ctx, planID); gerr == nil {
		headSha = p.RegressionAnchor.HeadSha
	}
	overall := internalplans.StalenessFresh
	anyKnown := false
	for i := range report.References {
		ref := report.References[i]
		if ref.Future {
			continue // proposed code is never "stale"
		}
		if s.staleness == nil {
			ref.Staleness = internalplans.StalenessUnknown
			report.Degraded = true
			report.References[i] = ref
			continue
		}
		tier, factor, err := s.staleness.Compute(ctx, ref)
		if err != nil {
			ref.Staleness = internalplans.StalenessUnknown
			report.Degraded = true
			report.References[i] = ref
			continue
		}
		// Refine the existence floor: a still-present reference (FRESH) whose code
		// changed since the anchor is LIGHTLY_STALE ("small diffs in referenced
		// code"). DEFINITELY_STALE (moved/deleted) is never downgraded. Absent a
		// HeadSha or a git runner, the floor's FRESH stands — honest, never guessed.
		if tier == internalplans.StalenessFresh {
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
		overall = internalplans.StalenessUnknown
	}
	report.Overall = overall
	return report, nil
}

// gitChangeTier upgrades a still-present (FRESH) reference to LIGHTLY_STALE when
// its location has changed since the anchor's HeadSha, with a change factor from
// the diff magnitude. Returns refined=false (keep FRESH) when there is no anchor
// sha, no runner, the tool is absent, the ref is not file-backed, or nothing
// changed. Uses `git diff --numstat <sha> -- <target>`: empty output (exit 0)
// means unchanged; non-empty means changed.
func (s *service) gitChangeTier(ctx context.Context, headSha string, ref internalplans.Reference) (internalplans.StalenessTier, float64, bool) {
	headSha = strings.TrimSpace(headSha)
	if headSha == "" || s.runner == nil {
		return "", 0, false
	}
	if ref.Kind != internalplans.ReferenceCode && ref.Kind != internalplans.ReferenceDoc {
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
	return internalplans.StalenessLightlyStale, factor, true
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
	// Persist for the cheap-read context path (status/next). Best-effort: a cache
	// write failure must not fail the live validation the agent asked for.
	if s.results != nil {
		_ = s.results.SaveResult(ctx, res)
	}
	return res, nil
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
		name, args := splitCommand(cmd)
		if name == "" {
			continue
		}
		oracle := isOracleCommand(cmd)
		_, err := s.runner(ctx, name, args...)
		switch {
		case err == nil:
			details = append(details, fmt.Sprintf("ok %s", cmd))
			if oracle {
				oraclePassed++
			}
		case errors.Is(err, ErrToolNotFound):
			details = append(details, fmt.Sprintf("unknown %s: tool not found", cmd))
			if oracle {
				oracleUnknown = true
			}
		default:
			var exitErr CommandExitError
			if errors.As(err, &exitErr) && exitErr.Code == 2 {
				details = append(details, fmt.Sprintf("unknown %s: not comparable (exit 2)", cmd))
				if oracle {
					oracleUnknown = true
				}
				continue
			}
			details = append(details, fmt.Sprintf("FAIL %s: %v", cmd, err))
			if oracle {
				oracleFailed = true
			}
			// An informational command failing does not flip the verdict.
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

func (s *service) now() string {
	return s.clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
}

// deriveScope computes the exact baseline/validation command set across all
// affected locations a plan/phase's references touch. Scenario-scoped code refs
// map to a git-control-tower scenario baseline diff; non-scenario code refs map
// to a repo-level git diff. The plan's own regression-anchor commands are folded
// in. Output is deduped and stably ordered so the command set is deterministic.
func deriveScope(p internalplans.Plan, refs []internalplans.Reference) BaselineScope {
	scenarios := map[string]bool{}
	repoLevel := false
	for _, ref := range refs {
		if ref.Kind != internalplans.ReferenceCode || ref.Future {
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
		commands = append(commands, fmt.Sprintf("git-control-tower baseline diff --scenario %s", name))
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

func stalenessRank(t internalplans.StalenessTier) int {
	switch t {
	case internalplans.StalenessFresh:
		return 1
	case internalplans.StalenessLightlyStale:
		return 2
	case internalplans.StalenessDefinitelyStale:
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
