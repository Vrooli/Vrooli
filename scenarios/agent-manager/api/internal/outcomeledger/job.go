package outcomeledger

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

// Store is what the ledger job reads and writes.
type Store interface {
	// LoadPending returns the runs since `since` changed after their row was
	// written, newest first, with their per-run facts filled in.
	LoadPending(ctx context.Context, since time.Time, limit int) ([]RunFacts, error)
	SaveFacts(ctx context.Context, rows []RunFacts) error
	Facts(ctx context.Context, since time.Time) ([]RunFacts, error)
	// SaveUnits writes the fields the unit pass derives across runs.
	SaveUnits(ctx context.Context, rows []RunFacts) error
	Signals(ctx context.Context, since time.Time) (map[string]Signal, error)
	ObserveModes(ctx context.Context, active []ModePeriod, now time.Time) error
	ModePeriods(ctx context.Context) ([]ModePeriod, error)
	MarkBackfilled(ctx context.Context) error
}

// Reader is what measures read.
type Reader interface {
	FactsInWindow(ctx context.Context, from, to time.Time) ([]RunFacts, error)
	BackfillCount(ctx context.Context, from, to time.Time) (int64, error)
	PendingCount(ctx context.Context, since time.Time) (int64, error)
	ModePeriods(ctx context.Context) ([]ModePeriod, error)
}

// Signal is what classified operator messages say about one run.
type Signal struct {
	Messages    int64
	Corrections []Correction
}

// Lookback bounds how far back the ledger keeps rows fresh and links units.
const Lookback = 90 * 24 * time.Hour

// Job refreshes run_ledger_facts in small passes. It never reads run_events.
type Job struct {
	Store Store
	Log   *slog.Logger
	// Modes reads the active usage modes; tests replace it.
	Modes func(time.Time) []ModePeriod
	// Markers reads the launcher's randomized-session markers; tests replace it.
	Markers   func(since time.Time) []usagemode.LaunchMarker
	Now       func() time.Time
	BatchSize int
	// Fast is the interval while a backlog remains; Slow once caught up.
	Fast, Slow time.Duration
}

func (j *Job) defaults() {
	if j.Now == nil {
		j.Now = time.Now
	}
	if j.Log == nil {
		j.Log = slog.Default()
	}
	if j.BatchSize <= 0 {
		j.BatchSize = 500
	}
	if j.Fast <= 0 {
		j.Fast = 15 * time.Second
	}
	if j.Slow <= 0 {
		j.Slow = 5 * time.Minute
	}
	if j.Modes == nil {
		j.Modes = ActiveModePeriods
	}
	if j.Markers == nil {
		j.Markers = usagemode.ReadLaunches
	}
}

// ActiveModePeriods reads the shared mode file.
func ActiveModePeriods(now time.Time) []ModePeriod {
	modes := usagemode.Read(now)
	out := make([]ModePeriod, 0, len(modes))
	for _, mode := range modes {
		period := ModePeriod{ID: mode.ID, Label: mode.Label, Runner: mode.Scope.Runner, Role: mode.Scope.Role, Profile: mode.Scope.Profile, StartedAt: mode.StartedAt}
		if mode.ExpiresAt != nil {
			period.ExpiresAt = *mode.ExpiresAt
		}
		settingsByRunner := mode.Runners
		if mode.IsLearn() && mode.Learn != nil {
			period.Kind, period.Split = usagemode.KindLearn, mode.Learn.Split()
			if mode.Learn.Sessions != nil && mode.Learn.Sessions.OptIn {
				period.SessionOptIn = true
				period.BlockDays = mode.Learn.Sessions.BlockDays
			}
			settingsByRunner = mode.Learn.Runners
		}
		for _, settings := range settingsByRunner {
			if settings.SessionModel != "" {
				period.Targets = append(period.Targets, settings.SessionModel)
			}
			for _, target := range settings.ModelMap {
				period.Targets = append(period.Targets, target)
			}
		}
		sort.Strings(period.Targets)
		out = append(out, period)
	}
	return out
}

// Start runs passes in the background until ctx ends.
func (j *Job) Start(ctx context.Context) {
	if j == nil || j.Store == nil {
		return
	}
	j.defaults()
	go func() {
		for {
			pending, err := j.Pass(ctx)
			if err != nil && ctx.Err() == nil {
				j.Log.Warn("outcome ledger pass degraded; it retries", "error", err.Error())
			}
			wait := j.Slow
			// Live sessions always leave a few changed runs; only a full batch
			// means a backlog worth the shorter interval.
			if pending >= j.BatchSize || err != nil {
				wait = j.Fast
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}
	}()
}

// Pass refreshes one batch of changed runs, then re-derives the facts that
// depend on several runs: attached classes, work units and outcome estimates.
// It returns how many changed runs the batch held.
func (j *Job) Pass(ctx context.Context) (int, error) {
	j.defaults()
	now := j.Now().UTC()
	since := now.Add(-Lookback)
	if err := j.Store.ObserveModes(ctx, j.Modes(now), now); err != nil {
		return 0, err
	}
	pending, err := j.Store.LoadPending(ctx, since, j.BatchSize)
	if err != nil {
		return 0, err
	}
	if len(pending) > 0 {
		if err := j.Store.SaveFacts(ctx, pending); err != nil {
			return len(pending), err
		}
	}
	rows, err := j.Store.Facts(ctx, since)
	if err != nil {
		return len(pending), err
	}
	periods, err := j.Store.ModePeriods(ctx)
	if err != nil {
		return len(pending), err
	}
	signals, err := j.Store.Signals(ctx, since)
	if err != nil {
		return len(pending), err
	}
	if changed := DeriveWithMarkers(rows, periods, signals, j.Markers(since)); len(changed) > 0 {
		if err := j.Store.SaveUnits(ctx, changed); err != nil {
			return len(pending), err
		}
	}
	if len(pending) < j.BatchSize {
		if err := j.Store.MarkBackfilled(ctx); err != nil {
			return len(pending), err
		}
	}
	return len(pending), nil
}

// Derive computes the cross-run fields of rows and returns the rows that changed.
func Derive(rows []RunFacts, periods []ModePeriod, signals map[string]Signal) []RunFacts {
	return DeriveWithMarkers(rows, periods, signals, nil)
}

// DeriveWithMarkers is Derive with the launcher's session markers, which settle
// the arm and exception of operator sessions in a Learn comparison.
func DeriveWithMarkers(rows []RunFacts, periods []ModePeriod, signals map[string]Signal, markers []usagemode.LaunchMarker) []RunFacts {
	byID := make(map[string]*RunFacts, len(rows))
	var attached, imported, work []RunFacts
	for i := range rows {
		byID[rows[i].RunID] = &rows[i]
	}
	before := make(map[string]RunFacts, len(rows))
	for i := range rows {
		run := &rows[i]
		before[run.RunID] = *run
		relation := ResolveMode(*run, periods)
		run.ModeID, run.ModeRelation, run.ModeBasis, run.ModeConfidence = relation.ID, relation.Relation, relation.Basis, relation.Confidence
		run.OperatorMessages = signals[run.RunID].Messages
		arm := ResolveArm(*run, periods, markers)
		run.Arm, run.ArmMode, run.ArmBasis, run.ArmException = arm.Arm, arm.ModeID, arm.Basis, arm.Exception
		switch {
		case run.Path == PathAttached:
			attached = append(attached, *run)
		case run.HasRequests():
			work = append(work, *run)
		}
		if strings.HasPrefix(run.Path, PathImported) {
			imported = append(imported, *run)
		}
	}
	for id, classified := range ClassifyAttached(attached, imported) {
		run := byID[id]
		run.AttachedClass, run.LinkedRunID, run.LinkConfidence = classified.AttachedClass, classified.LinkedRunID, classified.LinkConfidence
	}
	assignments := LinkWorkUnits(work)
	members := map[string][]RunFacts{}
	corrections := map[string][]Correction{}
	for _, run := range work {
		assignment := assignments[run.RunID]
		current := byID[run.RunID]
		current.WorkUnitID, current.LinkKind, current.LinkConfidence = assignment.WorkUnitID, assignment.LinkKind, assignment.LinkConfidence
		members[assignment.WorkUnitID] = append(members[assignment.WorkUnitID], *current)
		corrections[assignment.WorkUnitID] = append(corrections[assignment.WorkUnitID], signals[run.RunID].Corrections...)
	}
	for unit, list := range members {
		estimate := EstimateUnit(list, corrections[unit])
		for _, member := range list {
			current := byID[member.RunID]
			current.UnitOutcome, current.UnitSource = estimate.Outcome, estimate.Source
		}
	}
	var changed []RunFacts
	for i := range rows {
		run := rows[i]
		previous := before[run.RunID]
		if run.ModeID != previous.ModeID || run.ModeRelation != previous.ModeRelation || run.ModeBasis != previous.ModeBasis || run.OperatorMessages != previous.OperatorMessages ||
			run.Arm != previous.Arm || run.ArmMode != previous.ArmMode || run.ArmBasis != previous.ArmBasis || run.ArmException != previous.ArmException ||
			run.AttachedClass != previous.AttachedClass || run.LinkedRunID != previous.LinkedRunID || run.WorkUnitID != previous.WorkUnitID || run.LinkKind != previous.LinkKind ||
			run.LinkConfidence != previous.LinkConfidence || run.UnitOutcome != previous.UnitOutcome || run.UnitSource != previous.UnitSource {
			changed = append(changed, run)
		}
	}
	return changed
}
