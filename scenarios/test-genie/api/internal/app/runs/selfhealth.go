package runs

import (
	"context"
	"path/filepath"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/selfhealth"
	"test-genie/internal/selfhealthsnapshots"

	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// snapshotTimeLayout matches the persisted snapshot encoding so trend
// timestamps round-trip as text in the proto response.
const snapshotTimeLayout = time.RFC3339Nano

// attachTrend fills the ledger's captured_at + trend delta (current ledger vs
// the latest persisted snapshot) and, when requested, the windowed trend
// series. It is a no-op when no snapshot store is wired or none exist yet, so
// the compute-on-read path is unchanged without the sweeper.
func (s *Service) attachTrend(ctx context.Context, sh *runspb.SelfHealth, includeTrend bool, window time.Duration) {
	if s.snapshotReader == nil || sh.GetLedger() == nil {
		return
	}
	latest, ok, err := s.snapshotReader.Latest(ctx)
	if err == nil && ok {
		sh.Ledger.CapturedAt = latest.CapturedAt.UTC().Format(snapshotTimeLayout)
		sh.Ledger.Trend = &runspb.TrendDelta{
			PreviousCapturedAt:   latest.CapturedAt.UTC().Format(snapshotTimeLayout),
			PreviousAvailability: latest.Availability,
			PreviousRunCount:     int32(latest.RunCount),
			AvailabilityDelta:    sh.Ledger.GetAvailability() - latest.Availability,
			RunCountDelta:        sh.Ledger.GetRunCount() - int32(latest.RunCount),
		}
	}

	if !includeTrend {
		return
	}
	q := selfhealthsnapshots.SeriesQuery{Limit: 500}
	if window > 0 {
		q.Since = time.Now().Add(-window)
	}
	series, err := s.snapshotReader.Series(ctx, q)
	if err != nil {
		return
	}
	for _, snap := range series {
		sh.TrendSeries = append(sh.TrendSeries, &runspb.SelfHealthTrendPoint{
			CapturedAt:     snap.CapturedAt.UTC().Format(snapshotTimeLayout),
			Availability:   snap.Availability,
			RunCount:       int32(snap.RunCount),
			HardViolations: int32(snap.HardViolations),
			MetricsAdopted: int32(snap.MetricsAdopted),
		})
	}
}

// GetSelfHealth assembles Test Genie's self-observability snapshot: the phase
// catalog summary, per-provider conformance (probed live + time-boxed unless
// skipped), and the compute-on-read reliability ledger over a recent window.
func (s *Service) GetSelfHealth(ctx context.Context, req *connect.Request[runspb.GetSelfHealthRequest]) (*connect.Response[runspb.GetSelfHealthResponse], error) {
	var window time.Duration
	if days := int(req.Msg.GetWindowDays()); days > 0 {
		window = time.Duration(days) * 24 * time.Hour
	}

	catalog := phases.DefaultCatalog()
	catalogSummary, phaseMeta := buildCatalogSummary(catalog)

	sh := &runspb.SelfHealth{Catalog: catalogSummary}

	if s.ledgerSource != nil {
		ledger, err := selfhealth.NewBuilder(s.ledgerSource, window).Build(ctx, phaseMeta)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		sh.Ledger = ledgerToProto(ledger)
		s.attachTrend(ctx, sh, req.Msg.GetIncludeTrend(), window)
	}

	if req.Msg.GetSkipConformance() {
		sh.ConformanceFreshness = "skipped"
	} else {
		scanner := selfhealth.ConformanceScanner{
			RepoRoot: s.repoRoot(),
			Target:   selfhealth.DefaultScanTarget,
		}
		if s.storedMetrics != nil {
			scanner.StoredMetrics = func(probeCtx context.Context, _ string, phase string) bool {
				return s.storedMetrics(probeCtx, phase)
			}
		}
		report := scanner.Scan(ctx)
		sh.Conformance = conformanceToProto(report)
		sh.ConformanceFreshness = "live"
	}

	return connect.NewResponse(&runspb.GetSelfHealthResponse{SelfHealth: sh}), nil
}

// repoRoot derives the repository root from scenariosRoot (its parent) so the
// conformance scan can load each provider's shipped Test Genie descriptor.
func (s *Service) repoRoot() string {
	return filepath.Dir(s.scenariosRoot)
}

// buildCatalogSummary produces the proto catalog summary and the phase→meta map
// the reliability ledger uses to attribute phases to providers.
func buildCatalogSummary(catalog *phases.Catalog) (*runspb.CatalogSummary, map[string]selfhealth.PhaseMeta) {
	specs := catalog.All()
	summary := &runspb.CatalogSummary{}
	meta := make(map[string]selfhealth.PhaseMeta, len(specs))

	for _, spec := range specs {
		name := spec.Name.String()
		findingSource := findingid.SourceToken(spec.FindingSource)
		delegated := spec.Delegated != nil
		provider := ""
		if delegated {
			provider = spec.Delegated.ProviderScenario
		}

		summary.TotalPhases++
		if delegated {
			summary.DelegatedPhases++
		} else {
			summary.NativePhases++
		}
		summary.Phases = append(summary.Phases, &runspb.CatalogPhase{
			Name:          name,
			Optional:      spec.Optional,
			Source:        spec.Source,
			Delegated:     delegated,
			Provider:      provider,
			FindingSource: findingSource,
		})

		meta[name] = selfhealth.PhaseMeta{
			Provider:      provider,
			FindingSource: findingSource,
			Delegated:     delegated,
		}
	}
	return summary, meta
}

func conformanceToProto(report selfhealth.ConformanceReport) []*runspb.ProviderConformance {
	out := make([]*runspb.ProviderConformance, 0, len(report.Providers))
	for _, pr := range report.Providers {
		out = append(out, &runspb.ProviderConformance{
			Provider:            pr.Provider,
			Phase:               pr.Phase,
			Classification:      string(pr.Classification),
			ReasonCodes:         append([]string(nil), pr.ReasonCodes...),
			Reachable:           pr.Reachable,
			ContractValid:       pr.ContractValid,
			IdentityOk:          pr.IdentityOK,
			SpecValid:           pr.SpecValid,
			MetricsAdopted:      pr.MetricsAdopted,
			MetricsReachable:    pr.MetricsReachable,
			FixContractRequired: pr.FixContractRequired,
			FixContractValid:    pr.FixContractValid,
			ConcurrencyDeclared: pr.ConcurrencyDeclared,
			AdoptionScore:       pr.AdoptionScore,
			Violations:          pr.Violations,
			Autofix:             autofixCoverageToProto(pr.Autofix),
		})
	}
	return out
}

func autofixCoverageToProto(c assessment.AutofixCoverage) *runspb.AutofixCoverage {
	return &runspb.AutofixCoverage{
		Total:               int32(c.Total),
		FixableUniverse:     int32(c.FixableUniverse),
		Implemented:         int32(c.Implemented),
		Pending:             int32(c.Pending),
		Manual:              int32(c.Manual),
		Declared:            int32(c.Declared),
		DeclarationComplete: c.DeclarationComplete,
		ImplementationRate:  c.ImplementationRate(),
	}
}

func ledgerToProto(l *selfhealth.Ledger) *runspb.ReliabilityLedger {
	if l == nil {
		return nil
	}
	out := &runspb.ReliabilityLedger{
		WindowDays:   int32(l.WindowDays),
		RunCount:     int32(l.RunCount),
		Availability: l.Availability,
	}
	for _, o := range l.RunOutcomes {
		out.RunOutcomes = append(out.RunOutcomes, &runspb.RunOutcomeCount{
			Outcome: o.Outcome,
			Count:   int32(o.Count),
		})
	}
	for _, p := range l.Phases {
		out.Phases = append(out.Phases, phaseReliabilityToProto(p))
	}
	for _, pr := range l.Providers {
		out.Providers = append(out.Providers, &runspb.ProviderReliability{
			Provider:          pr.Provider,
			Phases:            pr.Phases,
			TotalObservations: int32(pr.TotalObservations),
			Passed:            int32(pr.Passed),
			Failed:            int32(pr.Failed),
			Skipped:           int32(pr.Skipped),
			Availability:      pr.Availability,
			FailureRate:       pr.FailureRate,
			MetricsAdopted:    int32(pr.MetricsAdopted),
			Duration:          durationToProto(pr.Duration),
		})
	}
	return out
}

func phaseReliabilityToProto(p selfhealth.PhaseReliability) *runspb.PhaseReliability {
	out := &runspb.PhaseReliability{
		Phase:             p.Phase,
		Provider:          p.Provider,
		FindingSource:     p.FindingSource,
		TotalObservations: int32(p.TotalObservations),
		Passed:            int32(p.Passed),
		Failed:            int32(p.Failed),
		Skipped:           int32(p.Skipped),
		Degraded:          int32(p.Degraded),
		Availability:      p.Availability,
		FailureRate:       p.FailureRate,
		MetricsAdopted:    int32(p.MetricsAdopted),
		Duration:          durationToProto(p.Duration),
	}
	for _, lc := range p.SkipReasons {
		out.SkipReasons = append(out.SkipReasons, &runspb.LabeledCount{Label: lc.Label, Count: int32(lc.Count)})
	}
	for _, lc := range p.Classifications {
		out.Classifications = append(out.Classifications, &runspb.LabeledCount{Label: lc.Label, Count: int32(lc.Count)})
	}
	for _, ws := range p.WorstScenarios {
		out.WorstScenarios = append(out.WorstScenarios, &runspb.ScenarioFailureRate{
			Scenario:    ws.Scenario,
			Executed:    int32(ws.Executed),
			Failures:    int32(ws.Failures),
			FailureRate: ws.FailureRate,
		})
	}
	if p.SecurityFriction.FailedAttempts > 0 || p.SecurityFriction.GreenTransitions > 0 {
		out.SecurityFriction = &runspb.SecurityFriction{
			FailedAttempts:     int32(p.SecurityFriction.FailedAttempts),
			GreenTransitions:   int32(p.SecurityFriction.GreenTransitions),
			RecurringFailures:  int32(p.SecurityFriction.RecurringFailures),
			TimeToGreenSamples: int32(p.SecurityFriction.TimeToGreenSamples),
			TimeToGreen:        durationToProto(p.SecurityFriction.TimeToGreen),
		}
	}
	return out
}

func durationToProto(d selfhealth.DurationStats) *runspb.DurationStats {
	return &runspb.DurationStats{
		Samples: int32(d.Samples),
		P50:     int32(d.P50),
		P95:     int32(d.P95),
		Min:     int32(d.Min),
		Max:     int32(d.Max),
		Avg:     int32(d.Avg),
	}
}
