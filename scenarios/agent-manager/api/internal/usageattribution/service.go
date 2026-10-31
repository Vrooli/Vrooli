package usageattribution

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/interventions"
	"agent-manager/internal/opportunities"
	"agent-manager/internal/pricing"
	"agent-manager/internal/toolfriction"
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
	// Tools reads the projected command facts tool friction profiles.
	Tools ToolCallStore
	// Library and Interventions feed opportunities and operator corrections
	// into Recommendations.
	Library       LibrarySource
	Interventions InterventionSummarizer
}

// Attribution is the current windows' results and the runs they name, with
// the forecasts of those windows.
type Attribution struct {
	Results   []Result
	Runs      map[string]Run
	Spans     int
	Forecasts []usageforecast.Forecast
}

// Current attributes every current window of provider (all when empty) after
// since (each window's start when zero).
func (s Service) Current(ctx context.Context, provider string, since, now time.Time) (Attribution, error) {
	spans, err := s.Readings.QuotaReadingSpans(ctx, provider, now.Add(-usageforecast.CurrentLookback))
	if err != nil {
		return Attribution{}, err
	}
	var windows []Window
	var forecasts []usageforecast.Forecast
	for _, forecast := range usageforecast.Current(spans, now) {
		// A window whose reset has passed is history, not a current window.
		if forecast.Window != nil && forecast.Window.HasPercent && (forecast.Window.ResetAt.IsZero() || forecast.Window.ResetAt.After(now)) {
			windows = append(windows, FromForecast(forecast.Window))
			forecasts = append(forecasts, forecast)
		}
	}
	out, err := s.attribute(ctx, windows, since, now, len(spans))
	out.Forecasts = forecasts
	return out, err
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
// inbox and returns the bug id; the inbox deduplicates on the fingerprint.
type BugFiler interface {
	FileFrictionBug(ctx context.Context, bug FrictionBug) (string, error)
}

// ToolCallStore reads the command facts of a window projected at the current
// fact version, with the features stored for them, and counts what the window
// holds at older versions.
type ToolCallStore interface {
	ToolCalls(ctx context.Context, from, to time.Time) (toolfriction.Window, error)
}

// ToolFrictionWindow is the default tool-friction window: a week.
const ToolFrictionWindow = 7 * 24 * time.Hour

// MaxLookback is the longest window the friction list, the opportunities and
// a finding's lookup read: an older start is clamped to it.
const MaxLookback = 90 * 24 * time.Hour

// ToolWindowStart is the start of a tool window that asks for since: the
// default week when since is zero, at most MaxLookback before now, and never
// at or after now.
func ToolWindowStart(since, now time.Time) (time.Time, error) {
	switch {
	case since.IsZero():
		return now.Add(-ToolFrictionWindow), nil
	case !since.Before(now):
		return time.Time{}, fmt.Errorf("since %s is not before now (%s)", since.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339))
	case since.Before(now.Add(-MaxLookback)):
		return now.Add(-MaxLookback), nil
	}
	return since, nil
}

// LibrarySource reads the program library sequences are matched to.
type LibrarySource interface {
	Library(ctx context.Context) opportunities.Library
}

// InterventionSummarizer summarizes classified operator messages under a gate.
type InterventionSummarizer interface {
	Summarize(ctx context.Context, version string, gate interventions.Gate, from, to time.Time) (interventions.Summary, error)
}

// Recommendations is the one ranked list of what to change: the quota levers
// of the current windows (each provider's longest, or only the window named by
// only) merged with the tool friction findings, opportunities and gated
// operator interventions of the last ToolFrictionWindow. A source that cannot
// be read is left out and named in the coverage.
func (s Service) Recommendations(ctx context.Context, provider string, only WindowKey, now time.Time) (Report, Attribution, error) {
	attribution, err := s.Current(ctx, provider, time.Time{}, now)
	if err != nil {
		return Report{}, attribution, err
	}
	in := Inputs{Results: attribution.Results, Runs: attribution.Runs, Only: only, Now: now}
	if len(attribution.Results) > 0 {
		if in.Outcomes, err = s.Store.UsageOutcomes(ctx, now.Add(-Lookback), now); err != nil {
			return Report{}, attribution, err
		}
		in.Prices = s.Pricer.prices(ctx, nil, in.Outcomes)
	}
	from := now.Add(-ToolFrictionWindow)
	if s.Tools != nil {
		window, err := s.Tools.ToolCalls(ctx, from, now)
		if err != nil {
			return Report{}, attribution, err
		}
		in.Calls = append([]toolfriction.Call{}, window.Calls...) // non-nil: the source was read
		in.Stale = window.Stale
		if s.Statuses != nil {
			if in.Statuses, err = s.Statuses.FrictionStatuses(ctx); err != nil {
				return Report{}, attribution, err
			}
		}
		if s.Library != nil {
			library := s.Library.Library(ctx)
			in.Library = &library
			in.Outputs = opportunities.OutputsFrom(ctx, s.Tools)
		}
	} else {
		in.Coverage = append(in.Coverage, "tool calls are unavailable: tool friction, skill weight and opportunities cannot tell")
	}
	if s.Interventions != nil {
		summary, err := s.Interventions.Summarize(ctx, interventions.ClassifierVersion, interventions.ProvenGate, from, now)
		if err != nil {
			return Report{}, attribution, err
		}
		in.Interventions = &summary
	}
	report := Recommend(in)
	report.Outcome = outcome(report.Actions, attribution.Forecasts, only)
	return report, attribution, nil
}

// outcome is the named window's forecast with its listed quota levers
// applied (CombinedRecoverable); nil when no window is named or it has no
// current forecast.
func outcome(actions []Action, forecasts []usageforecast.Forecast, only WindowKey) *WindowOutcome {
	if only.Pool == "" {
		return nil
	}
	for _, f := range forecasts {
		if f.Window.Provider == only.Provider && f.Window.Pool == only.Pool && f.Window.Window == only.Window {
			saving, levers := CombinedRecoverable(actions, only)
			return &WindowOutcome{Outcome: f.WithSaving(saving), Levers: levers}
		}
	}
	return nil
}

// FrictionReport is the window's owner-level tool findings from the tool
// profiles since Since, with the coverage that says how much of it they could
// see. Operator interventions are classified per message by package
// interventions and read beside it.
type FrictionReport struct {
	Since        time.Time
	Tools        toolfriction.Report
	ToolFindings []ToolFinding
}

// Friction lists the window's tool findings with their durable status. The
// window starts at ToolWindowStart(since, now).
func (s Service) Friction(ctx context.Context, since, now time.Time) (FrictionReport, error) {
	since, err := ToolWindowStart(since, now)
	if err != nil {
		return FrictionReport{}, err
	}
	report := FrictionReport{Since: since}
	statuses := map[string]ProblemStatus{}
	if s.Statuses != nil {
		var err error
		if statuses, err = s.Statuses.FrictionStatuses(ctx); err != nil {
			return report, err
		}
	}
	if s.Tools != nil {
		window, err := s.Tools.ToolCalls(ctx, since, now)
		if err != nil {
			return report, err
		}
		report.Tools = toolfriction.Analyze(window)
		for _, finding := range report.Tools.Findings {
			item := ToolFinding{Finding: finding, Status: StatusOpen}
			if status, ok := statuses[finding.Fingerprint]; ok {
				item.Status, item.BugID, item.StatusSetAt = status.Status, status.BugID, status.At
			}
			report.ToolFindings = append(report.ToolFindings, item)
		}
	}
	return report, nil
}

// FrictionItem is a tool finding as an operator acts on it.
type FrictionItem struct {
	Fingerprint, Description, Status, BugID string
	bug                                     FrictionBug
}

// ErrUnknownFinding is a fingerprint no tool finding of the window has.
var ErrUnknownFinding = errors.New("no tool finding has this fingerprint")

// ErrFrictionUnavailable is a status store or bug inbox that cannot be
// reached: retrying later can succeed.
var ErrFrictionUnavailable = errors.New("friction reporting is unavailable")

// item finds a tool finding by fingerprint in the window the caller listed it
// in (ToolWindowStart(since, now)).
func (s Service) item(ctx context.Context, fingerprint string, since, now time.Time) (FrictionItem, error) {
	report, err := s.Friction(ctx, since, now)
	if err != nil {
		return FrictionItem{}, err
	}
	for _, finding := range report.ToolFindings {
		if finding.Fingerprint == fingerprint {
			return FrictionItem{Fingerprint: fingerprint, Description: finding.Description, Status: finding.Status, BugID: finding.BugID, bug: finding.bug()}, nil
		}
	}
	return FrictionItem{}, fmt.Errorf("%w %q since %s", ErrUnknownFinding, fingerprint, report.Since.UTC().Format(time.RFC3339))
}

// reporting serializes the actions on one fingerprint in this process, so two
// concurrent reports of a finding file it once.
var reporting sync.Map

func lockFingerprint(fingerprint string) func() {
	mu, _ := reporting.LoadOrStore(fingerprint, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}

// ReportFriction files the finding listed since in the bug inbox and records
// the bug id on it. It files at most once per fingerprint: a reported finding
// returns its bug unchanged, concurrent calls are serialized, and the filing
// is keyed on the fingerprint, so a retry after the status write failed gets
// the bug the inbox already holds.
func (s Service) ReportFriction(ctx context.Context, fingerprint string, since, now time.Time) (FrictionItem, error) {
	if s.Statuses == nil || s.Bugs == nil {
		return FrictionItem{}, ErrFrictionUnavailable
	}
	defer lockFingerprint(fingerprint)()
	item, err := s.item(ctx, fingerprint, since, now)
	if err != nil || item.Status == StatusReported {
		return item, err
	}
	id, err := s.Bugs.FileFrictionBug(ctx, item.bug)
	if err != nil {
		return FrictionItem{}, fmt.Errorf("%w: %v", ErrFrictionUnavailable, err)
	}
	item.Status, item.BugID = StatusReported, id
	return item, s.Statuses.SetFrictionStatus(ctx, ProblemStatus{Fingerprint: fingerprint, Status: StatusReported, BugID: id, At: now})
}

// DismissFriction sets the finding listed since aside: it stays listed as
// dismissed and is no longer recommended. A reported problem keeps its bug id.
func (s Service) DismissFriction(ctx context.Context, fingerprint string, since, now time.Time) error {
	if s.Statuses == nil {
		return ErrFrictionUnavailable
	}
	defer lockFingerprint(fingerprint)()
	item, err := s.item(ctx, fingerprint, since, now)
	if err != nil {
		return err
	}
	return s.Statuses.SetFrictionStatus(ctx, ProblemStatus{Fingerprint: fingerprint, Status: StatusDismissed, BugID: item.BugID, At: now})
}

// Side is one cohort's quota per accepted work unit. Completed counts lifecycle
// completion only; an imported run is complete when its transcript ends.
type Side struct {
	Runs, Completed, Priced int
	// Accepted and UnknownUnits count distinct work units by outcome estimate.
	Accepted, UnknownUnits int
	Tokens                 domain.UsageCounters
	Weighted, Points       float64
	From, To               time.Time
}

// PointsPerAccepted is the cohort's attributed points per accepted work unit.
func (s Side) PointsPerAccepted() float64 {
	if s.Accepted == 0 {
		return 0
	}
	return s.Points / float64(s.Accepted)
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
	accepted, unknown := map[string]bool{}, map[string]bool{}
	for _, id := range runIDs {
		run := runs[id]
		if run.Completed {
			side.Completed++
		}
		unit := run.WorkUnit
		if unit == "" {
			unit = id
		}
		switch {
		case run.Accepted:
			accepted[unit] = true
		case run.OutcomeUnknown:
			unknown[unit] = true
		}
	}
	side.Accepted, side.UnknownUnits = len(accepted), len(unknown)
	period, err := s.PeriodUse(ctx, provider, from, to)
	if err != nil {
		return side, err
	}
	for _, id := range runIDs {
		priced := false
		for _, use := range period.Runs[id] {
			side.Points += use.Points
			side.Weighted += use.Weighted
			side.Tokens = addCounters(side.Tokens, use.Tokens)
			priced = priced || use.Priced
		}
		if priced {
			side.Priced++
		}
	}
	return side, nil
}

// ModelUse is one run's use on one model in a period.
type ModelUse struct {
	Points, Weighted float64
	Tokens           domain.UsageCounters
	// Priced reports usage a calibrated window explained.
	Priced bool
}

// PeriodUse is every run's use in a period, per model, with the calibration of
// the windows that priced it.
type PeriodUse struct {
	Runs    map[string]map[string]ModelUse
	Sources map[string]string
	Windows []Window
	Turns   []Turn
	Prices  Prices
}

// PeriodUse attributes the turns in [from, to] across every window of at least
// a day that overlaps it. Windows of one provider can overlap (a model pool
// beside the account pool), so a turn's points are its largest attribution, not
// their sum.
func (s Service) PeriodUse(ctx context.Context, provider string, from, to time.Time) (PeriodUse, error) {
	out := PeriodUse{Runs: map[string]map[string]ModelUse{}, Sources: map[string]string{}}
	// A window that began up to its length before from still prices usage in
	// the period.
	spans, err := s.Readings.QuotaReadingSpans(ctx, provider, from.Add(-8*24*time.Hour))
	if err != nil {
		return out, err
	}
	turns, err := s.Store.UsageTurns(ctx, from, to)
	if err != nil {
		return out, err
	}
	prices := s.Pricer.prices(ctx, turns, nil)
	out.Turns, out.Prices = turns, prices
	type turnKey struct {
		run, model string
		at         time.Time
	}
	best := map[turnKey][2]float64{}
	for _, window := range usageforecast.Windows(spans) {
		if !window.HasPercent || window.LastAt.Before(from) || window.Start.After(to) || window.WindowMinutes < 24*60 {
			continue
		}
		if provider != "" && window.Provider != provider {
			continue
		}
		start := window.Start
		if from.After(start) {
			start = from
		}
		w := FromForecast(window)
		result := Attribute(w, start, turns, prices)
		out.Windows = append(out.Windows, w)
		for _, weight := range result.Calibration.Weights {
			if weight.Source != "" {
				out.Sources[weight.Model] = weight.Source
			}
		}
		for i, turn := range result.Turns {
			if turn.At.After(to) {
				continue
			}
			key := turnKey{turn.RunID, turn.Model, turn.At}
			if current := best[key]; result.Points[i] > current[0] {
				best[key] = [2]float64{result.Points[i], result.Weighted[i]}
			} else if _, seen := best[key]; !seen {
				best[key] = [2]float64{result.Points[i], result.Weighted[i]}
			}
		}
	}
	for _, turn := range turns {
		if turn.At.Before(from) || turn.At.After(to) {
			continue
		}
		models := out.Runs[turn.RunID]
		if models == nil {
			models = map[string]ModelUse{}
			out.Runs[turn.RunID] = models
		}
		use := models[turn.Model]
		use.Tokens = addCounters(use.Tokens, turn.Tokens)
		if attributed, ok := best[turnKey{turn.RunID, turn.Model, turn.At}]; ok {
			use.Points += attributed[0]
			use.Weighted += attributed[1]
			use.Priced = true
		}
		models[turn.Model] = use
	}
	return out, nil
}

// Consumption is price-independent use: tokens at fixed category weights, so a
// cheaper model's extra work reads as more consumption rather than less cost.
func Consumption(tokens domain.UsageCounters) float64 {
	return float64(tokens.InputTokens) + 0.1*float64(tokens.CacheReadTokens) + 1.25*float64(tokens.CacheCreationTokens) + 5*float64(tokens.OutputTokens)
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
