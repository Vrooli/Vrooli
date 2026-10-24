package goalhome

import (
	"fmt"
	"strings"
	"time"
)

// Thresholds are the step-back trigger settings; the CLI defaults follow the
// worker card.
type Thresholds struct {
	FlatlineK      int
	OverrunFactor  float64
	RefactorGrowth int
	FeatureFactor  float64
	WallClock      time.Duration
	SpendThreshold float64
}

// Trigger is one step-back (or review-only) trigger.
type Trigger struct {
	Name   string `json:"name"`
	Fired  bool   `json:"fired"`
	Review bool   `json:"review_only,omitempty"`
	Detail string `json:"detail"`
}

// EpochReport is the epoch-check result. The fields through
// MalformedLogLines are the original report; the rest add acceptance, typed
// gates and yield.
type EpochReport struct {
	File              string    `json:"file"`
	Kind              string    `json:"kind"`
	WorkUnits         int       `json:"work_units"`
	Estimate          int       `json:"estimate"`
	MetricHistory     []string  `json:"metric_history"`
	NetRuntimeLines   int       `json:"net_runtime_lines"`
	NetTestLines      int       `json:"net_test_lines"`
	OpenDirectives    []string  `json:"open_directives"`
	WeightedTokens    *float64  `json:"weighted_tokens,omitempty"`
	Accepted          string    `json:"accepted,omitempty"`
	Triggers          []Trigger `json:"triggers"`
	StepBack          []string  `json:"step_back"`
	Review            []string  `json:"review"`
	MalformedLogLines []string  `json:"malformed_log_lines,omitempty"`

	// AcceptedAt is set for an accepted epoch. Its fired triggers move to
	// AfterAcceptance: reported, never a step-back, a review or a wake.
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	ReopenedBy      []string   `json:"reopened_by,omitempty"`
	AfterAcceptance []string   `json:"after_acceptance,omitempty"`
	GateReport
	YieldEstimate *Quantity `json:"yield_estimate,omitempty"`
	// Yield is measured (ACCEPTED yield= or an inventory gate), never the
	// sum of slice-log deltas that NetRuntimeLines reports.
	Yield       *Quantity `json:"yield,omitempty"`
	YieldSource string    `json:"yield_source,omitempty"`
}

// EvaluateEpoch applies the step-back triggers and judges the gates.
// weighted is nil when spend was not measured; the spend trigger then reports
// that it was not evaluated.
func EvaluateEpoch(path string, epoch *Epoch, limits Thresholds, now time.Time, weighted *float64) *EpochReport {
	report := &EpochReport{
		File: path, Kind: strings.ToLower(epoch.Fields["kind"]), WorkUnits: len(epoch.Log),
		Estimate: leadingNumber(epoch.Fields["estimate"]), WeightedTokens: weighted,
		MetricHistory: []string{}, OpenDirectives: []string{}, StepBack: []string{}, Review: []string{}, MalformedLogLines: epoch.Malformed,
	}
	if accepted := epoch.LastAcceptance(); accepted != nil {
		report.Accepted = accepted.Raw
	}
	acked := map[string]bool{}
	for _, entry := range epoch.Log {
		report.MetricHistory = append(report.MetricHistory, entry.Metric)
		report.NetRuntimeLines += entry.RuntimeLines
		report.NetTestLines += entry.TestLines
		for _, id := range entry.Acks {
			acked[id] = true
		}
	}
	for _, directive := range epoch.Directives {
		if !acked[directive.ID] {
			report.OpenDirectives = append(report.OpenDirectives, directive.ID)
		}
	}
	accepted := epoch.IsAccepted()
	if accepted {
		at := epoch.LastAcceptance().At
		report.AcceptedAt = &at
	}
	report.ReopenedBy = epoch.ReopenedBy()
	add := func(trigger Trigger) {
		report.Triggers = append(report.Triggers, trigger)
		switch {
		case !trigger.Fired:
		case accepted:
			report.AfterAcceptance = append(report.AfterAcceptance, trigger.Name+": "+trigger.Detail)
		case trigger.Review:
			report.Review = append(report.Review, trigger.Name+": "+trigger.Detail)
		default:
			report.StepBack = append(report.StepBack, trigger.Name+": "+trigger.Detail)
		}
	}

	flat := Trigger{Name: "flatline", Detail: fmt.Sprintf("needs %d log lines, have %d", limits.FlatlineK, len(epoch.Log))}
	if limits.FlatlineK > 0 && len(epoch.Log) >= limits.FlatlineK {
		tail := epoch.Log[len(epoch.Log)-limits.FlatlineK:]
		flat.Fired = true
		for _, entry := range tail[1:] {
			if entry.Metric != tail[0].Metric {
				flat.Fired = false
			}
		}
		flat.Detail = fmt.Sprintf("exit metric over the last %d lines: %s", limits.FlatlineK, map[bool]string{true: "unchanged at " + tail[0].Metric, false: "moved"}[flat.Fired])
	}
	add(flat)

	overrun := Trigger{Name: "overrun", Detail: "no estimate in the brief"}
	if report.Estimate > 0 {
		limit := limits.OverrunFactor * float64(report.Estimate)
		overrun.Fired = float64(len(epoch.Log)) > limit
		overrun.Detail = fmt.Sprintf("%d log lines against %.0f (%.1fx estimate %d)", len(epoch.Log), limit, limits.OverrunFactor, report.Estimate)
	}
	add(overrun)

	growth := Trigger{Name: "growth", Detail: "kind is not refactor or feature"}
	switch report.Kind {
	case "refactor":
		growth.Fired = report.NetRuntimeLines > limits.RefactorGrowth
		growth.Detail = fmt.Sprintf("net runtime lines %+d against +%d", report.NetRuntimeLines, limits.RefactorGrowth)
	case "feature":
		expected := leadingNumber(epoch.Fields["expected size"])
		if expected > 0 {
			limit := limits.FeatureFactor * float64(expected)
			growth.Fired = float64(report.NetRuntimeLines) > limit
			growth.Detail = fmt.Sprintf("net runtime lines %+d against %.0f (%.1fx expected %d)", report.NetRuntimeLines, limit, limits.FeatureFactor, expected)
		} else {
			growth.Detail = "feature epoch has no expected size"
		}
	}
	add(growth)

	spend := Trigger{Name: "spend", Detail: "not evaluated: set --spend-threshold after calibrating on 3 epochs"}
	if limits.SpendThreshold > 0 && weighted == nil {
		spend.Detail = "not evaluated: no worker runs to measure"
	} else if limits.SpendThreshold > 0 {
		spend.Fired = *weighted > limits.SpendThreshold
		spend.Detail = fmt.Sprintf("weighted non-cache tokens %.0f against %.0f", *weighted, limits.SpendThreshold)
	}
	add(spend)

	wall := Trigger{Name: "wall-clock", Review: true, Detail: "no Started time in the brief"}
	if started, err := time.Parse(time.RFC3339, epoch.Fields["started"]); err == nil {
		elapsed := now.Sub(started)
		wall.Fired = limits.WallClock > 0 && elapsed > limits.WallClock
		wall.Detail = fmt.Sprintf("%s since start against %s (prompts a review; never forces a stop)", elapsed.Round(time.Minute), limits.WallClock)
	}
	add(wall)

	report.GateReport = EvaluateGates(epoch)
	report.YieldEstimate = epoch.YieldEstimate()
	report.Yield, report.YieldSource = epoch.Yield()
	return report
}
