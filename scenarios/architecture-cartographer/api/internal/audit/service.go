package audit

import (
	"context"
	"errors"
	"strings"
	"time"

	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/suppressions"
)

// Service is the application-layer surface for the audit domain.
type Service interface {
	Run(ctx context.Context, in RunInput) (Report, error)
}

// SuppressionLoader resolves the set of in-repo `// arch:allow` markers
// for a scenario. Matches the production suppressions.Provider seam.
type SuppressionLoader interface {
	Active(ctx context.Context, scenario string) ([]suppressions.Marker, error)
}

type service struct {
	graph        graph.Service
	domains      domains.Service
	conflicts    conflicts.Service
	verdicts     conflicts.VerdictProvider
	suppressions SuppressionLoader
	clock        clock.Clock
}

// NewService constructs the audit orchestrator. verdicts may be nil
// (the mislocated_file detector skips when no provider is wired —
// see conflicts.DetectInput.VerdictProvider).
func NewService(g graph.Service, d domains.Service, c conflicts.Service, verdicts conflicts.VerdictProvider, sup SuppressionLoader, clk clock.Clock) Service {
	return &service{graph: g, domains: d, conflicts: c, verdicts: verdicts, suppressions: sup, clock: clk}
}

func (s *service) Run(ctx context.Context, in RunInput) (Report, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return Report{}, ErrInvalidRunRequest{Field: "scenario", Reason: "required"}
	}
	failOn := in.FailOn
	if failOn == conflicts.SeverityUnspecified {
		failOn = conflicts.SeverityWarn
	}
	start := s.clock.Now()
	rep := Report{Scenario: scenario}

	snap, err := s.latestSnapshot(ctx, scenario)
	if err != nil {
		return s.toolError(rep, "graph extract failed: "+err.Error(), start), nil
	}
	rep.Graph = graphSummary(snap)

	dmap, dErr := s.domains.GetDomainMap(ctx, scenario)
	if dErr != nil {
		// A scenario without an authoritative map is not a tool error;
		// detectors that don't consult the map still produce findings.
		// We surface the authority_fallback info conflict via the
		// detector chain rather than failing here.
		var (
			noAuthority domains.ErrNoAuthority
			notFound    domains.ErrScenarioNotFound
		)
		if !errors.As(dErr, &noAuthority) && !errors.As(dErr, &notFound) {
			return s.toolError(rep, "domain derivation failed: "+dErr.Error(), start), nil
		}
	}
	rep.Domains = derivedSummary(dmap)

	var markers []suppressions.Marker
	if s.suppressions != nil {
		markers, _ = s.suppressions.Active(ctx, scenario)
	}

	found, dErr := s.conflicts.DetectConflicts(ctx, conflicts.DetectOrchestrationInput{
		Scenario:        scenario,
		Snapshot:        snap,
		DomainMap:       dmap,
		VerdictProvider: s.verdicts,
		Suppressions:    markers,
	})
	if dErr != nil {
		return s.toolError(rep, "conflict detection failed: "+dErr.Error(), start), nil
	}

	filtered := applyFilters(found, in.IncludeTypes, in.ExcludeTypes)
	rep.Findings = filtered
	rep.TotalFindings = len(filtered)
	rep.BySeverity = countBySeverity(filtered)
	rep.ByType = countByType(filtered)
	rep.Outcome = decideOutcome(filtered, failOn)
	rep.Duration = s.clock.Now().Sub(start)
	return rep, nil
}

// latestSnapshot returns the most recent snapshot for the scenario,
// triggering an extract if none exists yet.
func (s *service) latestSnapshot(ctx context.Context, scenario string) (graph.GraphSnapshot, error) {
	page, err := s.graph.ListSnapshots(ctx, graph.ListSnapshotsFilter{Scenario: scenario, PageSize: 1})
	if err != nil {
		return graph.GraphSnapshot{}, err
	}
	if len(page.Snapshots) > 0 {
		return page.Snapshots[0], nil
	}
	snap, _, err := s.graph.ExtractGraph(ctx, graph.ExtractGraphInput{Scenario: scenario})
	if err != nil {
		return graph.GraphSnapshot{}, err
	}
	return snap, nil
}

func (s *service) toolError(rep Report, msg string, start time.Time) Report {
	rep.Outcome = OutcomeToolError
	rep.Error = msg
	rep.Duration = s.clock.Now().Sub(start)
	return rep
}

// applyFilters returns the conflict subset matching include_types
// (whitelist) and not in exclude_types (blacklist).
func applyFilters(in []conflicts.Conflict, include, exclude []string) []conflicts.Conflict {
	includeSet := newSet(include)
	excludeSet := newSet(exclude)
	out := make([]conflicts.Conflict, 0, len(in))
	for _, c := range in {
		if len(includeSet) > 0 {
			if _, ok := includeSet[c.Type]; !ok {
				continue
			}
		}
		if _, ok := excludeSet[c.Type]; ok {
			continue
		}
		out = append(out, c)
	}
	return out
}

func newSet(in []string) map[string]struct{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

func countBySeverity(in []conflicts.Conflict) map[string]int {
	out := make(map[string]int)
	for _, c := range in {
		if c.Severity == "" {
			continue
		}
		out[string(c.Severity)]++
	}
	return out
}

func countByType(in []conflicts.Conflict) map[string]int {
	out := make(map[string]int)
	for _, c := range in {
		if c.Type == "" {
			continue
		}
		out[c.Type]++
	}
	return out
}

// severityRank orders severities; higher rank = more blocking.
func severityRank(s conflicts.Severity) int {
	switch s {
	case conflicts.SeverityBlocker:
		return 4
	case conflicts.SeverityError:
		return 3
	case conflicts.SeverityWarn:
		return 2
	case conflicts.SeverityInfo:
		return 1
	default:
		return 0
	}
}

// decideOutcome returns Findings when ANY conflict's severity is ≥
// failOn; otherwise Clean (even if lower-severity advisory findings
// exist — they are reported but do not flip the outcome).
func decideOutcome(in []conflicts.Conflict, failOn conflicts.Severity) Outcome {
	threshold := severityRank(failOn)
	for _, c := range in {
		if severityRank(c.Severity) >= threshold {
			return OutcomeFindings
		}
	}
	return OutcomeClean
}
