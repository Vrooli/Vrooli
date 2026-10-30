package usageattribution

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"
	"agent-manager/internal/usageforecast"
)

// Service attributes the provider windows read from quota observations to
// the usage in Store.
type Service struct {
	Store    Store
	Readings pricing.QuotaReadingSource
	Pricer   *PriceBook
	// Statuses and Bugs are set where friction problems are acted on.
	Statuses StatusStore
	Bugs     BugFiler
}

// Attribution is the current windows' results and the runs they name.
type Attribution struct {
	Results []Result
	Runs    map[string]Run
	Spans   int
}

// Current attributes every current window of provider (all when empty) after
// since (each window's start when zero).
func (s Service) Current(ctx context.Context, provider string, since, now time.Time) (Attribution, error) {
	spans, err := s.Readings.QuotaReadingSpans(ctx, provider, now.Add(-usageforecast.CurrentLookback))
	if err != nil {
		return Attribution{}, err
	}
	var windows []Window
	for _, forecast := range usageforecast.Current(spans, now) {
		// A window whose reset has passed is history, not a current window.
		if forecast.Window != nil && forecast.Window.HasPercent && (forecast.Window.ResetAt.IsZero() || forecast.Window.ResetAt.After(now)) {
			windows = append(windows, FromForecast(forecast.Window))
		}
	}
	return s.attribute(ctx, windows, since, now, len(spans))
}

func (s Service) attribute(ctx context.Context, windows []Window, since, now time.Time, spans int) (Attribution, error) {
	out := Attribution{Spans: spans}
	if len(windows) == 0 {
		out.Runs = map[string]Run{}
		return out, nil
	}
	from := now
	for _, w := range windows {
		start := w.Start
		if since.After(start) {
			start = since
		}
		if start.Before(from) {
			from = start
		}
	}
	turns, err := s.Store.UsageTurns(ctx, from, now)
	if err != nil {
		return out, err
	}
	ids := map[string]bool{}
	prices := s.Pricer.prices(ctx, turns, nil)
	for _, w := range windows {
		start := w.Start
		if since.After(start) {
			start = since
		}
		result := Attribute(w, start, turns, prices)
		for _, turn := range result.Turns {
			ids[turn.RunID] = true
		}
		out.Results = append(out.Results, result)
	}
	out.Runs, err = s.Store.UsageRuns(ctx, keys(ids))
	return out, err
}

// StatusStore keeps what the operator did with friction problems.
type StatusStore interface {
	FrictionStatuses(ctx context.Context) (map[string]ProblemStatus, error)
	SetFrictionStatus(ctx context.Context, status ProblemStatus) error
}

// BugFiler files a friction problem with its evidence in the scenario-qa bug
// inbox and returns the bug id.
type BugFiler interface {
	FileFrictionBug(ctx context.Context, problem Problem) (string, error)
}

// inputs gathers what the rules read: the current windows' attribution, the
// friction episodes since the earliest priced window, and the run outcomes.
func (s Service) inputs(ctx context.Context, provider string, now time.Time, only WindowKey) (Inputs, Attribution, error) {
	attribution, err := s.Current(ctx, provider, time.Time{}, now)
	if err != nil || len(attribution.Results) == 0 {
		return Inputs{}, attribution, err
	}
	from := now
	for _, result := range selectResults(attribution.Results, only) {
		if result.From.Before(from) {
			from = result.From
		}
	}
	episodes, err := s.Store.UsageEpisodes(ctx, from, now)
	if err != nil {
		return Inputs{}, attribution, err
	}
	outcomes, err := s.Store.UsageOutcomes(ctx, now.Add(-Lookback), now)
	if err != nil {
		return Inputs{}, attribution, err
	}
	// Runs named only by episodes still need their runner for pricing.
	missing := map[string]bool{}
	for _, episode := range episodes {
		if _, ok := attribution.Runs[episode.RunID]; !ok {
			missing[episode.RunID] = true
		}
	}
	if len(missing) > 0 {
		extra, err := s.Store.UsageRuns(ctx, keys(missing))
		if err != nil {
			return Inputs{}, attribution, err
		}
		for id, run := range extra {
			attribution.Runs[id] = run
		}
	}
	in := Inputs{Results: attribution.Results, Runs: attribution.Runs, Episodes: episodes, Outcomes: outcomes, Prices: s.Pricer.prices(ctx, nil, outcomes), Only: only, Now: now}
	if s.Statuses != nil {
		statuses, err := s.Statuses.FrictionStatuses(ctx)
		if err != nil {
			return Inputs{}, attribution, err
		}
		in.Dismissed = map[string]bool{}
		for fingerprint, status := range statuses {
			in.Dismissed[fingerprint] = status.Status == StatusDismissed
		}
	}
	return in, attribution, nil
}

// Recommendations evaluates the rules over the current windows: each
// provider's longest window, or only the window named by only.
func (s Service) Recommendations(ctx context.Context, provider string, only WindowKey, now time.Time) (Report, Attribution, error) {
	in, attribution, err := s.inputs(ctx, provider, now, only)
	if err != nil || len(attribution.Results) == 0 {
		return Report{Rules: emptyRules()}, attribution, err
	}
	return Recommend(in), attribution, nil
}

// TopPlan is the largest plan of levers for one window, evaluated for that
// window alone: the action an alert for the window names.
func (s Service) TopPlan(ctx context.Context, key WindowKey, now time.Time) (Plan, bool, error) {
	report, _, err := s.Recommendations(ctx, key.Provider, key, now)
	if err != nil {
		return Plan{}, false, err
	}
	plan, ok := report.Top(key)
	return plan, ok, nil
}

// FrictionReport is the friction problems with how much of the window's runs
// the current episode classifier has analysed: problems count only analysed
// runs, so while history is re-projected their totals are partial.
type FrictionReport struct {
	Problems []Problem
	// RunsAnalysed of RunsTotal runs that used quota in the windows have
	// episodes from the current classifier.
	RunsAnalysed, RunsTotal int
	Attribution             Attribution
}

// Friction lists the recurring friction problems of the current windows with
// their durable status, largest saving first.
func (s Service) Friction(ctx context.Context, provider string, now time.Time) (FrictionReport, error) {
	in, attribution, err := s.inputs(ctx, provider, now, WindowKey{})
	report := FrictionReport{Attribution: attribution}
	if err != nil || len(attribution.Results) == 0 {
		return report, err
	}
	primary, _ := prepare(in)
	report.Problems = frictionProblems(in, primary)
	runs := map[string]bool{}
	for _, result := range primary {
		for _, turn := range result.Turns {
			runs[turn.RunID] = true
		}
	}
	report.RunsTotal = len(runs)
	if report.RunsAnalysed, err = s.Store.UsageEpisodeCoverage(ctx, keys(runs)); err != nil {
		return report, err
	}
	if s.Statuses == nil {
		return report, nil
	}
	statuses, err := s.Statuses.FrictionStatuses(ctx)
	if err != nil {
		return report, err
	}
	for i, problem := range report.Problems {
		if status, ok := statuses[problem.Fingerprint]; ok {
			report.Problems[i].Status, report.Problems[i].BugID, report.Problems[i].StatusSetAt = status.Status, status.BugID, status.At
		}
	}
	return report, nil
}

// problem finds a friction problem by fingerprint among the current windows'.
func (s Service) problem(ctx context.Context, fingerprint string, now time.Time) (Problem, error) {
	report, err := s.Friction(ctx, "", now)
	if err != nil {
		return Problem{}, err
	}
	for _, problem := range report.Problems {
		if problem.Fingerprint == fingerprint {
			return problem, nil
		}
	}
	return Problem{}, fmt.Errorf("no recurring friction problem has fingerprint %q in the current windows", fingerprint)
}

// ReportFriction files the problem in the bug inbox and records the bug id on
// it. Reporting a problem that was already reported returns it unchanged.
func (s Service) ReportFriction(ctx context.Context, fingerprint string, now time.Time) (Problem, error) {
	if s.Statuses == nil || s.Bugs == nil {
		return Problem{}, errors.New("friction reporting is unavailable")
	}
	problem, err := s.problem(ctx, fingerprint, now)
	if err != nil {
		return Problem{}, err
	}
	if problem.Status == StatusReported {
		return problem, nil
	}
	id, err := s.Bugs.FileFrictionBug(ctx, problem)
	if err != nil {
		return Problem{}, err
	}
	problem.Status, problem.BugID, problem.StatusSetAt = StatusReported, id, now
	return problem, s.Statuses.SetFrictionStatus(ctx, ProblemStatus{Fingerprint: fingerprint, Status: StatusReported, BugID: id, At: now})
}

// DismissFriction sets a problem aside: it stays listed as dismissed and is
// no longer recommended. A reported problem keeps its bug id.
func (s Service) DismissFriction(ctx context.Context, fingerprint string, now time.Time) error {
	if s.Statuses == nil {
		return errors.New("friction status is unavailable")
	}
	if strings.TrimSpace(fingerprint) == "" {
		return errors.New("fingerprint is required")
	}
	problem, err := s.problem(ctx, fingerprint, now)
	if err != nil {
		return err
	}
	return s.Statuses.SetFrictionStatus(ctx, ProblemStatus{Fingerprint: fingerprint, Status: StatusDismissed, BugID: problem.BugID, At: now})
}

func emptyRules() []RuleStatus {
	rules := make([]RuleStatus, 0, len(Rules))
	for _, rule := range Rules {
		rules = append(rules, RuleStatus{Rule: rule, Reason: "no provider window has used-percent readings"})
	}
	return rules
}

// Side is one cohort's quota per successful workload unit.
type Side struct {
	Runs, Succeeded, Priced int
	Tokens                  domain.UsageCounters
	Weighted, Points        float64
	From, To                time.Time
}

// PointsPerSuccess is the cohort's attributed points per successful run.
func (s Side) PointsPerSuccess() float64 {
	if s.Succeeded == 0 {
		return 0
	}
	return s.Points / float64(s.Succeeded)
}

// Compare prices runIDs' usage in [from, to] with the calibration of every
// window that overlaps it. A run's points are its attributed share of the
// observed increase, so the comparison is in provider quota, not tokens.
func (s Service) Compare(ctx context.Context, provider string, runIDs []string, from, to time.Time) (Side, error) {
	side := Side{Runs: len(runIDs), From: from, To: to}
	runs, err := s.Store.UsageRuns(ctx, runIDs)
	if err != nil {
		return side, err
	}
	members := map[string]bool{}
	for _, id := range runIDs {
		members[id] = true
		if runs[id].Succeeded {
			side.Succeeded++
		}
	}
	// A window that began up to its length before from still prices usage in
	// the period.
	spans, err := s.Readings.QuotaReadingSpans(ctx, provider, from.Add(-8*24*time.Hour))
	if err != nil {
		return side, err
	}
	turns, err := s.Store.UsageTurns(ctx, from, to)
	if err != nil {
		return side, err
	}
	priced := map[string]bool{}
	prices := s.Pricer.prices(ctx, turns, nil)
	for _, window := range usageforecast.Windows(spans) {
		if !window.HasPercent || window.LastAt.Before(from) || window.Start.After(to) || window.WindowMinutes < 24*60 {
			continue
		}
		start := window.Start
		if from.After(start) {
			start = from
		}
		result := Attribute(FromForecast(window), start, turns, prices)
		for i, turn := range result.Turns {
			if members[turn.RunID] && !turn.At.After(to) {
				side.Points += result.Points[i]
				side.Weighted += result.Weighted[i]
				priced[turn.RunID] = true
			}
		}
	}
	for _, turn := range turns {
		if members[turn.RunID] {
			side.Tokens = addCounters(side.Tokens, turn.Tokens)
		}
	}
	side.Priced = len(priced)
	return side, nil
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
