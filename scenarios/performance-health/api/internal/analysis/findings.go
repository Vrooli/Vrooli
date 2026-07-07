package analysis

import (
	"fmt"
	"slices"

	mga "github.com/vrooli/maturity-go/assessment"
)

// DeriveFindings turns the per-component table into deterministic findings: a
// component whose average commit time exceeds budgetMs is flagged with
// quantified evidence and its located definition. NO AI — pure thresholding.
//
// A component over budget whose definition couldn't be located still emits a
// finding (metrics + a "definition not located" note in the message), rather
// than being dropped.
func DeriveFindings(components []ComponentTiming, budgetMs float64) []Finding {
	var out []Finding
	for _, c := range components {
		if budgetMs <= 0 || c.AvgMs <= budgetMs {
			continue
		}
		msg := fmt.Sprintf("%s averages %.1fms per commit, over the %.1fms budget", c.Component, c.AvgMs, budgetMs)
		if c.Definition == "" {
			msg += " (definition not located)"
		}
		out = append(out, Finding{
			Code:       "PERF_COMPONENT_COMMIT_OVER_BUDGET",
			Component:  c.Component,
			Definition: c.Definition,
			Message:    msg,
			Evidence:   fmt.Sprintf("count=%d avg=%.1fms max=%.1fms over budget by %.1fms", c.CommitCount, c.AvgMs, c.MaxMs, c.AvgMs-budgetMs),
			Severity:   "warning",
		})
	}
	return out
}

const (
	minInteractionInputEvents = 10
	minDrawnFPS               = 45.0
	maxDroppedFrameRate       = 0.20
	maxRasterTotalMs          = 500.0
	maxLayoutTotalMs          = 100.0
)

// DeriveInteractionFindings flags trace-level interaction evidence that React
// commit timing cannot explain: absent/insufficient input, poor frame health,
// and expensive browser raster/layout work.
func DeriveInteractionFindings(frames FrameSummary, browserWork, inputEvents []EventSummary) []Finding {
	var out []Finding
	totalInput := totalEventCount(inputEvents)
	if totalInput == 0 {
		out = append(out, Finding{
			Code:     "PERF_INTERACTION_INPUT_MISSING",
			Message:  "trace contains no input EventDispatch evidence for the measured interaction",
			Evidence: "input_event_count=0",
			Severity: "warning",
		})
	} else if totalInput < minInteractionInputEvents {
		out = append(out, Finding{
			Code:     "PERF_INTERACTION_INPUT_TOO_SMALL",
			Message:  fmt.Sprintf("trace contains only %d input event(s), below the %d-event interaction floor", totalInput, minInteractionInputEvents),
			Evidence: fmt.Sprintf("input_event_count=%d min=%d", totalInput, minInteractionInputEvents),
			Severity: "warning",
		})
	}

	if frames.BeginFrameCount == 0 && frames.DrawnFrameCount == 0 && frames.DroppedFrameCount == 0 {
		out = append(out, Finding{
			Code:     "PERF_FRAME_HEALTH_MISSING",
			Message:  "trace contains no frame events, so pan/zoom smoothness cannot be proven",
			Evidence: "begin_frame=0 drawn_frame=0 dropped_frame=0",
			Severity: "warning",
		})
	} else {
		if frames.ApproxDrawnFPS > 0 && frames.ApproxDrawnFPS < minDrawnFPS {
			out = append(out, Finding{
				Code:     "PERF_LOW_DRAWN_FPS",
				Message:  fmt.Sprintf("drawn frame rate %.1ffps is below the %.1ffps interaction floor", frames.ApproxDrawnFPS, minDrawnFPS),
				Evidence: fmt.Sprintf("drawn_fps=%.1f duration=%.1fms drawn_frames=%d", frames.ApproxDrawnFPS, frames.TraceDurationMs, frames.DrawnFrameCount),
				Severity: "warning",
			})
		}
		if frames.DroppedFrameRate > maxDroppedFrameRate {
			out = append(out, Finding{
				Code:     "PERF_HIGH_DROPPED_FRAME_RATE",
				Message:  fmt.Sprintf("dropped-frame rate %.1f%% exceeds the %.1f%% interaction ceiling", frames.DroppedFrameRate*100, maxDroppedFrameRate*100),
				Evidence: fmt.Sprintf("dropped_rate=%.3f drawn_frames=%d dropped_frames=%d", frames.DroppedFrameRate, frames.DrawnFrameCount, frames.DroppedFrameCount),
				Severity: "warning",
			})
		}
	}

	for _, work := range browserWork {
		switch {
		case work.Name == "RasterTask" && work.TotalMs > maxRasterTotalMs:
			out = append(out, browserWorkFinding("PERF_HIGH_RASTER_COST", work, maxRasterTotalMs))
		case slices.Contains([]string{"Layout", "UpdateLayoutTree"}, work.Name) && work.TotalMs > maxLayoutTotalMs:
			out = append(out, browserWorkFinding("PERF_HIGH_LAYOUT_COST", work, maxLayoutTotalMs))
		}
	}
	return out
}

func totalEventCount(events []EventSummary) int {
	total := 0
	for _, e := range events {
		total += e.Count
	}
	return total
}

func browserWorkFinding(code string, work EventSummary, maxTotalMs float64) Finding {
	return Finding{
		Code:     code,
		Message:  fmt.Sprintf("%s totals %.1fms, over the %.1fms interaction ceiling", work.Name, work.TotalMs, maxTotalMs),
		Evidence: fmt.Sprintf("%s count=%d total=%.1fms avg=%.1fms max=%.1fms", work.Name, work.Count, work.TotalMs, work.AvgMs, work.MaxMs),
		Severity: "warning",
	}
}

// AssessmentFindings projects deterministic perf findings into the shared
// maturity-go finding shape so they flow through the same finding pipeline as
// readiness (packages/maturity-go/assessment). The located definition becomes
// the finding Location and the quantified evidence is folded into the message;
// AutofixAvailable is false (perf hotspots are not deterministically autofixed).
func AssessmentFindings(findings []Finding) []mga.Finding {
	out := make([]mga.Finding, 0, len(findings))
	for _, f := range findings {
		out = append(out, mga.Finding{
			Code:             f.Code,
			Severity:         f.Severity,
			Title:            f.Component,
			Message:          f.Message + " [" + f.Evidence + "]",
			Location:         f.Definition,
			AutofixAvailable: false,
		})
	}
	return out
}
