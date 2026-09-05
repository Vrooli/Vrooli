package wizard

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Apply can take many minutes: a scenario start is bounded at five, and a
// selection may carry dozens of them. The wizard used to poll the run in
// silence for that entire time and print only the final report, so the operator
// saw a bare prompt and no output -- indistinguishable from a hang. These
// cadences match `vrooli setup`, which the operator has already seen, so the
// two long-running surfaces read the same way.
const (
	applyFirstHeartbeat = 10 * time.Second
	applyHeartbeatEvery = 30 * time.Second

	// applyOutcomeApplying mirrors the API's marker for the item a runner is
	// executing right now, and applyOutcomePending its marker for one not yet
	// reached. These are the two non-terminal outcomes; every other value means
	// the item is done. The wire contract is pinned by a test rather than
	// shared as a symbol, because the two live in different modules.
	applyOutcomeApplying = "applying"
	applyOutcomePending  = "pending"
)

// applyRunItem is the subset of an apply run item this reporter renders.
type applyRunItem struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Outcome string `json:"outcome"`
	Error   string `json:"error,omitempty"`
}

func (i applyRunItem) label() string {
	kind := strings.TrimSpace(i.Kind)
	name := strings.TrimSpace(i.Name)
	switch {
	case kind == "" && name == "":
		return strings.TrimSpace(i.ID)
	case kind == "":
		return name
	default:
		return kind + " " + name
	}
}

// applyProgressReporter turns successive poll snapshots into a live log.
//
// It is driven by the poll loop rather than by a timer of its own so that it
// cannot outlive the run or emit after the wizard has moved on, and so the
// whole thing stays testable with a fake clock.
type applyProgressReporter struct {
	out   io.Writer
	now   func() time.Time
	total int

	started       bool
	announced     map[string]bool
	inFlight      string
	inFlightSince time.Time
	lastHeartbeat time.Time
}

func newApplyProgressReporter(out io.Writer, now func() time.Time) *applyProgressReporter {
	if now == nil {
		now = time.Now
	}
	return &applyProgressReporter{out: out, now: now, announced: map[string]bool{}}
}

// Observe renders whatever changed since the previous snapshot.
func (r *applyProgressReporter) Observe(items []applyRunItem) {
	if len(items) == 0 {
		return
	}
	if !r.started {
		r.started = true
		r.total = len(items)
		fmt.Fprintf(r.out, "APPLY  · Applying %d item(s)\n", r.total)
	}

	for index, item := range items {
		key := r.key(item, index)
		outcome := strings.TrimSpace(item.Outcome)

		if outcome == applyOutcomeApplying {
			if r.inFlight != key {
				r.inFlight = key
				r.inFlightSince = r.now()
				r.lastHeartbeat = time.Time{}
				fmt.Fprintf(r.out, "         ↳ %s (%d/%d)\n", item.label(), index+1, r.total)
			}
			continue
		}
		// A run is written with every item already marked pending, so an
		// unrecognised-as-unstarted outcome would render the whole selection as
		// finished the instant it was accepted.
		if outcome == "" || outcome == applyOutcomePending || r.announced[key] {
			continue
		}

		r.announced[key] = true
		elapsed := ""
		if r.inFlight == key {
			elapsed = " (" + formatApplyDuration(r.now().Sub(r.inFlightSince)) + ")"
			r.inFlight = ""
		}
		switch outcome {
		case "applied", "already_satisfied", "skipped_self":
			fmt.Fprintf(r.out, "         ✓ %s %s%s\n", item.label(), outcome, elapsed)
		default:
			fmt.Fprintf(r.out, "         ! %s %s%s\n", item.label(), outcome, elapsed)
			if detail := summarizeApplyError(item.Error); detail != "" {
				fmt.Fprintf(r.out, "           %s\n", detail)
			}
		}
	}

	r.heartbeat(items)
}

// heartbeat proves a long step is still working. Without it the gap between
// "↳ starting" and "✓ done" is silent for minutes, which is the exact shape of
// the hang the operator reported.
func (r *applyProgressReporter) heartbeat(items []applyRunItem) {
	if r.inFlight == "" {
		return
	}
	now := r.now()
	due := r.inFlightSince.Add(applyFirstHeartbeat)
	if !r.lastHeartbeat.IsZero() {
		due = r.lastHeartbeat.Add(applyHeartbeatEvery)
	}
	if now.Before(due) {
		return
	}
	r.lastHeartbeat = now

	label := r.inFlight
	for index, item := range items {
		if r.key(item, index) == r.inFlight {
			label = item.label()
			break
		}
	}
	fmt.Fprintf(r.out, "         · still working: %s (%s elapsed)\n", label, formatApplyDuration(now.Sub(r.inFlightSince)))
}

// Finish prints the closing tally. It reports what actually happened per
// outcome rather than a bare "done", so a partially applied run cannot read as
// a clean one.
func (r *applyProgressReporter) Finish(status string, items []applyRunItem) {
	if !r.started {
		return
	}
	counts := map[string]int{}
	for _, item := range items {
		outcome := strings.TrimSpace(item.Outcome)
		if outcome == "" {
			outcome = "not reached"
		}
		counts[outcome]++
	}
	summary := make([]string, 0, len(counts))
	for outcome, count := range counts {
		summary = append(summary, fmt.Sprintf("%d %s", count, outcome))
	}
	sort.Strings(summary)
	fmt.Fprintf(r.out, "APPLY  · %s — %s\n", status, strings.Join(summary, ", "))
}

// Reconnected clears the in-flight marker after a gap in observation. The API
// restarting mid-apply is expected on this path, and the elapsed time measured
// across that gap would be the wizard's downtime, not the item's.
func (r *applyProgressReporter) Reconnected() {
	r.inFlight = ""
	r.lastHeartbeat = time.Time{}
}

// key identifies an item across snapshots. Position is the fallback because an
// item without an ID must still not be confused with its neighbours.
func (r *applyProgressReporter) key(item applyRunItem, index int) string {
	if id := strings.TrimSpace(item.ID); id != "" {
		return id
	}
	return fmt.Sprintf("%d:%s", index, item.label())
}

func formatApplyDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	return d.Round(time.Second).String()
}

// applyErrorSummaryLimit bounds the inline failure line.
//
// A control-plane failure carries the whole probe result: the remote-desktop
// safeguard alone returns about 1.5KB of JSON on one line. Printing that raw
// buries the progress log it was meant to annotate. The full text still reaches
// the operator in the final report, so this line only has to be enough to
// recognise the failure by.
const applyErrorSummaryLimit = 160

func summarizeApplyError(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	if index := strings.IndexAny(detail, "\r\n"); index >= 0 {
		detail = strings.TrimSpace(detail[:index])
	}
	// A control-plane error often ends with the JSON payload it collected.
	// The prose before it is the part worth showing.
	if index := strings.Index(detail, ": {"); index >= 0 {
		detail = strings.TrimSpace(detail[:index])
	}
	if len(detail) > applyErrorSummaryLimit {
		detail = strings.TrimSpace(detail[:applyErrorSummaryLimit]) + "..."
	}
	return detail
}
