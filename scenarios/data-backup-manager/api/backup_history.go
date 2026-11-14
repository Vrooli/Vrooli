package main

import (
	"context"
	"errors"

	destinationsH "data-backup-manager/handlers/destinations"
	plansint "data-backup-manager/internal/plans"
	runsint "data-backup-manager/internal/runs"
	targetsint "data-backup-manager/internal/targets"
)

// backupHistory adapts the plans, runs and targets domains to the backup-failure
// monitor's BackupHistory seam. It reads; its one action is asking for a run.
type backupHistory struct {
	plans   plansint.Service
	runs    runsint.Service
	repo    runsint.Repository
	targets targetsint.Service
}

var _ destinationsH.BackupHistory = backupHistory{}

func (a backupHistory) Plans(ctx context.Context) ([]destinationsH.PlanRecord, error) {
	plans, err := a.plans.List(ctx, 500)
	if err != nil {
		return nil, err
	}
	out := make([]destinationsH.PlanRecord, 0, len(plans))
	for _, p := range plans {
		out = append(out, destinationsH.PlanRecord{
			ID: p.ID, Name: p.Name, Enabled: p.Enabled, Schedule: p.Schedule,
			TargetIDs: p.TargetIDs, DestinationIDs: p.DestinationIDs, CreatedAt: p.CreatedAt,
		})
	}
	return out, nil
}

func (a backupHistory) LatestRun(ctx context.Context, planID string) (destinationsH.RunRecord, bool, error) {
	runs, err := a.runs.ListRuns(ctx, planID, 1)
	if err != nil || len(runs) == 0 {
		return destinationsH.RunRecord{}, false, err
	}
	r := runs[0]
	rec := destinationsH.RunRecord{ID: r.ID, Status: string(r.Status), StartedAt: r.StartedAt, FinishedAt: r.FinishedAt}
	switch r.Status {
	case runsint.RunCompleted, runsint.RunPartialFailed, runsint.RunFailed:
		rec.Terminal = true
	}
	for _, cause := range r.Preflight {
		if cause.DestinationID != "" {
			rec.Failures = append(rec.Failures, destinationsH.DestinationFailure{DestinationID: cause.DestinationID, Code: string(cause.Code)})
		}
	}
	for _, o := range r.Outcomes {
		if o.FailureCode != "" && o.DestinationID != "" && o.Status != runsint.OutcomeSucceeded {
			rec.Failures = append(rec.Failures, destinationsH.DestinationFailure{DestinationID: o.DestinationID, Code: string(o.FailureCode)})
		}
	}
	return rec, true, nil
}

func (a backupHistory) PlanSuccess(ctx context.Context, planID string) (destinationsH.PlanSuccess, error) {
	reader, ok := a.repo.(runsint.PlanSuccessReader)
	if !ok {
		return destinationsH.PlanSuccess{}, errors.New("run history cannot report plan success")
	}
	s, err := reader.PlanSuccess(ctx, planID)
	return destinationsH.PlanSuccess{At: s.At, ByTarget: s.ByTarget}, err
}

func (a backupHistory) TargetLabels(ctx context.Context, ids []string) (map[string]destinationsH.TargetLabel, error) {
	out := make(map[string]destinationsH.TargetLabel, len(ids))
	for _, id := range ids {
		t, err := a.targets.Get(ctx, id)
		if err != nil {
			var missing targetsint.ErrTargetNotFound
			if errors.As(err, &missing) {
				continue
			}
			return nil, err
		}
		out[id] = destinationsH.TargetLabel{Owner: t.Owner, Name: t.Name}
	}
	return out, nil
}

func (a backupHistory) ActiveRun(ctx context.Context, destinationID string) (bool, error) {
	active, err := a.repo.ListNonTerminalRuns(ctx)
	if err != nil || len(active) == 0 {
		return false, err
	}
	plans, err := a.plans.List(ctx, 500)
	if err != nil {
		return false, err
	}
	writes := map[string]bool{}
	for _, p := range plans {
		for _, d := range p.DestinationIDs {
			if d == destinationID {
				writes[p.ID] = true
			}
		}
	}
	for _, r := range active {
		if writes[r.PlanID] {
			return true, nil
		}
	}
	return false, nil
}

func (a backupHistory) TriggerRun(ctx context.Context, planID string) error {
	_, err := a.runs.TriggerRun(ctx, planID, runsint.TriggerManual)
	var active runsint.ErrRunAlreadyActive
	if errors.As(err, &active) {
		return nil
	}
	return err
}
