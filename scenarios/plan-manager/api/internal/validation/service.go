package validation

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"plan-manager/internal/clock"
	internalplans "plan-manager/internal/plans"
)

// Service is the validation application surface.
type Service interface {
	ResolveReferences(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	ComputeStaleness(ctx context.Context, planID, phaseID string) (ReferenceReport, error)
	DeriveBaselineScope(ctx context.Context, planID, phaseID string) (BaselineScope, error)
	RunValidation(ctx context.Context, planID, phaseID string) (Result, error)
	VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error)
}

type service struct {
	plans     PlanSource
	resolver  ReferenceResolver
	staleness StalenessComputer
	runner    CommandRunner
	clock     clock.Clock
}

// Deps wires the validation Service. plans is required; resolver/staleness/runner
// are optional (nil => that capability degrades to a marked gap, never a false
// positive).
type Deps struct {
	Plans     PlanSource
	Resolver  ReferenceResolver
	Staleness StalenessComputer
	Runner    CommandRunner
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
	scope, err := s.DeriveBaselineScope(ctx, planID, phaseID)
	if err != nil {
		return Result{}, err
	}
	staleReport, _ := s.ComputeStaleness(ctx, planID, phaseID)
	res := Result{
		PlanID:      planID,
		PhaseID:     phaseID,
		CommandsRun: scope.Commands,
		Staleness:   staleReport.Overall,
		RanAt:       s.now(),
	}
	res.Verdict, res.Detail = s.runCommands(ctx, scope.Commands)
	return res, nil
}

func (s *service) VerifyDefinitionOfDone(ctx context.Context, planID string) (Result, bool, error) {
	p, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Result{}, false, err
	}
	commands := p.RegressionAnchor.Commands
	res := Result{PlanID: planID, CommandsRun: commands, RanAt: s.now()}
	if p.RegressionAnchor.Unavailable || len(commands) == 0 {
		res.Verdict = VerdictUnknown
		res.Detail = "regression anchor unavailable; DoD cannot be verified against an oracle"
		return res, false, nil
	}
	res.Verdict, res.Detail = s.runCommands(ctx, commands)
	return res, res.Verdict == VerdictPass, nil
}

// runCommands runs the derived command set as the diff oracle. PASS iff every
// command exits 0; FAIL on a non-zero exit; UNKNOWN when there is no runner or no
// command to run (degrade honestly — never a fabricated pass).
func (s *service) runCommands(ctx context.Context, commands []string) (Verdict, string) {
	if len(commands) == 0 {
		return VerdictUnknown, "no baseline commands derived"
	}
	if s.runner == nil {
		return VerdictUnknown, "no command runner configured (git-control-tower unavailable)"
	}
	var details []string
	for _, cmd := range commands {
		name, args := splitCommand(cmd)
		if name == "" {
			continue
		}
		out, err := s.runner(ctx, name, args...)
		if err != nil {
			details = append(details, fmt.Sprintf("FAIL %s: %v", cmd, err))
			return VerdictFail, strings.Join(append(details, strings.TrimSpace(string(out))), "\n")
		}
		details = append(details, fmt.Sprintf("ok %s", cmd))
	}
	return VerdictPass, strings.Join(details, "\n")
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
		commands = append(commands, "git diff --stat")
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
