package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Defaults. They are deliberately conservative: a caller that constructs a
// Policy with only a Mode gets bounded batches, bounded retries, and a real
// deferral ceiling, not an unbounded loop.
const (
	// DefaultMaxBatch is the largest number of jobs one admission may cover.
	DefaultMaxBatch = 32
	// DefaultMaxAttempts is the number of execution attempts before a job fails.
	DefaultMaxAttempts = 5
	// DefaultLeaseTTL is how long a leased job survives without renewal.
	DefaultLeaseTTL = 5 * time.Minute
	// DefaultModelDwell is how long a local model is treated as resident after
	// its last use. Jobs inside the dwell reuse the loaded model.
	DefaultModelDwell = 15 * time.Minute
	// DefaultDeferBase is the first backoff after a denial or failure.
	DefaultDeferBase = 30 * time.Second
	// DefaultDeferCap bounds exponential deferral backoff.
	DefaultDeferCap = 30 * time.Minute
	// DefaultInterval is the recommended tick cadence.
	DefaultInterval = 30 * time.Second

	// bookkeepingTimeout bounds the cancellation-proof write that records a
	// finished job's outcome.
	bookkeepingTimeout = 15 * time.Second
)

// Window is a recurring local-time interval in which background work may run.
//
// Days selects the weekdays the window starts on; an empty Days means every day.
// Start and End are "HH:MM" (24-hour). End may be "24:00". When End is before
// Start the window crosses midnight and is attributed to the day it starts on,
// so a Saturday 22:00-02:00 window includes Sunday 01:00.
type Window struct {
	Days  []time.Weekday `json:"days,omitempty"`
	Start string         `json:"start"`
	End   string         `json:"end"`

	startMin int
	endMin   int
}

// ParseWindow validates and compiles a window.
func ParseWindow(days []time.Weekday, start, end string) (Window, error) {
	w := Window{Days: append([]time.Weekday(nil), days...), Start: start, End: end}
	if err := w.compile(); err != nil {
		return Window{}, err
	}
	return w, nil
}

// Validate compiles a window in place and reports malformed values.
func (w *Window) Validate() error { return w.compile() }

func (w *Window) compile() error {
	start, err := parseClock(w.Start)
	if err != nil {
		return fmt.Errorf("%w: window start: %v", ErrInvalidPolicy, err)
	}
	end, err := parseClock(w.End)
	if err != nil {
		return fmt.Errorf("%w: window end: %v", ErrInvalidPolicy, err)
	}
	if start == end {
		return fmt.Errorf("%w: window start and end are identical (%s); an empty window cannot express a schedule", ErrInvalidPolicy, w.Start)
	}
	for _, day := range w.Days {
		if day < time.Sunday || day > time.Saturday {
			return fmt.Errorf("%w: window day %d is not a weekday", ErrInvalidPolicy, day)
		}
	}
	w.startMin, w.endMin = start, end
	return nil
}

func parseClock(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	hourPart, minutePart, ok := strings.Cut(raw, ":")
	if !ok || len(hourPart) == 0 || len(minutePart) != 2 {
		return 0, fmt.Errorf("time %q is not HH:MM", raw)
	}
	hour, err := strconv.Atoi(hourPart)
	if err != nil || hour < 0 || hour > 24 {
		return 0, fmt.Errorf("time %q has an invalid hour", raw)
	}
	minute, err := strconv.Atoi(minutePart)
	if err != nil || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("time %q has an invalid minute", raw)
	}
	if hour == 24 {
		if minute != 0 {
			return 0, fmt.Errorf("time %q is outside 00:00-24:00", raw)
		}
		return 24 * 60, nil
	}
	return hour*60 + minute, nil
}

func (w Window) allowsDay(day time.Weekday) bool {
	if len(w.Days) == 0 {
		return true
	}
	for _, d := range w.Days {
		if d == day {
			return true
		}
	}
	return false
}

// Contains reports whether t (converted to loc) falls inside the window. An
// uncompiled window is compiled on first use; a malformed one contains nothing.
func (w Window) Contains(t time.Time, loc *time.Location) bool {
	if w.startMin == 0 && w.endMin == 0 {
		if err := w.compile(); err != nil {
			return false
		}
	}
	if loc == nil {
		loc = time.Local
	}
	local := t.In(loc)
	minute := local.Hour()*60 + local.Minute()
	if w.startMin < w.endMin {
		return w.allowsDay(local.Weekday()) && minute >= w.startMin && minute < w.endMin
	}
	// Crosses midnight: before-end belongs to the previous day's window.
	if minute >= w.startMin {
		return w.allowsDay(local.Weekday())
	}
	if minute < w.endMin {
		return w.allowsDay(local.AddDate(0, 0, -1).Weekday())
	}
	return false
}

// nextOpen returns the next instant strictly after t at which the window opens.
func (w Window) nextOpen(t time.Time, loc *time.Location) (time.Time, bool) {
	if w.startMin == 0 && w.endMin == 0 {
		if err := w.compile(); err != nil {
			return time.Time{}, false
		}
	}
	if loc == nil {
		loc = time.Local
	}
	local := t.In(loc)
	for offset := 0; offset <= 8; offset++ {
		day := local.AddDate(0, 0, offset)
		if !w.allowsDay(day.Weekday()) {
			continue
		}
		candidate := time.Date(day.Year(), day.Month(), day.Day(), w.startMin/60, w.startMin%60, 0, 0, loc)
		if candidate.After(local) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

// Policy is the scheduler's configuration. Every zero value has a conservative
// default except the limits, where zero means "no limit": a silent cap would be
// indistinguishable from a bug when work stops.
type Policy struct {
	Mode Mode

	// Windows gates idle and balanced modes. An empty set means the mode is
	// always open; configure a window to actually defer work to off-hours.
	Windows []Window
	// Location is the timezone windows are evaluated in. Nil means local time.
	Location *time.Location

	// RequireIdle additionally requires an idle host observation in any mode.
	RequireIdle bool
	// IdleLoadBelow is the highest 1-minute load average still considered idle.
	IdleLoadBelow float64
	// IdleRAMFreeBytes is the minimum available RAM still considered idle.
	IdleRAMFreeBytes uint64

	// MaxSecondsPerHour caps estimated/actual seconds started in a rolling hour.
	MaxSecondsPerHour float64
	// MaxModelLoadsPerHour caps cold model loads in a rolling hour.
	MaxModelLoadsPerHour int
	// MaxJobsPerHour caps jobs started in a rolling hour.
	MaxJobsPerHour int

	// ModelDwell is how long a model stays resident after its last use.
	ModelDwell time.Duration

	// MaxSharePerOwner caps one owner's share of the seconds started in the
	// rolling budget window. Zero means unlimited; otherwise (0, 1].
	MaxSharePerOwner float64

	// DefaultMaxBatch bounds a batch when a job does not set MaxBatch.
	DefaultMaxBatch int
	// DeferBase and DeferCap shape exponential deferral backoff.
	DeferBase time.Duration
	DeferCap  time.Duration
	// LeaseTTL bounds how long a leased job survives a crashed runner.
	LeaseTTL time.Duration
	// Interval is the cadence used by Scheduler.Run.
	Interval time.Duration
}

// Normalized validates the policy and fills defaults.
func (p Policy) Normalized() (Policy, error) {
	if p.Mode == "" {
		p.Mode = ModeBalanced
	}
	if !p.Mode.Valid() {
		return Policy{}, fmt.Errorf("%w: unknown mode %q", ErrInvalidPolicy, p.Mode)
	}
	if p.Location == nil {
		p.Location = time.Local
	}
	if p.ModelDwell < 0 {
		return Policy{}, fmt.Errorf("%w: model dwell cannot be negative", ErrInvalidPolicy)
	}
	if p.ModelDwell == 0 {
		p.ModelDwell = DefaultModelDwell
	}
	if p.MaxSecondsPerHour < 0 {
		return Policy{}, fmt.Errorf("%w: max seconds per hour cannot be negative", ErrInvalidPolicy)
	}
	if p.MaxModelLoadsPerHour < 0 || p.MaxJobsPerHour < 0 {
		return Policy{}, fmt.Errorf("%w: max loads/jobs per hour cannot be negative", ErrInvalidPolicy)
	}
	if p.MaxSharePerOwner < 0 || p.MaxSharePerOwner > 1 {
		return Policy{}, fmt.Errorf("%w: max owner share must be in (0, 1] or zero for unlimited", ErrInvalidPolicy)
	}
	if p.IdleLoadBelow < 0 || p.IdleRAMFreeBytes < 0 {
		return Policy{}, fmt.Errorf("%w: idle thresholds cannot be negative", ErrInvalidPolicy)
	}
	if p.DefaultMaxBatch <= 0 {
		p.DefaultMaxBatch = DefaultMaxBatch
	}
	if p.DeferBase <= 0 {
		p.DeferBase = DefaultDeferBase
	}
	if p.DeferCap <= 0 {
		p.DeferCap = DefaultDeferCap
	}
	if p.DeferCap < p.DeferBase {
		p.DeferCap = p.DeferBase
	}
	if p.LeaseTTL <= 0 {
		p.LeaseTTL = DefaultLeaseTTL
	}
	if p.Interval <= 0 {
		p.Interval = DefaultInterval
	}
	windows := make([]Window, len(p.Windows))
	for i, window := range p.Windows {
		compiled := Window{Days: append([]time.Weekday(nil), window.Days...), Start: window.Start, End: window.End}
		if err := compiled.Validate(); err != nil {
			return Policy{}, err
		}
		windows[i] = compiled
	}
	p.Windows = windows
	return p, nil
}

// windowsOpen reports whether at is inside any configured window. An empty
// window set is open all day.
func (p Policy) windowsOpen(at time.Time) bool {
	if len(p.Windows) == 0 {
		return true
	}
	for _, window := range p.Windows {
		if window.Contains(at, p.Location) {
			return true
		}
	}
	return false
}

// nextWindowOpen returns the next opening after at across all windows.
func (p Policy) nextWindowOpen(at time.Time) *time.Time {
	if len(p.Windows) == 0 {
		return nil
	}
	var best time.Time
	for _, window := range p.Windows {
		if next, ok := window.nextOpen(at, p.Location); ok && (best.IsZero() || next.Before(best)) {
			best = next
		}
	}
	if best.IsZero() {
		return nil
	}
	return &best
}

// idleObservation reports whether an observation satisfies the idle thresholds.
// An observation without constraints is idle; a nil observation never is.
func (p Policy) idleObservation(state *HostState) bool {
	if state == nil {
		return false
	}
	if p.IdleLoadBelow > 0 && state.Load1 > p.IdleLoadBelow {
		return false
	}
	if p.IdleRAMFreeBytes > 0 && state.AvailableRAMBytes < p.IdleRAMFreeBytes {
		return false
	}
	return true
}

// maxBatchFor returns the batch limit for a group: the smallest positive
// MaxBatch named by its jobs, bounded by the policy default. A job asking for a
// larger batch than the policy allows is clamped, never widened.
func (p Policy) maxBatchFor(jobs []Job) int {
	limit := p.DefaultMaxBatch
	for _, job := range jobs {
		if job.MaxBatch > 0 && job.MaxBatch < limit {
			limit = job.MaxBatch
		}
	}
	if limit <= 0 {
		limit = 1
	}
	return limit
}

// backoff is the deferral delay after the given number of deferrals. It grows
// exponentially from DeferBase and is clamped at DeferCap. There is no jitter:
// a deterministic delay is easier to reason about, and a thundering herd is not
// a concern in a single-host scheduler.
func (p Policy) backoff(deferrals int) time.Duration {
	if deferrals < 1 {
		deferrals = 1
	}
	delay := p.DeferBase
	for i := 1; i < deferrals; i++ {
		delay *= 2
		if delay >= p.DeferCap {
			return p.DeferCap
		}
	}
	if delay > p.DeferCap {
		delay = p.DeferCap
	}
	return delay
}

// ErrInvalidPolicy marks a malformed scheduler policy.
var ErrInvalidPolicy = fmt.Errorf("schedule: invalid policy")
