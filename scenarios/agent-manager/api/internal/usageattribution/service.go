package usageattribution

import (
	"context"
	"sort"
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

// Recommendations evaluates the rules over the current windows.
func (s Service) Recommendations(ctx context.Context, provider string, now time.Time) (Report, Attribution, error) {
	attribution, err := s.Current(ctx, provider, time.Time{}, now)
	if err != nil || len(attribution.Results) == 0 {
		return Report{Rules: emptyRules()}, attribution, err
	}
	from := now
	for _, result := range primaryResults(attribution.Results) {
		if result.From.Before(from) {
			from = result.From
		}
	}
	episodes, err := s.Store.UsageEpisodes(ctx, from, now)
	if err != nil {
		return Report{}, attribution, err
	}
	outcomes, err := s.Store.UsageOutcomes(ctx, now.Add(-Lookback), now)
	if err != nil {
		return Report{}, attribution, err
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
			return Report{}, attribution, err
		}
		for id, run := range extra {
			attribution.Runs[id] = run
		}
	}
	report := Recommend(Inputs{Results: attribution.Results, Runs: attribution.Runs, Episodes: episodes, Outcomes: outcomes, Prices: s.Pricer.prices(ctx, nil, outcomes)})
	return report, attribution, nil
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
