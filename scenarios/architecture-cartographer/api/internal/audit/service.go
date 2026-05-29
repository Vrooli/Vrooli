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
	RunAll(ctx context.Context, in RunAllInput) (SweepReport, error)
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
	scenarios    ScenarioLister
	clock        clock.Clock
}

// NewService constructs the audit orchestrator. verdicts may be nil
// (the mislocated_file detector skips when no provider is wired —
// see conflicts.DetectInput.VerdictProvider). scenarios may be nil
// when the caller never invokes RunAll.
func NewService(g graph.Service, d domains.Service, c conflicts.Service, verdicts conflicts.VerdictProvider, sup SuppressionLoader, scenarios ScenarioLister, clk clock.Clock) Service {
	return &service{graph: g, domains: d, conflicts: c, verdicts: verdicts, suppressions: sup, scenarios: scenarios, clock: clk}
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

	snap, freshness, err := s.freshSnapshot(ctx, scenario, in.SkipTS)
	if err != nil {
		return s.toolError(rep, "graph extract failed: "+err.Error(), start), nil
	}
	rep.Graph = graphSummary(snap)
	rep.SnapshotFreshness = freshness

	dmap, dErr := s.domains.GetDomainMap(ctx, scenario)
	missingAuthority := false
	if dErr != nil {
		// A scenario without an authoritative map is not a tool error;
		// detectors that don't consult the map still produce findings.
		// We surface authority status through Domains.Confidence so the
		// audit gate can render a remediation-led message.
		var (
			noAuthority domains.ErrNoAuthority
			notFound    domains.ErrScenarioNotFound
		)
		switch {
		case errors.As(dErr, &noAuthority):
			missingAuthority = true
		case errors.As(dErr, &notFound):
			// keep dmap zero; detectors still run on the graph
		default:
			return s.toolError(rep, "domain derivation failed: "+dErr.Error(), start), nil
		}
	}
	rep.Domains = derivedSummary(dmap)
	if missingAuthority {
		// ErrNoAuthority produces a zero-valued DerivedDomainMap; mark
		// the summary explicitly so decideOutcomeWithAuthority can gate
		// on it without re-running the ladder.
		rep.Domains.Confidence = string(domains.ConfidenceMissing)
	}

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
	rep.ByDomain = countByDomain(filtered)
	rep.SuppressedFindings = countSuppressed(filtered)
	rep.Outcome, rep.OutcomeReason = decideOutcomeWithAuthority(scenario, filtered, failOn, rep.Domains.Confidence, in.AllowLowAuthority)
	if rep.Outcome == OutcomeClean && len(snap.SkippedAdapters) > 0 {
		// Graph extract degraded silently (--skip-ts or workspace_unsupported
		// from an adapter). Mark partial so the operator sees the gap rather
		// than mistaking a half-analysis for a clean run.
		rep.Outcome = OutcomePartial
		rep.OutcomeReason = "skipped adapters: " + strings.Join(snap.SkippedAdapters, ", ")
	}
	rep.Duration = s.clock.Now().Sub(start)
	return rep, nil
}

// decideOutcomeWithAuthority extends decideOutcome with the authority-
// confidence axis. Missing or low confidence (no DOMAINS.md or only
// advisory rungs supplied an authority) is a first-class failure axis:
// detectors run without a curated set, so absence-of-findings is
// meaningless. The caller opts back in to the lax behavior with
// AllowLowAuthority — but the remediation message names the fix
// (writing DOMAINS.md) before the bypass.
func decideOutcomeWithAuthority(scenario string, in []conflicts.Conflict, failOn conflicts.Severity, confidence string, allowLow bool) (Outcome, string) {
	if o := decideOutcome(in, failOn); o == OutcomeFindings {
		return o, ""
	}
	if allowLow {
		return OutcomeClean, ""
	}
	switch confidence {
	case string(domains.ConfidenceMissing):
		return OutcomeFindings, missingAuthorityMessage(scenario)
	case string(domains.ConfidenceLow):
		return OutcomeFindings, lowAuthorityMessage(scenario)
	}
	return OutcomeClean, ""
}

// missingAuthorityMessage is rendered when no ladder rung declared any
// domain. Leads with the fix; names the bypass last on purpose so agents
// reach for the fix first.
func missingAuthorityMessage(scenario string) string {
	return "no domain authority for scenario " + scenario +
		" — write scenarios/" + scenario + "/docs/concepts/DOMAINS.md " +
		"(see docs/concepts/DOMAINS.template.md), or pass " +
		"--allow-low-authority for advisory mode"
}

// lowAuthorityMessage is rendered when the ladder fell through to a
// derived rung (folder/cli) instead of a curated DOMAINS.md.
func lowAuthorityMessage(scenario string) string {
	return "scenario " + scenario + " has only derived domain authority " +
		"(no docs/concepts/DOMAINS.md) — promote the inferred domains to " +
		"a curated DOMAINS.md, or pass --allow-low-authority for advisory mode"
}

// freshSnapshot guarantees the audit operates on a snapshot whose
// content-hash matches the current source tree. It always invokes
// ExtractGraph (which is hash-aware: the graph service computes the
// current source-tree hash and reuses a persisted snapshot whose
// content_hash matches). The cacheHit signal returned by ExtractGraph
// distinguishes a reused snapshot from a freshly normalized one.
//
// freshness values:
//   - CACHED:      ExtractGraph found a snapshot whose hash matches.
//   - RE_EXTRACTED: there was a prior persisted snapshot whose hash
//     differed from the current source tree.
//   - FRESH:        no prior persisted snapshot existed.
func (s *service) freshSnapshot(ctx context.Context, scenario string, skipTS bool) (graph.GraphSnapshot, SnapshotFreshness, error) {
	page, lsErr := s.graph.ListSnapshots(ctx, graph.ListSnapshotsFilter{Scenario: scenario, PageSize: 1})
	priorExists := lsErr == nil && len(page.Snapshots) > 0
	snap, cacheHit, err := s.graph.ExtractGraph(ctx, graph.ExtractGraphInput{Scenario: scenario, SkipTS: skipTS})
	if err != nil {
		return graph.GraphSnapshot{}, SnapshotFreshnessUnspecified, err
	}
	switch {
	case cacheHit:
		return snap, SnapshotFreshnessCached, nil
	case priorExists:
		return snap, SnapshotFreshnessReExtracted, nil
	default:
		return snap, SnapshotFreshnessFresh, nil
	}
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

// countByDomain buckets findings by every domain they're tagged with —
// a finding tagged with two domains contributes to both buckets.
// Findings with no Domains contribute to a "—" bucket so the operator
// still sees how many domain-less findings exist.
func countByDomain(in []conflicts.Conflict) map[string]int {
	out := make(map[string]int)
	for _, c := range in {
		if len(c.Domains) == 0 {
			out["—"]++
			continue
		}
		for _, d := range c.Domains {
			out[d]++
		}
	}
	return out
}

// countSuppressed counts findings sanctioned by an active in-repo
// `// arch:allow` marker — reported but not counted toward outcome.
func countSuppressed(in []conflicts.Conflict) int {
	n := 0
	for _, c := range in {
		if c.Suppressed {
			n++
		}
	}
	return n
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
