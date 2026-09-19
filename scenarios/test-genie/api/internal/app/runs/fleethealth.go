package runs

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/selfhealth"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// GetFleetHealth aggregates stored runs across the whole fleet into a fleet-wide
// health snapshot (compute-on-read; it never launches a fleet run). It is the
// read side of the fleet backbone — the default-OFF background scheduler keeps
// the stored runs fresh, and this verb turns them into importance/staleness-aware
// fleet insight with every datum as-of stamped.
func (s *Service) GetFleetHealth(ctx context.Context, req *connect.Request[runspb.GetFleetHealthRequest]) (*connect.Response[runspb.GetFleetHealthResponse], error) {
	if s.fleetSource == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("fleet ledger source is not configured"))
	}

	var window time.Duration
	if days := int(req.Msg.GetWindowDays()); days > 0 {
		window = time.Duration(days) * 24 * time.Hour
	}

	var roster []string
	if req.Msg.GetIncludeRoster() && s.fleetRoster != nil {
		names, err := s.fleetRoster(ctx)
		if err != nil {
			// A roster failure must not sink the whole rollup — degrade to "no
			// roster" (never-tested stays empty, an honest unknown) rather than
			// erroring the call.
			roster = nil
		} else {
			roster = names
		}
	}

	ledger, err := selfhealth.NewFleetBuilder(s.fleetSource, window).Build(ctx, roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&runspb.GetFleetHealthResponse{
		FleetHealth: fleetLedgerToProto(ledger),
	}), nil
}

func fleetLedgerToProto(l *selfhealth.FleetLedger) *runspb.FleetHealth {
	out := &runspb.FleetHealth{
		WindowDays:              int32(l.WindowDays),
		CapturedAt:              l.CapturedAt.UTC().Format(snapshotTimeLayout),
		ScenariosTested:         int32(l.ScenariosTested),
		ScenariosTotal:          int32(l.ScenariosTotal),
		TotalRuns:               int32(l.TotalRuns),
		TotalIssues:             int32(l.FailedPhaseObservations),
		FailedPhaseObservations: int32(l.FailedPhaseObservations),
		NeverTestedInWindow:     l.NeverTestedInWindow,
	}
	for _, bucket := range l.FailureClassifications {
		out.FailureClassifications = append(out.FailureClassifications, &runspb.FailureClassificationCount{
			Classification: bucket.Label,
			Count:          int32(bucket.Count),
		})
	}
	out.FindingQuality = &runspb.FleetFindingQuality{
		Blockers: int32(l.FindingQuality.Blockers),
		Errors:   int32(l.FindingQuality.Errors),
		Warnings: int32(l.FindingQuality.Warnings),
		Infos:    int32(l.FindingQuality.Infos),
		Total:    int32(l.FindingQuality.Total),
	}
	for _, sc := range l.Scenarios {
		fs := &runspb.FleetScenarioHealth{
			Target:       sc.Scenario,
			Runs:         int32(sc.Runs),
			PassedRuns:   int32(sc.PassedRuns),
			FailedRuns:   int32(sc.FailedRuns),
			Availability: sc.Availability,
			FailureRate:  sc.FailureRate,
			Issues:       int32(sc.Issues),
			LastOutcome:  sc.LastOutcome,
			AgeDays:      sc.AgeDays,
		}
		if !sc.LastRunAt.IsZero() {
			fs.LastRunAt = sc.LastRunAt.UTC().Format(snapshotTimeLayout)
		}
		out.Scenarios = append(out.Scenarios, fs)
	}
	for _, src := range l.TopFindingSources {
		out.TopFindingSources = append(out.TopFindingSources, &runspb.FleetFindingSource{
			Source: src.Source,
			Issues: int32(src.Issues),
		})
	}
	for _, alert := range l.Alerts {
		out.Alerts = append(out.Alerts, &runspb.FleetAlert{
			Code: alert.Code, Severity: alert.Severity, Target: alert.Scenario,
			Source: alert.Source, Message: alert.Message, EvidenceAgeDays: alert.EvidenceAgeDays,
			Owner: alert.Owner, NextAction: alert.NextAction, RollbackPath: alert.RollbackPath,
		})
	}
	return out
}
