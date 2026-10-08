package goalhome

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Diminishing-returns rules (DL-3, decision P-06).
const (
	// LowYieldRatio and LowYieldRun: this many accepted epochs in a row, each
	// below this fraction of its yield estimate.
	LowYieldRatio = 0.25
	LowYieldRun   = 3
	// EmptyCensusRun: this many latest censuses in a row with no admissible slice.
	EmptyCensusRun = 2
	// GapHorizonEpochs: the remaining gap needs more epochs than this at the
	// mean yield of the last HorizonWindow accepted epochs.
	GapHorizonEpochs = 30
	HorizonWindow    = 3
)

// EpochYield is one accepted epoch's estimate and measured yield.
type EpochYield struct {
	Epoch       string    `json:"epoch"`
	AcceptedAt  time.Time `json:"accepted_at"`
	Estimate    *Quantity `json:"estimate,omitempty"`
	Yield       *Quantity `json:"yield,omitempty"`
	YieldSource string    `json:"yield_source,omitempty"`
}

// Rule is one diminishing-returns rule outcome.
type Rule struct {
	Name   string `json:"rule"`
	Fired  bool   `json:"fired"`
	Detail string `json:"detail"`
}

// Forecast is the goal-level diminishing-returns check over the epochs
// accepted since GOAL.md's "Destination set" time.
type Forecast struct {
	Accepted []EpochYield `json:"accepted"`
	Rules    []Rule       `json:"rules"`
	// Acknowledged is true when ## Needs operator holds a [re-aim] item.
	Acknowledged bool `json:"acknowledged"`
}

// Fired lists the rules that fired.
func (f *Forecast) Fired() []Rule {
	var fired []Rule
	for _, rule := range f.Rules {
		if rule.Fired {
			fired = append(fired, rule)
		}
	}
	return fired
}

// Finding is the diminishing-returns finding, or nil when no rule fired. It
// blocks until ## Needs operator holds a [re-aim] decision or the destination
// is reset.
func (f *Forecast) Finding() *Finding {
	fired := f.Fired()
	if len(fired) == 0 {
		return nil
	}
	reasons := make([]string, 0, len(fired))
	for _, rule := range fired {
		reasons = append(reasons, rule.Name+": "+rule.Detail)
	}
	detail := strings.Join(reasons, "; ")
	if f.Acknowledged {
		detail += " (a [re-aim] decision is under ## Needs operator)"
	} else {
		detail += ". Record a re-aim decision with options under ## Needs operator, tagged [re-aim], before admitting another slice"
	}
	return &Finding{Code: "diminishing-returns", Blocking: !f.Acknowledged, Detail: detail, File: QueueFile}
}

// ForecastOf evaluates the diminishing-returns rules for a goal home.
func ForecastOf(home *Home) *Forecast {
	forecast := &Forecast{Accepted: []EpochYield{}, Acknowledged: home.Queue.HasReaimDecision()}
	for _, file := range home.Epochs {
		if !file.Epoch.IsAccepted() {
			continue
		}
		accepted := file.Epoch.LastAcceptance()
		if !home.Goal.DestinationSet.IsZero() && accepted.At.Before(home.Goal.DestinationSet) {
			continue
		}
		yield, source := file.Epoch.Yield()
		forecast.Accepted = append(forecast.Accepted, EpochYield{Epoch: strings.TrimSuffix(file.Name, ".md"), AcceptedAt: accepted.At, Estimate: file.Epoch.YieldEstimate(), Yield: yield, YieldSource: source})
	}
	sort.SliceStable(forecast.Accepted, func(i, j int) bool { return forecast.Accepted[i].AcceptedAt.Before(forecast.Accepted[j].AcceptedAt) })
	forecast.Rules = []Rule{lowYieldRule(forecast.Accepted), emptyCensusRule(home.Queue.Censuses), gapHorizonRule(home.Goal, forecast.Accepted)}
	return forecast
}

func lowYieldRule(accepted []EpochYield) Rule {
	rule := Rule{Name: "low-yield"}
	if len(accepted) < LowYieldRun {
		rule.Detail = fmt.Sprintf("not evaluated: %d accepted epochs since the destination was set, need %d", len(accepted), LowYieldRun)
		return rule
	}
	var ratios, unknown []string
	for _, epoch := range accepted[len(accepted)-LowYieldRun:] {
		switch {
		case epoch.Estimate == nil || epoch.Estimate.Value == 0:
			unknown = append(unknown, epoch.Epoch+" has no Yield estimate")
		case epoch.Yield == nil:
			unknown = append(unknown, epoch.Epoch+" has no measured yield")
		case !sameUnit(epoch.Estimate.Unit, epoch.Yield.Unit):
			unknown = append(unknown, fmt.Sprintf("%s estimate unit %q differs from yield unit %q", epoch.Epoch, epoch.Estimate.Unit, epoch.Yield.Unit))
		default:
			ratio := epoch.Yield.Value / epoch.Estimate.Value
			if ratio >= LowYieldRatio {
				rule.Detail = fmt.Sprintf("%s delivered %.0f%% of its estimate", epoch.Epoch, ratio*100)
				return rule
			}
			ratios = append(ratios, fmt.Sprintf("%s %s of %s", epoch.Epoch, epoch.Yield, epoch.Estimate))
		}
	}
	if len(unknown) > 0 {
		rule.Detail = "not evaluated: " + strings.Join(unknown, "; ")
		return rule
	}
	rule.Fired = true
	rule.Detail = fmt.Sprintf("the last %d accepted epochs each delivered under %.0f%% of estimate (%s)", LowYieldRun, LowYieldRatio*100, strings.Join(ratios, ", "))
	return rule
}

func emptyCensusRule(censuses []Census) Rule {
	rule := Rule{Name: "empty-censuses"}
	if len(censuses) < EmptyCensusRun {
		rule.Detail = fmt.Sprintf("not evaluated: %d censuses under ## Censuses, need %d", len(censuses), EmptyCensusRun)
		return rule
	}
	for _, census := range censuses[len(censuses)-EmptyCensusRun:] {
		if !census.Empty() {
			rule.Detail = fmt.Sprintf("the census at line %d found %s admissible", census.Line, census.Admissible)
			return rule
		}
	}
	rule.Fired = true
	rule.Detail = fmt.Sprintf("the last %d censuses found no admissible slice", EmptyCensusRun)
	return rule
}

func gapHorizonRule(goal Goal, accepted []EpochYield) Rule {
	rule := Rule{Name: "gap-horizon"}
	if goal.Destination == nil || goal.Current == nil {
		rule.Detail = "not evaluated: GOAL.md needs \"- Destination: <metric> <op> <n>\" and \"- Current: <n> @ <date>\""
		return rule
	}
	destination := goal.Destination
	if compare(goal.Current.Value, destination.Op, destination.Target) {
		rule.Detail = fmt.Sprintf("destination met: %s %s %s %s", destination.Metric, formatNumber(goal.Current.Value), destination.Op, formatNumber(destination.Target))
		return rule
	}
	gap := destination.Target - goal.Current.Value
	direction := math.Copysign(1, gap)
	window := accepted
	if len(window) > HorizonWindow {
		window = window[len(window)-HorizonWindow:]
	}
	var progress []float64
	for _, epoch := range window {
		if epoch.Yield != nil && sameUnit(epoch.Yield.Unit, destination.Metric) {
			progress = append(progress, epoch.Yield.Value*direction)
		}
	}
	if len(progress) == 0 {
		rule.Detail = fmt.Sprintf("not evaluated: none of the last %d accepted epochs has a measured yield in %s", HorizonWindow, destination.Metric)
		return rule
	}
	mean := 0.0
	for _, value := range progress {
		mean += value
	}
	mean /= float64(len(progress))
	if mean <= 0 {
		rule.Fired = true
		rule.Detail = fmt.Sprintf("the gap of %s %s is not closing: mean yield %s over %d epochs", formatNumber(math.Abs(gap)), destination.Metric, formatNumber(mean*direction), len(progress))
		return rule
	}
	epochs := math.Abs(gap) / mean
	rule.Fired = epochs > GapHorizonEpochs
	rule.Detail = fmt.Sprintf("the gap of %s %s needs about %.0f epochs at the mean yield of %s over %d epochs (limit %d)", formatNumber(math.Abs(gap)), destination.Metric, epochs, formatNumber(mean), len(progress), GapHorizonEpochs)
	return rule
}
