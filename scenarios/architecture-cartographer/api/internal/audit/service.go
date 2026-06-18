package audit

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
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

	var verdicts conflicts.VerdictProvider
	if s.verdicts != nil {
		verdicts = newCachedVerdictProvider(s.verdicts)
	}
	found, dErr := s.conflicts.DetectConflicts(ctx, conflicts.DetectOrchestrationInput{
		Scenario:        scenario,
		Snapshot:        snap,
		DomainMap:       dmap,
		VerdictProvider: verdicts,
		Suppressions:    markers,
	})
	if dErr != nil {
		return s.toolError(rep, "conflict detection failed: "+dErr.Error(), start), nil
	}
	coverage, coverageErr := coverageSummary(ctx, scenario, snap, verdicts)
	if coverageErr != nil {
		return s.toolError(rep, "coverage scoring failed: "+coverageErr.Error(), start), nil
	}
	rep.Coverage = coverage

	filtered := applyFilters(found, in.IncludeTypes, in.ExcludeTypes)
	rep.Findings = filtered
	rep.TotalFindings = len(filtered)
	rep.BySeverity = countBySeverity(filtered)
	rep.ByType = countByType(filtered)
	rep.ByDomain = countByDomain(filtered)
	rep.SuppressedFindings = countSuppressed(filtered)
	rep.Categories = scoreCategories(rep.Coverage, filtered, rep.Domains.Confidence)
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

type cachedVerdictProvider struct {
	upstream conflicts.VerdictProvider
	mu       sync.Mutex
	cache    map[string][]conflicts.Verdict
	content  map[string][]conflicts.Verdict
}

func newCachedVerdictProvider(upstream conflicts.VerdictProvider) *cachedVerdictProvider {
	return &cachedVerdictProvider{
		upstream: upstream,
		cache:    make(map[string][]conflicts.Verdict),
		content:  make(map[string][]conflicts.Verdict),
	}
}

func (p *cachedVerdictProvider) VerdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk) ([]conflicts.Verdict, error) {
	return p.verdictsFor(ctx, scenario, chunks, false)
}

func (p *cachedVerdictProvider) ContentVerdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk) ([]conflicts.Verdict, error) {
	return p.verdictsFor(ctx, scenario, chunks, true)
}

func (p *cachedVerdictProvider) verdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk, contentOnly bool) ([]conflicts.Verdict, error) {
	if p == nil || p.upstream == nil || len(chunks) == 0 {
		return nil, nil
	}
	key := verdictCacheKey(scenario, chunks)
	cache := p.cache
	fetch := p.upstream.VerdictsFor
	if contentOnly {
		cache = p.content
		fetch = p.upstream.ContentVerdictsFor
	}
	p.mu.Lock()
	if got, ok := cache[key]; ok {
		cp := append([]conflicts.Verdict(nil), got...)
		p.mu.Unlock()
		return cp, nil
	}
	p.mu.Unlock()

	got, err := fetch(ctx, scenario, chunks)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	cache[key] = append([]conflicts.Verdict(nil), got...)
	p.mu.Unlock()
	return got, nil
}

func verdictCacheKey(scenario string, chunks []graph.Chunk) string {
	var b strings.Builder
	b.WriteString(scenario)
	for _, c := range chunks {
		b.WriteByte('\x00')
		b.WriteString(c.ID)
		b.WriteByte(':')
		b.WriteString(c.FileID)
	}
	return b.String()
}

func coverageSummary(ctx context.Context, scenario string, snap graph.GraphSnapshot, verdicts conflicts.VerdictProvider) (CoverageSummary, error) {
	chunks := snap.Chunks()
	out := CoverageSummary{TotalFiles: len(chunks)}
	if len(chunks) == 0 {
		return out.withPercents(), nil
	}
	if verdicts == nil {
		out.AllAbstained.Count = len(chunks)
		return out.withPercents(), nil
	}
	scored, err := verdicts.VerdictsFor(ctx, scenario, chunks)
	if err != nil {
		return CoverageSummary{}, err
	}
	for i, v := range scored {
		if i >= len(chunks) {
			break
		}
		switch {
		case v.AllAbstained || (strings.TrimSpace(v.TopDomain) == "" && strings.TrimSpace(v.Tier) == ""):
			out.AllAbstained.Count++
		case v.Tier == "auto_place":
			out.AutoPlace.Count++
		case v.Tier == "suggest":
			out.Suggest.Count++
		default:
			out.Conflict.Count++
		}
	}
	if len(scored) < len(chunks) {
		out.AllAbstained.Count += len(chunks) - len(scored)
	}
	return out.withPercents(), nil
}

func (s CoverageSummary) withPercents() CoverageSummary {
	if s.TotalFiles == 0 {
		return s
	}
	denom := float64(s.TotalFiles)
	s.AutoPlace.Percent = percent(s.AutoPlace.Count, denom)
	s.Suggest.Percent = percent(s.Suggest.Count, denom)
	s.Conflict.Percent = percent(s.Conflict.Count, denom)
	s.AllAbstained.Percent = percent(s.AllAbstained.Count, denom)
	return s
}

func percent(count int, denom float64) float64 {
	return float64(count) * 100 / denom
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

func scoreCategories(coverage CoverageSummary, findings []conflicts.Conflict, authorityConfidence string) []AuditCategory {
	return []AuditCategory{
		{
			Key:      "placement_legibility",
			Label:    "Placement legibility",
			Score:    placementScore(coverage),
			TopItems: topItemsFor(findings, 3, "mislocated_file"),
		},
		{
			Key:      "surface_alignment",
			Label:    "Surface alignment",
			Score:    penaltyScore(findings, "surface_coherence"),
			TopItems: topItemsFor(findings, 3, "surface_coherence"),
		},
		{
			Key:      "boundary_cleanliness",
			Label:    "Boundary cleanliness",
			Score:    penaltyScore(findings, "layering", "cycle", "cross_scenario"),
			TopItems: topItemsFor(findings, 3, "layering", "cycle", "cross_scenario"),
		},
		{
			Key:      "naming_clarity",
			Label:    "Naming clarity",
			Score:    penaltyScore(findings, "naming", "glossary_drift"),
			TopItems: topItemsFor(findings, 3, "naming", "glossary_drift"),
		},
		{
			Key:      "authority",
			Label:    "Authority",
			Score:    authorityScore(authorityConfidence),
			TopItems: topItemsFor(findings, 3, "domain_sparse_warning", "convergence_drift"),
		},
	}
}

func placementScore(coverage CoverageSummary) float64 {
	if coverage.TotalFiles == 0 {
		return 1
	}
	return clamp01((coverage.AutoPlace.Percent + coverage.Suggest.Percent) / 100)
}

func penaltyScore(findings []conflicts.Conflict, types ...string) float64 {
	if len(findings) == 0 {
		return 1
	}
	typeSet := newSet(types)
	penalty := 0.0
	for _, finding := range findings {
		if _, ok := typeSet[finding.Type]; !ok {
			continue
		}
		switch finding.Severity {
		case conflicts.SeverityBlocker:
			penalty += 0.35
		case conflicts.SeverityError:
			penalty += 0.25
		case conflicts.SeverityWarn:
			penalty += 0.12
		case conflicts.SeverityInfo:
			penalty += 0.05
		default:
			penalty += 0.05
		}
	}
	return clamp01(1 - penalty)
}

func authorityScore(confidence string) float64 {
	switch confidence {
	case string(domains.ConfidenceHigh):
		return 1
	case "medium":
		return 0.75
	case string(domains.ConfidenceLow):
		return 0.4
	case string(domains.ConfidenceMissing):
		return 0
	default:
		return 0.5
	}
}

func topItemsFor(findings []conflicts.Conflict, limit int, types ...string) []CategoryTopItem {
	if limit <= 0 {
		return nil
	}
	typeSet := newSet(types)
	matches := make([]conflicts.Conflict, 0, len(findings))
	for _, finding := range findings {
		if _, ok := typeSet[finding.Type]; ok {
			matches = append(matches, finding)
		}
	}
	sortConflicts(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]CategoryTopItem, 0, len(matches))
	for _, finding := range matches {
		out = append(out, CategoryTopItem{
			ID:           finding.ID,
			StableID:     finding.StableID,
			Type:         finding.Type,
			Subtype:      finding.Subtype,
			Severity:     finding.Severity,
			FindingClass: finding.FindingClass,
			Locations:    append([]string(nil), finding.Locations...),
			Headline:     headlineForCategory(finding),
		})
	}
	return out
}

func sortConflicts(findings []conflicts.Conflict) {
	sort.SliceStable(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if left.Type != right.Type {
			return left.Type < right.Type
		}
		if left.Subtype != right.Subtype {
			return left.Subtype < right.Subtype
		}
		return strings.Join(left.Locations, "\x00") < strings.Join(right.Locations, "\x00")
	})
}

func headlineForCategory(c conflicts.Conflict) string {
	if len(c.Locations) == 0 {
		return c.Type
	}
	return c.Type + " @ " + c.Locations[0]
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
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

// decideOutcome returns Findings only when a deterministic conflict reaches
// ERROR/BLOCKER severity. Heuristic findings are reported but never gate.
func decideOutcome(in []conflicts.Conflict, failOn conflicts.Severity) Outcome {
	threshold := severityRank(failOn)
	if threshold < severityRank(conflicts.SeverityError) {
		threshold = severityRank(conflicts.SeverityError)
	}
	for _, c := range in {
		if c.FindingClass == conflicts.FindingClassDeterministic && severityRank(c.Severity) >= threshold {
			return OutcomeFindings
		}
	}
	return OutcomeClean
}
